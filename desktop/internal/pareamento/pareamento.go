// Package pareamento cuida das operações, dos tokens do QR, dos pedidos de
// pareamento (que esperam a confirmação do operador) e da autenticação dos
// aparelhos pela credencial (docs/synchronization/protocolo.md §2, ADR-0012).
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
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"sitposse7/desktop/internal/banco"
	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/importacao"
)

// Erros com significado HTTP (o syncserver traduz).
var (
	ErrSemOperacao      = errors.New("nenhuma operação ativa na Central")
	ErrOperacaoAtiva    = errors.New("já existe uma operação ativa; encerre-a antes de criar outra")
	ErrTokenInvalido    = errors.New("QR de pareamento expirado ou já usado; gere um novo QR na Central")
	ErrOutraOperacao    = errors.New("esta equipe já pertence a outra operação")
	ErrRecusado         = errors.New("pareamento recusado pelo operador da Central")
	ErrTempoEsgotado    = errors.New("o operador não confirmou a tempo; leia o QR de novo")
	ErrNaoAutorizado    = errors.New("aparelho não pareado ou credencial revogada")
	ErrPedidoInexistente = errors.New("pedido de pareamento não encontrado")
)

// Prazos (protocolo.md §2).
const (
	ValidadeToken   = 10 * time.Minute
	EsperaOperador  = 90 * time.Second
)

// Operacao é uma operação de campo (ADR-0012).
type Operacao struct {
	UUID      string `json:"uuid"`
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
	Status    string `json:"status"`
	CriadaEm  int64  `json:"criada_em"`
}

// PedidoResumo é o que a tela da Central mostra para o operador decidir.
type PedidoResumo struct {
	ID                string  `json:"id"`
	EquipeUUID        string  `json:"equipe_uuid"`
	EquipeNome        string  `json:"equipe_nome"`
	Responsavel       string  `json:"responsavel"`
	IdentificacaoArea *string `json:"identificacao_area"`
	RotuloAparelho    *string `json:"rotulo_aparelho"`
	InstalacaoUUID    string  `json:"instalacao_uuid"`
	EquipeConhecida   bool    `json:"equipe_conhecida"`
	RecebidoEm        int64   `json:"recebido_em"`
}

type pedido struct {
	resumo  PedidoResumo
	req     contratos.PedidoPareamento
	decisao chan bool
}

// Servico concentra operações, pareamento e autenticação.
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

// ---------- operações ----------

// OperacaoAtiva devolve a operação ativa (nil se não houver).
func (s *Servico) OperacaoAtiva(ctx context.Context) (*Operacao, error) {
	var o Operacao
	var desc sql.NullString
	err := s.B.Leitura.QueryRowContext(ctx, `SELECT uuid, nome, descricao, status, created_at FROM operacoes WHERE status = 'ativa'`).
		Scan(&o.UUID, &o.Nome, &desc, &o.Status, &o.CriadaEm)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	o.Descricao = desc.String
	return &o, err
}

