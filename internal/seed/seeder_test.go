package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/Everything-is-Code/3scaleextract/internal/admin"
)

type seedCall struct {
	Method string
	Path   string
	Form   url.Values
}

type seedMockClient struct {
	getResponses   map[string]any
	postHandlers   map[string]func(path string, form url.Values, dst any) error
	putHandlers    map[string]func(path string, form url.Values, dst any) error
	deleteHandlers map[string]func(path string) error
	defaultPost    func(path string, form url.Values, dst any) error
	defaultPut     func(path string, form url.Values, dst any) error
	calls          []seedCall
}

func newSeedMock() *seedMockClient {
	return &seedMockClient{
		getResponses:   make(map[string]any),
		postHandlers:   make(map[string]func(path string, form url.Values, dst any) error),
		putHandlers:    make(map[string]func(path string, form url.Values, dst any) error),
		deleteHandlers: make(map[string]func(path string) error),
		defaultPost:    func(string, url.Values, any) error { return nil },
		defaultPut:     func(string, url.Values, any) error { return nil },
	}
}

func (m *seedMockClient) Get(_ context.Context, path string, dst any) error {
	m.calls = append(m.calls, seedCall{Method: "GET", Path: path})
	val, ok := m.getResponses[path]
	if !ok {
		return fmt.Errorf("%w: missing GET %s", admin.ErrUnrecoverable, path)
	}
	return fillJSON(dst, val)
}

func (m *seedMockClient) GetAllPages(_ context.Context, _ string, _ int, fn func([]json.RawMessage) error) error {
	return nil
}

func (m *seedMockClient) PostForm(_ context.Context, path string, form url.Values, dst any) error {
	cloned := url.Values{}
	for k, v := range form {
		cloned[k] = append([]string(nil), v...)
	}
	m.calls = append(m.calls, seedCall{Method: "POST", Path: path, Form: cloned})
	if h, ok := m.postHandlers[path]; ok {
		return h(path, form, dst)
	}
	if m.defaultPost != nil {
		return m.defaultPost(path, form, dst)
	}
	return nil
}

func (m *seedMockClient) PutForm(_ context.Context, path string, form url.Values, dst any) error {
	cloned := url.Values{}
	for k, v := range form {
		cloned[k] = append([]string(nil), v...)
	}
	m.calls = append(m.calls, seedCall{Method: "PUT", Path: path, Form: cloned})
	if h, ok := m.putHandlers[path]; ok {
		return h(path, form, dst)
	}
	if m.defaultPut != nil {
		return m.defaultPut(path, form, dst)
	}
	return nil
}

func (m *seedMockClient) PutJSON(_ context.Context, path string, _ any, _ any) error {
	m.calls = append(m.calls, seedCall{Method: "PUT", Path: path})
	return nil
}

func (m *seedMockClient) Delete(_ context.Context, path string) error {
	m.calls = append(m.calls, seedCall{Method: "DELETE", Path: path})
	if h, ok := m.deleteHandlers[path]; ok {
		return h(path)
	}
	return nil
}

