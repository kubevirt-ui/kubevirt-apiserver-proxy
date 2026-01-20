package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	cache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/config"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/handlers"
)

const (
	healthCacheTime = 30 * time.Second
	apiCacheTime    = 15 * time.Second
)

func main() {
	flag.Parse()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	router := gin.Default()

	memoryStore := persist.NewMemoryStore(1 * time.Minute)

	router.Use(gzip.Gzip(gzip.DefaultCompression))

	router.GET("/health", cache.CacheByRequestURI(memoryStore, healthCacheTime), handlers.HealthHandler)
	router.GET("/apis/*path", cache.CacheByRequestURI(memoryStore, apiCacheTime), handlers.RequestHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
		TLSConfig: &tls.Config{
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		},
	}

	if minTLSVer := cfg.GetMinTLSVersion(); minTLSVer != 0 {
		server.TLSConfig.MinVersion = minTLSVer
	}

	if ciphers := cfg.GetTLSCipherSuites(); len(ciphers) > 0 {
		server.TLSConfig.CipherSuites = ciphers
	}

	log.Printf("listening for server 8080 - v0.0.10 - API cache time: %v", apiCacheTime)

	if os.Getenv("APP_ENV") == "dev" {
		err = server.ListenAndServe()
	} else {
		err = server.ListenAndServeTLS("./cert/tls.crt", "./cert/tls.key")
	}

	if err != nil {
		log.Println("Failed to start server: ", err.Error())
	}
}
