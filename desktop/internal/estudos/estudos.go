// Package estudos gere a hierarquia Estudo de Situação → Jornada → Equipe da Central
// (RN-01, RN-02, ADR-0023) e resolve o código de equipe vindo do celular no destino
// (estudo, jornada e equipe) mostrado ao operador antes de importar (ADR-0024).
package estudos

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"sitposse7/desktop/internal/banco"
	"sitposse7/desktop/internal/codigos"
	"sitposse7/desktop/internal/dominio"
)

// Erros mostrados ao operador.
var (
	ErrNomeVazio          = errors.New("informe o nome")
	ErrNomeLongo          = errors.New("nome com mais de 200 caracteres")
	ErrData               = errors.New("data inválida")
	ErrNomeRepetido       = errors.New("já existe uma equipe com este nome nesta jornada")
	ErrComColetas         = errors.New("há cadastros recebidos para esta equipe ou jornada; ela não pode ser excluída")
	ErrNaoEncontrado      = errors.New("não encontrado")
	ErrCodigoIndisponivel = errors.New("não foi possível gerar um código único; tente de novo")
)

const maxNome = 200

// TentativasCodigo é quantas vezes um código é sorteado de novo em caso de colisão.
const TentativasCodigo = 5

// Servico concentra as operações de escrita e leitura.
type Servico struct {
	B *banco.Banco
	// Agora e Aleatorio podem ser trocados nos testes.
	Agora     func() time.Time
	Aleatorio io.Reader
}

func (s *Servico) agora() int64 {
	if s.Agora != nil {
		return s.Agora().UnixMilli()
	}
	return time.Now().UnixMilli()
}

// Equipe de uma jornada.
type Equipe struct {
	UUID             string `json:"uuid"`
	JornadaUUID      string `json:"jornada_uuid"`
	Nome             string `json:"nome"`
	Codigo           string `json:"codigo"`
	CodigoExibicao   string `json:"codigo_exibicao"`
	Coletas          int    `json:"coletas"`
	Familias         int    `json:"familias"`
	UltimaImportacao *int64 `json:"ultima_importacao"`
}

// Jornada de levantamento.
type Jornada struct {
	UUID       string   `json:"uuid"`
	EstudoUUID string   `json:"estudo_uuid"`
	Numero     int      `json:"numero"`
	Data       string   `json:"data"`
	Familias   int      `json:"familias"`
	Equipes    []Equipe `json:"equipes"`
}

// Estudo de situação (resumo para listas).
type Estudo struct {
	UUID             string `json:"uuid"`
	Codigo           string `json:"codigo"`
	CodigoExibicao   string `json:"codigo_exibicao"`
	Nome             string `json:"nome"`
	CriadoEm         int64  `json:"criado_em"`
	Jornadas         int    `json:"jornadas"`
	Equipes          int    `json:"equipes"`
	Familias         int    `json:"familias"`
	PrimeiraData     string `json:"primeira_data"`
	UltimaData       string `json:"ultima_data"`
	UltimaImportacao *int64 `json:"ultima_importacao"`
}

// EstudoDetalhe traz as jornadas com as equipes.
type EstudoDetalhe struct {
	Estudo
	ListaJornadas []Jornada `json:"lista_jornadas"`
}

// Destino é a equipe (com jornada e estudo) para onde vai uma coleta.
type Destino struct {
	EstudoUUID    string `json:"estudo_uuid"`
	EstudoCodigo  string `json:"estudo_codigo"`
	EstudoNome    string `json:"estudo_nome"`
	JornadaUUID   string `json:"jornada_uuid"`
	JornadaNumero int    `json:"jornada_numero"`
	JornadaData   string `json:"jornada_data"`
	EquipeUUID    string `json:"equipe_uuid"`
	EquipeNome    string `json:"equipe_nome"`
	EquipeCodigo  string `json:"equipe_codigo"`
}

// ---------- validação ----------

func limparNome(nome string) (string, error) {
	n := strings.Join(strings.Fields(nome), " ")
	if n == "" {
		return "", ErrNomeVazio
	}
	if utf8.RuneCountInString(n) > maxNome {
		return "", ErrNomeLongo
	}
	return n, nil
}

// ValidarData confere AAAA-MM-DD (data civil da jornada).
func ValidarData(data string) error {
	t, err := time.Parse("2006-01-02", data)
	if err != nil || t.Format("2006-01-02") != data || t.Year() < 2000 || t.Year() > 2100 {
		return fmt.Errorf("%w: use dia/mês/ano válido", ErrData)
	}
	return nil
}

