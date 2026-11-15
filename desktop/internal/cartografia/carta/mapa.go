// Package carta monta a carta-imagem (A27, ADR-0022; RN-92..94): o mapa (imagem de
// satélite + pinos + norte + escala + atribuição) e as folhas PNG e PDF.
//
// O mapa é desenhado uma única vez e entra igual no PNG e no PDF: a posição dos pinos é
// geo.Pixel, a mesma função da prévia na tela.
package carta

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/geo"
	"sitposse7/desktop/internal/fontes"
)

// Marcador é uma propriedade com coordenada válida.
type Marcador struct {
	Numero      int       `json:"numero"` // número fixo da propriedade no estudo (RN-96)
	UUID        string    `json:"uuid"`
	Nome        string    `json:"nome"`
	Responsavel string    `json:"responsavel"`
	Ponto       geo.Ponto `json:"ponto"`
	PrecisaoM   *float64  `json:"precisao_m"`
	Origem      string    `json:"origem"` // gnss | manual
	Equipe      string    `json:"equipe"`
	Jornada     string    `json:"jornada"`
}

// Proveniencia de uma parte da imagem (quadro da carta; RN-97).
type Proveniencia struct {
	Rotulo     string  `json:"rotulo"`
	Periodo    string  `json:"periodo"`
	ResolucaoM float64 `json:"resolucao_m"`
	Fracao     float64 `json:"fracao"`
	Estimada   bool    `json:"estimada"` // nuvem estimada (sem máscara da fonte)
}

// SetorIndice: retângulo de um setor desenhado na folha geral.
type SetorIndice struct {
	Rotulo    string
	Retangulo geo.Retangulo
}

// Dados de uma folha da carta.
type Dados struct {
	Tipo           string              // individual | geral
	Subtitulo      string              // nome da propriedade ou "N propriedades"
	Imagem         *cartografia.Imagem // Ret, Largura e Altura da folha; PNG se Base for nil
	Base           *image.NRGBA        // imagem já composta da folha (A30)
	Partes         []Proveniencia      // fontes da imagem; vazio = dados de Imagem (A27)
	Mosaico        bool
	Folha          string        // "" na geral; "Setor A"…
	Referencia     string        // "Setor A — ver a carta geral (folha 1)"
	Setores        []SetorIndice // índice dos setores (só na geral)
	Atribuicao     string
	Marcadores     []Marcador
	SemLocalizacao []string // propriedades que não podem ser representadas (RN-94)
	Criterios      []string
	Estudo         string
	GeradoEm       time.Time
	NomesNoMapa    bool // escrever o nome ao lado do pino (senão, só o número)
}

// Cores de alto contraste sobre vegetação e solo exposto.
var (
	corPino    = color.NRGBA{0xF2, 0xB7, 0x05, 0xFF} // ipê
	corBorda   = color.NRGBA{0x10, 0x14, 0x12, 0xFF}
	corBranca  = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
	corTinta   = color.NRGBA{0x1B, 0x26, 0x20, 0xFF}
	corCaixa   = color.NRGBA{0xFF, 0xFF, 0xFF, 0xE6}
	corSemDado = color.NRGBA{0xD9, 0xDE, 0xD6, 0xFF}
)

type tipos struct{ negrito, normal *opentype.Font }

var fontesCarta = func() tipos {
	n, _ := opentype.Parse(fontes.Bold)
	r, _ := opentype.Parse(fontes.Regular)
	return tipos{n, r}
}()

func face(f *opentype.Font, px float64) font.Face {
	fc, _ := opentype.NewFace(f, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingFull})
	return fc
}

func largura(fc font.Face, s string) float64 { return float64(font.MeasureString(fc, s)) / 64 }

// texto com halo (contorno) para ler sobre a imagem.
func texto(dst draw.Image, fc font.Face, s string, x, y float64, cor, halo color.Color, espessura int) {
	d := &font.Drawer{Dst: dst, Face: fc}
	if halo != nil {
		d.Src = image.NewUniform(halo)
		for dx := -espessura; dx <= espessura; dx++ {
			for dy := -espessura; dy <= espessura; dy++ {
				if dx*dx+dy*dy > espessura*espessura+1 {
					continue
				}
				d.Dot = fixed.P(int(math.Round(x))+dx, int(math.Round(y))+dy)
				d.DrawString(s)
			}
		}
	}
	d.Src = image.NewUniform(cor)
	d.Dot = fixed.P(int(math.Round(x)), int(math.Round(y)))
	d.DrawString(s)
}

