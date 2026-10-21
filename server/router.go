package server

import (
	"time"

	cache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/handlers"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/handlers/allowednamespaces"
)

const (
	HealthCacheTime = 30 * time.Second
	APICacheTime    = 15 * time.Second
)

// NewRouter returns the HTTP router used by the proxy process.
func NewRouter() *gin.Engine {
	router := gin.Default()

	memoryStore := persist.NewMemoryStore(1 * time.Minute)

	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("/health", cache.CacheByRequestURI(memoryStore, HealthCacheTime), handlers.HealthHandler)
	router.POST("/apis/allowednamespaces", allowednamespaces.CacheMiddleware(memoryStore, APICacheTime), allowednamespaces.Handler)
	router.GET("/apis/*path", cache.CacheByRequestURI(memoryStore, APICacheTime), handlers.RequestHandler)

	return router
}
