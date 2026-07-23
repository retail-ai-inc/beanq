package bredis

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisHolderReloadCA(t *testing.T) {
	caOne, leafOne := testCertificateChain(t, 1)
	caTwo, leafTwo := testCertificateChain(t, 2)
	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, caOne, 0600); err != nil {
		t.Fatalf("write first CA: %v", err)
	}

	client := &closeCountingClient{}
	holder := &RedisHolder{UniversalClient: client, caFile: caFile, verifyCertificate: true}
	if err := holder.Reload(context.Background()); err != nil {
		t.Fatalf("reload first CA: %v", err)
	}
	config := holder.tlsConfig()
	verifyTLSState(t, config, leafOne, true)
	verifyTLSState(t, config, leafTwo, false)

	if err := os.WriteFile(caFile, caTwo, 0600); err != nil {
		t.Fatalf("write second CA: %v", err)
	}
	if err := holder.Reload(context.Background()); err != nil {
		t.Fatalf("reload second CA: %v", err)
	}
	if holder.UniversalClient != client {
		t.Fatal("Reload replaced the Redis client")
	}
	verifyTLSState(t, config, leafOne, false)
	verifyTLSState(t, config, leafTwo, true)

	if err := os.WriteFile(caFile, []byte("invalid CA"), 0600); err != nil {
		t.Fatalf("write invalid CA: %v", err)
	}
	if err := holder.Reload(context.Background()); err == nil {
		t.Fatal("Reload accepted an invalid CA")
	}
	verifyTLSState(t, config, leafTwo, true)
}

func TestRedisHolderReloadHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	holder := &RedisHolder{caFile: filepath.Join(t.TempDir(), "missing.pem")}
	if err := holder.Reload(ctx); err == nil {
		t.Fatal("Reload accepted a canceled context")
	}
}

func TestRedisHolderTLSConfigSkipsVerificationWhenDisabled(t *testing.T) {
	holder := &RedisHolder{verifyCertificate: false}
	if err := holder.tlsConfig().VerifyConnection(tls.ConnectionState{}); err != nil {
		t.Fatalf("VerifyConnection returned error: %v", err)
	}
}

func TestRedisHolderConcurrentCAReload(t *testing.T) {
	_, leafOne := testCertificateChain(t, 11)
	_, leafTwo := testCertificateChain(t, 12)
	poolOne := x509.NewCertPool()
	poolOne.AddCert(leafOne)
	poolTwo := x509.NewCertPool()
	poolTwo.AddCert(leafTwo)

	holder := &RedisHolder{verifyCertificate: true}
	holder.caPool.Store(poolOne)
	config := holder.tlsConfig()
	state := tls.ConnectionState{PeerCertificates: []*x509.Certificate{leafOne}, ServerName: "redis.test"}

	var wait sync.WaitGroup
	for range 8 {
		wait.Go(func() {
			for range 1000 {
				_ = config.VerifyConnection(state)
			}
		})
	}
	for range 1000 {
		holder.caPool.Store(poolTwo)
		holder.caPool.Store(poolOne)
	}
	wait.Wait()
}

func TestRedisAddresses(t *testing.T) {
	tests := []struct {
		name        string
		hosts       string
		defaultPort string
		want        []string
		wantErr     bool
	}{
		{name: "hostname with default port", hosts: "redis", defaultPort: "6379", want: []string{"redis:6379"}},
		{name: "preserve explicit port", hosts: "redis:6380", defaultPort: "6379", want: []string{"redis:6380"}},
		{name: "trim multiple hosts", hosts: " redis-1 , redis-2:6380 ", defaultPort: "6379", want: []string{"redis-1:6379", "redis-2:6380"}},
		{name: "IPv6 with default port", hosts: "2001:db8::1", defaultPort: "6379", want: []string{"[2001:db8::1]:6379"}},
		{name: "bracketed IPv6 with port", hosts: "[2001:db8::1]:6380", defaultPort: "6379", want: []string{"[2001:db8::1]:6380"}},
		{name: "empty host", hosts: "", defaultPort: "6379", wantErr: true},
		{name: "empty host entry", hosts: "redis-1,,redis-2", defaultPort: "6379", wantErr: true},
		{name: "missing port", hosts: "redis", wantErr: true},
		{name: "invalid address", hosts: "redis:bad:address", defaultPort: "6379", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := redisAddresses(tt.hosts, tt.defaultPort)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("redisAddresses() = %v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("redisAddresses() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("redisAddresses() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("redisAddresses()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRedisHolderCloseOnce(t *testing.T) {
	client := &closeCountingClient{}
	holder := &RedisHolder{UniversalClient: client}

	if err := holder.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := holder.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if got := client.closes.Load(); got != 1 {
		t.Fatalf("underlying Close calls = %d, want 1", got)
	}
}

func TestRedisHolderClientReturnsUnderlyingClient(t *testing.T) {
	underlying := &closeCountingClient{}
	holder := &RedisHolder{UniversalClient: underlying}

	if got := holder.Client(); got != underlying {
		t.Fatalf("Client() = %T, want original underlying client", got)
	}
}

type closeCountingClient struct {
	redis.UniversalClient
	closes atomic.Int32
}

func (c *closeCountingClient) Close() error {
	c.closes.Add(1)
	return nil
}

func verifyTLSState(t *testing.T, config *tls.Config, certificate *x509.Certificate, wantValid bool) {
	t.Helper()
	err := config.VerifyConnection(tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{certificate},
		ServerName:       "redis.test",
	})
	if (err == nil) != wantValid {
		t.Fatalf("VerifyConnection() error = %v, want valid = %v", err, wantValid)
	}
}

func testCertificateChain(t *testing.T, serial int64) ([]byte, *x509.Certificate) {
	t.Helper()
	now := time.Now()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: "BeanQ test CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create CA: %v", err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse CA: %v", err)
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate leaf key: %v", err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(serial + 100),
		Subject:      pkix.Name{CommonName: "redis.test"},
		DNSNames:     []string{"redis.test"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("create leaf certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("parse leaf certificate: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), leaf
}
