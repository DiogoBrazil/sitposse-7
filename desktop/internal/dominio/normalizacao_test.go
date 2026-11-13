package dominio

import (
	"testing"

	"sitposse7/desktop/internal/testutil"
)

func TestNormalizacaoFixtures(t *testing.T) {
	var fx struct {
		Casos []struct {
			Original    string `json:"original"`
			Normalizado string `json:"normalizado"`
			Valido      bool   `json:"valido"`
		} `json:"casos"`
	}
	testutil.LerFixture(t, "normalizacao/nomes.json", &fx)
	if len(fx.Casos) == 0 {
		t.Fatal("fixture sem casos")
	}
	for _, c := range fx.Casos {
		n, ok := NormalizarNome(c.Original)
		if n != c.Normalizado || ok != c.Valido {
			t.Errorf("%q: obteve (%q,%v), esperado (%q,%v)", c.Original, n, ok, c.Normalizado, c.Valido)
		}
	}
}
