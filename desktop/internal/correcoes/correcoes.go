// Package correcoes aplica correções locais com auditoria e concorrência otimista.
package correcoes

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"sitposse7/desktop/internal/banco"
	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/dominio"
)

type Servico struct{ B *banco.Banco }
type Documento struct {
	Propriedade map[string]any   `json:"propriedade"`
	Itens       []map[string]any `json:"itens"`
	Fotos       []map[string]any `json:"fotos"`
	Catalogo    []map[string]any `json:"catalogo"`
	Token       string           `json:"token"`
}
type leitor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func linhas(ctx context.Context, db leitor, q string, args ...any) ([]map[string]any, error) {
	rows, e := db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	cols, e := rows.Columns()
	if e != nil {
		return nil, e
	}
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptr := make([]any, len(cols))
		for i := range vals {
			ptr[i] = &vals[i]
		}
		if e = rows.Scan(ptr...); e != nil {
			return nil, e
		}
		m := map[string]any{}
		for i, c := range cols {
			m[c] = vals[i]
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func carregar(ctx context.Context, db leitor, id string) (*Documento, error) {
	ps, e := linhas(ctx, db, `SELECT * FROM propriedades WHERE uuid=? AND excluido_em IS NULL`, id)
	if e != nil {
		return nil, e
	}
	if len(ps) != 1 {
		return nil, errors.New("propriedade não encontrada ou excluída")
	}
	d := &Documento{Propriedade: ps[0]}
	d.Itens, e = linhas(ctx, db, `SELECT * FROM itens_propriedade WHERE propriedade_uuid=? AND excluido_em IS NULL ORDER BY uuid`, id)
	if e != nil {
		return nil, e
	}
	d.Fotos, e = linhas(ctx, db, `SELECT * FROM fotografias WHERE propriedade_uuid=? AND excluido_em IS NULL ORDER BY uuid`, id)
	if e != nil {
		return nil, e
	}
	b, _ := json.Marshal(d)
	h := sha256.Sum256(b)
	d.Token = hex.EncodeToString(h[:])
	d.Catalogo, e = linhas(ctx, db, `SELECT codigo,categoria,nome_exibicao FROM catalogo_itens WHERE ativo=1 ORDER BY categoria,ordem`)
	return d, e
}
func (s *Servico) Carregar(ctx context.Context, id string) (*Documento, error) {
	tx, e := s.B.Leitura.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	return carregar(ctx, tx, id)
}
func auditar(ctx context.Context, tx *sql.Tx, ent, id string, antes, depois any) error {
	a, _ := json.Marshal(antes)
	b, _ := json.Marshal(depois)
	acao := "corrigir"
	if m, ok := depois.(map[string]any); ok && m["excluido_em"] != nil {
		acao = "excluir"
	}
	_, e := tx.ExecContext(ctx, `INSERT INTO auditoria(quando,origem,entidade,entidade_uuid,acao,antes_json,depois_json) VALUES(?,'central',?,?,?,?,?)`, time.Now().UnixMilli(), ent, id, acao, string(a), string(b))
	return e
}
func tokenIgual(d *Documento, t string) error {
	if t == "" || d.Token != t {
		return errors.New("o cadastro foi alterado enquanto o formulário estava aberto; feche e recarregue antes de tentar novamente")
	}
	return nil
}

var campos = strings.Fields(`nome_propriedade responsavel_familia observacoes_relevantes pessoas_adultos pessoas_jovens pessoas_criancas veic_caminhoes veic_automoveis_utilitarios veic_motos_ciclomotores bov_adultos bov_bezerros equ_adultos equ_potros area_valor_milesimos area_unidade lat lon gps_origem gps_precisao_m gps_capturado_em gps_satelites`)

func validar(p map[string]any) error {
	cp := map[string]any{}
	for k, v := range p {
		cp[k] = v
	}
	cp["rev"] = int64(1)
	for _, k := range []string{"created_at", "updated_at", "gps_capturado_em", "excluido_em"} {
		if cp[k] != nil {
			switch v := cp[k].(type) {
			case int64:
				cp[k] = contratos.FormatarInstante(v)
			case float64:
				cp[k] = contratos.FormatarInstante(int64(v))
			}
		}
	}
	b, _ := json.Marshal(cp)
	var c contratos.Propriedade
	if e := json.Unmarshal(b, &c); e != nil {
		return e
	}
	return contratos.ValidarPropriedade(&c)
}
func (s *Servico) Salvar(ctx context.Context, id string, d *Documento) error {
	tx, e := s.B.Escrita.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	atual, e := carregar(ctx, tx, id)
	if e != nil {
		return e
	}
	if e = tokenIgual(atual, d.Token); e != nil {
		return e
	}
	antesJSON, _ := json.Marshal(atual)
	p := atual.Propriedade
	for _, k := range campos {
		p[k] = d.Propriedade[k]
	}
	nome, ok := p["nome_propriedade"].(string)
	if !ok {
		return errors.New("informe o nome da propriedade")
	}
	p["nome_propriedade"] = strings.TrimSpace(nome)
	resp, ok := p["responsavel_familia"].(string)
	if !ok {
		return errors.New("informe o responsável")
	}
	p["responsavel_familia"] = strings.TrimSpace(resp)
	if e = validar(p); e != nil {
		return e
	}
	agora := time.Now().UnixMilli()
	set := []string{}
	args := []any{}
	for _, k := range campos {
		set = append(set, k+"=?")
		args = append(args, p[k])
	}
	args = append(args, agora, id)
	if _, e = tx.ExecContext(ctx, `UPDATE propriedades SET `+strings.Join(set, ",")+`,updated_at=?,rev_central=rev_central+1,corrigido_desde_rev_origem=1 WHERE uuid=?`, args...); e != nil {
		return e
	}
	existentes := map[string]map[string]any{}
	for _, it := range atual.Itens {
		existentes[it["uuid"].(string)] = it
	}
	vistos := map[string]bool{}
	predef := map[string]bool{}
	// Retirar primeiro libera o índice único dos itens predefinidos.
	desejados := map[string]bool{}
	for _, it := range d.Itens {
		u, _ := it["uuid"].(string)
		desejados[u] = true
	}
	for u, it := range existentes {
		if !desejados[u] {
			if _, e = tx.ExecContext(ctx, `UPDATE itens_propriedade SET excluido_em=?,updated_at=?,rev_central=rev_central+1,corrigido_desde_rev_origem=1 WHERE uuid=?`, agora, agora, u); e != nil {
				return e
			}
			if e = auditar(ctx, tx, "item", u, it, map[string]any{"excluido_em": agora}); e != nil {
				return e
			}
		}
	}
	for _, it := range d.Itens {
		u, _ := it["uuid"].(string)
		if u == "" {
			u = uuid.NewString()
		}
		if vistos[u] {
			return errors.New("item repetido")
		}
		vistos[u] = true
		codigo, _ := it["codigo_catalogo"].(string)
		var cat string
		if e = tx.QueryRowContext(ctx, `SELECT categoria FROM catalogo_itens WHERE codigo=? AND ativo=1`, codigo).Scan(&cat); e != nil {
			return errors.New("item do catálogo inválido")
		}
		if !strings.HasSuffix(codigo, ".outros") {
			if predef[codigo] {
				return errors.New("item predefinido repetido")
			}
			predef[codigo] = true
		}
		var nome *string
		if n, ok := it["nome_personalizado"].(string); ok {
			n = strings.TrimSpace(n)
			if n != "" {
				nome = &n
			}
		}
		raw, _ := json.Marshal(it["quantidade"])
		var q int64
		if e = json.Unmarshal(raw, &q); e != nil || q < 1 {
			return errors.New("quantidade de item deve ser um inteiro maior que zero")
		}
		c := contratos.Item{UUID: u, PropriedadeUUID: id, Categoria: cat, CodigoCatalogo: codigo, NomePersonalizado: nome, Quantidade: q, Rev: 1, CreatedAt: contratos.FormatarInstante(agora), UpdatedAt: contratos.FormatarInstante(agora)}
		if e = contratos.ValidarItem(&c); e != nil {
			return e
		}
		var norm any
		if nome != nil {
			norm, _ = dominio.NormalizarNome(*nome)
		}
		if old, ok := existentes[u]; ok {
			_, e = tx.ExecContext(ctx, `UPDATE itens_propriedade SET categoria=?,codigo_catalogo=?,nome_personalizado=?,nome_normalizado=?,quantidade=?,updated_at=?,rev_central=rev_central+1,corrigido_desde_rev_origem=1 WHERE uuid=?`, cat, codigo, nome, norm, q, agora, u)
			if e == nil {
				e = auditar(ctx, tx, "item", u, old, c)
			}
		} else {
			js, _ := json.Marshal(c)
			_, e = tx.ExecContext(ctx, `INSERT INTO itens_propriedade(uuid,propriedade_uuid,categoria,codigo_catalogo,nome_personalizado,nome_normalizado,quantidade,created_at,updated_at,rev_origem,rev_central,corrigido_desde_rev_origem,recebido_em,origem_instalacao,registro_json) VALUES(?,?,?,?,?,?,?,?,?,1,1,1,?,?,?)`, u, id, cat, codigo, nome, norm, q, agora, agora, agora, "central", string(js))
			if e == nil {
				e = auditar(ctx, tx, "item", u, nil, c)
			}
		}
		if e != nil {
			return e
		}
	}
	fotos := map[string]bool{}
	for _, f := range atual.Fotos {
		fotos[f["uuid"].(string)] = true
	}
	for _, f := range d.Fotos {
		u, _ := f["uuid"].(string)
		if !fotos[u] {
			return errors.New("foto não pertence ao cadastro")
		}
		delete(fotos, u)
	}
	for u := range fotos {
		if _, e = tx.ExecContext(ctx, `UPDATE fotografias SET excluido_em=?,updated_at=?,rev_central=rev_central+1,corrigido_desde_rev_origem=1 WHERE uuid=?`, agora, agora, u); e != nil {
			return e
		}
		var antesFoto any
		for _, f := range atual.Fotos {
			if f["uuid"] == u {
				antesFoto = f
			}
		}
		if e = auditar(ctx, tx, "foto", u, antesFoto, map[string]any{"excluido_em": agora}); e != nil {
			return e
		}
	}
	// Recarregar o antes: a estrutura local p foi alterada apenas em memória.
	if e = auditar(ctx, tx, "propriedade", id, json.RawMessage(antesJSON), d); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Servico) Excluir(ctx context.Context, id, token string) error {
	tx, e := s.B.Escrita.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	d, e := carregar(ctx, tx, id)
	if e != nil {
		return e
	}
	if e = tokenIgual(d, token); e != nil {
		return e
	}
	if e = ExcluirCadastro(ctx, tx, id); e != nil {
		return e
	}
	return tx.Commit()
}
func ExcluirCadastro(ctx context.Context, tx *sql.Tx, id string) error {
	agora := time.Now().UnixMilli()
	for _, t := range []struct{ t, where, ent string }{{"itens_propriedade", "propriedade_uuid", "item"}, {"fotografias", "propriedade_uuid", "foto"}, {"propriedades", "uuid", "propriedade"}} {
		antes, e := linhas(ctx, tx, "SELECT * FROM "+t.t+" WHERE "+t.where+"=? AND excluido_em IS NULL", id)
		if e != nil {
			return e
		}
		for _, a := range antes {
			u := a["uuid"].(string)
			if _, e = tx.ExecContext(ctx, "UPDATE "+t.t+" SET excluido_em=?,updated_at=?,rev_central=rev_central+1,corrigido_desde_rev_origem=1 WHERE uuid=?", agora, agora, u); e != nil {
				return e
			}
			if e = auditar(ctx, tx, t.ent, u, a, map[string]any{"excluido_em": agora}); e != nil {
				return e
			}
		}
	}
	return nil
}

// ExcluirEstrutura recebe apenas consultas fixas internas, nunca identificadores da UI.
func ExcluirEstrutura(ctx context.Context, tx *sql.Tx, tabela, where, id string) error {
	if tabela != "estudos" && tabela != "jornadas" && tabela != "equipes" {
		return errors.New("estrutura inválida")
	}
	antes, e := linhas(ctx, tx, "SELECT * FROM "+tabela+" WHERE "+where+" AND excluido_em IS NULL", id)
	if e != nil {
		return e
	}
	agora := time.Now().UnixMilli()
	for _, a := range antes {
		if e = auditar(ctx, tx, tabela, a["uuid"].(string), a, map[string]any{"excluido_em": agora}); e != nil {
			return e
		}
	}
	_, e = tx.ExecContext(ctx, "UPDATE "+tabela+" SET excluido_em=?,updated_at=? WHERE "+where+" AND excluido_em IS NULL", agora, agora, id)
	return e
}
