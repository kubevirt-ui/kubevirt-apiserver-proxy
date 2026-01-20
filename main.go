package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	cache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/handlers"
)

const (
	healthCacheTime = 30 * time.Second
	apiCacheTime    = 15 * time.Second
)

var (
	minTLSVersion   uint16
	tlsCipherSuites []uint16

	minTLSVersionFlag   = flag.Uint("tls-min-version", 0, "The minimum TLS version to use")
	tlsCipherSuitesFlag = flag.String("tls-cipher-suites", "", "A comma-separated list of cipher suites to use")
)

func init() {
	flag.Parse()

	if *minTLSVersionFlag > 0 {
		minTLSVersion = uint16(*minTLSVersionFlag)
	}

	if *tlsCipherSuitesFlag != "" {
		ciphers := strings.Split(*tlsCipherSuitesFlag, ",")
		tlsCipherSuites = make([]uint16, 0, len(ciphers))

		for _, cipherStr := range ciphers {
			cipher, err := strconv.ParseUint(cipherStr, 10, 16)
			if err != nil {
				panic(fmt.Errorf("can't parse cipher %q; %w", cipherStr, err))
			}

			tlsCipherSuites = append(tlsCipherSuites, uint16(cipher))
		}
	}
}

func main() {

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

	if minTLSVersion != 0 {
		server.TLSConfig.MinVersion = minTLSVersion
	}

	if len(tlsCipherSuites) > 0 {
		server.TLSConfig.CipherSuites = tlsCipherSuites
	}

	log.Printf("listening for server 8080 - v0.0.10 - API cache time: %v", apiCacheTime)

	var err error
	if os.Getenv("APP_ENV") == "dev" {
		err = server.ListenAndServe()
	} else {
		err = server.ListenAndServeTLS("./cert/tls.crt", "./cert/tls.key")
	}

	if err != nil {
		log.Println("Failed to start server: ", err.Error())
	}
}
