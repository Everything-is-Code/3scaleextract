package stats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxRetries  = 5
	initialBackoff     = time.Second
	maxBackoff         = 32 * time.Second
	defaultConcurrency = 4
)

var (
	ErrUnrecoverable = errors.New("unrecoverable Analytics API error")
	ErrRateLimited   = errors.New("Analytics API rate limit exceeded after retries")
)

type UsageQuery struct {
	Since       string
	Until       string
	Granularity string
	MetricName  string
}

type Client interface {
	GetUsage(ctx context.Context, serviceID int, q UsageQuery) (json.RawMessage, error)
}

type HTTPClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	semaphore  chan struct{}
	maxRetries int
}

type Options struct {
	BaseURL       string
	Token         string
	HTTPClient    *http.Client
	MaxConcurrent int
	MaxRetries    int
}

func DeriveStatsBaseURL(adminURL string) (string, error) {
	normalized, err := normalizeAdminURL(adminURL)
	if err != nil {
		return "", err
	}
	base := strings.TrimSuffix(normalized, "/admin/api")
	base = strings.TrimRight(base, "/")
	if base == "" {
		return "", fmt.Errorf("invalid admin URL for stats derivation: %q", adminURL)
	}
	return base + "/stats", nil
}

func normalizeAdminURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return "", errors.New("admin URL is required")
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "", fmt.Errorf("admin URL must start with http:// or https://: %q", raw)
	}
	return raw, nil
}

func NewClient(opts Options) (*HTTPClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("stats base URL is required")
	}
	if strings.TrimSpace(opts.Token) == "" {
		return nil, errors.New("access token is required")
	}

	maxConcurrent := opts.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = defaultConcurrency
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	return &HTTPClient{
		baseURL:    baseURL,
		token:      opts.Token,
		httpClient: httpClient,
		semaphore:  make(chan struct{}, maxConcurrent),
		maxRetries: maxRetries,
	}, nil
}

func (c *HTTPClient) GetUsage(ctx context.Context, serviceID int, q UsageQuery) (json.RawMessage, error) {
	path := fmt.Sprintf("/services/%d/usage.json", serviceID)
	return c.doGet(ctx, path, q)
}

func (c *HTTPClient) doGet(ctx context.Context, path string, q UsageQuery) (json.RawMessage, error) {
	if err := c.acquire(ctx); err != nil {
		return nil, err
	}
	defer c.release()

	reqURL, err := c.buildURL(path, q)
	if err != nil {
		return nil, err
	}

	var lastErr error
	backoff := initialBackoff
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, backoff); err != nil {
				return nil, err
			}
			backoff = minDuration(backoff*2, maxBackoff)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			lastErr = fmt.Errorf("%w: HTTP 429", ErrRateLimited)
			continue
		case resp.StatusCode >= 500:
			lastErr = fmt.Errorf("server error HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
			continue
		case resp.StatusCode == http.StatusForbidden:
			return nil, fmt.Errorf("%w: HTTP 403: Analytics API access denied — verify Enterprise tier and PAT Analytics scope: %s",
				ErrUnrecoverable, truncate(string(body), 200))
		case resp.StatusCode >= 400:
			return nil, fmt.Errorf("%w: HTTP %d: %s", ErrUnrecoverable, resp.StatusCode, truncate(string(body), 200))
		}

		return json.RawMessage(body), nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnrecoverable, lastErr)
	}
	return nil, ErrUnrecoverable
}

func (c *HTTPClient) buildURL(path string, q UsageQuery) (string, error) {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return "", err
	}
	values := u.Query()
	values.Set("access_token", c.token)
	if q.Since != "" {
		values.Set("since", q.Since)
	}
	if q.Until != "" {
		values.Set("until", q.Until)
	}
	if q.Granularity != "" {
		values.Set("granularity", q.Granularity)
	}
	if q.MetricName != "" {
		values.Set("metric_name", q.MetricName)
	}
	u.RawQuery = values.Encode()
	return u.String(), nil
}

func (c *HTTPClient) acquire(ctx context.Context) error {
	select {
	case c.semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *HTTPClient) release() {
	<-c.semaphore
}

func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var _ Client = (*HTTPClient)(nil)

// ParseServiceID extracts service_id from a raw Admin API /services list item.
func ParseServiceID(raw json.RawMessage) (int, string, error) {
	var envelope struct {
		Service struct {
			ID         int    `json:"id"`
			SystemName string `json:"system_name"`
		} `json:"service"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return 0, "", err
	}
	if envelope.Service.ID == 0 {
		return 0, "", fmt.Errorf("service id missing")
	}
	if strings.TrimSpace(envelope.Service.SystemName) == "" {
		return 0, "", fmt.Errorf("service system_name missing")
	}
	return envelope.Service.ID, envelope.Service.SystemName, nil
}

// FormatUsagePath returns the relative stats API path for tests and logging.
func FormatUsagePath(serviceID int) string {
	return "/services/" + strconv.Itoa(serviceID) + "/usage.json"
}
