package nucleo

import (
	"context"
	"testing"

	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/correcoes"
	"sitposse7/desktop/internal/estatisticas"
)

// RN-96: número fixo da propriedade no estudo, atribuído na chegada, estável e nunca reutilizado.
func TestNumeracaoFixaNoEstudo(t *testing.T) {
	a := novoAmb(t)
	a.parear()
	ctx := context.Background()
	numero := func(uuid string) int {
		return a.contar(`SELECT coalesce((SELECT numero FROM numeracao_propriedades WHERE propriedade_uuid = ?), 0)`, uuid)
	}

	p1, p2 := a.prop("Sítio Jaraguá", 1), a.prop("Chácara Boa Vista", 1)
	l := a.novoLote([]contratos.Propriedade{p1, p2}, nil, nil)
	a.putLote(l)
	a.importarConfirmado(l.LoteUUID)
	n1, n2 := numero(p1.UUID), numero(p2.UUID)
	if n1 < 1 || n2 < 1 || n1 == n2 || n1+n2 != 3 {
		t.Fatalf("números iniciais: %d e %d", n1, n2)
	}

	// Nova versão do mesmo cadastro não renumera.
	p1.Rev, p1.NomePropriedade = 2, "Sítio Jaraguá II"
	l2 := a.novoLote([]contratos.Propriedade{p1}, nil, nil)
	a.putLote(l2)
	a.importarConfirmado(l2.LoteUUID)
	if numero(p1.UUID) != n1 {
		t.Fatal("número mudou com a nova versão")
	}

	// Excluir na Central não libera o número.
	d, err := (&correcoes.Servico{B: a.n.B}).Carregar(ctx, p2.UUID)
	if err != nil {
		t.Fatal(err)
	}
	if err := (&correcoes.Servico{B: a.n.B}).Excluir(ctx, p2.UUID, d.Token); err != nil {
		t.Fatal(err)
	}
	p3 := a.prop("Fazenda Nova", 1)
	l3 := a.novoLote([]contratos.Propriedade{p3}, nil, nil)
	a.putLote(l3)
	a.importarConfirmado(l3.LoteUUID)
	if n3 := numero(p3.UUID); n3 != 3 {
		t.Fatalf("número da terceira propriedade = %d, esperado 3 (o 2 não é reaproveitado)", n3)
	}

	// O número aparece nas famílias das estatísticas (cartas e relatórios).
	base, err := estatisticas.Carregar(ctx, a.n.B.Leitura)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range base.Familias {
		if f.Numero != numero(f.UUID) || f.Numero == 0 {
			t.Fatalf("família %s com número %d", f.Nome, f.Numero)
		}
	}

	// O número não pode ser alterado nem apagado.
	if _, err := a.n.B.Escrita.Exec(`UPDATE numeracao_propriedades SET numero = 99 WHERE propriedade_uuid = ?`, p1.UUID); err == nil {
		t.Fatal("número alterado")
	}
	if _, err := a.n.B.Escrita.Exec(`DELETE FROM numeracao_propriedades WHERE propriedade_uuid = ?`, p1.UUID); err == nil {
		t.Fatal("número apagado")
	}
}
