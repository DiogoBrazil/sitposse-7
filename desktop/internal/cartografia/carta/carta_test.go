package carta

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/geo"
)

// imagem sintética de "satélite" (verde com variação), totalmente com dados.
func satelite(w, h int) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(60 + 40*math.Sin(float64(x)/37)*math.Cos(float64(y)/23))
			img.Set(x, y, color.NRGBA{v / 2, v + 20, v / 3, 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func dados(t *testing.T, n int, tipo string) Dados {
	t.Helper()
	var ms []Marcador
	var pts []geo.Ponto
	for i := 0; i < n; i++ {
		p := geo.Ponto{Lat: -10.40 - 0.013*float64(i%5), Lon: -62.50 + 0.017*float64(i/5) + 0.004*float64(i%3)}
		prec := 6.0
		ms = append(ms, Marcador{Numero: i + 1, Nome: "Sítio São João " + string(rune('A'+i%26)), Ponto: p, PrecisaoM: &prec, Origem: "gnss", Equipe: "Bravo", Jornada: "Jornada 1 · 10/10/2026"})
		pts = append(pts, p)
	}
	r, err := geo.Enquadrar(pts, 400, 1200, 2000, 1500)
	if err != nil {
		t.Fatal(err)
	}
	d := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	img := &cartografia.Imagem{PNG: satelite(2000, 1500), Ret: r, Largura: 2000, Altura: 1500, Data: &d, ResolucaoNativaM: 10}
	return Dados{Tipo: tipo, Subtitulo: "Teste", Imagem: img, Atribuicao: "Contains modified Copernicus Sentinel data 2026", Marcadores: ms,
		SemLocalizacao: []string{"Chácara Sem GPS"}, Criterios: []string{"Equipe: Bravo"}, Estudo: "Levantamento Linha C-25 (SITPOSSE7-A82F-91C6)", GeradoEm: time.Date(2026, 11, 14, 22, 0, 0, 0, time.Local), NomesNoMapa: n <= 12}
}

// Correspondência entre a prévia e o arquivo: o centro de cada pino desenhado está em geo.Pixel.
func TestPinoNoPixelDaProjecao(t *testing.T) {
	d := dados(t, 9, "geral")
	m, err := Mapa(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, mk := range d.Marcadores {
		x, y := geo.Pixel(mk.Ponto, d.Imagem.Ret, 2000, 1500)
		// O número escuro fica no meio; a cor do pino aparece num anel a 0,75 do raio.
		r := RaioPino(2000) * 0.75
		achou := 0
		for _, a := range []float64{0, math.Pi / 2, math.Pi, 3 * math.Pi / 2} {
			c := m.NRGBAAt(int(math.Round(x+r*math.Cos(a))), int(math.Round(y+r*math.Sin(a))))
			if c == corPino {
				achou++
			}
		}
		if achou < 3 {
			t.Errorf("pino %d: cor do pino não encontrada em volta de (%.1f, %.1f)", mk.Numero, x, y)
		}
		// Fora do pino (1,6 raio), não é a cor do pino.
		if c := m.NRGBAAt(int(x+RaioPino(2000)*1.6), int(y)); c == corPino {
			t.Errorf("pino %d maior que o esperado", mk.Numero)
		}
	}
}

func TestFolhas(t *testing.T) {
	for _, c := range []struct {
		nome string
		d    Dados
	}{{"individual", dados(t, 1, "individual")}, {"geral", dados(t, 9, "geral")}, {"geral-grande", dados(t, 40, "geral")}} {
		t.Run(c.nome, func(t *testing.T) {
			mp, m, err := MapaPNG(c.d)
			if err != nil {
				t.Fatal(err)
			}
			folha, err := FolhaPNG(c.d, m)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := png.Decode(bytes.NewReader(folha)); err != nil {
				t.Fatal(err)
			}
			pdf, err := FolhaPDF(c.d, m)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
				t.Fatal("PDF inválido")
			}
			if c.nome == "geral" && !regexp.MustCompile(`/Count 2\b`).Match(pdf) {
				t.Error("carta geral deve ter a página de coordenadas")
			}
			if dir := os.Getenv("SITPOSSE7_AMOSTRAS"); dir != "" {
				_ = os.WriteFile(filepath.Join(dir, "carta-"+c.nome+"-mapa.png"), mp, 0o644)
				_ = os.WriteFile(filepath.Join(dir, "carta-"+c.nome+".png"), folha, 0o644)
				_ = os.WriteFile(filepath.Join(dir, "carta-"+c.nome+".pdf"), pdf, 0o644)
			}
		})
	}
}

func TestInformacoes(t *testing.T) {
	d := dados(t, 3, "geral")
	info := map[string]string{}
	for _, p := range Informacoes(d) {
		info[p[0]] = p[1]
	}
	if info["Data da imagem"] != "Aquisição de 28/10/2026" {
		t.Errorf("data = %q", info["Data da imagem"])
	}
	d.Imagem.Data = nil
	d.Imagem.De, d.Imagem.Ate = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	for _, p := range Informacoes(d) {
		if p[0] == "Data da imagem" && p[1] != "Mosaico de aquisições entre 01/08/2026 e 30/10/2026 (menor nebulosidade)" {
			t.Errorf("mosaico = %q (nunca uma data única)", p[1])
		}
	}
	if info["Sem localização"] == "" || info["Fonte"] == "" || info["Referência"] == "" {
		t.Errorf("faltam campos obrigatórios: %v", info)
	}
}

// A30: carta com folha geral + setores (PDF multipágina), quadro de proveniência do mosaico,
// índice de setores na geral e numeração igual em todas as folhas.
func TestCartaMultifolhaEProveniencia(t *testing.T) {
	g := dados(t, 9, "geral")
	g.Partes = []Proveniencia{
		{Rotulo: "CBERS-4A/WPM (INPE)", Periodo: "13/10/2026", ResolucaoM: 2, Fracao: 0.94, Estimada: true},
		{Rotulo: "Sentinel-2 L2A (Copernicus/INPE)", Periodo: "16/09/2026", ResolucaoM: 10, Fracao: 0.06},
		{Rotulo: "Mosaico Sentinel-2 trimestral (INPE)", Periodo: "01/04/2026 a 30/06/2026", ResolucaoM: 10, Fracao: 0},
	}
	g.Mosaico = true
	r := g.Imagem.Ret
	setor := geo.Retangulo{MinX: r.MinX, MaxX: (r.MinX + r.MaxX) / 2, MinY: (r.MinY + r.MaxY) / 2, MaxY: r.MaxY}
	g.Setores = []SetorIndice{{Rotulo: "A", Retangulo: setor}}
	info := map[string]string{}
	for _, p := range Informacoes(g) {
		info[p[0]] = p[1]
	}
	if !strings.Contains(info["Mosaico"], "não é uma fotografia de uma única data") || !strings.Contains(info["Mosaico"], "CBERS-4A/WPM (INPE), 13/10/2026, 2 m (94% da área)") ||
		strings.Contains(info["Mosaico"], "trimestral") || info["Nuvens"] == "" || !strings.HasPrefix(info["Resolução"], "2 m") {
		t.Fatalf("informações: %v", info)
	}
	if !strings.Contains(notaImagem(g), "resolução nominal de 2 m") {
		t.Fatal(notaImagem(g))
	}
	mg, err := Mapa(g)
	if err != nil {
		t.Fatal(err)
	}
	s := g
	s.Imagem = &cartografia.Imagem{PNG: g.Imagem.PNG, Ret: setor, Largura: 2000, Altura: 1500}
	s.Setores, s.Folha, s.Referencia = nil, "Setor A", "Setor A (folha 2 de 2)"
	s.Marcadores = nil
	for _, m := range g.Marcadores {
		if setor.Contem(m.Ponto) {
			s.Marcadores = append(s.Marcadores, m) // mesmos números da geral
		}
	}
	ms, err := Mapa(s)
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := CartaPDF([]Pagina{{g, mg}, {s, ms}})
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(pdf, []byte("/Type /Page\n")) + bytes.Count(pdf, []byte("/Type /Page ")); n < 3 {
		t.Fatalf("PDF com %d páginas; esperado geral + setor + tabela", n)
	}
	// Na geral, o contorno do setor A aparece (pixel branco/escuro na borda do setor).
	x, y := geo.Pixel(geo.LatLon(setor.MaxX, setor.MinY), r, 2000, 1500)
	c := mg.NRGBAAt(int(x), int(y)-30)
	if c.R != c.G || c.G != c.B {
		t.Fatalf("contorno do setor ausente: %v", c)
	}
}