// preencher rasteriza o caminho só no retângulo que ele ocupa (rápido em imagens grandes).
func preencher(dst draw.Image, cor color.Color, pts [][2]float64, caminho func(z *vector.Rasterizer, ox, oy float32)) {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		minX, minY = math.Min(minX, p[0]), math.Min(minY, p[1])
		maxX, maxY = math.Max(maxX, p[0]), math.Max(maxY, p[1])
	}
	r := image.Rect(int(math.Floor(minX))-1, int(math.Floor(minY))-1, int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1).Intersect(dst.Bounds())
	if r.Empty() {
		return
	}
	z := vector.NewRasterizer(r.Dx(), r.Dy())
	caminho(z, float32(r.Min.X), float32(r.Min.Y))
	z.Draw(dst, r, image.NewUniform(cor), image.Point{})
}

func circulo(dst draw.Image, cx, cy, r float64, cor color.Color) {
	const k = 0.5522847498 // aproximação de círculo por cúbicas
	preencher(dst, cor, [][2]float64{{cx - r, cy - r}, {cx + r, cy + r}}, func(z *vector.Rasterizer, ox, oy float32) {
		x, y, rr := float32(cx)-ox, float32(cy)-oy, float32(r)
		kr := float32(k) * rr
		z.MoveTo(x+rr, y)
		z.CubeTo(x+rr, y+kr, x+kr, y+rr, x, y+rr)
		z.CubeTo(x-kr, y+rr, x-rr, y+kr, x-rr, y)
		z.CubeTo(x-rr, y-kr, x-kr, y-rr, x, y-rr)
		z.CubeTo(x+kr, y-rr, x+rr, y-kr, x+rr, y)
		z.ClosePath()
	})
}

func poligono(dst draw.Image, cor color.Color, pts ...[2]float64) {
	preencher(dst, cor, pts, func(z *vector.Rasterizer, ox, oy float32) {
		z.MoveTo(float32(pts[0][0])-ox, float32(pts[0][1])-oy)
		for _, p := range pts[1:] {
			z.LineTo(float32(p[0])-ox, float32(p[1])-oy)
		}
		z.ClosePath()
	})
}

func retangulo(dst draw.Image, x0, y0, x1, y1 float64, cor color.Color) {
	poligono(dst, cor, [2]float64{x0, y0}, [2]float64{x1, y0}, [2]float64{x1, y1}, [2]float64{x0, y1})
}

// RaioPino em pixels, proporcional ao tamanho da imagem.
func RaioPino(larg int) float64 { return math.Max(11, float64(larg)/115) }

func formatarDistancia(m float64) string {
	if m >= 1000 {
		return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.1f", m/1000), "0"), ".") + " km"
	}
	return fmt.Sprintf("%.0f m", m)
}

