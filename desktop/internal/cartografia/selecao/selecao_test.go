package selecao

import (
	"reflect"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/geo"
	"sitposse7/desktop/internal/cartografia/raster"
)

var agora = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

// paineis: geral 4 × 1 células e um setor 2 × 1.
func paineis() []raster.Painel {
	g := raster.NovasCelulas(geo.Retangulo{MinX: 0, MinY: 0, MaxX: 1000, MaxY: 250}, 250)
	s := raster.NovasCelulas(geo.Retangulo{MinX: 0, MinY: 0, MaxX: 500, MaxY: 250}, 250)
	return []raster.Painel{raster.NovoPainel(g, 4, 1000), raster.NovoPainel(s, 4, 1000)}
}

// ent cria uma candidata com sem dados / nuvem por célula: geral (4) + setor (2).
func ent(id, fonte string, dias int, sd, nv []float32, r2 float64) Entrada {
	c := cartografia.Candidata{ID: id, Fonte: fonte, Data: agora.AddDate(0, 0, -dias)}
	return Entrada{Candidata: c, Av: &raster.Avaliacao{SemDados: [][]float32{sd[:4], sd[4:]}, Nuvem: [][]float32{nv[:4], nv[4:]}, Cor: raster.Cor{Tipo: "afim", R2: r2}}}
}

func TestEscolhaPrefere2mSemNuvemEDepois10m(t *testing.T) {
	limpo := []float32{0, 0, 0, 0, 0, 0}
	cbers := ent("cbers", cartografia.FonteCBERS, 30,
		[]float32{0, 0, 0, 1},                // última célula da geral fora da faixa do CBERS
		[]float32{0, 0.5, 0, 0, 0, 0.5}, 0.8) // nuvem na 2ª célula da geral e na 2ª do setor
	cbers.Av.SemDados = [][]float32{{0, 0, 0, 1}, {0, 0}}
	s2velho := ent("s2-velho", cartografia.FonteS2, 200, limpo, limpo, 0.9)
	s2novo := ent("s2-novo", cartografia.FonteS2, 20, limpo, limpo, 0.9)
	mos := ent("mosaico", cartografia.FonteMosaico, 120, limpo, limpo, 1)
	es := Escolher([]Entrada{mos, s2velho, cbers, s2novo}, paineis(), agora, Padrao())
	var ids []string
	for _, c := range es.Receita.Cenas {
		ids = append(ids, c.Candidata.ID)
	}
	if !reflect.DeepEqual(ids, []string{"cbers", "s2-novo", "mosaico"}) {
		t.Fatalf("cenas %v", ids)
	}
	if !reflect.DeepEqual(es.Receita.Geral.Cena, []int16{0, 1, 0, 1}) || !reflect.DeepEqual(es.Receita.Detalhe[0].Cena, []int16{0, 1}) {
		t.Fatalf("atribuição geral %v setor %v", es.Receita.Geral.Cena, es.Receita.Detalhe[0].Cena)
	}
	if es.Limitadas != 0 || es.SemImagem != 0 || es.FracaoNivel(1, 2) != 0.5 {
		t.Fatal(es.Limitadas, es.SemImagem, es.FracaoNivel(1, 2))
	}
	// Reproduzível: a mesma escolha com as entradas em outra ordem.
	es2 := Escolher([]Entrada{s2novo, cbers, mos, s2velho}, paineis(), agora, Padrao())
	if !reflect.DeepEqual(es.Receita, es2.Receita) {
		t.Fatal("escolha depende da ordem do catálogo")
	}
}

func TestEscolhaCelulasLimitadasESemImagem(t *testing.T) {
	nublado := []float32{0.3, 0.3, 0.3, 0.3, 0.3, 0.3}
	semNada := []float32{1, 1, 1, 1, 1, 1}
	a := ent("s2-a", cartografia.FonteS2, 10, []float32{0, 0, 0, 1, 0, 0}, nublado, 0.9)
	b := ent("s2-b", cartografia.FonteS2, 40, []float32{0, 0, 0, 1, 0, 0}, []float32{0.1, 0.1, 0.1, 0.1, 0.1, 0.1}, 0.9)
	es := Escolher([]Entrada{a, b, ent("x", cartografia.FonteCBERS, 5, semNada, semNada, 0)}, paineis(), agora, Padrao())
	if es.Limitadas != 5 || es.SemImagem != 1 {
		t.Fatalf("limitadas %d, sem imagem %d", es.Limitadas, es.SemImagem)
	}
	if es.Receita.Cenas[0].Candidata.ID != "s2-b" || es.Receita.Geral.Cena[3] != -1 {
		t.Fatalf("%+v", es.Receita.Geral)
	}
}

func TestRetalhoPequenoNaoEntra(t *testing.T) {
	// CBERS limpo em só 1 de 6 células (abaixo do mínimo de 3): não vale uma cena a mais.
	limpo := []float32{0, 0, 0, 0, 0, 0}
	cb := ent("cbers", cartografia.FonteCBERS, 10, limpo, []float32{0, 1, 1, 1, 1, 1}, 0.8)
	s2 := ent("s2", cartografia.FonteS2, 10, limpo, limpo, 0.9)
	es := Escolher([]Entrada{cb, s2}, paineis(), agora, Padrao())
	if es.Receita.Cenas[0].Candidata.ID != "s2" || len(es.Receita.Cenas) != 1 {
		t.Fatalf("%v", es.Receita.Cenas)
	}
}
