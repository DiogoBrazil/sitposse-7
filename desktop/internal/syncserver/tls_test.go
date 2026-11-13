package syncserver

import (
	"crypto/tls"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"
)

// ADR-0014: TLS 1.3 preferido; TLS 1.2 só com ECDHE + AEAD (Android 8/9); nada abaixo.

func clienteTLS(t *testing.T, pin string, ajustar func(*tls.Config)) *http.Client {
	t.Helper()
	cfg := ConfigClienteComPin(pin)
	ajustar(cfg)
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: cfg}}
}

func TestTLS12ComSuiteAEADAceito(t *testing.T) {
	s := novoServidor(t, t.TempDir())
	for _, suite := range SuitesTLS12 {
		c := clienteTLS(t, s.Identidade.Pin(), func(c *tls.Config) {
			c.MaxVersion = tls.VersionTLS12
			c.CipherSuites = []uint16{suite}
		})
		resp, err := c.Get("https://127.0.0.1:" + strconv.Itoa(s.Porta()) + "/v1/status")
		if err != nil {
			t.Fatalf("%s: %v", tls.CipherSuiteName(suite), err)
		}
		resp.Body.Close()
		if resp.TLS.Version != tls.VersionTLS12 || resp.TLS.CipherSuite != suite {
			t.Fatalf("negociado %x/%s, esperado TLS 1.2/%s", resp.TLS.Version, tls.CipherSuiteName(resp.TLS.CipherSuite), tls.CipherSuiteName(suite))
		}
	}
}

func TestTLS12SemAEADRecusado(t *testing.T) {
	s := novoServidor(t, t.TempDir())
	fracas := []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA}
	for _, suite := range fracas {
		if slices.Contains(SuitesTLS12, suite) {
			t.Fatalf("suíte CBC na lista aceita: %s", tls.CipherSuiteName(suite))
		}
		c := clienteTLS(t, s.Identidade.Pin(), func(c *tls.Config) {
			c.MaxVersion = tls.VersionTLS12
			c.CipherSuites = []uint16{suite}
		})
		if resp, err := c.Get("https://127.0.0.1:" + strconv.Itoa(s.Porta()) + "/v1/status"); err == nil {
			resp.Body.Close()
			t.Fatalf("suíte %s aceita", tls.CipherSuiteName(suite))
		}
	}
}

func TestTLSAbaixoDe12Recusado(t *testing.T) {
	s := novoServidor(t, t.TempDir())
	c := clienteTLS(t, s.Identidade.Pin(), func(c *tls.Config) {
		c.MinVersion = tls.VersionTLS10
		c.MaxVersion = tls.VersionTLS11
		c.CipherSuites = nil
	})
	if resp, err := c.Get("https://127.0.0.1:" + strconv.Itoa(s.Porta()) + "/v1/status"); err == nil {
		resp.Body.Close()
		t.Fatal("TLS 1.1 aceito")
	}
}