// CriarOperacao cria e ativa uma operação (só uma ativa por vez).
func (s *Servico) CriarOperacao(ctx context.Context, nome, descricao string) (*Operacao, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return nil, errors.New("informe o nome da operação")
	}
	if len([]rune(nome)) > contratos.MaxTexto {
		return nil, fmt.Errorf("nome com mais de %d caracteres", contratos.MaxTexto)
	}
	if atv, err := s.OperacaoAtiva(ctx); err != nil {
		return nil, err
	} else if atv != nil {
		return nil, ErrOperacaoAtiva
	}
	o := &Operacao{UUID: uuid.NewString(), Nome: nome, Descricao: strings.TrimSpace(descricao), Status: "ativa", CriadaEm: s.agora().UnixMilli()}
	tx, err := s.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var desc any
	if o.Descricao != "" {
		desc = o.Descricao
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO operacoes (uuid, nome, descricao, status, created_at, updated_at) VALUES (?, ?, ?, 'ativa', ?, ?)`,
		o.UUID, o.Nome, desc, o.CriadaEm, o.CriadaEm); err != nil {
		return nil, err
	}
	depois, _ := json.Marshal(o)
	if _, err := tx.ExecContext(ctx, `INSERT INTO auditoria (quando, operador, origem, entidade, entidade_uuid, acao, depois_json) VALUES (?, NULL, 'central', 'operacao', ?, 'criar', ?)`,
		o.CriadaEm, o.UUID, string(depois)); err != nil {
		return nil, err
	}
	return o, tx.Commit()
}

// ---------- tokens ----------

// GerarToken cria um token de uso único para o QR (10 min).
func (s *Servico) GerarToken(ctx context.Context) (token string, op *Operacao, expira time.Time, err error) {
	op, err = s.OperacaoAtiva(ctx)
	if err != nil {
		return "", nil, time.Time{}, err
	}
	if op == nil {
		return "", nil, time.Time{}, ErrSemOperacao
	}
	token, err = aleatorio32()
	if err != nil {
		return "", nil, time.Time{}, err
	}
	agora := s.agora()
	expira = agora.Add(ValidadeToken)
	_, err = s.B.Escrita.ExecContext(ctx, `INSERT INTO tokens_pareamento (token_hash, operacao_uuid, criado_em, expira_em) VALUES (?, ?, ?, ?)`,
		HashSegredo(token), op.UUID, agora.UnixMilli(), expira.UnixMilli())
	return token, op, expira, err
}

func (s *Servico) conferirToken(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, token string) (operacao string, err error) {
	var expira int64
	var usado sql.NullInt64
	err = q.QueryRowContext(ctx, `SELECT t.operacao_uuid, t.expira_em, t.usado_em FROM tokens_pareamento t
		JOIN operacoes o ON o.uuid = t.operacao_uuid WHERE t.token_hash = ? AND o.status = 'ativa'`, HashSegredo(token)).Scan(&operacao, &expira, &usado)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (usado.Valid || s.agora().UnixMilli() > expira)) {
		return "", ErrTokenInvalido
	}
	return operacao, err
}

// ---------- pedidos ----------

// Pedir registra o pedido do aparelho e espera a decisão do operador.
func (s *Servico) Pedir(ctx context.Context, req contratos.PedidoPareamento) (*contratos.RespostaPareamento, error) {
	if err := contratos.ValidarPedidoPareamento(&req); err != nil {
		return nil, fmt.Errorf("%w: %v", importacao.ErrInvalido, err)
	}
	op, err := s.conferirToken(ctx, s.B.Leitura, req.Token)
	if err != nil {
		return nil, err
	}
	var opEquipe string
	err = s.B.Leitura.QueryRowContext(ctx, `SELECT operacao_uuid FROM equipes WHERE uuid = ?`, req.Equipe.UUID).Scan(&opEquipe)
	conhecida := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if conhecida && opEquipe != op {
		return nil, ErrOutraOperacao // P6: decidir depois; nunca mudar de operação em silêncio
	}

	p := &pedido{req: req, decisao: make(chan bool, 1), resumo: PedidoResumo{
		ID: uuid.NewString(), EquipeUUID: req.Equipe.UUID, EquipeNome: req.Equipe.Nome, Responsavel: req.Equipe.Responsavel,
		IdentificacaoArea: req.Equipe.IdentificacaoArea, RotuloAparelho: req.RotuloAparelho, InstalacaoUUID: req.InstalacaoUUID,
		EquipeConhecida: conhecida, RecebidoEm: s.agora().UnixMilli(),
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
	op, err := s.conferirToken(ctx, tx, req.Token) // de novo, agora dentro da transação
	if err != nil {
		return nil, err
	}
	var opNome string
	if err := tx.QueryRowContext(ctx, `SELECT nome FROM operacoes WHERE uuid = ?`, op).Scan(&opNome); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tokens_pareamento SET usado_em = ? WHERE token_hash = ?`, agora, HashSegredo(req.Token)); err != nil {
		return nil, err
	}
	e := req.Equipe
	js, _ := json.Marshal(e)
	criado, _ := contratos.LerInstante(e.CreatedAt)
	atualizado, _ := contratos.LerInstante(e.UpdatedAt)
	var opEquipe string
	err = tx.QueryRowContext(ctx, `SELECT operacao_uuid FROM equipes WHERE uuid = ?`, e.UUID).Scan(&opEquipe)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx, `INSERT INTO equipes (uuid, operacao_uuid, nome, responsavel, identificacao_area, created_at, updated_at, rev_origem, recebido_em, registro_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.UUID, op, e.Nome, e.Responsavel, e.IdentificacaoArea, criado, atualizado, e.Rev, agora, string(js)); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	case opEquipe != op:
		return nil, ErrOutraOperacao
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO dispositivos (instalacao_uuid, equipe_uuid, credencial_hash, rotulo_aparelho, pareado_em) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (instalacao_uuid) DO UPDATE SET equipe_uuid = excluded.equipe_uuid, credencial_hash = excluded.credencial_hash,
		rotulo_aparelho = excluded.rotulo_aparelho, pareado_em = excluded.pareado_em, revogado_em = NULL`,
		req.InstalacaoUUID, e.UUID, HashSegredo(credencial), req.RotuloAparelho, agora); err != nil {
		return nil, err
	}
	depois, _ := json.Marshal(map[string]any{"instalacao_uuid": req.InstalacaoUUID, "equipe_uuid": e.UUID, "rotulo_aparelho": req.RotuloAparelho, "operacao_uuid": op})
	if _, err := tx.ExecContext(ctx, `INSERT INTO auditoria (quando, operador, origem, entidade, entidade_uuid, acao, depois_json) VALUES (?, NULL, 'central', 'dispositivo', ?, 'parear', ?)`,
		agora, req.InstalacaoUUID, string(depois)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &contratos.RespostaPareamento{ProtocoloVersao: contratos.ProtocoloVersao, CentralID: s.CentralID,
		CredencialDispositivo: credencial, Operacao: contratos.OperacaoResumo{UUID: op, Nome: opNome}}, nil
}

// ---------- autenticação ----------

// Autenticar identifica o aparelho pela credencial (Authorization: Bearer).
func (s *Servico) Autenticar(ctx context.Context, credencial string) (importacao.Dispositivo, error) {
	var d importacao.Dispositivo
	if credencial == "" {
		return d, ErrNaoAutorizado
	}
	err := s.B.Leitura.QueryRowContext(ctx, `SELECT d.instalacao_uuid, d.equipe_uuid, e.operacao_uuid FROM dispositivos d
		JOIN equipes e ON e.uuid = d.equipe_uuid WHERE d.credencial_hash = ? AND d.revogado_em IS NULL`, HashSegredo(credencial)).
		Scan(&d.InstalacaoUUID, &d.EquipeUUID, &d.OperacaoUUID)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNaoAutorizado
	}
	return d, err
}
