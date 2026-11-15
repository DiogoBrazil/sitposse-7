package raster

import (
	"context"
	"math"
	"math/rand"
	"testing"

	"sitposse7/desktop/internal/cartografia/cog"
	"sitposse7/desktop/internal/cartografia/cog/cogteste"
	"sitposse7/desktop/internal/cartografia/geo"
)

// Cena sintética em UTM 20S: R = coluna%251+1, G = linha%251+1 (permite saber de que pixel
// da origem veio cada pixel da grade).
func cenaUTM(t *testing.T, l, a int, ox, oy, res float64, mexer func(*cogteste.Imagem)) *cog.Arquivo {
	t.Helper()
	im := cogteste.Gradiente(l, a)
	if mexer != nil {
		mexer(&im)
	}
	zero := 0
	arq := cogteste.Escrever([]cogteste.Imagem{im, cogteste.Reduzir(im)}, cogteste.Opcoes{Big: true, EPSG: 32720, OrigemX: ox, OrigemY: oy, ResX: res, ResY: res, NoData: &zero})
	f, err := cog.Abrir(context.Background(), cogteste.Memoria(arq))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func gradeEm(p geo.Ponto, ladoM float64, px int) Grade {
	x, y := geo.XY(p)
	k := 1 / math.Cos(p.Lat*math.Pi/180)
	h := ladoM * k / 2
	return Grade{Ret: geo.Retangulo{MinX: x - h, MaxX: x + h, MinY: y - h, MaxY: y + h}, L: px, A: px}
}

func TestReprojecaoPreservaPosicao(t *testing.T) {
	centro := geo.Ponto{Lat: -10.075, Lon: -63.169}
	z, _ := geo.ZonaEPSG(32720)
	le, no := geo.UTM(centro, z)
	ox, oy := math.Floor(le)-400, math.Floor(no)+400 // cena de 800 × 800 m com 2 m/px
	f := cenaUTM(t, 400, 400, ox, oy, 2, nil)
	g := gradeEm(centro, 500, 250) // 2 m por pixel
	a, err := (&FonteCOG{RGB: f, Cache: NovoCacheBlocos(0)}).Pintar(context.Background(), g)
	if err != nil {
		t.Fatal(err)
	}
	for _, ij := range [][2]int{{0, 0}, {125, 125}, {249, 0}, {13, 200}, {249, 249}} {
		x, y := g.Centro(float64(ij[0]), float64(ij[1]))
		e, n := geo.UTM(geo.LatLon(x, y), z)
		col, lin := (e-ox)/2-0.5, (oy-n)/2-0.5
		k := ij[1]*g.L + ij[0]
		if a.Classe[k] != ClasseOK {
			t.Fatalf("pixel %v sem dados", ij)
		}
		// Bilinear sobre o gradiente: R ≈ col mod 251 + 1, G ≈ lin mod 251 + 1 (erro < 1,5 nível ≈ 3 m).
		er, eg := math.Mod(col, 251)+1, math.Mod(lin, 251)+1
		if math.Abs(float64(a.RGB[3*k])-er) > 1.5 || math.Abs(float64(a.RGB[3*k+1])-eg) > 1.5 {
			t.Fatalf("pixel %v: R=%d G=%d, esperado ≈ %.1f %.1f", ij, a.RGB[3*k], a.RGB[3*k+1], er, eg)
		}
	}
	// Grade maior que a cena: fora dela é "sem dados".
	g2 := gradeEm(centro, 2000, 100)
	a2, err := (&FonteCOG{RGB: f, Cache: NovoCacheBlocos(0)}).Pintar(context.Background(), g2)
	if err != nil {
		t.Fatal(err)
	}
	if a2.Classe[0] != ClasseSemDado || a2.Classe[50*100+50] != ClasseOK {
		t.Fatal("cobertura fora/dentro da cena")
	}
}

func TestSCLMarcaNuvemESombra(t *testing.T) {
	centro := geo.Ponto{Lat: -10.075, Lon: -63.169}
	z, _ := geo.ZonaEPSG(32720)
	le, no := geo.UTM(centro, z)
	ox, oy := math.Floor(le)-1000, math.Floor(no)+1000
	rgb := cenaUTM(t, 200, 200, ox, oy, 10, nil)
	// SCL 20 m: metade oeste = 4 (vegetação), quadrante NE = 9 (nuvem), SE = 3 (sombra).
	scl := cogteste.Imagem{L: 100, A: 100, Amostras: 1, Pix: make([]byte, 100*100)}
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			v := byte(4)
			if x >= 50 {
				v = 9
				if y >= 50 {
					v = 3
				}
			}
			scl.Pix[y*100+x] = v
		}
	}
	arq := cogteste.Escrever([]cogteste.Imagem{scl}, cogteste.Opcoes{EPSG: 32720, OrigemX: ox, OrigemY: oy, ResX: 20, ResY: 20})
	sf, err := cog.Abrir(context.Background(), cogteste.Memoria(arq))
	if err != nil {
		t.Fatal(err)
	}
	g := gradeEm(centro, 1600, 80)
	a, err := (&FonteCOG{RGB: rgb, SCL: sf, Cache: NovoCacheBlocos(0)}).Pintar(context.Background(), g)
	if err != nil {
		t.Fatal(err)
	}
	if a.Classe[40*80+10] != ClasseOK || a.Classe[10*80+70] != ClasseNuvem || a.Classe[70*80+70] != ClasseSombra {
		t.Fatal("classes SCL", a.Classe[40*80+10], a.Classe[10*80+70], a.Classe[70*80+70])
	}
}