// ---------- escrita ----------

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func auditar(ctx context.Context, tx execer, quando int64, entidade, uuid, acao string, antes, depois any) error {
	var a, d any
	if antes != nil {
		b, _ := json.Marshal(antes)
		a = string(b)
	}
	if depois != nil {
		b, _ := json.Marshal(depois)
		d = string(b)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO auditoria (quando, operador, origem, entidade, entidade_uuid, acao, antes_json, depois_json)
		VALUES (?, NULL, 'central', ?, ?, ?, ?, ?)`, quando, entidade, uuid, acao, a, d)
	return err
}

// codigoLivre sorteia um código que ainda não existe na tabela (até TentativasCodigo vezes).
func (s *Servico) codigoLivre(ctx context.Context, tx execer, prefixo, tabela string) (string, error) {
	for i := 0; i < TentativasCodigo; i++ {
		c, err := codigos.Gerar(prefixo, s.Aleatorio)
		if err != nil {
			return "", err
		}
		var n int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+tabela+" WHERE codigo = ?", c).Scan(&n); err != nil {
			return "", err
		}
		if n == 0 {
			return c, nil
		}
	}
	return "", ErrCodigoIndisponivel
}

func (s *Servico) inserirEquipes(ctx context.Context, tx execer, jornada string, nomes []string, agora int64) ([]Equipe, error) {
	vistos := map[string]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT nome_normalizado FROM equipes WHERE jornada_uuid = ? AND excluido_em IS NULL`, jornada)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var n string
		rows.Scan(&n)
		vistos[n] = true
	}
	rows.Close()
	var criadas []Equipe
	for _, bruto := range nomes {
		nome, err := limparNome(bruto)
		if err != nil {
			return nil, fmt.Errorf("equipe: %w", err)
		}
		norm, _ := dominio.NormalizarNome(nome)
		if vistos[norm] {
			return nil, fmt.Errorf("%w: %s", ErrNomeRepetido, nome)
		}
		vistos[norm] = true
		cod, err := s.codigoLivre(ctx, tx, codigos.PrefixoEquipe, "equipes")
		if err != nil {
			return nil, err
		}
		e := Equipe{UUID: uuid.NewString(), JornadaUUID: jornada, Nome: nome, Codigo: cod, CodigoExibicao: codigos.Exibir(cod)}
		if _, err := tx.ExecContext(ctx, `INSERT INTO equipes (uuid, jornada_uuid, nome, nome_normalizado, codigo, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			e.UUID, jornada, nome, norm, cod, agora, agora); err != nil {
			return nil, err
		}
		if err := auditar(ctx, tx, agora, "equipe", e.UUID, "criar", nil, e); err != nil {
			return nil, err
		}
		criadas = append(criadas, e)
	}
	return criadas, nil
}

func (s *Servico) inserirJornada(ctx context.Context, tx execer, estudo, data string, agora int64) (*Jornada, error) {
	if err := ValidarData(data); err != nil {
		return nil, err
	}
	var numero int
	if err := tx.QueryRowContext(ctx, `SELECT coalesce(max(numero), 0) + 1 FROM jornadas WHERE estudo_uuid = ?`, estudo).Scan(&numero); err != nil {
		return nil, err
	}
	j := &Jornada{UUID: uuid.NewString(), EstudoUUID: estudo, Numero: numero, Data: data, Equipes: []Equipe{}}
	if _, err := tx.ExecContext(ctx, `INSERT INTO jornadas (uuid, estudo_uuid, numero, data, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		j.UUID, estudo, numero, data, agora, agora); err != nil {
		return nil, err
	}
	return j, auditar(ctx, tx, agora, "jornada", j.UUID, "criar", nil, j)
}

// CriarEstudo cria o estudo com a primeira jornada e, opcionalmente, as equipes dela.
func (s *Servico) CriarEstudo(ctx context.Context, nome, dataPrimeiraJornada string, equipes []string) (*EstudoDetalhe, error) {
	nome, err := limparNome(nome)
	if err != nil {
		return nil, err
	}
	if err := ValidarData(dataPrimeiraJornada); err != nil {
		return nil, err
	}
	agora := s.agora()
	tx, err := s.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	cod, err := s.codigoLivre(ctx, tx, codigos.PrefixoEstudo, "estudos")
	if err != nil {
		return nil, err
	}
	id := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `INSERT INTO estudos (uuid, codigo, nome, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, id, cod, nome, agora, agora); err != nil {
		return nil, err
	}
	if err := auditar(ctx, tx, agora, "estudo", id, "criar", nil, map[string]string{"codigo": cod, "nome": nome}); err != nil {
		return nil, err
	}
	j, err := s.inserirJornada(ctx, tx, id, dataPrimeiraJornada, agora)
	if err != nil {
		return nil, err
	}
	if _, err := s.inserirEquipes(ctx, tx, j.UUID, equipes, agora); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Detalhe(ctx, id)
}

// RenomearEstudo troca o nome (o código não muda).
func (s *Servico) RenomearEstudo(ctx context.Context, estudo, nome string) error {
	nome, err := limparNome(nome)
	if err != nil {
		return err
	}
	return s.atualizar(ctx, "estudos", "estudo", estudo, "nome", nome)
}

// atualizar troca uma coluna de texto de uma entidade vigente, com auditoria.
func (s *Servico) atualizar(ctx context.Context, tabela, entidade, id, coluna, valor string) error {
	agora := s.agora()
	tx, err := s.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var antes string
	err = tx.QueryRowContext(ctx, "SELECT "+coluna+" FROM "+tabela+" WHERE uuid = ? AND excluido_em IS NULL", id).Scan(&antes)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNaoEncontrado
	}
	if err != nil {
		return err
	}
	if antes == valor {
		return nil
	}
	extra := ""
	args := []any{valor, agora}
	if tabela == "equipes" {
		norm, _ := dominio.NormalizarNome(valor)
		extra = ", nome_normalizado = ?"
		args = append(args, norm)
		var jornada string
		tx.QueryRowContext(ctx, `SELECT jornada_uuid FROM equipes WHERE uuid = ?`, id).Scan(&jornada)
		var n int
		tx.QueryRowContext(ctx, `SELECT count(*) FROM equipes WHERE jornada_uuid = ? AND nome_normalizado = ? AND uuid <> ? AND excluido_em IS NULL`, jornada, norm, id).Scan(&n)
		if n > 0 {
			return ErrNomeRepetido
		}
	}
	if _, err := tx.ExecContext(ctx, "UPDATE "+tabela+" SET "+coluna+" = ?, updated_at = ?"+extra+" WHERE uuid = ?", append(args, id)...); err != nil {
		return err
	}
	if err := auditar(ctx, tx, agora, entidade, id, "atualizar", map[string]string{coluna: antes}, map[string]string{coluna: valor}); err != nil {
		return err
	}
	return tx.Commit()
}

// CriarJornada acrescenta uma jornada (número seguinte) ao estudo, com as equipes informadas.
// Cada equipe recebe código novo, mesmo que o nome repita o de outra jornada (RN-01).
func (s *Servico) CriarJornada(ctx context.Context, estudo, data string, equipes []string) (*Jornada, error) {
	agora := s.agora()
	tx, err := s.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM estudos WHERE uuid = ? AND excluido_em IS NULL`, estudo).Scan(&n); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrNaoEncontrado
	}
	j, err := s.inserirJornada(ctx, tx, estudo, data, agora)
	if err != nil {
		return nil, err
	}
	if j.Equipes, err = s.inserirEquipes(ctx, tx, j.UUID, equipes, agora); err != nil {
		return nil, err
	}
	if j.Equipes == nil {
		j.Equipes = []Equipe{}
	}
	return j, tx.Commit()
}

