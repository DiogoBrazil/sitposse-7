package banco

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbrirAplicaPragmasEMigracoes(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "central.db")
	b, err := Abrir(ctx, caminho)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerificarPragmas(ctx, b.Escrita); err != nil {
		t.Fatal(err)
	}
	if err := VerificarPragmas(ctx, b.Leitura); err != nil {
		t.Fatal(err)
	}
	var v int
	if err := b.Escrita.QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != VersaoMaxima() || v < 1 {
		t.Fatalf("user_version=%d err=%v", v, err)
	}
	if _, err := b.Escrita.Exec(`INSERT INTO meta VALUES ('k','v',1,1)`); err != nil {
		t.Fatal(err)
	}
	// Leitura não escreve.
	if _, err := b.Leitura.Exec(`INSERT INTO meta VALUES ('k2','v',1,1)`); err == nil {
		t.Fatal("conexão de leitura aceitou escrita")
	}
	b.Fechar()

	// Reabrir é idempotente e preserva dados.
	b, err = Abrir(ctx, caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	var n int
	b.Leitura.QueryRow(`SELECT count(*) FROM meta`).Scan(&n)
	if n != 1 {
		t.Fatalf("dados perdidos após reabrir: %d", n)
	}
}

func TestRecusaBancoMaisNovo(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "central.db")
	b, err := Abrir(ctx, caminho)
	if err != nil {
		t.Fatal(err)
	}
	b.Escrita.Exec("PRAGMA user_version = 999")
	b.Fechar()
	_, err = Abrir(ctx, caminho)
	if err == nil || !strings.Contains(err.Error(), "mais novo") {
		t.Fatalf("esperado erro de versão mais nova, obteve %v", err)
	}
}

func TestStrictRejeitaTipoErrado(t *testing.T) {
	b, err := Abrir(context.Background(), filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	if _, err := b.Escrita.Exec(`INSERT INTO meta VALUES ('k','v','não é inteiro',1)`); err == nil {
		t.Fatal("STRICT não rejeitou texto em coluna INTEGER")
	}
}

// Banco criado pela versão da Fase 0 (só a migração 0001, com dados) migra para a
// versão atual sem perder nada (AGENTS.md §3/§4: migração não destrutiva testada).
func TestMigraBancoDaFase0SemPerderDados(t *testing.T) {
	ctx := context.Background()
	caminho := filepath.Join(t.TempDir(), "fase0.db")
	db, err := abrir(caminho, false)
	if err != nil {
		t.Fatal(err)
	}
	m1, err := migracoesFS.ReadFile("migracoes/0001_meta.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{string(m1), "PRAGMA user_version = 1",
		`INSERT INTO meta VALUES ('central_id','3f1c9a52-6a0e-4c1b-9d55-2f7f1f0b8e21',1,1)`} {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	b, err := Abrir(ctx, caminho)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	var id string
	if err := b.Leitura.QueryRow(`SELECT valor FROM meta WHERE chave='central_id'`).Scan(&id); err != nil || id != "3f1c9a52-6a0e-4c1b-9d55-2f7f1f0b8e21" {
		t.Fatalf("central_id perdido na migração: %q %v", id, err)
	}
	var v, n int
	b.Leitura.QueryRow("PRAGMA user_version").Scan(&v)
	b.Leitura.QueryRow("SELECT count(*) FROM catalogo_itens").Scan(&n)
	if v != VersaoMaxima() || n == 0 {
		t.Fatalf("user_version=%d catálogo=%d", v, n)
	}
}

func TestAuditoriaSomenteAcrescimo(t *testing.T) {
	b, err := Abrir(context.Background(), filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	if _, err := b.Escrita.Exec(`INSERT INTO auditoria (quando, origem, entidade, entidade_uuid, acao) VALUES (1,'central','x','u','teste')`); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Escrita.Exec(`UPDATE auditoria SET acao='outra'`); err == nil {
		t.Fatal("UPDATE em auditoria aceito")
	}
	if _, err := b.Escrita.Exec(`DELETE FROM auditoria`); err == nil {
		t.Fatal("DELETE em auditoria aceito")
	}
}