func fillJSON(dst any, val any) error {
	if dst == nil {
		return nil
	}
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func (m *seedMockClient) postCount(path string) int {
	n := 0
	for _, c := range m.calls {
		if c.Method == "POST" && c.Path == path {
			n++
		}
	}
	return n
}

func (m *seedMockClient) deleteCount(pathPrefix string) int {
	n := 0
	for _, c := range m.calls {
		if c.Method == "DELETE" && strings.HasPrefix(c.Path, pathPrefix) {
			n++
		}
	}
	return n
}

func (m *seedMockClient) putFormField(pathSuffix, key, want string) bool {
	for _, c := range m.calls {
		if c.Method != "PUT" || !strings.HasSuffix(c.Path, pathSuffix) {
			continue
		}
		if c.Form.Get(key) == want {
			return true
		}
	}
	return false
}

func (m *seedMockClient) hasPutPath(suffix string) bool {
	for _, c := range m.calls {
		if c.Method == "PUT" && strings.HasSuffix(c.Path, suffix) {
			return true
		}
	}
	return false
}

func mockWithExistingService(m *seedMockClient, systemName string, id int) {
	m.getResponses["/services"] = map[string]any{
		"services": []map[string]any{
			{"service": map[string]any{"id": id, "system_name": systemName}},
		},
	}
}

func mockWithExistingBackend(m *seedMockClient, systemName string, id int) {
	m.getResponses["/backend_apis"] = map[string]any{
		"backend_apis": []map[string]any{
			{"backend_api": map[string]any{"id": id, "system_name": systemName}},
		},
	}
}

func mockProductHappyPath(m *seedMockClient, serviceID, planID int) {
	m.getResponses["/services"] = map[string]any{"services": []any{}}
	metricsPath := fmt.Sprintf("/services/%d/metrics", serviceID)
	m.getResponses[metricsPath] = map[string]any{
		"metrics": []map[string]any{
			{"metric": map[string]any{"id": 1, "system_name": "hits"}},
		},
	}
	m.defaultPost = func(path string, form url.Values, dst any) error {
		switch {
		case path == "/services":
			return fillJSON(dst, map[string]any{"service": map[string]any{"id": serviceID}})
		case strings.HasSuffix(path, "/application_plans"):
			return fillJSON(dst, map[string]any{"application_plan": map[string]any{"id": planID}})
		case strings.HasSuffix(path, "/applications"):
			appName := form.Get("name")
			if appName == "" {
				appName = form.Get("application[name]")
			}
			return fillJSON(dst, map[string]any{
				"application": map[string]any{
					"id":            501,
					"name":          appName,
					"client_id":     "oidc-client-id",
					"client_secret": "oidc-client-secret",
				},
			})
		default:
			return nil
		}
	}
}

func minimalAPIKeyProduct() ProductFixture {
	return ProductFixture{
		SystemName:  "test_api_key",
		Name:        "Test API Key Product",
		Description: "Unit test fixture",
		AuthMode:    "api_key",
		BackendRefs: []string{"test_backend"},
		Plans: []PlanFixture{
			{SystemName: "basic", Name: "Basic Plan"},
		},
		Applications: []ApplicationFixture{
			{Name: "test-app", Plan: "basic"},
		},
	}
}

func minimalOIDCProduct() ProductFixture {
	return ProductFixture{
		SystemName:  "test_oidc",
		Name:        "Test OIDC Product",
		Description: "Unit test fixture",
		AuthMode:    "oidc",
		BackendRefs: []string{"test_backend"},
		OIDC: &OIDCFixture{
			RealmURL:         "https://sso.example.com/auth/realms/seed-demo",
			ZyncClientID:     "zync-admin",
			ZyncClientSecret: "ZyncSecret123",
			StandardFlow:     true,
			ServiceAccounts:  true,
		},
		Plans: []PlanFixture{
			{SystemName: "enterprise", Name: "Enterprise Plan"},
		},
		Applications: []ApplicationFixture{
			{Name: "oidc-app", Plan: "enterprise", RedirectURL: "https://cb.example.com/callback"},
		},
	}
}

func resultWithBackend(name string, id int) *Result {
	return &Result{
		Backends: map[string]int{name: id},
		Services: map[string]int{},
		Plans:    map[string]int{},
	}
}

func TestSeedAPIKeyAuth(t *testing.T) {
	const (
		serviceID = 101
		planID    = 201
		accountID = 42
	)
	mock := newSeedMock()
	mockProductHappyPath(mock, serviceID, planID)

	s := NewSeeder(mock, Options{})
	result := resultWithBackend("test_backend", 10)
	if err := s.seedProduct(context.Background(), minimalAPIKeyProduct(), result, accountID); err != nil {
		t.Fatal(err)
	}
	if !mock.putFormField("/proxy", "auth_user_key", "true") {
		t.Fatal("expected auth_user_key=true on proxy PUT")
	}
	if !mock.putFormField("/proxy", "auth_app_id", "false") {
		t.Fatal("expected auth_app_id=false on proxy PUT")
	}
	if mock.postCount("/accounts/42/applications") != 1 {
		t.Fatalf("applications POST count = %d", mock.postCount("/accounts/42/applications"))
	}
	if len(result.Applications) != 1 || result.Applications[0] != "test-app" {
		t.Fatalf("applications = %v", result.Applications)
	}
}

func TestSeedOIDCRefreshApps(t *testing.T) {
	const (
		serviceID = 101
		planID    = 201
		accountID = 42
	)
	mock := newSeedMock()
	mockProductHappyPath(mock, serviceID, planID)
	appsPath := fmt.Sprintf("/accounts/%d/applications", accountID)
	mock.getResponses[appsPath] = map[string]any{
		"applications": []map[string]any{
			{"application": map[string]any{
				"id": 900, "name": "stale-app", "user_key": "uk-1", "service_id": serviceID,
			}},
		},
	}

	s := NewSeeder(mock, Options{})
	result := resultWithBackend("test_backend", 10)
	if err := s.seedProduct(context.Background(), minimalOIDCProduct(), result, accountID); err != nil {
		t.Fatal(err)
	}
	if mock.deleteCount(fmt.Sprintf("/accounts/%d/applications/", accountID)) != 1 {
		t.Fatalf("delete count = %d", mock.deleteCount(fmt.Sprintf("/accounts/%d/applications/", accountID)))
	}
	if len(result.Applications) != 1 || result.Applications[0] != "oidc-app" {
		t.Fatalf("applications = %v", result.Applications)
	}
}

func TestSeedOIDCProxyConfiguration(t *testing.T) {
	const (
		serviceID = 101
		planID    = 201
		accountID = 42
	)
	mock := newSeedMock()
	mockProductHappyPath(mock, serviceID, planID)
	appsPath := fmt.Sprintf("/accounts/%d/applications", accountID)
	mock.getResponses[appsPath] = map[string]any{"applications": []any{}}

	product := minimalOIDCProduct()
	s := NewSeeder(mock, Options{})
	result := resultWithBackend("test_backend", 10)
	if err := s.seedProduct(context.Background(), product, result, accountID); err != nil {
		t.Fatal(err)
	}
	if !mock.putFormField("/proxy", "oidc_issuer_endpoint", product.OIDC.IssuerEndpoint()) {
		t.Fatal("expected oidc_issuer_endpoint on proxy PUT")
	}
	if !mock.hasPutPath("/proxy/oidc_configuration") {
		t.Fatal("expected PUT to oidc_configuration")
	}
}

func TestSkipExistingService(t *testing.T) {
	const existingID = 55
	mock := newSeedMock()
	mockWithExistingService(mock, "test_api_key", existingID)

	s := NewSeeder(mock, Options{SkipExisting: true})
	id, err := s.ensureService(context.Background(), minimalAPIKeyProduct())
	if err != nil {
		t.Fatal(err)
	}
	if id != existingID {
		t.Fatalf("service id = %d", id)
	}
	if mock.postCount("/services") != 0 {
		t.Fatalf("POST /services count = %d", mock.postCount("/services"))
	}
	if len(s.skipped) != 1 || s.skipped[0] != "service:test_api_key" {
		t.Fatalf("skipped = %v", s.skipped)
	}
}

func TestSkipExistingBackend(t *testing.T) {
	const existingID = 77
	backend := BackendFixture{
		SystemName:      "test_backend",
		Name:            "Test Backend",
		PrivateEndpoint: "https://upstream.example.com",
		Description:     "test",
	}
	mock := newSeedMock()
	mockWithExistingBackend(mock, backend.SystemName, existingID)

	s := NewSeeder(mock, Options{SkipExisting: true})
	id, err := s.ensureBackend(context.Background(), backend)
	if err != nil {
		t.Fatal(err)
	}
	if id != existingID {
		t.Fatalf("backend id = %d", id)
	}
	if mock.postCount("/backend_apis") != 0 {
		t.Fatalf("POST /backend_apis count = %d", mock.postCount("/backend_apis"))
	}
	if len(s.skipped) != 1 || s.skipped[0] != "backend:test_backend" {
		t.Fatalf("skipped = %v", s.skipped)
	}
}

func TestHTTPErrorBackend(t *testing.T) {
	backend := BackendFixture{
		SystemName:      "fail_backend",
		Name:            "Fail Backend",
		PrivateEndpoint: "https://upstream.example.com",
		Description:     "test",
	}
	mock := newSeedMock()
	mock.getResponses["/backend_apis"] = map[string]any{"backend_apis": []any{}}
	mock.postHandlers["/backend_apis"] = func(string, url.Values, any) error {
		return fmt.Errorf("%w: HTTP 403 forbidden", admin.ErrUnrecoverable)
	}

	s := NewSeeder(mock, Options{})
	_, err := s.ensureBackend(context.Background(), backend)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Fatalf("error = %v", err)
	}

	s2 := NewSeeder(mock, Options{})
	_, runErr := s2.Run(context.Background())
	if runErr == nil {
		t.Fatal("expected Run error")
	}
	if !strings.Contains(runErr.Error(), "backend") || !strings.Contains(runErr.Error(), "seed_payments") {
		t.Fatalf("Run error = %v", runErr)
	}
}