// AlterarDataJornada corrige a data da jornada.
func (s *Servico) AlterarDataJornada(ctx context.Context, jornada, data string) error {
	if err := ValidarData(data); err != nil {
		return err
	}
	return s.atualizar(ctx, "jornadas", "jornada", jornada, "data", data)
}

// AdicionarEquipes cria equipes (com códigos novos) numa jornada existente.
func (s *Servico) AdicionarEquipes(ctx context.Context, jornada string, nomes []string) ([]Equipe, error) {
	agora := s.agora()
	tx, err := s.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM jornadas WHERE uuid = ? AND excluido_em IS NULL`, jornada).Scan(&n); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrNaoEncontrado
	}
	es, err := s.inserirEquipes(ctx, tx, jornada, nomes, agora)
	if err != nil {
		return nil, err
	}
	return es, tx.Commit()
}

// RenomearEquipe troca o nome (o código não muda).
func (s *Servico) RenomearEquipe(ctx context.Context, equipe, nome string) error {
	nome, err := limparNome(nome)
	if err != nil {
		return err
	}
	return s.atualizar(ctx, "equipes", "equipe", equipe, "nome", nome)
}

// ExcluirEquipe marca a equipe como excluída (tombstone), só se nenhuma coleta foi ligada a ela.
func (s *Servico) ExcluirEquipe(ctx context.Context, equipe string) error {
	return s.excluir(ctx, "equipe", equipe,
		`SELECT count(*) FROM coletas WHERE equipe_uuid = ?`,
		`UPDATE equipes SET excluido_em = ?, updated_at = ? WHERE uuid = ? AND excluido_em IS NULL`)
}

// ExcluirJornada marca a jornada e as equipes dela como excluídas, só se nenhuma coleta foi ligada.
func (s *Servico) ExcluirJornada(ctx context.Context, jornada string) error {
	return s.excluir(ctx, "jornada", jornada,
		`SELECT count(*) FROM coletas c JOIN equipes e ON e.uuid = c.equipe_uuid WHERE e.jornada_uuid = ?`,
		`UPDATE jornadas SET excluido_em = ?, updated_at = ? WHERE uuid = ? AND excluido_em IS NULL`,
		`UPDATE equipes SET excluido_em = ?1, updated_at = ?2 WHERE jornada_uuid = ?3 AND excluido_em IS NULL`)
}

func (s *Servico) excluir(ctx context.Context, entidade, id, contarVinculos string, updates ...string) error {
	agora := s.agora()
	tx, err := s.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRowContext(ctx, contarVinculos, id).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrComColetas
	}
	res, err := tx.ExecContext(ctx, updates[0], agora, agora, id)
	if err != nil {
		return err
	}
	if k, _ := res.RowsAffected(); k == 0 {
		return ErrNaoEncontrado
	}
	for _, u := range updates[1:] {
		if _, err := tx.ExecContext(ctx, u, agora, agora, id); err != nil {
			return err
		}
	}
	if err := auditar(ctx, tx, agora, entidade, id, "excluir", nil, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// ---------- leitura ----------

type consultor interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

const sqlDestino = `SELECT s.uuid, s.codigo, s.nome, j.uuid, j.numero, j.data, e.uuid, e.nome, e.codigo
	FROM equipes e JOIN jornadas j ON j.uuid = e.jornada_uuid JOIN estudos s ON s.uuid = j.estudo_uuid `

func lerDestino(r *sql.Row) (*Destino, error) {
	var d Destino
	err := r.Scan(&d.EstudoUUID, &d.EstudoCodigo, &d.EstudoNome, &d.JornadaUUID, &d.JornadaNumero, &d.JornadaData, &d.EquipeUUID, &d.EquipeNome, &d.EquipeCodigo)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &d, err
}

// DestinoPorCodigo resolve um código de equipe (normalizado) numa equipe vigente; nil se não existe.
func DestinoPorCodigo(ctx context.Context, db consultor, codigo string) (*Destino, error) {
	return lerDestino(db.QueryRowContext(ctx, sqlDestino+`WHERE e.codigo = ? AND e.excluido_em IS NULL AND j.excluido_em IS NULL AND s.excluido_em IS NULL`, codigo))
}

// DestinoPorEquipe devolve o destino de uma equipe (mesmo excluída, para histórico); nil se não existe.
func DestinoPorEquipe(ctx context.Context, db consultor, equipe string) (*Destino, error) {
	return lerDestino(db.QueryRowContext(ctx, sqlDestino+`WHERE e.uuid = ?`, equipe))
}

// EquipeVigente diz se a equipe existe e não foi excluída.
func EquipeVigente(ctx context.Context, db consultor, equipe string) (*Destino, error) {
	return lerDestino(db.QueryRowContext(ctx, sqlDestino+`WHERE e.uuid = ? AND e.excluido_em IS NULL AND j.excluido_em IS NULL AND s.excluido_em IS NULL`, equipe))
}

const sqlEstudos = `SELECT s.uuid, s.codigo, s.nome, s.created_at,
	(SELECT count(*) FROM jornadas j WHERE j.estudo_uuid = s.uuid AND j.excluido_em IS NULL),
	(SELECT count(*) FROM equipes e JOIN jornadas j ON j.uuid = e.jornada_uuid WHERE j.estudo_uuid = s.uuid AND e.excluido_em IS NULL AND j.excluido_em IS NULL),
	(SELECT count(*) FROM v_propriedades_vigentes p WHERE p.estudo_uuid = s.uuid),
	coalesce((SELECT min(data) FROM jornadas j WHERE j.estudo_uuid = s.uuid AND j.excluido_em IS NULL), ''),
	coalesce((SELECT max(data) FROM jornadas j WHERE j.estudo_uuid = s.uuid AND j.excluido_em IS NULL), ''),
	(SELECT max(l.importado_em) FROM lotes_recebidos l JOIN equipes e ON e.uuid = l.equipe_destino_uuid JOIN jornadas j ON j.uuid = e.jornada_uuid WHERE j.estudo_uuid = s.uuid)
	FROM estudos s WHERE s.excluido_em IS NULL `

func lerEstudo(sc interface{ Scan(...any) error }) (Estudo, error) {
	var e Estudo
	var ult sql.NullInt64
	err := sc.Scan(&e.UUID, &e.Codigo, &e.Nome, &e.CriadoEm, &e.Jornadas, &e.Equipes, &e.Familias, &e.PrimeiraData, &e.UltimaData, &ult)
	e.CodigoExibicao = codigos.Exibir(e.Codigo)
	if ult.Valid {
		e.UltimaImportacao = &ult.Int64
	}
	return e, err
}

// Listar devolve os estudos, do mais recente para o mais antigo.
func Listar(ctx context.Context, db consultor) ([]Estudo, error) {
	rows, err := db.QueryContext(ctx, sqlEstudos+`ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Estudo{}
	for rows.Next() {
		e, err := lerEstudo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Detalhe devolve o estudo com jornadas e equipes vigentes.
func Detalhe(ctx context.Context, db consultor, estudo string) (*EstudoDetalhe, error) {
	e, err := lerEstudo(db.QueryRowContext(ctx, sqlEstudos+`AND s.uuid = ?`, estudo))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	d := &EstudoDetalhe{Estudo: e, ListaJornadas: []Jornada{}}
	rows, err := db.QueryContext(ctx, `SELECT j.uuid, j.numero, j.data, (SELECT count(*) FROM v_propriedades_vigentes p WHERE p.jornada_uuid = j.uuid)
		FROM jornadas j WHERE j.estudo_uuid = ? AND j.excluido_em IS NULL ORDER BY j.numero`, estudo)
	if err != nil {
		return nil, err
	}
	idx := map[string]int{}
	for rows.Next() {
		j := Jornada{EstudoUUID: estudo, Equipes: []Equipe{}}
		if err := rows.Scan(&j.UUID, &j.Numero, &j.Data, &j.Familias); err != nil {
			rows.Close()
			return nil, err
		}
		idx[j.UUID] = len(d.ListaJornadas)
		d.ListaJornadas = append(d.ListaJornadas, j)
	}
	rows.Close()
	rows, err = db.QueryContext(ctx, `SELECT e.uuid, e.jornada_uuid, e.nome, e.codigo,
		(SELECT count(*) FROM coletas c WHERE c.equipe_uuid = e.uuid),
		(SELECT count(*) FROM v_propriedades_vigentes p WHERE p.equipe_uuid = e.uuid),
		(SELECT max(l.importado_em) FROM lotes_recebidos l WHERE l.equipe_destino_uuid = e.uuid)
		FROM equipes e JOIN jornadas j ON j.uuid = e.jornada_uuid
		WHERE j.estudo_uuid = ? AND e.excluido_em IS NULL AND j.excluido_em IS NULL ORDER BY e.created_at, e.nome`, estudo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var q Equipe
		var ult sql.NullInt64
		if err := rows.Scan(&q.UUID, &q.JornadaUUID, &q.Nome, &q.Codigo, &q.Coletas, &q.Familias, &ult); err != nil {
			return nil, err
		}
		q.CodigoExibicao = codigos.Exibir(q.Codigo)
		if ult.Valid {
			q.UltimaImportacao = &ult.Int64
		}
		if i, ok := idx[q.JornadaUUID]; ok {
			d.ListaJornadas[i].Equipes = append(d.ListaJornadas[i].Equipes, q)
		}
	}
	return d, rows.Err()
}

// Detalhe pela conexão de leitura do serviço.
func (s *Servico) Detalhe(ctx context.Context, estudo string) (*EstudoDetalhe, error) {
	return Detalhe(ctx, s.B.Leitura, estudo)
}

// Listar pela conexão de leitura do serviço.
func (s *Servico) Listar(ctx context.Context) ([]Estudo, error) { return Listar(ctx, s.B.Leitura) }
