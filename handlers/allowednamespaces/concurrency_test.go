package allowednamespaces

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolveMaxConcurrencyUsesDefault(t *testing.T) {
	maxConcurrency, err := resolveMaxConcurrency(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if maxConcurrency != defaultMaxConcurrentSSARChecks {
		t.Fatalf("expected default %d, got %d", defaultMaxConcurrentSSARChecks, maxConcurrency)
	}
}

func TestResolveMaxConcurrencyUsesRequestBody(t *testing.T) {
	value := 5
	maxConcurrency, err := resolveMaxConcurrency(&value)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if maxConcurrency != 5 {
		t.Fatalf("expected 5, got %d", maxConcurrency)
	}
}

func TestResolveMaxConcurrencyClampsToCap(t *testing.T) {
	value := 9999
	maxConcurrency, err := resolveMaxConcurrency(&value)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if maxConcurrency != maxConcurrentSSARChecksCap {
		t.Fatalf("expected cap %d, got %d", maxConcurrentSSARChecksCap, maxConcurrency)
	}
}

func TestResolveMaxConcurrencyRejectsInvalidValue(t *testing.T) {
	value := 0
	_, err := resolveMaxConcurrency(&value)
	if err == nil {
		t.Fatal("expected error for invalid maxConcurrency")
	}
}

func TestHandlerRejectsInvalidMaxConcurrencyInBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/apis/allowednamespaces", Handler)

	req := httptest.NewRequest(
		http.MethodPost,
		"/apis/allowednamespaces",
		strings.NewReader(`{"maxConcurrency":0,"resourceAttributes":{"verb":"get","resource":"pods"}}`),
	)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
