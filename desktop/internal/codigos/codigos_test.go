package codigos

import (
	"bytes"
	"errors"
	"testing"

	"sitposse7/desktop/internal/testutil"
)

type fixture struct {
	Alfabeto string `json:"alfabeto"`
	Digitos  []struct {
		Corpo  string `json:"corpo"`
		Digito string `json:"digito"`
	} `json:"digitos"`
	Normalizacao []struct {
		Entrada     string `json:"entrada"`
		Prefixo     string `json:"prefixo"`
		Normalizado string `json:"normalizado"`
		Exibicao    string `json:"exibicao"`
	} `json:"normalizacao"`
	Erros []struct {
		Entrada string `json:"entrada"`
		Prefixo string `json:"prefixo"`
		Erro    string `json:"erro"`
	} `json:"erros"`
}

func TestFixtureCompartilhada(t *testing.T) {
	var fx fixture
	testutil.LerFixture(t, "codigos/codigos.v1.json", &fx)
	if fx.Alfabeto != Alfabeto {
		t.Fatalf("alfabeto divergente: %q", fx.Alfabeto)
	}
	for _, d := range fx.Digitos {
		got, err := DigitoVerificador(d.Corpo)
		if err != nil || string(got) != d.Digito {
			t.Errorf("dígito de %s = %q (%v), esperado %s", d.Corpo, got, err, d.Digito)
		}
	}
	for _, c := range fx.Normalizacao {
		got, err := Normalizar(c.Entrada, c.Prefixo)
		if err != nil || got != c.Normalizado {
			t.Errorf("Normalizar(%q) = %q, %v; esperado %q", c.Entrada, got, err, c.Normalizado)
		}
		if Exibir(got) != c.Exibicao {
			t.Errorf("Exibir(%q) = %q, esperado %q", got, Exibir(got), c.Exibicao)
		}
		if !Valido(c.Normalizado, c.Prefixo) {
			t.Errorf("%q deveria ser válido", c.Normalizado)
		}
	}
	esperado := map[string]error{"vazio": ErrVazio, "tamanho": ErrTamanho, "caractere": ErrCaractere, "digito": ErrDigito}
	for _, c := range fx.Erros {
		_, err := Normalizar(c.Entrada, c.Prefixo)
		if !errors.Is(err, esperado[c.Erro]) {
			t.Errorf("Normalizar(%q) erro = %v, esperado %s", c.Entrada, err, c.Erro)
		}
	}
}

func TestGerarProduzCodigosValidos(t *testing.T) {
	vistos := map[string]bool{}
	for i := 0; i < 2000; i++ {
		c, err := Gerar(PrefixoEquipe, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !Valido(c, PrefixoEquipe) {
			t.Fatalf("código gerado inválido: %s", c)
		}
		vistos[c] = true
	}
	if len(vistos) < 1990 {
		t.Fatalf("códigos repetidos demais: %d distintos em 2000", len(vistos))
	}
	e, _ := Gerar(PrefixoEstudo, bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6}))
	if e != "SITPOSSE7"+"0123456"+string(must(DigitoVerificador("0123456"))) || !Valido(e, PrefixoEstudo) {
		t.Fatalf("estudo: %s", e)
	}
}

// Toda troca de um caractere é detectada pelo dígito (propriedade do Luhn mod N).
func TestTrocaDeUmCaractereSempreDetectada(t *testing.T) {
	c, _ := Gerar(PrefixoEquipe, nil)
	corpo := []byte(c[2:])
	for i := range corpo {
		for _, x := range []byte(Alfabeto) {
			if x == corpo[i] {
				continue
			}
			m := append([]byte(nil), corpo...)
			m[i] = x
			if digitoConfere(string(m)) {
				t.Fatalf("troca não detectada: %s → %s", corpo, m)
			}
		}
	}
}

func must(b byte, err error) byte {
	if err != nil {
		panic(err)
	}
	return b
}
