package composicao

import (
	"os"
	"path/filepath"
	"testing"

	"sitposse7/desktop/internal/cartografia/geo"
)

// ret: retângulo em torno de um ponto, com meia largura em metros de projeção.
func ret(p geo.Ponto, meia float64) geo.Retangulo {
	x, y := geo.XY(p)
	return geo.Retangulo{MinX: x - meia, MinY: y - meia, MaxX: x + meia, MaxY: y + meia}
}

func TestCobreExigeGeralESetor(t *testing.T) {
	a, b := geo.Ponto{Lat: -10.40, Lon: -62.50}, geo.Ponto{Lat: -10.80, Lon: -62.05}
	meio := geo.Ponto{Lat: -10.60, Lon: -62.27}
	geral := ret(meio, 80000)
	sem := &Composicao{Folhas: geo.Setorizacao{Geral: geo.Folha{Retangulo: geral}}}
	if !sem.Cobre([]geo.Ponto{a, b, meio}) {
		t.Error("sem setores, basta a geral")
	}
	if sem.Cobre([]geo.Ponto{{Lat: -12, Lon: -62.5}}) {
		t.Error("ponto fora da geral")
	}
	com := &Composicao{Folhas: geo.Setorizacao{Geral: geo.Folha{Retangulo: geral},
		Setores: []geo.Folha{{Rotulo: "A", Retangulo: ret(a, 3000)}, {Rotulo: "B", Retangulo: ret(b, 3000)}}}}
	if !com.Cobre([]geo.Ponto{a, b}) {
		t.Error("cada ponto no seu setor")
	}
	// Na geral, mas fora de todos os setores: sumiria das folhas de setor.
	if com.Cobre([]geo.Ponto{a, meio}) {
		t.Error("ponto fora dos setores deveria impedir")
	}
}

func TestCorrespondePelaChaveOuPeloIndice(t *testing.T) {
	s := &Servico{Dir: t.TempDir()}
	cfg := ConfigFolhas(func(r float64) geo.ConfigFolhas { return geo.ConfigPadrao(r) })
	pts := []geo.Ponto{{Lat: -10.40, Lon: -62.50}, {Lat: -10.42, Lon: -62.47}}
	movidos := []geo.Ponto{{Lat: -10.405, Lon: -62.50}, {Lat: -10.42, Lon: -62.47}}
	id := "0123456789abcdef0123456789abcdef"

	// Composição nova: a chave gravada nela.
	c := &Composicao{ID: id}
	if err := s.gravar(c, ChaveArea(pts, cfg(2))); err != nil {
		t.Fatal(err)
	}
	lida, err := s.Carregar(id)
	if err != nil || lida.Chave == "" {
		t.Fatalf("chave não gravada: %+v %v", lida, err)
	}
	if !s.Corresponde(lida, pts, cfg) || s.Corresponde(lida, movidos, cfg) {
		t.Error("correspondência pela chave")
	}
	// A ordem dos pontos não importa.
	if !s.Corresponde(lida, []geo.Ponto{pts[1], pts[0]}, cfg) {
		t.Error("ordem dos pontos")
	}

	// Composição antiga (0.8.0, sem chave): vale o índice da área.
	antiga := &Composicao{ID: id}
	if !s.Corresponde(antiga, pts, cfg) || s.Corresponde(antiga, movidos, cfg) {
		t.Error("correspondência pelo índice")
	}
	_ = os.Remove(filepath.Join(s.Dir, "composicoes", "areas", ChaveArea(pts, cfg(2))))
	if s.Corresponde(antiga, pts, cfg) {
		t.Error("sem chave e sem índice não pode corresponder")
	}
}

func TestLembrarSelecao(t *testing.T) {
	s := &Servico{Dir: t.TempDir()}
	id, sel := "0123456789abcdef0123456789abcdef", "fedcba9876543210fedcba9876543210"
	if _, ok := s.DaSelecao(sel); ok {
		t.Fatal("seleção vazia")
	}
	if err := s.gravar(&Composicao{ID: id}, "chave"); err != nil {
		t.Fatal(err)
	}
	if err := s.LembrarSelecao(sel, id); err != nil {
		t.Fatal(err)
	}
	if c, ok := s.DaSelecao(sel); !ok || c.ID != id {
		t.Fatalf("seleção lembrada: %+v", c)
	}
	if s.LembrarSelecao("../../x", id) == nil || s.LembrarSelecao(sel, "../x") == nil {
		t.Error("nomes inválidos aceitos")
	}
	if _, ok := s.DaSelecao("../areas"); ok {
		t.Error("seleção inválida lida")
	}
}