// Mapa desenha a imagem de satélite com os pinos, o norte, a escala e a atribuição.
func Mapa(d Dados) (*image.NRGBA, error) {
	var img image.Image = d.Base
	if d.Base == nil {
		var err error
		img, err = png.Decode(bytes.NewReader(d.Imagem.PNG))
		if err != nil {
			return nil, fmt.Errorf("imagem de satélite ilegível: %w", err)
		}
	}
	w, h := d.Imagem.Largura, d.Imagem.Altura
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(m, m.Bounds(), image.NewUniform(corSemDado), image.Point{}, draw.Src)
	draw.Draw(m, m.Bounds(), img, img.Bounds().Min, draw.Over)

	u := float64(w) / 2000 // unidade relativa (tamanhos pensados para 2000 px)
	r := RaioPino(w)
	fNum := face(fontesCarta.negrito, r*1.05)
	fNome := face(fontesCarta.negrito, 26*u)
	fPeq := face(fontesCarta.normal, 20*u)
	fNorte := face(fontesCarta.negrito, 30*u)

	// Índice dos setores (na geral): contorno com halo e a letra no canto.
	for _, st := range d.Setores {
		x0, y0 := geo.Pixel(geo.LatLon(st.Retangulo.MinX, st.Retangulo.MaxY), d.Imagem.Ret, w, h)
		x1, y1 := geo.Pixel(geo.LatLon(st.Retangulo.MaxX, st.Retangulo.MinY), d.Imagem.Ret, w, h)
		for _, c := range []struct {
			cor color.Color
			e   float64
		}{{corBorda, 5 * u}, {corBranca, 2.5 * u}} {
			retangulo(m, x0-c.e/2, y0-c.e/2, x1+c.e/2, y0+c.e/2, c.cor)
			retangulo(m, x0-c.e/2, y1-c.e/2, x1+c.e/2, y1+c.e/2, c.cor)
			retangulo(m, x0-c.e/2, y0-c.e/2, x0+c.e/2, y1+c.e/2, c.cor)
			retangulo(m, x1-c.e/2, y0-c.e/2, x1+c.e/2, y1+c.e/2, c.cor)
		}
		fSet := face(fontesCarta.negrito, 30*u)
		lw := largura(fSet, st.Rotulo)
		retangulo(m, x0, y0, x0+lw+24*u, y0+42*u, corCaixa)
		texto(m, fSet, st.Rotulo, x0+12*u, y0+32*u, corTinta, nil, 0)
	}

	// Pinos: primeiro todos os círculos, depois rótulos (o rótulo não some sob outro pino).
	for _, mk := range d.Marcadores {
		x, y := geo.Pixel(mk.Ponto, d.Imagem.Ret, w, h)
		circulo(m, x, y, r+3*u, corBorda)
		circulo(m, x, y, r, corPino)
		num := fmt.Sprint(mk.Numero)
		texto(m, fNum, num, x-largura(fNum, num)/2, y+r*0.37, corBorda, nil, 0)
	}
	if d.NomesNoMapa {
		for _, mk := range d.Marcadores {
			x, y := geo.Pixel(mk.Ponto, d.Imagem.Ret, w, h)
			nome := mk.Nome
			lx := x + r + 8*u
			if lx+largura(fNome, nome) > float64(w)-10 { // perto da borda direita: à esquerda do pino
				lx = x - r - 8*u - largura(fNome, nome)
			}
			texto(m, fNome, nome, lx, y+9*u, corBranca, corBorda, int(math.Max(2, 3*u)))
		}
	}

	// Norte (canto superior direito): a imagem é EPSG:3857 com o norte para cima.
	nx, ny := float64(w)-70*u, 40*u
	retangulo(m, nx-38*u, ny-12*u, nx+38*u, ny+120*u, corCaixa)
	poligono(m, corBorda, [2]float64{nx, ny + 4*u}, [2]float64{nx + 22*u, ny + 80*u}, [2]float64{nx, ny + 64*u}, [2]float64{nx - 22*u, ny + 80*u})
	poligono(m, corBranca, [2]float64{nx, ny + 14*u}, [2]float64{nx - 14*u, ny + 70*u}, [2]float64{nx, ny + 60*u})
	texto(m, fNorte, "N", nx-largura(fNorte, "N")/2, ny+110*u, corBorda, nil, 0)

	// Escala (canto inferior esquerdo), só se tecnicamente correta (RN-92).
	esc := geo.CalcularEscala(d.Imagem.Ret, w)
	if esc.Valida {
		metros, px := geo.Barra(esc.MetrosPorPixel, 360*u)
		bx, by := 34*u, float64(h)-46*u
		retangulo(m, bx-16*u, by-48*u, bx+px+20*u, by+22*u, corCaixa)
		retangulo(m, bx, by, bx+px, by+10*u, corBorda)
		retangulo(m, bx+px/2, by+2*u, bx+px-2*u, by+8*u, corBranca)
		retangulo(m, bx, by-6*u, bx+3*u, by+10*u, corBorda)
		retangulo(m, bx+px-3*u, by-6*u, bx+px, by+10*u, corBorda)
		texto(m, fPeq, "0", bx, by-14*u, corTinta, nil, 0)
		rot := formatarDistancia(metros)
		texto(m, fPeq, rot, bx+px-largura(fPeq, rot), by-14*u, corTinta, nil, 0)
	}

	// Atribuição (canto inferior direito), exigida pela licença dos dados.
	at := d.Atribuicao
	aw := largura(fPeq, at)
	retangulo(m, float64(w)-aw-26*u, float64(h)-36*u, float64(w), float64(h), corCaixa)
	texto(m, fPeq, at, float64(w)-aw-14*u, float64(h)-12*u, corTinta, nil, 0)
	return m, nil
}

// MapaPNG devolve o mapa codificado em PNG.
func MapaPNG(d Dados) ([]byte, *image.NRGBA, error) {
	m, err := Mapa(d)
	if err != nil {
		return nil, nil, err
	}
	var b bytes.Buffer
	if err := png.Encode(&b, m); err != nil {
		return nil, nil, err
	}
	return b.Bytes(), m, nil
}
