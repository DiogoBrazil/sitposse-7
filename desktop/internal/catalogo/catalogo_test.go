// Package catalogo só contém o teste que confere o catálogo semeado pela migração
// com a fixture compartilhada (o Android confere a mesma fixture).
package catalogo

import (
	"context"
	"path/filepath"
	"testing"

	"sitposse7/desktop/internal/banco"
	"sitposse7/desktop/internal/testutil"
)

func TestCatalogoIgualAFixture(t *testing.T) {
	var fx struct {
		Versao int `json:"versao_catalogo"`
		Itens  []struct {
			Codigo       string `json:"codigo"`
			Categoria    string `json:"categoria"`
			NomeExibicao string `json:"nome_exibicao"`
			Ordem        int    `json:"ordem"`
		} `json:"itens"`
	}
	testutil.LerFixture(t, "catalogo/itens.v1.json", &fx)
	b, err := banco.Abrir(context.Background(), filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Fechar()
	rows, err := b.Leitura.Query(`SELECT codigo, categoria, nome_exibicao, ordem, versao_catalogo FROM catalogo_itens ORDER BY categoria, ordem`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type linha struct {
		codigo, categoria, nome string
		ordem, versao           int
	}
	banco := map[string]linha{}
	for rows.Next() {
		var l linha
		rows.Scan(&l.codigo, &l.categoria, &l.nome, &l.ordem, &l.versao)
		banco[l.codigo] = l
	}
	if len(banco) != len(fx.Itens) {
		t.Fatalf("banco tem %d itens, fixture %d", len(banco), len(fx.Itens))
	}
	for _, it := range fx.Itens {
		l, ok := banco[it.Codigo]
		if !ok || l.categoria != it.Categoria || l.nome != it.NomeExibicao || l.ordem != it.Ordem || l.versao != fx.Versao {
			t.Errorf("divergência em %s: banco=%+v fixture=%+v", it.Codigo, l, it)
		}
	}
}
