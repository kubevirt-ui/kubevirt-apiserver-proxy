package allowednamespaces

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	gincache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-gonic/gin"
)

const cacheKeyPrefix = "allowednamespaces:"

// CacheMiddleware caches successful responses keyed by the Authorization header and request body.
func CacheMiddleware(store persist.CacheStore, ttl time.Duration) gin.HandlerFunc {
	return gincache.Cache(store, ttl, gincache.WithCacheStrategyByRequest(cacheStrategyByRequest))
}

func cacheStrategyByRequest(c *gin.Context) (bool, gincache.Strategy) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false, gincache.Strategy{}
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	digest := sha256.New()
	digest.Write([]byte(c.GetHeader("Authorization")))
	digest.Write(bodyBytes)

	return true, gincache.Strategy{
		CacheKey: cacheKeyPrefix + hex.EncodeToString(digest.Sum(nil)),
	}
}
