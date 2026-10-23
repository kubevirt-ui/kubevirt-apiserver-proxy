package server

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"net/http"
	"sort"
	"strings"
	"time"

	gincache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-gonic/gin"
)

const apiProxyCacheKeyPrefix = "apiproxy:"

func apiProxyCacheMiddleware(store persist.CacheStore, ttl time.Duration) gin.HandlerFunc {
	return gincache.Cache(store, ttl, gincache.WithCacheStrategyByRequest(apiProxyCacheStrategy))
}

func apiProxyCacheStrategy(c *gin.Context) (bool, gincache.Strategy) {
	shouldCache, strategy := gincache.CacheStrategyRequestURI(c)
	if !shouldCache {
		return false, strategy
	}

	digest := sha256.New()
	digest.Write([]byte(strategy.CacheKey))
	digest.Write([]byte{0})
	digest.Write([]byte(c.GetHeader("Authorization")))
	digest.Write([]byte{0})
	writeImpersonationHeadersDigest(digest, c.Request)

	return true, gincache.Strategy{
		CacheKey: apiProxyCacheKeyPrefix + hex.EncodeToString(digest.Sum(nil)),
	}
}

func writeImpersonationHeadersDigest(digest hash.Hash, req *http.Request) {
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
