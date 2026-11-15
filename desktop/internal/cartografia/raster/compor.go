package raster

import (
	"context"
	"image"
	"image/color"
	"math"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/geo"
)

// Celulas é uma grade de células quadradas (lado Tam, em metros projetados 3857) alinhada à
// origem, sobre uma área. Cena guarda, por célula, o índice da cena escolhida (-1 = nenhuma).
type Celulas struct {
	Tam  float64 `json:"tam"`
	I0   int     `json:"i0"` // índice da 1ª coluna (x = I0·Tam)
	J0   int     `json:"j0"` // índice da 1ª linha de baixo (y = J0·Tam)
	NX   int     `json:"nx"`
	NY   int     `json:"ny"`
	Cena []int16 `json:"cena"`
}

// NovasCelulas cobre o retângulo com células de lado tam.
func NovasCelulas(r geo.Retangulo, tam float64) Celulas {
	i0, i1 := int(math.Floor(r.MinX/tam)), int(math.Ceil(r.MaxX/tam))
	j0, j1 := int(math.Floor(r.MinY/tam)), int(math.Ceil(r.MaxY/tam))
	c := Celulas{Tam: tam, I0: i0, J0: j0, NX: max(1, i1-i0), NY: max(1, j1-j0)}
	c.Cena = make([]int16, c.NX*c.NY)
	for i := range c.Cena {
		c.Cena[i] = -1
	}
	return c
}

// Ret da grade de células.
func (c Celulas) Ret() geo.Retangulo {
	return geo.Retangulo{MinX: float64(c.I0) * c.Tam, MinY: float64(c.J0) * c.Tam, MaxX: float64(c.I0+c.NX) * c.Tam, MaxY: float64(c.J0+c.NY) * c.Tam}
}

// Indice da célula que contém (x, y) (-1 se fora). Linhas de cima para baixo.
func (c Celulas) Indice(x, y float64) int {
	i := int(math.Floor(x/c.Tam)) - c.I0
	j := int(math.Floor(y/c.Tam)) - c.J0
	if i < 0 || j < 0 || i >= c.NX || j >= c.NY {
		return -1
	}
	return (c.NY-1-j)*c.NX + i
}

// RetCelula de um índice.
func (c Celulas) RetCelula(k int) geo.Retangulo {
	i, jTopo := k%c.NX, k/c.NX
	j := c.NY - 1 - jTopo
	x0, y0 := float64(c.I0+i)*c.Tam, float64(c.J0+j)*c.Tam
	return geo.Retangulo{MinX: x0, MinY: y0, MaxX: x0 + c.Tam, MaxY: y0 + c.Tam}
}

// CenaReceita: uma cena usada na composição, com a sua correção de cor.
type CenaReceita struct {
	Candidata     cartografia.Candidata `json:"candidata"`
	Cor           Cor                   `json:"cor"`
	NuvemEstimada bool                  `json:"nuvem_estimada"` // a nuvem desta fonte foi estimada (CBERS)
}

// Receita da composição: tudo o que é preciso para reproduzir os mesmos pixels.
type Receita struct {
	Versao  int           `json:"versao"`
	Cenas   []CenaReceita `json:"cenas"` // ordem de prioridade (para preencher buracos)
	Geral   Celulas       `json:"geral"`
	Detalhe []Celulas     `json:"detalhe"`
	Tom     Tom           `json:"tom"`
}

// VersaoReceita entra no identificador: mudar o processamento invalida composições antigas.
const VersaoReceita = 1

// cenaEm: cena atribuída à posição e a distância (m projetados) até a borda da célula mais
// próxima com outra cena (para suavizar a costura), com essa outra cena.
func (rc *Receita) cenaEm(x, y float64) (cena int, dist float64, vizinha int) {
	for _, g := range rc.Detalhe {
		if k := g.Indice(x, y); k >= 0 && g.Cena[k] >= 0 {
			return rc.vizinhanca(g, k, x, y)
		}
	}
	if k := rc.Geral.Indice(x, y); k >= 0 && rc.Geral.Cena[k] >= 0 {
		return rc.vizinhanca(rc.Geral, k, x, y)
	}
	return -1, math.Inf(1), -1
}

func (rc *Receita) vizinhanca(g Celulas, k int, x, y float64) (int, float64, int) {
	s := int(g.Cena[k])
	r := g.RetCelula(k)
	melhor, viz := math.Inf(1), -1
	for _, d := range [][3]float64{{x - r.MinX, -1, 0}, {r.MaxX - x, 1, 0}, {r.MaxY - y, 0, 1}, {y - r.MinY, 0, -1}} {
		if d[0] >= melhor {
			continue
		}
		kk := g.Indice(x+d[1]*g.Tam, y+d[2]*g.Tam)
		if kk >= 0 && g.Cena[kk] >= 0 && int(g.Cena[kk]) != s {
			melhor, viz = d[0], int(g.Cena[kk])
		}
	}
	return s, melhor, viz
}