func TestAjustarCorRecuperaTransformacao(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	m := [3][4]float64{{1.4, -0.2, 0.1, -20}, {-0.1, 1.6, -0.15, -15}, {0.05, -0.3, 1.1, -10}}
	var ps []Par
	for i := 0; i < 3000; i++ {
		f := [3]float64{40 + r.Float64()*120, 50 + r.Float64()*100, 60 + r.Float64()*90}
		var o [3]float64
		for k := 0; k < 3; k++ {
			o[k] = m[k][0]*f[0] + m[k][1]*f[1] + m[k][2]*f[2] + m[k][3] + r.NormFloat64()*2
		}
		if i%10 == 0 { // 10% de mudanças no terreno / nuvens que escaparam
			o = [3]float64{250, 250, 250}
		}
		ps = append(ps, Par{Fonte: f, Ref: o})
	}
	c := AjustarCor(ps)
	if c.Tipo != "afim" || c.R2 < 0.9 {
		t.Fatalf("ajuste %s R²=%.2f", c.Tipo, c.R2)
	}
	for k := 0; k < 3; k++ {
		for j := 0; j < 4; j++ {
			tol := 0.03
			if j == 3 {
				tol = 3
			}
			if math.Abs(c.M[k][j]-m[k][j]) > tol {
				t.Fatalf("M[%d][%d] = %.3f, esperado %.3f", k, j, c.M[k][j], m[k][j])
			}
		}
	}
	if AjustarCor(ps[:50]).Tipo != "identidade" {
		t.Fatal("poucos pares deveria manter as cores")
	}
	// Relação não linear e invertida: o afim é recusado e vale o casamento de percentis.
	var inv []Par
	for i := 0; i < 1000; i++ {
		v := float64(i % 250)
		inv = append(inv, Par{Fonte: [3]float64{v, v, v}, Ref: [3]float64{255 - v, 255 - v, 255 - v}})
	}
	if AjustarCor(inv).Tipo != "lut" {
		t.Fatal("transformação invertida aceita")
	}
}

// fonteFixa: cor constante, com uma região sem dados (x < semDadosAteX, em m projetados).
type fonteFixa struct {
	cor          [3]uint8
	semDadosAteX float64
}

func (f fonteFixa) Pintar(_ context.Context, g Grade) (*Amostras, error) {
	a := novasAmostras(g)
	for j := 0; j < g.A; j++ {
		for i := 0; i < g.L; i++ {
			x, _ := g.Centro(float64(i), float64(j))
			k := j*g.L + i
			if x < f.semDadosAteX {
				continue
			}
			copy(a.RGB[3*k:], f.cor[:])
			a.Classe[k] = ClasseOK
		}
	}
	return a, nil
}

func TestRenderizarCelulasCosturaEBuracos(t *testing.T) {
	g := Grade{Ret: geo.Retangulo{MinX: 0, MinY: 0, MaxX: 1000, MaxY: 500}, L: 200, A: 100} // 5 m/px
	cel := NovasCelulas(g.Ret, 500)                                                         // 2 células lado a lado
	cel.Cena = []int16{0, 1}
	rc := &Receita{Cenas: []CenaReceita{{Cor: Identidade()}, {Cor: Identidade()}, {Cor: Identidade()}}, Geral: cel, Tom: TomNeutro()}
	fontes := []Fonte{
		fonteFixa{cor: [3]uint8{200, 0, 0}, semDadosAteX: 100}, // cena 0 sem dados nos primeiros 100 m
		fonteFixa{cor: [3]uint8{0, 0, 200}},
		fonteFixa{cor: [3]uint8{0, 200, 0}},
	}
	r, err := Renderizar(context.Background(), rc, fontes, g)
	if err != nil {
		t.Fatal(err)
	}
	px := func(i, j int) []uint8 { k := 4 * (j*200 + i); return r.Img.Pix[k : k+4] }
	if p := px(50, 50); p[0] != 200 || p[2] != 0 { // célula 0, dados da cena 0
		t.Fatal("célula 0", p)
	}
	if p := px(150, 50); p[2] != 200 || p[0] != 0 { // célula 1 → cena 1
		t.Fatal("célula 1", p)
	}
	if p := px(5, 50); p[2] != 200 { // buraco da cena 0 → próxima em ordem (cena 1)
		t.Fatal("buraco", p)
	}
	if p := px(99, 50); p[0] == 200 || p[2] == 200 || p[0] == 0 { // costura: mistura das duas
		t.Fatal("costura", p)
	}
	if r.Cobertura != 1 || len(r.Usadas) != 2 {
		t.Fatal(r.Cobertura, r.Usadas)
	}
}

func TestTilesEZoom(t *testing.T) {
	r := RetTile(16, 21000, 34000)
	x0, y0, x1, y1 := TilesEm(r, 16)
	if x0 != 21000 || y0 != 34000 || x1 != 21000 || y1 != 34000 {
		t.Fatal(x0, y0, x1, y1)
	}
	if z := ZoomNativo(2, -10); z != 16 {
		t.Fatal("2 m →", z)
	}
	if z := ZoomNativo(10, -10); z != 14 {
		t.Fatal("10 m →", z)
	}
	tom := CalcularTom(make([][3]uint8, 10), 1, 1)
	if tom != TomNeutro() {
		t.Fatal("tom com poucas amostras")
	}
}
