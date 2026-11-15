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
		ms = append(ms, Marcador{Numero: i + 1, Nome: "Sítio São João " + string(rune('A'+i%26)), Ponto: p, PrecisaoM: &prec, Origem: "gnss", Equipe: "Bravo", AreaIdentificada: "Linha C-25"})
		pts = append(pts, p)
	}
	r, err := geo.Enquadrar(pts, 400, 1200, 2000, 1500)
	if err != nil {
		t.Fatal(err)
	}
	d := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	img := &cartografia.Imagem{PNG: satelite(2000, 1500), Ret: r, Largura: 2000, Altura: 1500, Data: &d, ResolucaoNativaM: 10}
	return Dados{Tipo: tipo, Subtitulo: "Teste", Imagem: img, Atribuicao: "Contains modified Copernicus Sentinel data 2026", Marcadores: ms,
		SemLocalizacao: []string{"Chácara Sem GPS"}, Criterios: []string{"Equipe: Bravo"}, Operacao: "Levantamento Linha C-25", GeradoEm: time.Date(2026, 11, 14, 22, 0, 0, 0, time.Local), NomesNoMapa: n <= 12}
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
