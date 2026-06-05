package admin

import (
	"bytes"
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
	defaultPerPage     = 500
	maxPerPage         = 500
	defaultMaxRetries  = 5
	initialBackoff     = time.Second
	maxBackoff         = 32 * time.Second
	defaultConcurrency = 4
)

var (
	ErrUnrecoverable = errors.New("unrecoverable Admin API error")
	ErrRateLimited   = errors.New("Admin API rate limit exceeded after retries")
)

type Client interface {
	Get(ctx context.Context, path string, dst any) error
	GetAllPages(ctx context.Context, path string, perPage int, fn func([]json.RawMessage) error) error
	PostForm(ctx context.Context, path string, form url.Values, dst any) error
	PutForm(ctx context.Context, path string, form url.Values, dst any) error
	PutJSON(ctx context.Context, path string, payload any, dst any) error
	Delete(ctx context.Context, path string) error
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

func NewClient(opts Options) (*HTTPClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("base URL is required")
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

func (c *HTTPClient) Get(ctx context.Context, path string, dst any) error {
	body, err := c.doRequest(ctx, path, 0, 0)
	if err != nil {
		return err
	}
	if dst == nil {
		return nil
	}
	return json.Unmarshal(body, dst)
}

func (c *HTTPClient) GetAllPages(ctx context.Context, path string, perPage int, fn func([]json.RawMessage) error) error {
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	for page := 1; ; page++ {
		body, err := c.doRequest(ctx, path, page, perPage)
		if err != nil {
			return err
		}

		items, err := extractPageItems(body)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		if err := fn(items); err != nil {
			return err
		}
		if len(items) < perPage {
			return nil
		}
	}
}

func (c *HTTPClient) PutForm(ctx context.Context, path string, form url.Values, dst any) error {
	return c.writeForm(ctx, http.MethodPut, path, form, dst)
}

func (c *HTTPClient) PostForm(ctx context.Context, path string, form url.Values, dst any) error {
	return c.writeForm(ctx, http.MethodPost, path, form, dst)
}

func (c *HTTPClient) writeForm(ctx context.Context, method, path string, form url.Values, dst any) error {
	if form == nil {
		form = url.Values{}
	}
	payload := []byte(form.Encode())
	body, err := c.doWrite(ctx, method, path, payload, "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	if dst == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, dst)
}

func (c *HTTPClient) doWrite(ctx context.Context, method, path string, payload []byte, contentType string) ([]byte, error) {
	if err := c.acquire(ctx); err != nil {
		return nil, err
	}
	defer c.release()

	reqURL, err := c.buildURL(path, 0, 0)
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

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
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
			lastErr = fmt.Errorf("server error HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 200))
			continue
		case resp.StatusCode >= 400:
			return nil, fmt.Errorf("%w: HTTP %d: %s", ErrUnrecoverable, resp.StatusCode, truncate(string(respBody), 200))
		}

		return respBody, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnrecoverable, lastErr)
	}
	return nil, ErrUnrecoverable
}

func (c *HTTPClient) PutJSON(ctx context.Context, path string, payload any, dst any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	body, err := c.doWrite(ctx, http.MethodPut, path, data, "application/json")
	if err != nil {
		return err
	}
	if dst == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, dst)
}

func (c *HTTPClient) Delete(ctx context.Context, path string) error {
	_, err := c.doWrite(ctx, http.MethodDelete, path, nil, "")
	return err
}

func (c *HTTPClient) doRequest(ctx context.Context, path string, page, perPage int) ([]byte, error) {
	if err := c.acquire(ctx); err != nil {
		return nil, err
	}
	defer c.release()

	reqURL, err := c.buildURL(path, page, perPage)
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
		case resp.StatusCode >= 400:
			return nil, fmt.Errorf("%w: HTTP %d: %s", ErrUnrecoverable, resp.StatusCode, truncate(string(body), 200))
		}

		return body, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnrecoverable, lastErr)
	}
	return nil, ErrUnrecoverable
}

func (c *HTTPClient) buildURL(path string, page, perPage int) (string, error) {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, ".json") {
		path = strings.TrimSuffix(path, "/") + ".json"
	}

	u, err := url.Parse(c.baseURL + "/admin/api" + path)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("access_token", c.token)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
		q.Set("per_page", strconv.Itoa(perPage))
	}
	u.RawQuery = q.Encode()
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

func extractPageItems(body []byte) ([]json.RawMessage, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}

	for _, key := range []string{"applications", "services", "backend_apis", "items"} {
		if raw, ok := root[key]; ok {
			return decodeArray(raw)
		}
	}

	for _, raw := range root {
		items, err := decodeArray(raw)
		if err == nil && len(items) > 0 {
			return items, nil
		}
	}
	return nil, nil
}

func decodeArray(raw json.RawMessage) ([]json.RawMessage, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
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
