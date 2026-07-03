package btls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestLoadTLSConfigFromCA(t *testing.T) {
	certPEM := testCAPEM(t)
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, certPEM, 0600); err != nil {
		t.Fatalf("write CA file: %v", err)
	}

	cfg, err := LoadTLSConfigFromCA(caFile, true)
	if err != nil {
		t.Fatalf("LoadTLSConfigFromCA returned error: %v", err)
	}
	if cfg.RootCAs == nil {
		t.Fatal("RootCAs is nil")
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion = %v, want %v", cfg.MinVersion, tls.VersionTLS12)
	}
	if !cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify should preserve verifyCertificate=true behavior")
	}
}

func TestLoadTLSConfigFromCARejectsInvalidPEM(t *testing.T) {
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, []byte("not a certificate"), 0600); err != nil {
		t.Fatalf("write CA file: %v", err)
	}

	if _, err := LoadTLSConfigFromCA(caFile, false); err == nil {
		t.Fatal("expected invalid PEM error")
	}
}

func TestIsTargetCAEvent(t *testing.T) {
	watchDir := filepath.Clean("/etc/certs")
	watchBase := "ca.pem"

	tests := []struct {
		name  string
		event fsnotify.Event
		want  bool
	}{
		{name: "target file", event: fsnotify.Event{Name: filepath.Join(watchDir, watchBase)}, want: true},
		{name: "kubernetes symlink", event: fsnotify.Event{Name: filepath.Join(watchDir, "..data")}, want: true},
		{name: "kubernetes temp symlink", event: fsnotify.Event{Name: filepath.Join(watchDir, "..data_tmp")}, want: true},
		{name: "other file", event: fsnotify.Event{Name: filepath.Join(watchDir, "other.pem")}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isTargetCAEvent(tt.event, watchDir, watchBase); got != tt.want {
				t.Fatalf("isTargetCAEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testCAPEM(t *testing.T) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "beanq test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
