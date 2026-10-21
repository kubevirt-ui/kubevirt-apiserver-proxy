package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/config"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/handlers"
	"github.com/kubevirt-ui/kubevirt-apiserver-proxy/server"
)

func main() {
	flag.Parse()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	router := server.NewRouter()

	httpServer := &http.Server{
		Addr:      ":8080",
		Handler:   router,
		TLSConfig: &tls.Config{},
	}

	minTLSVer := cfg.GetMinTLSVersion()
	if minTLSVer != 0 {
		httpServer.TLSConfig.MinVersion = minTLSVer
	}

	if minTLSVer < tls.VersionTLS13 {
		if ciphers := cfg.GetTLSCipherSuites(); len(ciphers) > 0 {
			httpServer.TLSConfig.CipherSuites = ciphers
		}
	}

	httpServer.TLSConfig.CurvePreferences = cfg.GetTLSCurveIDs()

	clientCfg := httpServer.TLSConfig.Clone()
	httpServer.TLSConfig.GetConfigForClient = func(_ *tls.ClientHelloInfo) (*tls.Config, error) {
		return clientCfg.Clone(), nil
	}

	log.Printf("listening for server 8080 - v0.0.10 - API cache time: %v", server.APICacheTime)

	if os.Getenv("APP_ENV") == "dev" {
		err = httpServer.ListenAndServe()
	} else {
		err = httpServer.ListenAndServeTLS(handlers.ServingCertPathForListen(), handlers.ServingKeyPathForListen())
	}

	if err != nil {
		log.Println("Failed to start server: ", err.Error())
	}
}
