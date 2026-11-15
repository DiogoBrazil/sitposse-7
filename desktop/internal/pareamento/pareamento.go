// Package pareamento cuida dos tokens do QR, dos pedidos de pareamento (que esperam a
// confirmação do operador) e da autenticação dos aparelhos pela credencial
// (docs/synchronization/protocolo.md §2). O pareamento é só do aparelho: a equipe vem
// de cada coleta, pelo código (ADR-0024).
package pareamento

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"sitposse7/desktop/internal/banco"
	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/importacao"
)

// Erros com significado HTTP (o syncserver traduz).
var (
	ErrTokenInvalido     = errors.New("QR de pareamento expirado ou já usado; gere um novo QR na Central")
	ErrRecusado          = errors.New("pareamento recusado pelo operador da Central")
	ErrTempoEsgotado     = errors.New("o operador não confirmou a tempo; leia o QR de novo")
	ErrNaoAutorizado     = errors.New("aparelho não pareado ou credencial revogada")
	ErrPedidoInexistente = errors.New("pedido de pareamento não encontrado")
)

// Prazos (protocolo.md §2).
const (
	ValidadeToken  = 10 * time.Minute
	EsperaOperador = 90 * time.Second
)

// PedidoResumo é o que a tela da Central mostra para o operador decidir.
type PedidoResumo struct {
	ID             string  `json:"id"`
	RotuloAparelho *string `json:"rotulo_aparelho"`
	InstalacaoUUID string  `json:"instalacao_uuid"`
	// AparelhoConhecido: este aparelho já foi pareado antes (o novo pareamento troca a credencial).
	AparelhoConhecido bool  `json:"aparelho_conhecido"`
	RecebidoEm        int64 `json:"recebido_em"`
}

type pedido struct {
	resumo  PedidoResumo
	req     contratos.PedidoPareamento
	decisao chan bool
}

// Servico concentra pareamento e autenticação.
type Servico struct {
	B         *banco.Banco
	CentralID string
	Agora     func() time.Time
	// Espera é o tempo máximo aguardando o operador (padrão EsperaOperador).
	Espera time.Duration

	mu        sync.Mutex
	pendentes map[string]*pedido
}

func (s *Servico) agora() time.Time {
	if s.Agora != nil {
		return s.Agora()
	}
	return time.Now()
}

