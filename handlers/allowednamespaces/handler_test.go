package allowednamespaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-gonic/gin"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/handlers"
)

func TestMain(m *testing.M) {
	os.Setenv("KUBE_TLS_INSECURE", "true")
	os.Exit(m.Run())
}

func TestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"default"}},{"metadata":{"name":"other"}}]}`))
		case r.Method == http.MethodPost && r.URL.Path == ssarAPIPath:
			var ssar selfSubjectAccessReview
			if err := json.NewDecoder(r.Body).Decode(&ssar); err != nil {
				t.Fatalf("failed to decode SSAR request: %v", err)
			}

			namespace := ssar.Spec.ResourceAttributes.Namespace
			allowed := namespace == "default"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":{"allowed":` + boolString(allowed) + `}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer apiServer.Close()

	originalAPIServerURL := handlers.APIServerURL
	handlers.APIServerURL = strings.TrimPrefix(apiServer.URL, "https://")
	t.Cleanup(func() {
		handlers.APIServerURL = originalAPIServerURL
	})

	router := gin.New()
	router.POST("/apis/allowednamespaces", Handler)

	body := `{
		"resourceAttributes":{
			"verb":"get",
			"group":"kubevirt.io",
			"resource":"virtualmachines"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/apis/allowednamespaces", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response AllowedNamespacesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.AllowedNamespaces) != 1 || response.AllowedNamespaces[0] != "default" {
		t.Fatalf("expected allowedNamespaces [default], got %v", response.AllowedNamespaces)
	}

	if len(response.Failures) != 0 {
		t.Fatalf("expected no failures, got %v", response.Failures)
	}
}

func TestHandlerRetriesFailedSSARAndSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var attemptsMu sync.Mutex
	ssarAttempts := map[string]int{}

	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"flaky"}},{"metadata":{"name":"stable"}}]}`))
		case r.Method == http.MethodPost && r.URL.Path == ssarAPIPath:
			var ssar selfSubjectAccessReview
			if err := json.NewDecoder(r.Body).Decode(&ssar); err != nil {
				t.Errorf("failed to decode SSAR request: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}

			namespace := ssar.Spec.ResourceAttributes.Namespace
			attemptsMu.Lock()
			ssarAttempts[namespace]++
			attempt := ssarAttempts[namespace]
			attemptsMu.Unlock()

			if namespace == "flaky" && attempt == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"message":"temporary unavailable"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":{"allowed":true}}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	originalAPIServerURL := handlers.APIServerURL
	handlers.APIServerURL = strings.TrimPrefix(apiServer.URL, "https://")
	t.Cleanup(func() {
		handlers.APIServerURL = originalAPIServerURL
	})

	router := gin.New()
	router.POST("/apis/allowednamespaces", Handler)

	req := httptest.NewRequest(
		http.MethodPost,
		"/apis/allowednamespaces",
		strings.NewReader(`{"resourceAttributes":{"verb":"get","resource":"pods"}}`),
	)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response AllowedNamespacesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.AllowedNamespaces) != 2 {
		t.Fatalf("expected 2 allowed namespaces after retry, got %v", response.AllowedNamespaces)
	}

	if len(response.Failures) != 0 {
		t.Fatalf("expected no failures, got %v", response.Failures)
	}

	if ssarAttempts["flaky"] != 2 {
		t.Fatalf("expected 2 SSAR attempts for flaky namespace, got %d", ssarAttempts["flaky"])
	}
}

func TestHandlerReportsFailuresAfterRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"broken"}}]}`))
		case r.Method == http.MethodPost && r.URL.Path == ssarAPIPath:
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"forbidden"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer apiServer.Close()

	originalAPIServerURL := handlers.APIServerURL
	handlers.APIServerURL = strings.TrimPrefix(apiServer.URL, "https://")
	t.Cleanup(func() {
		handlers.APIServerURL = originalAPIServerURL
	})

	router := gin.New()
	router.POST("/apis/allowednamespaces", Handler)

	req := httptest.NewRequest(
		http.MethodPost,
		"/apis/allowednamespaces",
		strings.NewReader(`{"resourceAttributes":{"verb":"get","resource":"pods"}}`),
	)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response AllowedNamespacesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response.AllowedNamespaces) != 0 {
		t.Fatalf("expected no allowed namespaces, got %v", response.AllowedNamespaces)
	}

	if len(response.Failures) != 1 {
		t.Fatalf("expected 1 failure, got %v", response.Failures)
	}

	if response.Failures[0].Namespace != "broken" {
		t.Fatalf("expected failure for broken namespace, got %q", response.Failures[0].Namespace)
	}

	if response.Failures[0].Error != "kubernetes API returned status 403: forbidden" {
		t.Fatalf("unexpected failure error: %q", response.Failures[0].Error)
	}
}

