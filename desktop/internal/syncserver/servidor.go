package syncserver

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/importacao"
	"sitposse7/desktop/internal/pareamento"
)

// Versões de protocolo e esquema aceitas por esta Central (só a 2: ADR-0024, ADR-0025).
const (
	ProtocoloVersao = contratos.ProtocoloVersao
	SchemaVersao    = contratos.SchemaVersao
	PortaPadrao     = 38807
)

// Status é a resposta de GET /v1/status (shared/schemas/status.v2.schema.json).
type Status struct {
	ProtocoloVersao    int    `json:"protocolo_versao"`
	ProtocoloVersaoMin int    `json:"protocolo_versao_min"`
	ProtocoloVersaoMax int    `json:"protocolo_versao_max"`
	SchemaVersaoMin    int    `json:"schema_versao_min"`
	SchemaVersaoMax    int    `json:"schema_versao_max"`
	CentralID          string `json:"central_id"`
	Horario            string `json:"horario"`
}

// PayloadQR é o conteúdo do QR de pareamento (shared/schemas/qr-pareamento.v2.schema.json).
type PayloadQR struct {
	V          int      `json:"v"`
	CentralID  string   `json:"central_id"`
	IPs        []string `json:"ips"`
	Porta      int      `json:"porta"`
	SPKISHA256 string   `json:"spki_sha256"`
	Token      string   `json:"token_pareamento,omitempty"`
}

// Servidor é o servidor HTTPS local da Central.
type Servidor struct {
	Identidade *Identidade
	CentralID  string
	// Pareamento e Importador habilitam a API v1 completa (nil = só /v1/status).
	Pareamento *pareamento.Servico
	Importador *importacao.Importador
	// AoReceber é chamado (sem bloquear) quando chega lote, foto ou importação, para a tela ao vivo.
	AoReceber func(evento string)
	http      *http.Server
	ln        net.Listener
}

// Handler devolve o roteador HTTP (exposto para testes).
func (s *Servidor) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/status", s.status)
	if s.Pareamento != nil && s.Importador != nil {
		s.registrarAPI(mux)
	}
	return somenteRedeLocal(mux)
}

func (s *Servidor) status(w http.ResponseWriter, r *http.Request) {
	escreverJSON(w, http.StatusOK, Status{
		ProtocoloVersao: ProtocoloVersao, ProtocoloVersaoMin: ProtocoloVersao, ProtocoloVersaoMax: ProtocoloVersao,
		SchemaVersaoMin: SchemaVersao, SchemaVersaoMax: SchemaVersao,
		CentralID: s.CentralID,
		Horario:   time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	})
}

func somenteRedeLocal(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !EnderecoPermitido(r.RemoteAddr) {
			escreverJSON(w, http.StatusForbidden, map[string]string{"erro": "somente rede local"})
			return
		}
		h.ServeHTTP(w, r)
	})
}

func escreverJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// SuitesTLS12 são as únicas suítes aceitas em TLS 1.2 (ADR-0014): ECDHE (sigilo
// futuro) com AEAD, para o certificado ECDSA da Central. TLS 1.3 tem suítes fixas.
var SuitesTLS12 = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
}

// configTLS: TLS 1.3 preferido; TLS 1.2 restrito como mínimo porque Android 8 e 9
// (minSdk 26, A15) não têm TLS 1.3 na plataforma (ADR-0014).
func configTLS(cert tls.Certificate) *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		CipherSuites: SuitesTLS12,
		Certificates: []tls.Certificate{cert},
	}
}

// Iniciar escuta em endereco (ex.: ":38807") com TLS (ADR-0007/0014) e retorna imediatamente.
func (s *Servidor) Iniciar(endereco string) error {
	cfg := configTLS(s.Identidade.Certificado)
	ln, err := net.Listen("tcp", endereco)
	if err != nil {
		return err
	}
	s.ln = ln
	s.http = &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 15 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	go func() {
		if err := s.http.Serve(tls.NewListener(ln, cfg)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			_ = err // registrado pelo chamador via logs na Fase 2
		}
	}()
	return nil
}

// Porta devolve a porta efetivamente em uso.
func (s *Servidor) Porta() int {
	if s.ln == nil {
		return 0
	}
	_, p, _ := net.SplitHostPort(s.ln.Addr().String())
	n, _ := strconv.Atoi(p)
	return n
}

// Parar encerra o servidor aguardando requisições em andamento.
func (s *Servidor) Parar(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

// QR monta o payload de pareamento com os IPs atuais.
func (s *Servidor) QR(token string) PayloadQR {
	ips := make([]string, 0, len(s.Identidade.IPs))
	for _, ip := range s.Identidade.IPs {
		if !ip.IsLoopback() {
			ips = append(ips, ip.String())
		}
	}
	return PayloadQR{V: 2, CentralID: s.CentralID, IPs: ips, Porta: s.Porta(), SPKISHA256: s.Identidade.Pin(), Token: token}
}
