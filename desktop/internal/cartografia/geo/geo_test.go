package geo

import (
	"errors"
	"math"
	"testing"
)

func perto(t *testing.T, nome string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %.9f, esperado %.9f (±%g)", nome, got, want, tol)
	}
}

// Valores de referência do EPSG:3857 (calculados de forma independente pela fórmula
// esférica, em Python, fora deste código): Porto Velho e o meridiano/equador.
func TestProjecaoConhecida(t *testing.T) {
	x, y := XY(Ponto{Lat: 0, Lon: 0.0001})
	perto(t, "x(0,0.0001)", x, 11.131949079, 1e-6)
	perto(t, "y(0,…)", y, 0, 1e-9)
	x, y = XY(Ponto{Lat: -8.76194, Lon: -63.90389})   // Porto Velho
	perto(t, "x Porto Velho", x, -7113748.4945, 0.01) // = −63,90389° × 111.319,4908 m/°
	perto(t, "y Porto Velho", y, -979198.7574, 0.01)
	x, _ = XY(Ponto{Lon: 180})
	perto(t, "x(180)", x, math.Pi*Raio, 1e-6)
	// Ida e volta.
	for _, p := range []Ponto{{-10.4321, -62.4567}, {85, 179.9}, {-45.1, 10}} {
		q := LatLon(XY(p))
		perto(t, "lat ida-volta", q.Lat, p.Lat, 1e-10)
		perto(t, "lon ida-volta", q.Lon, p.Lon, 1e-10)
	}
}

func TestValidar(t *testing.T) {
	for _, p := range []Ponto{{-91, 0}, {91, 0}, {0, -181}, {0, 181}, {0, 0}, {86, 0}, {math.NaN(), 1}} {
		if err := Validar(p); !errors.Is(err, ErrCoordenada) {
			t.Errorf("%v deveria ser inválido", p)
		}
	}
	if err := Validar(Ponto{-10.43, -62.47}); err != nil {
		t.Error(err)
	}
}

// A ordem lat/lon trocada levaria Rondônia para o Oceano Índico: o teste fixa a ordem.
func TestOrdemLatLon(t *testing.T) {
	r, _ := Enquadrar([]Ponto{{Lat: -10.43, Lon: -62.47}}, 500, 1000, 400, 300)
	c := r.Centro()
	perto(t, "lat do centro", c.Lat, -10.43, 1e-9)
	perto(t, "lon do centro", c.Lon, -62.47, 1e-9)
	if r.MaxX >= 0 || r.MaxY >= 0 {
		t.Error("Rondônia deve ficar no quadrante oeste/sul (x<0, y<0)")
	}
}

func TestPixelDoPino(t *testing.T) {
	pts := []Ponto{{-10.40, -62.50}, {-10.45, -62.42}, {-10.43, -62.47}}
	r, err := Enquadrar(pts, 300, 0, 2000, 1500)
	if err != nil {
		t.Fatal(err)
	}
	// Proporção da imagem respeitada (pixels quadrados).
	perto(t, "proporção", r.Largura()/r.Altura(), 2000.0/1500, 1e-9)
	for _, p := range pts {
		px, py := Pixel(p, r, 2000, 1500)
		if px < 0 || px > 2000 || py < 0 || py > 1500 {
			t.Errorf("%v fora da imagem: (%.1f, %.1f)", p, px, py)
		}
		q := PontoDoPixel(px, py, r, 2000, 1500)
		perto(t, "lat do pixel", q.Lat, p.Lat, 1e-10)
		perto(t, "lon do pixel", q.Lon, p.Lon, 1e-10)
	}
	// Norte para cima: o ponto mais ao norte tem y menor; o mais a leste, x maior.
	_, yN := Pixel(pts[0], r, 2000, 1500)
	_, yS := Pixel(pts[1], r, 2000, 1500)
	xO, _ := Pixel(pts[0], r, 2000, 1500)
	xL, _ := Pixel(pts[1], r, 2000, 1500)
	if !(yN < yS && xO < xL) {
		t.Error("orientação errada (norte para cima, leste à direita)")
	}
	// Centro do retângulo cai no centro da imagem.
	cx, cy := Pixel(r.Centro(), r, 2000, 1500)
	perto(t, "centro x", cx, 1000, 1e-6)
	perto(t, "centro y", cy, 750, 1e-6)
}

