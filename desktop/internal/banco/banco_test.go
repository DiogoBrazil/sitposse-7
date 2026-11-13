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
