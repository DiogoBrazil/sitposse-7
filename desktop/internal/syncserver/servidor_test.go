package syncserver

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func itoa(n int) string { return strconv.Itoa(n) }

func novoServidor(t *testing.T, dir string) *Servidor {
	t.Helper()
	id, err := CarregarOuCriarIdentidade(dir, []net.IP{net.IPv4(192, 168, 43, 10)})
	if err != nil {
		t.Fatal(err)
	}
	s := &Servidor{Identidade: id, CentralID: "central-teste"}
	if err := s.Iniciar("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Parar(context.Background()) })
	return s
}

func clienteComPin(pin string) *http.Client {
	return &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{TLSClientConfig: ConfigClienteComPin(pin)}}
}

func TestStatusComPinCorreto(t *testing.T) {
	s := novoServidor(t, t.TempDir())
	resp, err := clienteComPin(s.Identidade.Pin()).Get("https://127.0.0.1:" + strconv.Itoa(s.Porta()) + "/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var st Status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || st.CentralID != "central-teste" || st.ProtocoloVersao != ProtocoloVersao {
		t.Fatalf("status inesperado: %d %+v", resp.StatusCode, st)
	}
	if resp.TLS.Version != 0x0304 {
		t.Fatalf("esperado TLS 1.3, obteve %x", resp.TLS.Version)
	}
}

func TestPinErradoRecusado(t *testing.T) {
	s := novoServidor(t, t.TempDir())
	outro := novoServidor(t, t.TempDir()) // outra Central, outra chave
	_, err := clienteComPin(outro.Identidade.Pin()).Get("https://127.0.0.1:" + strconv.Itoa(s.Porta()) + "/v1/status")
	if err == nil || !errors.Is(err, ErrPinDivergente) {
		t.Fatalf("esperado ErrPinDivergente, obteve %v", err)
	}
}

func TestClienteSemPinRecusaCertificadoAutoassinado(t *testing.T) {
	s := novoServidor(t, t.TempDir())
	_, err := (&http.Client{Timeout: 5 * time.Second}).Get("https://127.0.0.1:" + strconv.Itoa(s.Porta()) + "/v1/status")
	if err == nil {
		t.Fatal("cliente padrão aceitou certificado autoassinado")
	}
}

func TestIdentidadePersistenteEPinEstavelAoMudarIPs(t *testing.T) {
	dir := t.TempDir()
	a, err := CarregarOuCriarIdentidade(dir, []net.IP{net.IPv4(192, 168, 43, 10)})
	if err != nil {
		t.Fatal(err)
	}
	b, err := CarregarOuCriarIdentidade(dir, []net.IP{net.IPv4(10, 0, 0, 5)}) // IP novo → reemite cert
	if err != nil {
		t.Fatal(err)
	}
	if a.Pin() != b.Pin() {
		t.Fatal("pin mudou ao reemitir certificado; pareamentos seriam perdidos")
	}
	if !cobreIPs(b.Certificado.Leaf, []net.IP{net.IPv4(10, 0, 0, 5)}) {
		t.Fatal("certificado reemitido não cobre o IP novo")
	}
}

// Ao reaproveitar o certificado salvo, os IPs anunciados (tela e QR) são os atuais,
// nunca o loopback nem IPs de redes antigas que o certificado ainda cobre.
func TestIdentidadeReaproveitadaAnunciaSoIPsAtuais(t *testing.T) {
	dir := t.TempDir()
	antigo, atual := net.IPv4(192, 168, 0, 24), net.IPv4(192, 168, 137, 1)
	if _, err := CarregarOuCriarIdentidade(dir, []net.IP{antigo, atual}); err != nil {
		t.Fatal(err)
	}
	casos := map[string][]net.IP{"um IP antigo some": {atual}, "sem rede": nil}
	for nome, ips := range casos {
		id, err := CarregarOuCriarIdentidade(dir, ips)
		if err != nil {
			t.Fatal(err)
		}
		if !cobreIPs(id.Certificado.Leaf, []net.IP{antigo, atual}) {
			t.Fatalf("%s: certificado deveria ter sido reaproveitado", nome)
		}
		if len(id.IPs) != len(ips) || (len(ips) == 1 && !id.IPs[0].Equal(atual)) {
			t.Fatalf("%s: IPs anunciados %v, esperado %v", nome, id.IPs, ips)
		}
	}
}

func TestSomenteRedeLocal(t *testing.T) {
	s := &Servidor{CentralID: "x"}
	casos := map[string]int{"192.168.1.5:5000": 200, "10.1.2.3:1": 200, "172.16.0.9:1": 200, "127.0.0.1:1": 200, "[fe80::1]:1": 200, "8.8.8.8:1": 403, "[2001:db8::1]:1": 403, "lixo": 403}
	for addr, esperado := range casos {
		req := httptest.NewRequest("GET", "/v1/status", nil)
		req.RemoteAddr = addr
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != esperado {
			t.Errorf("%s: código %d, esperado %d", addr, rec.Code, esperado)
		}
	}
}

func TestPayloadQR(t *testing.T) {
	s := novoServidor(t, t.TempDir())
	q := s.QR("tok", &Operacao{UUID: "u", Nome: "Op"})
	if q.V != 1 || q.SPKISHA256 != s.Identidade.Pin() || len(q.IPs) != 1 || q.IPs[0] != "192.168.43.10" || q.Porta != s.Porta() || q.OperacaoNome != "Op" {
		t.Fatalf("payload inesperado: %+v", q)
	}
	if len(q.SPKISHA256) != 43 {
		t.Fatalf("pin deve ter 43 caracteres base64url, tem %d", len(q.SPKISHA256))
	}
}
