package allowednamespaces

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	gincache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-gonic/gin"
)

const (
	cacheKeyPrefix      = "allowednamespaces:"
	maxRequestBodyBytes = 64 << 10
)

// CacheMiddleware caches successful responses keyed by forwarded auth headers and request body.
func CacheMiddleware(store persist.CacheStore, ttl time.Duration) gin.HandlerFunc {
	return gincache.Cache(store, ttl, gincache.WithCacheStrategyByRequest(cacheStrategyByRequest))
}

func cacheStrategyByRequest(c *gin.Context) (bool, gincache.Strategy) {
	bodyBytes, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes))
	if err != nil {
		return false, gincache.Strategy{}
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	digest := sha256.New()
	digest.Write([]byte(c.GetHeader("Authorization")))
	digest.Write([]byte{0})
	writeForwardedHeadersDigest(digest, c.Request)
	digest.Write(bodyBytes)

	return true, gincache.Strategy{
		CacheKey: cacheKeyPrefix + hex.EncodeToString(digest.Sum(nil)),
	}
}

func writeForwardedHeadersDigest(digest hash.Hash, req *http.Request) {
	impersonationKeys := make([]string, 0)
	for name := range req.Header {
		if strings.HasPrefix(strings.ToUpper(name), "IMPERSONATE-") {
			impersonationKeys = append(impersonationKeys, name)
		}
	}
	sort.Strings(impersonationKeys)

	for _, name := range impersonationKeys {
		values := append([]string(nil), req.Header.Values(name)...)
		sort.Strings(values)
		digest.Write([]byte(strings.ToLower(name)))
		digest.Write([]byte{0})
		digest.Write([]byte(strings.Join(values, "\x00")))
		digest.Write([]byte{0})
	}
}
