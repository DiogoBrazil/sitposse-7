// Package banco abre o SQLite da Central com as garantias de durabilidade
// exigidas (ADR-0004, RNF-03) e aplica migrações não destrutivas.
package banco

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"modernc.org/sqlite"
)

//go:embed migracoes/*.sql
var migracoesFS embed.FS

// PragmasEsperados são verificados após a abertura; divergência é erro fatal.
var PragmasEsperados = map[string]string{
	"journal_mode": "wal",
	"synchronous":  "2", // FULL
	"foreign_keys": "1",
}

// Banco agrupa a conexão de escrita (única) e o pool de leitura.
type Banco struct {
	Escrita *sql.DB
	Leitura *sql.DB
	Caminho string
}

func dsn(caminho string, somenteLeitura bool) string {
	q := url.Values{}
	q.Set("_busy_timeout", "10000")
	q.Set("_journal_mode", "WAL")
	q.Set("_synchronous", "FULL")
	q.Set("_foreign_keys", "1")
	if somenteLeitura {
		q.Set("_query_only", "1")
	} else {
		q.Set("_txlock", "immediate")
	}
	return "file:" + filepath.ToSlash(caminho) + "?" + q.Encode()
}

func abrir(caminho string, somenteLeitura bool) (*sql.DB, error) {
	c, err := sqlite.NewConnector(dsn(caminho, somenteLeitura))
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(c), nil
}

// Abrir abre (ou cria) o banco, verifica os PRAGMAs e aplica as migrações pendentes.
func Abrir(ctx context.Context, caminho string) (*Banco, error) {
	escrita, err := abrir(caminho, false)
	if err != nil {
		return nil, fmt.Errorf("abrindo banco: %w", err)
	}
	escrita.SetMaxOpenConns(1) // um único escritor (AGENTS.md desktop)
	if err := VerificarPragmas(ctx, escrita); err != nil {
		escrita.Close()
		return nil, err
	}
	if err := Migrar(ctx, escrita); err != nil {
		escrita.Close()
		return nil, err
	}
	leitura, err := abrir(caminho, true)
	if err != nil {
		escrita.Close()
		return nil, err
	}
	leitura.SetMaxOpenConns(4)
	if err := VerificarPragmas(ctx, leitura); err != nil {
		escrita.Close()
		leitura.Close()
		return nil, err
	}
	return &Banco{Escrita: escrita, Leitura: leitura, Caminho: caminho}, nil
}

// Fechar encerra as conexões.
func (b *Banco) Fechar() error {
	return errors.Join(b.Leitura.Close(), b.Escrita.Close())
}

// VerificarPragmas confirma que os PRAGMAs de durabilidade estão efetivos.
func VerificarPragmas(ctx context.Context, db *sql.DB) error {
	for nome, esperado := range PragmasEsperados {
		var v string
		if err := db.QueryRowContext(ctx, "PRAGMA "+nome).Scan(&v); err != nil {
			return fmt.Errorf("lendo PRAGMA %s: %w", nome, err)
		}
		if strings.ToLower(v) != esperado {
			return fmt.Errorf("PRAGMA %s = %q, esperado %q", nome, v, esperado)
		}
	}
	return nil
}

type migracao struct {
	versao int
	nome   string
	sql    string
}

func listarMigracoes() ([]migracao, error) {
	entradas, err := fs.ReadDir(migracoesFS, "migracoes")
	if err != nil {
		return nil, err
	}
	var ms []migracao
	for _, e := range entradas {
		num, _, ok := strings.Cut(e.Name(), "_")
		v, err := strconv.Atoi(num)
		if !ok || err != nil || v <= 0 {
			return nil, fmt.Errorf("nome de migração inválido: %s (use NNNN_descricao.sql)", e.Name())
		}
		conteudo, err := migracoesFS.ReadFile("migracoes/" + e.Name())
		if err != nil {
			return nil, err
		}
		ms = append(ms, migracao{v, e.Name(), string(conteudo)})
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].versao < ms[j].versao })
	for i, m := range ms {
		if m.versao != i+1 {
			return nil, fmt.Errorf("migrações devem ser contínuas a partir de 1; encontrado %s", m.nome)
		}
	}
	return ms, nil
}

// VersaoMaxima é a versão de esquema que este binário conhece.
func VersaoMaxima() int {
	ms, err := listarMigracoes()
	if err != nil {
		return 0
	}
	return len(ms)
}

// Migrar aplica, cada uma em sua transação, as migrações acima de PRAGMA user_version.
// Recusa abrir banco de versão mais nova que o binário (nunca rebaixa nem destrói).
// O backup automático antes de migrar será acrescentado com o pacote backup (Fase 5);
// até lá nenhuma migração altera dados existentes.
func Migrar(ctx context.Context, db *sql.DB) error {
	ms, err := listarMigracoes()
	if err != nil {
		return err
	}
	var atual int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&atual); err != nil {
		return err
	}
	if atual > len(ms) {
		return fmt.Errorf("banco na versão %d é mais novo que este aplicativo (conhece até %d); atualize a Central", atual, len(ms))
	}
	for _, m := range ms[atual:] {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			tx.Rollback()
			return fmt.Errorf("migração %s: %w", m.nome, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", m.versao)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migração %s: %w", m.nome, err)
		}
	}
	return nil
}
