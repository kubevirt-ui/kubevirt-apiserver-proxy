package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/handlers"
)

func TestAPIProxyCacheDoesNotShareResponsesAcrossTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("APP_ENV", "dev")
	t.Setenv("SERVICE_ACCOUNT_CA_PATH", filepath.Join(t.TempDir(), "missing-ca.crt"))

	const vmListPath = "/apis/kubevirt.io/v1/namespaces/default/virtualmachines"

	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != vmListPath {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		token := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer ")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"apiVersion":"kubevirt.io/v1",
			"kind":"VirtualMachineList",
			"items":[{"metadata":{"name":"vm-` + token + `"},"status":{"printableStatus":"Running"}}]
		}`))
	}))
	defer apiServer.Close()

	originalAPIServerURL := handlers.APIServerURL
	handlers.APIServerURL = strings.TrimPrefix(apiServer.URL, "https://")
	t.Cleanup(func() {
		handlers.APIServerURL = originalAPIServerURL
	})

	router := NewRouter()

	fetchVMName := func(t *testing.T, token string) string {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, vmListPath, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		items, ok := payload["items"].([]interface{})
		if !ok || len(items) != 1 {
			t.Fatalf("expected one item, got %v", payload["items"])
		}
		first, ok := items[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map item, got %T", items[0])
		}
		metadata, ok := first["metadata"].(map[string]interface{})
		if !ok {
			t.Fatalf("expected metadata map, got %v", first)
		}
		name, ok := metadata["name"].(string)
		if !ok {
			t.Fatalf("expected metadata name string, got %v", metadata["name"])
		}
		return name
	}

	if fetchVMName(t, "alice") != "vm-alice" {
		t.Fatal("unexpected response for alice")
	}
	// Second call for alice should hit cache but still return alice's VM.
	if fetchVMName(t, "alice") != "vm-alice" {
		t.Fatal("unexpected cached response for alice")
	}
	if fetchVMName(t, "bob") != "vm-bob" {
		t.Fatal("expected bob's response, got another caller's cached payload")
	}
}