func aleatorio32() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// HashSegredo é o SHA-256 (hex) de um token ou credencial; só o hash é guardado.
func HashSegredo(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ---------- tokens ----------

// GerarToken cria um token de uso único para o QR (10 min).
func (s *Servico) GerarToken(ctx context.Context) (token string, expira time.Time, err error) {
	token, err = aleatorio32()
	if err != nil {
		return "", time.Time{}, err
	}
	agora := s.agora()
	expira = agora.Add(ValidadeToken)
	_, err = s.B.Escrita.ExecContext(ctx, `INSERT INTO tokens_pareamento (token_hash, criado_em, expira_em) VALUES (?, ?, ?)`,
		HashSegredo(token), agora.UnixMilli(), expira.UnixMilli())
	return token, expira, err
}

func (s *Servico) conferirToken(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, token string) error {
	var expira int64
	var usado sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT expira_em, usado_em FROM tokens_pareamento WHERE token_hash = ?`, HashSegredo(token)).Scan(&expira, &usado)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (usado.Valid || s.agora().UnixMilli() > expira)) {
		return ErrTokenInvalido
	}
	return err
}

// ---------- pedidos ----------

// Pedir registra o pedido do aparelho e espera a decisão do operador.
func (s *Servico) Pedir(ctx context.Context, req contratos.PedidoPareamento) (*contratos.RespostaPareamento, error) {
	if err := contratos.ValidarPedidoPareamento(&req); err != nil {
		return nil, fmt.Errorf("%w: %v", importacao.ErrInvalido, err)
	}
	if err := s.conferirToken(ctx, s.B.Leitura, req.Token); err != nil {
		return nil, err
	}
	var n int
	if err := s.B.Leitura.QueryRowContext(ctx, `SELECT count(*) FROM dispositivos WHERE instalacao_uuid = ?`, req.InstalacaoUUID).Scan(&n); err != nil {
		return nil, err
	}
	p := &pedido{req: req, decisao: make(chan bool, 1), resumo: PedidoResumo{
		ID: uuid.NewString(), RotuloAparelho: req.RotuloAparelho, InstalacaoUUID: req.InstalacaoUUID,
		AparelhoConhecido: n > 0, RecebidoEm: s.agora().UnixMilli(),
	}}
	s.mu.Lock()
	if s.pendentes == nil {
		s.pendentes = map[string]*pedido{}
	}
	s.pendentes[p.resumo.ID] = p
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.pendentes, p.resumo.ID)
		s.mu.Unlock()
	}()

	espera := s.Espera
	if espera == 0 {
		espera = EsperaOperador
	}
	timer := time.NewTimer(espera)
	defer timer.Stop()
	select {
	case aceito := <-p.decisao:
		if !aceito {
			return nil, ErrRecusado
		}
	case <-timer.C:
		return nil, ErrTempoEsgotado
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return s.efetivar(ctx, req)
}

// Pendentes lista os pedidos aguardando o operador.
func (s *Servico) Pendentes() []PedidoResumo {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PedidoResumo, 0, len(s.pendentes))
	for _, p := range s.pendentes {
		out = append(out, p.resumo)
	}
	return out
}

// Decidir confirma ou recusa um pedido pendente.
func (s *Servico) Decidir(id string, aceitar bool) error {
	s.mu.Lock()
	p, ok := s.pendentes[id]
	s.mu.Unlock()
	if !ok {
		return ErrPedidoInexistente
	}
	select {
	case p.decisao <- aceitar:
	default: // já decidido
	}
	return nil
}

func (s *Servico) efetivar(ctx context.Context, req contratos.PedidoPareamento) (*contratos.RespostaPareamento, error) {
	credencial, err := aleatorio32()
	if err != nil {
		return nil, err
	}
	agora := s.agora().UnixMilli()
	tx, err := s.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := s.conferirToken(ctx, tx, req.Token); err != nil { // de novo, agora dentro da transação
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tokens_pareamento SET usado_em = ? WHERE token_hash = ?`, agora, HashSegredo(req.Token)); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO dispositivos (instalacao_uuid, credencial_hash, rotulo_aparelho, pareado_em) VALUES (?, ?, ?, ?)
		ON CONFLICT (instalacao_uuid) DO UPDATE SET credencial_hash = excluded.credencial_hash,
		rotulo_aparelho = excluded.rotulo_aparelho, pareado_em = excluded.pareado_em, revogado_em = NULL`,
		req.InstalacaoUUID, HashSegredo(credencial), req.RotuloAparelho, agora); err != nil {
		return nil, err
	}
	depois, _ := json.Marshal(map[string]any{"instalacao_uuid": req.InstalacaoUUID, "rotulo_aparelho": req.RotuloAparelho})
	if _, err := tx.ExecContext(ctx, `INSERT INTO auditoria (quando, operador, origem, entidade, entidade_uuid, acao, depois_json) VALUES (?, NULL, 'central', 'dispositivo', ?, 'parear', ?)`,
		agora, req.InstalacaoUUID, string(depois)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &contratos.RespostaPareamento{ProtocoloVersao: contratos.ProtocoloVersao, CentralID: s.CentralID,
		CredencialDispositivo: credencial}, nil
}

// ---------- autenticação ----------

// Autenticar identifica o aparelho pela credencial (Authorization: Bearer).
func (s *Servico) Autenticar(ctx context.Context, credencial string) (importacao.Dispositivo, error) {
	var d importacao.Dispositivo
	if credencial == "" {
		return d, ErrNaoAutorizado
	}
	err := s.B.Leitura.QueryRowContext(ctx, `SELECT instalacao_uuid FROM dispositivos WHERE credencial_hash = ? AND revogado_em IS NULL`,
		HashSegredo(credencial)).Scan(&d.InstalacaoUUID)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNaoAutorizado
	}
	return d, err
}
