package handlers

import (
	"crypto/x509"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const (
	defaultServingCertPath = "./cert/tls.crt"
	defaultServingKeyPath  = "./cert/tls.key"
)

// HealthResponse describes proxy liveness and whether required TLS material is present.
type HealthResponse struct {
	Status string `json:"status"`
	Ready  bool   `json:"ready"`

	AppEnv        string `json:"appEnv"`
	KubeAPIServer string `json:"kubeAPIServer"`

	ServingTLS ServingTLSHealth `json:"servingTLS"`
	Kubernetes KubernetesHealth `json:"kubernetes"`
}

type ServingTLSHealth struct {
	Required   bool `json:"required"`
	CertPath   string `json:"certPath"`
	KeyPath    string `json:"keyPath"`
	CertLoaded bool `json:"certLoaded"`
	KeyLoaded  bool `json:"keyLoaded"`
}

type KubernetesHealth struct {
	ServiceAccountCAPath    string `json:"serviceAccountCAPath"`
	ServiceAccountCAPresent bool   `json:"serviceAccountCAPresent"`
	ServiceAccountCAValid   bool   `json:"serviceAccountCAValid"`
}

// ServingCertPathForListen returns the TLS certificate path used when starting the HTTPS server.
func ServingCertPathForListen() string {
	return servingCertPath()
}

// ServingKeyPathForListen returns the TLS private key path used when starting the HTTPS server.
func ServingKeyPathForListen() string {
	return servingKeyPath()
}

func servingCertPath() string {
	if path := os.Getenv("SERVING_TLS_CERT_PATH"); path != "" {
		return path
	}
	return defaultServingCertPath
}

func servingKeyPath() string {
	if path := os.Getenv("SERVING_TLS_KEY_PATH"); path != "" {
		return path
	}
	return defaultServingKeyPath
}

func isDevMode() bool {
	return os.Getenv("APP_ENV") == "dev"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func serviceAccountCAValid(path string) bool {
	caPEM, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	pool := x509.NewCertPool()
	return pool.AppendCertsFromPEM(caPEM)
}

// BuildHealthResponse collects runtime TLS and Kubernetes mount status for /health.
func BuildHealthResponse() HealthResponse {
	certPath := servingCertPath()
	keyPath := servingKeyPath()
	caPath := serviceAccountCAPath()

	certLoaded := fileExists(certPath)
	keyLoaded := fileExists(keyPath)
	caPresent := fileExists(caPath)
	caValid := caPresent && serviceAccountCAValid(caPath)

	servingRequired := !isDevMode()
	ready := true
	if servingRequired {
		ready = certLoaded && keyLoaded
	}

	return HealthResponse{
		Status:        "ok",
		Ready:         ready,
		AppEnv:        os.Getenv("APP_ENV"),
		KubeAPIServer: APIServerURL,
		ServingTLS: ServingTLSHealth{
			Required:   servingRequired,
			CertPath:   certPath,
			KeyPath:    keyPath,
			CertLoaded: certLoaded,
			KeyLoaded:  keyLoaded,
		},
		Kubernetes: KubernetesHealth{
			ServiceAccountCAPath:    caPath,
			ServiceAccountCAPresent: caPresent,
			ServiceAccountCAValid:   caValid,
		},
	}
}

func HealthHandler(c *gin.Context) {
	defer c.Request.Body.Close()

	response := BuildHealthResponse()
	if response.Ready {
		c.JSON(http.StatusOK, response)
		return
	}

	c.JSON(http.StatusServiceUnavailable, response)
}
