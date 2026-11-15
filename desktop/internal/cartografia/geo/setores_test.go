package geo

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
)

func TestUTMReferencias(t *testing.T) {
	// Valores de referência (série de Snyder/USGS e exemplo do GeoConvert/GeographicLib).
	casos := []struct {
		p      Ponto
		epsg   int
		le, no float64
	}{
		{Ponto{-10.075, -63.169}, 32720, 481482.5967, 8886290.4484},
		{Ponto{-10.5, -62.1}, 32720, 598485.2425, 8839163.9378},
		{Ponto{33.3, 44.4}, 32638, 444140.5449, 3684706.3556},
	}
	for _, c := range casos {
		z, _ := ZonaEPSG(c.epsg)
		le, no := UTM(c.p, z)
		if math.Abs(le-c.le) > 0.01 || math.Abs(no-c.no) > 0.01 {
			t.Errorf("%v: %.4f %.4f, esperado %.4f %.4f", c.p, le, no, c.le, c.no)
		}
		q := DeUTM(le, no, z)
		if math.Abs(q.Lat-c.p.Lat) > 1e-10 || math.Abs(q.Lon-c.p.Lon) > 1e-10 {
			t.Errorf("ida e volta %v → %v", c.p, q)
		}
	}
	// Ida e volta em toda a zona e além dela (imagens podem ter pixels a alguns graus).
	r := rand.New(rand.NewSource(1))
	z := Zona{20, true}
	for i := 0; i < 2000; i++ {
		p := Ponto{Lat: -30 + r.Float64()*35, Lon: -69 + r.Float64()*12}
		le, no := UTM(p, z)
		q := DeUTM(le, no, z)
		if DistanciaM(p, q) > 1e-6 {
			t.Fatalf("ida e volta %v → %v", p, q)
		}
	}
	if _, _, err := ParaSRC(31983, Ponto{-10, -63}); err == nil {
		t.Fatal("EPSG não suportado aceito")
	}
}

// grade gera pontos espalhados num retângulo de larguraKm × alturaKm a partir de (lat, lon).
func grade(lat, lon, larguraKm, alturaKm float64, n int, semente int64) []Ponto {
	r := rand.New(rand.NewSource(semente))
	dLat := alturaKm / 111.32
	dLon := larguraKm / (111.32 * math.Cos(lat*math.Pi/180))
	pts := make([]Ponto, n)
	for i := range pts {
		pts[i] = Ponto{Lat: lat - r.Float64()*dLat, Lon: lon + r.Float64()*dLon}
	}
	return pts
}

func conferirCobertura(t *testing.T, pts []Ponto, s *Setorizacao) {
	t.Helper()
	for i, p := range pts {
		if !s.Geral.Retangulo.Contem(p) {
			t.Fatalf("ponto %d fora da geral", i)
		}
	}
	if len(s.Setores) == 0 {
		return
	}
	visto := make([]int, len(pts))
	for _, f := range s.Setores {
		for _, i := range f.Indices {
			visto[i]++
			if !f.Retangulo.Contem(pts[i]) {
				t.Fatalf("ponto %d fora do setor %s", i, f.Rotulo)
			}
		}
		// Mesma escala em todos os setores.
		if math.Abs(f.Retangulo.Largura()-s.Setores[0].Retangulo.Largura()) > 1e-6 {
			t.Fatal("setores com escalas diferentes")
		}
	}
	for i, n := range visto {
		if n != 1 {
			t.Fatalf("ponto %d atribuído a %d setores", i, n)
		}
	}
}