// Interpolar linearmente em graus erraria o pino; em latitudes altas o erro é visível.
func TestNaoEhInterpolacaoLinearEmGraus(t *testing.T) {
	r, _ := Enquadrar([]Ponto{{60, 10}, {62, 12}}, 0, 0, 1000, 1000)
	p := Ponto{61, 11}
	_, py := Pixel(p, r, 1000, 1000)
	so, ne := r.SO(), r.NE()
	pyLinear := (ne.Lat - p.Lat) / (ne.Lat - so.Lat) * 1000
	if math.Abs(py-pyLinear) < 3 {
		t.Errorf("Mercator e linear em graus deveriam diferir em latitude alta (%.2f vs %.2f)", py, pyLinear)
	}
}

func TestEscalaEBarra(t *testing.T) {
	// 10 km reais de largura em 2000 px, perto de 10°S: 5 m/px.
	c := Ponto{-10.43, -62.47}
	cx, cy := XY(c)
	k := 1 / math.Cos(c.Lat*math.Pi/180)
	r := Retangulo{MinX: cx - 5000*k, MaxX: cx + 5000*k, MinY: cy - 3750*k, MaxY: cy + 3750*k}
	e := CalcularEscala(r, 2000)
	perto(t, "m/px", e.MetrosPorPixel, 5, 0.01)
	if !e.Valida {
		t.Errorf("escala de carta local deveria ser válida (variação %.4f)", e.Variacao)
	}
	// A distância real entre as bordas leste e oeste no centro confere com a escala.
	perto(t, "largura real (haversine)", DistanciaM(LatLon(r.MinX, cy), LatLon(r.MaxX, cy)), 10000, 10000*0.002)
	m, px := Barra(5, 300)
	if m != 1000 || px != 200 {
		t.Errorf("barra = %v m / %v px; esperado 1000 m / 200 px", m, px)
	}
	m, _ = Barra(2.3, 250) // alvo 575 m → 500 m
	if m != 500 {
		t.Errorf("barra = %v m; esperado 500", m)
	}
	// Carta continental: a escala varia demais e não pode ter barra (RN-92).
	g, _ := Enquadrar([]Ponto{{5, -60}, {-30, -50}}, 0, 0, 2000, 1500)
	if CalcularEscala(g, 2000).Valida {
		t.Error("carta de 35° de latitude não pode ter barra de escala")
	}
}

func TestAgrupar(t *testing.T) {
	pts := []Ponto{
		{-10.40, -62.50}, {-10.42, -62.48}, // grupo A (perto)
		{-10.40, -61.50}, // grupo B (~110 km a leste)
		{-10.41, -62.49}, // grupo A
	}
	g := Agrupar(pts, 40000)
	if len(g) != 2 {
		t.Fatalf("grupos = %v; esperado 2", g)
	}
	if len(g[0]) != 3 || len(g[1]) != 1 || g[1][0] != 2 {
		t.Errorf("grupos = %v", g)
	}
	// Todos os pontos aparecem exatamente uma vez.
	visto := map[int]int{}
	for _, gr := range g {
		for _, i := range gr {
			visto[i]++
		}
	}
	if len(visto) != len(pts) {
		t.Error("ponto perdido ou repetido no agrupamento")
	}
	// Determinístico.
	if g2 := Agrupar(pts, 40000); len(g2) != len(g) || g2[0][0] != g[0][0] {
		t.Error("agrupamento não determinístico")
	}
}

func TestGarantirLargura(t *testing.T) {
	c := Ponto{-10.43, -62.47}
	x, y := XY(c)
	r := Retangulo{MinX: x - 40, MaxX: x + 40, MinY: y - 30, MaxY: y + 30} // ~80 m: zoom máximo da prévia
	g := GarantirLargura(r, 1500)
	perto(t, "largura real", DistanciaM(LatLon(g.MinX, y), LatLon(g.MaxX, y)), 1500, 2)
	perto(t, "proporção", g.Largura()/g.Altura(), 4.0/3, 1e-9)
	cc := g.Centro()
	perto(t, "centro lat", cc.Lat, c.Lat, 1e-9)
	perto(t, "centro lon", cc.Lon, c.Lon, 1e-9)
	if big := (Retangulo{MinX: x - 5000, MaxX: x + 5000, MinY: y - 3750, MaxY: y + 3750}); GarantirLargura(big, 1500) != big {
		t.Error("retângulo já largo não deve mudar")
	}
}
