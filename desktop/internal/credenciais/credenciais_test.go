package credenciais

import (
	"errors"
	"testing"
)

// Roda no Linux (memória) e no Windows (Gerenciador de Credenciais real):
//
//	GOOS=windows go test -c ./internal/credenciais && ./credenciais.test.exe
func TestCofreNativo(t *testing.T) {
	c := Nativo()
	alvo := "SITPOSSE-7/teste-automatizado"
	_ = c.Apagar(alvo)
	if _, err := c.Ler(alvo); !errors.Is(err, ErrNaoEncontrada) {
		t.Fatalf("ler inexistente: %v", err)
	}
	want := Credencial{Usuario: "sh-cliente-ção", Segredo: "s3gr3d0/çá+="}
	if err := c.Gravar(alvo, want); err != nil {
		t.Fatal(err)
	}
	got, err := c.Ler(alvo)
	if err != nil || got != want {
		t.Fatalf("lido %+v, %v; esperado %+v", got, err, want)
	}
	if err := c.Apagar(alvo); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Ler(alvo); !errors.Is(err, ErrNaoEncontrada) {
		t.Fatalf("depois de apagar: %v", err)
	}
	if err := c.Apagar(alvo); err != nil {
		t.Errorf("apagar de novo deve ser silencioso: %v", err)
	}
}
