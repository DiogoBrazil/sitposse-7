package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGravarAtomicoSubstituiSemDeixarTemporario(t *testing.T) {
	dir := t.TempDir()
	destino := filepath.Join(dir, "Relatório São José.pdf")
	if err := os.WriteFile(destino, []byte("antigo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := GravarAtomico(destino, []byte("novo")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(destino); string(b) != "novo" {
		t.Fatalf("conteúdo = %q", b)
	}
	if es, _ := os.ReadDir(dir); len(es) != 1 {
		t.Fatalf("ficaram %d arquivos na pasta (temporário não removido?)", len(es))
	}
}