func TestSetorizarCenarios(t *testing.T) {
	cfg := ConfigPadrao(2)
	casos := []struct {
		nome        string
		pts         []Ponto
		minS, maxS  int
		motivoVazio bool
	}{
		{"5 próximas", grade(-10.075, -63.169, 0.8, 0.6, 5, 1), 0, 0, true},
		{"20 em 2 × 10 km (norte-sul)", grade(-10.0, -63.2, 2, 10, 20, 2), 2, 6, false},
		{"20 em 10 × 2 km (leste-oeste)", grade(-10.0, -63.2, 10, 2, 20, 3), 2, 6, false},
		{"100 em 10 × 30 km", grade(-10.0, -63.3, 10, 30, 100, 4), 6, 40, false},
	}
	for _, c := range casos {
		s, err := Setorizar(c.pts, cfg)
		if err != nil {
			t.Fatal(c.nome, err)
		}
		if len(s.Setores) < c.minS || len(s.Setores) > c.maxS || (s.Motivo == "") != c.motivoVazio {
			t.Errorf("%s: %d setores (motivo %q), esperado %d..%d", c.nome, len(s.Setores), s.Motivo, c.minS, c.maxS)
		}
		conferirCobertura(t, c.pts, s)
		// Determinístico.
		s2, _ := Setorizar(c.pts, cfg)
		if !reflect.DeepEqual(s, s2) {
			t.Errorf("%s: resultado não determinístico", c.nome)
		}
	}
}

func TestSetorizarGruposDistantes(t *testing.T) {
	a := grade(-10.0, -63.2, 1.5, 1, 6, 5)
	b := grade(-10.6, -62.5, 1.5, 1, 6, 6) // cerca de 100 km do primeiro grupo
	pts := append(append([]Ponto{}, a...), b...)
	s, err := Setorizar(pts, ConfigPadrao(2))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Setores) != 2 {
		t.Fatalf("%d setores, esperado 2 (um por grupo)", len(s.Setores))
	}
	conferirCobertura(t, pts, s)
	if s.Setores[0].Rotulo != "A" || s.Setores[1].Rotulo != "B" {
		t.Fatal("rótulos", s.Setores[0].Rotulo, s.Setores[1].Rotulo)
	}
	// O vazio entre os grupos não entra em nenhum setor.
	meio := Ponto{Lat: -10.3, Lon: -62.85}
	for _, f := range s.Setores {
		if f.Retangulo.Contem(meio) {
			t.Fatal("setor cobre o vazio entre os grupos")
		}
	}
}

func TestSetorizarRepetidosEImagem10m(t *testing.T) {
	p := Ponto{-10.075, -63.169}
	pts := []Ponto{p, p, p, {-10.0751, -63.1691}}
	s, err := Setorizar(pts, ConfigPadrao(2))
	if err != nil || len(s.Setores) != 0 {
		t.Fatalf("repetidos: %v %v", s, err)
	}
	conferirCobertura(t, pts, s)
	// Com imagem de 10 m, a mesma área extensa precisa de menos setores.
	g := grade(-10.0, -63.3, 10, 30, 100, 4)
	s2, _ := Setorizar(g, ConfigPadrao(2))
	s10, _ := Setorizar(g, ConfigPadrao(10))
	if len(s10.Setores) >= len(s2.Setores) {
		t.Fatalf("10 m: %d setores; 2 m: %d", len(s10.Setores), len(s2.Setores))
	}
	if _, err := Setorizar(nil, ConfigPadrao(2)); err == nil {
		t.Fatal("aceitou lista vazia")
	}
	if _, err := Setorizar([]Ponto{{0, 0}}, ConfigPadrao(2)); err == nil {
		t.Fatal("aceitou (0, 0)")
	}
}

func TestSuspeitasERotulos(t *testing.T) {
	pts := grade(-10.0, -63.2, 2, 2, 10, 7)
	pts = append(pts, Ponto{-10.0, -60.0}, Ponto{40.0, -63.0})
	got := Suspeitas(pts, 50)
	if !reflect.DeepEqual(got, []int{10, 11}) {
		t.Fatal(got)
	}
	if rotuloSetor(0) != "A" || rotuloSetor(25) != "Z" || rotuloSetor(26) != "AA" || rotuloSetor(27) != "AB" {
		t.Fatal("rótulos")
	}
}
