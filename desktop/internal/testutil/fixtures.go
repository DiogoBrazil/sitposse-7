// Package testutil localiza e lê as fixtures compartilhadas (shared/test-fixtures).
package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// FixturePath devolve o caminho absoluto de shared/test-fixtures/<rel>.
func FixturePath(t testing.TB, rel string) string {
	t.Helper()
	_, arquivo, _, _ := runtime.Caller(0)
	raiz := filepath.Join(filepath.Dir(arquivo), "..", "..", "..")
	return filepath.Join(raiz, "shared", "test-fixtures", filepath.FromSlash(rel))
}

// LerFixture decodifica a fixture JSON em v.
func LerFixture(t testing.TB, rel string, v any) {
	t.Helper()
	dados, err := os.ReadFile(FixturePath(t, rel))
	if err != nil {
		t.Fatalf("lendo fixture %s: %v", rel, err)
	}
	if err := json.Unmarshal(dados, v); err != nil {
		t.Fatalf("decodificando fixture %s: %v", rel, err)
	}
}
