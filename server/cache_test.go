package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIProxyCacheStrategy_DifferentiatesAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	reqA := httptest.NewRequest(http.MethodGet, "/apis/kubevirt.io/v1/virtualmachines", nil)
	reqA.Header.Set("Authorization", "Bearer token-a")
	ctxA, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctxA.Request = reqA

	reqB := httptest.NewRequest(http.MethodGet, "/apis/kubevirt.io/v1/virtualmachines", nil)
	reqB.Header.Set("Authorization", "Bearer token-b")
	ctxB, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctxB.Request = reqB

	_, strategyA := apiProxyCacheStrategy(ctxA)
	_, strategyB := apiProxyCacheStrategy(ctxB)

	if strategyA.CacheKey == strategyB.CacheKey {
		t.Fatalf("expected different cache keys for different bearer tokens, got %q", strategyA.CacheKey)
	}
}
