package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestK8sTLSConfigUsesInClusterCA(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(caPath, generateTestCAPEM(t), 0o600); err != nil {
		t.Fatalf("failed to write test CA: %v", err)
	}

	t.Setenv("KUBE_TLS_INSECURE", "")
	t.Setenv("APP_ENV", "dev")

	cfg := k8sTLSConfigFromPath(caPath)
	if cfg.InsecureSkipVerify {
		t.Fatal("expected certificate verification when a valid CA is provided")
	}
	if cfg.RootCAs == nil {
		t.Fatal("expected RootCAs to be set from the service account CA file")
	}
}

func TestK8sTLSConfigInvalidCAPresentDoesNotUseDevInsecure(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(caPath, []byte("not a pem certificate"), 0o600); err != nil {
		t.Fatalf("failed to write invalid CA: %v", err)
	}

	t.Setenv("KUBE_TLS_INSECURE", "")
	t.Setenv("APP_ENV", "dev")

	cfg := k8sTLSConfigFromPath(caPath)
	if cfg.InsecureSkipVerify {
		t.Fatal("APP_ENV=dev must not bypass TLS when an in-cluster CA file is present but invalid")
	}
}

func TestK8sTLSConfigInvalidCAAllowsExplicitInsecure(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.crt")
	if err := os.WriteFile(caPath, []byte("not a pem certificate"), 0o600); err != nil {
		t.Fatalf("failed to write invalid CA: %v", err)
	}

	t.Setenv("KUBE_TLS_INSECURE", "true")
	t.Setenv("APP_ENV", "")

	cfg := k8sTLSConfigFromPath(caPath)
	if !cfg.InsecureSkipVerify {
		t.Fatal("expected KUBE_TLS_INSECURE to allow insecure TLS when CA is invalid")
	}
}

func generateTestCAPEM(t *testing.T) []byte {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-root"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create test certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
}