func TestHTTPErrorService(t *testing.T) {
	mock := newSeedMock()
	mock.getResponses["/services"] = map[string]any{"services": []any{}}
	mock.postHandlers["/services"] = func(string, url.Values, any) error {
		return fmt.Errorf("%w: HTTP 422 validation failed", admin.ErrUnrecoverable)
	}

	s := NewSeeder(mock, Options{})
	result := resultWithBackend("test_backend", 10)
	if err := s.seedProduct(context.Background(), minimalAPIKeyProduct(), result, 42); err == nil {
		t.Fatal("expected error")
	}

	backendID := 1
	mock2 := newSeedMock()
	mock2.getResponses["/backend_apis"] = map[string]any{"backend_apis": []any{}}
	mock2.getResponses["/accounts"] = map[string]any{"accounts": []any{}}
	mock2.getResponses["/services"] = map[string]any{"services": []any{}}
	mock2.defaultPost = func(path string, _ url.Values, dst any) error {
		switch path {
		case "/backend_apis":
			id := backendID
			backendID++
			return fillJSON(dst, map[string]any{"backend_api": map[string]any{"id": id}})
		case "/signup":
			return fillJSON(dst, map[string]any{"account": map[string]any{"id": 42}})
		case "/services":
			return fmt.Errorf("%w: HTTP 422 validation failed", admin.ErrUnrecoverable)
		default:
			return nil
		}
	}
	s2 := NewSeeder(mock2, Options{})
	_, runErr := s2.Run(context.Background())
	if runErr == nil {
		t.Fatal("expected Run error")
	}
	if !strings.Contains(runErr.Error(), "product") || !strings.Contains(runErr.Error(), "seed_api_key") {
		t.Fatalf("Run error = %v", runErr)
	}
}