// Resultado de uma renderização.
type Resultado struct {
	Img       *image.NRGBA
	Cobertura float64 // fração de pixels com imagem
	Usadas    []int   // índices das cenas que contribuíram
}

// Renderizar compõe a grade a partir das fontes (alinhadas com rc.Cenas): cena atribuída à
// célula, costura suavizada com a vizinha, buracos preenchidos pela ordem de prioridade,
// correção de cor de cada cena e tom final único. Pixels sem nenhuma imagem ficam com alfa 0.
func Renderizar(ctx context.Context, rc *Receita, fontes []Fonte, g Grade) (*Resultado, error) {
	n := g.L * g.A
	prim := make([]int16, n)
	viz := make([]int16, n)
	peso := make([]float32, n) // peso da cena primária na costura
	mpp := g.Ret.Largura() / float64(g.L)
	faixa := math.Max(2, math.Min(24, 40*fatorLat(g)/mpp)) * mpp // ≈ 40 m reais, em m projetados
	precisa := map[int]bool{}
	for j := 0; j < g.A; j++ {
		for i := 0; i < g.L; i++ {
			x, y := g.Centro(float64(i), float64(j))
			s, d, v := rc.cenaEm(x, y)
			k := j*g.L + i
			prim[k], viz[k], peso[k] = int16(s), -1, 1
			if s >= 0 {
				precisa[s] = true
			}
			if v >= 0 && d < faixa {
				viz[k] = int16(v)
				peso[k] = float32(0.5 + 0.5*d/faixa)
				precisa[v] = true
			}
		}
	}
	pintadas := make([]*Amostras, len(fontes))
	pintar := func(s int) error {
		if pintadas[s] != nil || s < 0 || s >= len(fontes) {
			return nil
		}
		a, err := fontes[s].Pintar(ctx, g)
		if err != nil {
			return err
		}
		pintadas[s] = a
		return nil
	}
	for s := range fontes {
		if precisa[s] {
			if err := pintar(s); err != nil {
				return nil, err
			}
		}
	}
	cor := func(s, k int) (float64, float64, float64, bool) {
		a := pintadas[s]
		if a == nil || a.Classe[k] != ClasseOK {
			return 0, 0, 0, false
		}
		r, gg, b := rc.Cenas[s].Cor.Aplicar(a.RGB[3*k], a.RGB[3*k+1], a.RGB[3*k+2])
		return float64(r), float64(gg), float64(b), true
	}
	img := image.NewNRGBA(image.Rect(0, 0, g.L, g.A))
	usadas := map[int]bool{}
	validos := 0
	buracos := []int{}
	for k := 0; k < n; k++ {
		s := int(prim[k])
		if s < 0 {
			buracos = append(buracos, k)
			continue
		}
		r, gg, b, ok := cor(s, k)
		if !ok {
			buracos = append(buracos, k)
			continue
		}
		if v := int(viz[k]); v >= 0 {
			if r2, g2, b2, ok2 := cor(v, k); ok2 {
				w := float64(peso[k])
				r, gg, b = w*r+(1-w)*r2, w*gg+(1-w)*g2, w*b+(1-w)*b2
				usadas[v] = true
			}
		}
		usadas[s] = true
		img.Pix[4*k], img.Pix[4*k+1], img.Pix[4*k+2], img.Pix[4*k+3] = clamp8(r), clamp8(gg), clamp8(b), 255
		validos++
	}
	// Buracos (sem dados ou nuvem na cena da célula): próxima cena válida, em ordem.
	for s := 0; s < len(fontes) && len(buracos) > 0; s++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := pintar(s); err != nil {
			return nil, err
		}
		resto := buracos[:0]
		for _, k := range buracos {
			r, gg, b, ok := cor(s, k)
			if !ok {
				resto = append(resto, k)
				continue
			}
			usadas[s] = true
			img.Pix[4*k], img.Pix[4*k+1], img.Pix[4*k+2], img.Pix[4*k+3] = clamp8(r), clamp8(gg), clamp8(b), 255
			validos++
		}
		buracos = resto
	}
	for k := 0; k < n; k++ {
		if img.Pix[4*k+3] == 0 {
			continue
		}
		img.Pix[4*k], img.Pix[4*k+1], img.Pix[4*k+2] = rc.Tom.Aplicar(img.Pix[4*k], img.Pix[4*k+1], img.Pix[4*k+2])
	}
	res := &Resultado{Img: img, Cobertura: float64(validos) / float64(n)}
	for s := range fontes {
		if usadas[s] {
			res.Usadas = append(res.Usadas, s)
		}
	}
	return res, nil
}

func fatorLat(g Grade) float64 { return 1 / math.Cos(g.Ret.Centro().Lat*math.Pi/180) }

// CorFundo dos pixels sem imagem na prévia (igual ao fundo do mapa).
var CorFundo = color.NRGBA{0x2b, 0x33, 0x2d, 0xff}
