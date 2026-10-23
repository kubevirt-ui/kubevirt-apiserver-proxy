package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildHealthResponse_DevModeReady(t *testing.T) {
	t.Setenv("APP_ENV", "dev")

	resp := BuildHealthResponse()

	if !resp.Ready {
		t.Fatal("expected ready in dev mode")
	}
	if resp.Status != "ok" {
		t.Fatalf("expected status ok, got %q", resp.Status)
	}
	if resp.ServingTLS.Required {
		t.Fatal("expected serving TLS not required in dev")
	}
	if resp.Kubernetes.ServiceAccountCAPath != defaultServiceAccountCAPath {
		t.Fatalf("unexpected CA path: %q", resp.Kubernetes.ServiceAccountCAPath)
	}
}

func TestBuildHealthResponse_ProductionNotReady(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("SERVING_TLS_CERT_PATH", filepath.Join(t.TempDir(), "missing.crt"))
	t.Setenv("SERVING_TLS_KEY_PATH", filepath.Join(t.TempDir(), "missing.key"))

	resp := BuildHealthResponse()

	if resp.Ready {
		t.Fatal("expected not ready without serving certs")
	}
	if !resp.ServingTLS.Required {
		t.Fatal("expected serving TLS required in production")
	}
	if resp.ServingTLS.CertLoaded || resp.ServingTLS.KeyLoaded {
		t.Fatal("expected cert and key not loaded")
	}
}

func TestBuildHealthResponse_ProductionReady(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	dir := t.TempDir()
	certPath := filepath.Join(dir, "tls.crt")
	keyPath := filepath.Join(dir, "tls.key")
	if err := os.WriteFile(certPath, []byte("test-cert"), 0o644); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("test-key"), 0o644); err != nil {
		t.Fatalf("write key: %v", err)
	}
	t.Setenv("SERVING_TLS_CERT_PATH", certPath)
	t.Setenv("SERVING_TLS_KEY_PATH", keyPath)

	resp := BuildHealthResponse()

	if !resp.Ready {
		t.Fatal("expected ready when serving cert and key exist")
	}
	if !resp.ServingTLS.CertLoaded || !resp.ServingTLS.KeyLoaded {
		t.Fatal("expected cert and key loaded")
	}
}

func TestHealthHandler_StatusCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		setup      func(t *testing.T)
		wantStatus int
		wantReady  bool
	}{
		{
			name: "dev ready",
			setup: func(t *testing.T) {
				t.Setenv("APP_ENV", "dev")
			},
			wantStatus: http.StatusOK,
			wantReady:  true,
		},
		{
			name: "production not ready",
			setup: func(t *testing.T) {
				t.Setenv("APP_ENV", "production")
				dir := t.TempDir()
				t.Setenv("SERVING_TLS_CERT_PATH", filepath.Join(dir, "missing.crt"))
				t.Setenv("SERVING_TLS_KEY_PATH", filepath.Join(dir, "missing.key"))
			},
			wantStatus: http.StatusServiceUnavailable,
			wantReady:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

			HealthHandler(c)

			if w.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}

			var body HealthResponse
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Ready != tc.wantReady {
				t.Fatalf("expected ready %v, got %v", tc.wantReady, body.Ready)
			}
		})
	}
}