func TestHandlerRequiresResourceAttributes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/apis/allowednamespaces", Handler)

	req := httptest.NewRequest(
		http.MethodPost,
		"/apis/allowednamespaces",
		strings.NewReader(`{"resourceAttributes":{"group":"kubevirt.io"}}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandlerCachesSuccessfulResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	namespaceListCalls := 0
	ssarCalls := 0

	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces":
			namespaceListCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"default"}}]}`))
		case r.Method == http.MethodPost && r.URL.Path == ssarAPIPath:
			ssarCalls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":{"allowed":true}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer apiServer.Close()

	originalAPIServerURL := handlers.APIServerURL
	handlers.APIServerURL = strings.TrimPrefix(apiServer.URL, "https://")
	t.Cleanup(func() {
		handlers.APIServerURL = originalAPIServerURL
	})

	store := persist.NewMemoryStore(1 * time.Minute)
	router := gin.New()
	router.POST("/apis/allowednamespaces", CacheMiddleware(store, time.Minute), Handler)

	body := `{"resourceAttributes":{"verb":"get","resource":"pods"}}`
	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/apis/allowednamespaces", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	first := send()
	if first.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, first.Code, first.Body.String())
	}

	second := send()
	if second.Code != http.StatusOK {
		t.Fatalf("expected status %d on cache hit, got %d: %s", http.StatusOK, second.Code, second.Body.String())
	}

	if first.Body.String() != second.Body.String() {
		t.Fatalf("expected identical cached response, got %q and %q", first.Body.String(), second.Body.String())
	}

	if namespaceListCalls != 1 {
		t.Fatalf("expected namespace list to be called once, got %d", namespaceListCalls)
	}

	if ssarCalls != 1 {
		t.Fatalf("expected SSAR to be called once, got %d", ssarCalls)
	}
}

func TestHandlerDoesNotCacheResponsesWithFailures(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ssarCalls := 0

	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"recover"}}]}`))
		case r.Method == http.MethodPost && r.URL.Path == ssarAPIPath:
			ssarCalls++
			if ssarCalls <= 2 {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"message":"temporary unavailable"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":{"allowed":true}}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	originalAPIServerURL := handlers.APIServerURL
	handlers.APIServerURL = strings.TrimPrefix(apiServer.URL, "https://")
	t.Cleanup(func() {
		handlers.APIServerURL = originalAPIServerURL
	})

	store := persist.NewMemoryStore(1 * time.Minute)
	router := gin.New()
	router.POST("/apis/allowednamespaces", CacheMiddleware(store, time.Minute), Handler)

	body := `{"resourceAttributes":{"verb":"get","resource":"pods"}}`
	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/apis/allowednamespaces", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	first := send()
	if first.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, first.Code, first.Body.String())
	}

	var firstResponse AllowedNamespacesResponse
	if err := json.Unmarshal(first.Body.Bytes(), &firstResponse); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}
	if len(firstResponse.Failures) != 1 {
		t.Fatalf("expected one failure on first request, got %v", firstResponse.Failures)
	}

	second := send()
	if second.Code != http.StatusOK {
		t.Fatalf("expected status %d on retry, got %d: %s", http.StatusOK, second.Code, second.Body.String())
	}

	var secondResponse AllowedNamespacesResponse
	if err := json.Unmarshal(second.Body.Bytes(), &secondResponse); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}
	if len(secondResponse.AllowedNamespaces) != 1 || secondResponse.AllowedNamespaces[0] != "recover" {
		t.Fatalf("expected recovered namespace list, got %v", secondResponse.AllowedNamespaces)
	}
	if len(secondResponse.Failures) != 0 {
		t.Fatalf("expected no failures after recovery, got %v", secondResponse.Failures)
	}

	if ssarCalls != 3 {
		t.Fatalf("expected 3 SSAR calls (2 failed attempts + 1 success), got %d", ssarCalls)
	}
}

func TestHandlerCacheSeparatesImpersonationHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ssarCalls := 0

	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"default"}}]}`))
		case r.Method == http.MethodPost && r.URL.Path == ssarAPIPath:
			ssarCalls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"status":{"allowed":true}}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	originalAPIServerURL := handlers.APIServerURL
	handlers.APIServerURL = strings.TrimPrefix(apiServer.URL, "https://")
	t.Cleanup(func() {
		handlers.APIServerURL = originalAPIServerURL
	})

	store := persist.NewMemoryStore(1 * time.Minute)
	router := gin.New()
	router.POST("/apis/allowednamespaces", CacheMiddleware(store, time.Minute), Handler)

	body := `{"resourceAttributes":{"verb":"get","resource":"pods"}}`
	send := func(impersonateUser string) {
		req := httptest.NewRequest(http.MethodPost, "/apis/allowednamespaces", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer shared-token")
		req.Header.Set("Impersonate-User", impersonateUser)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
		}
	}

	send("alice")
	send("bob")

	if ssarCalls != 2 {
		t.Fatalf("expected separate cache entries per impersonated user, got %d SSAR calls", ssarCalls)
	}
}

func TestHandlerRequiresAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/apis/allowednamespaces", Handler)

	req := httptest.NewRequest(
		http.MethodPost,
		"/apis/allowednamespaces",
		strings.NewReader(`{"resourceAttributes":{"verb":"get","resource":"pods"}}`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
