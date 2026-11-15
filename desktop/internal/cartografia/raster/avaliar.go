package raster

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"math"
	"sort"

	"sitposse7/desktop/internal/cartografia/geo"
)

// Painel de avaliação: células e a grade de amostragem (K × K pixels por célula).
type Painel struct {
	Cel   Celulas
	Grade Grade
	K     int
}

// NovoPainel com K amostras por lado de célula, limitado a maxLado pixels por lado.
func NovoPainel(cel Celulas, k, maxLado int) Painel {
	for k > 2 && (cel.NX*k > maxLado || cel.NY*k > maxLado) {
		k--
	}
	return Painel{Cel: cel, Grade: Grade{Ret: cel.Ret(), L: cel.NX * k, A: cel.NY * k}, K: k}
}

// LimiaresNuvem para estimar nuvem e sombra quando a fonte não traz máscara (CBERS-4A).
// O brilho da cena (com a cor harmonizada) é previsto a partir do brilho da referência sem
// nuvens por um ajuste linear robusto; nuvem é um desvio para mais, e sombra para menos,
// maior que max(DiferencaMin, Sigmas × dispersão normal). É uma ESTIMATIVA e é apresentada
// assim ao operador (RN-95).
type LimiaresNuvem struct {
	DiferencaMin float64 // desvio mínimo de brilho (0–255)
	Sigmas       float64 // desvio em múltiplos da dispersão robusta (1,4826 × MAD)
	SaturacaoMax float64 // nuvem é pouco saturada
	RefMinSombra float64 // sombra só onde a referência não é escura (água, mata fechada)
}

// LimiaresPadrao (calibrados no recorte de Ariquemes de 15/11/2026: nuvem fina sobre pasto,
// lida no overview de ≈ 16 m com brilho ≈ 145 contra ≈ 100 da referência).
func LimiaresPadrao() LimiaresNuvem {
	return LimiaresNuvem{DiferencaMin: 25, Sigmas: 4, SaturacaoMax: 0.3, RefMinSombra: 60}
}

// modeloBrilho: brilho previsto = A·brilho da referência + B; Lim = desvio tolerado.
type modeloBrilho struct{ A, B, Lim float64 }

func ajustarBrilho(pares []Par, cor *Cor, lim LimiaresNuvem) modeloBrilho {
	if len(pares) < 50 {
		return modeloBrilho{A: 1, Lim: math.Inf(1)}
	}
	xs, ys := make([]float64, len(pares)), make([]float64, len(pares))
	for i, p := range pares {
		r, g, b := cor.Aplicar(uint8(p.Fonte[0]), uint8(p.Fonte[1]), uint8(p.Fonte[2]))
		xs[i], ys[i] = lum(p.Ref[0], p.Ref[1], p.Ref[2]), lum(float64(r), float64(g), float64(b))
	}
	usar := make([]bool, len(xs))
	for i := range usar {
		usar[i] = true
	}
	m := modeloBrilho{A: 1}
	for it := 0; it < 4; it++ {
		var sx, sy, sxx, sxy, n float64
		for i := range xs {
			if usar[i] {
				sx, sy, sxx, sxy, n = sx+xs[i], sy+ys[i], sxx+xs[i]*xs[i], sxy+xs[i]*ys[i], n+1
			}
		}
		if d := n*sxx - sx*sx; n > 10 && math.Abs(d) > 1e-9 {
			m.A = (n*sxy - sx*sy) / d
			m.B = (sy - m.A*sx) / n
		}
		res := make([]float64, 0, len(xs))
		for i := range xs {
			res = append(res, math.Abs(ys[i]-(m.A*xs[i]+m.B)))
		}
		ord := append([]float64(nil), res...)
		sort.Float64s(ord)
		sigma := 1.4826 * ord[len(ord)/2]
		m.Lim = math.Max(lim.DiferencaMin, lim.Sigmas*sigma)
		for i := range usar {
			usar[i] = res[i] <= m.Lim
		}
	}
	return m
}

// Avaliacao de uma candidata nos painéis.
type Avaliacao struct {
	SemDados      [][]float32 `json:"-"` // fração por célula, por painel
	Nuvem         [][]float32 `json:"-"` // nuvem + sombra, por célula
	Cor           Cor         `json:"cor"`
	NuvemEstimada bool        `json:"nuvem_estimada"`
}

func lum(r, g, b float64) float64 { return 0.2126*r + 0.7152*g + 0.0722*b }

// Avaliar pinta a candidata nos painéis, ajusta a cor contra a referência (ref, pintada nos
// mesmos painéis; nil quando a candidata é a própria referência) e mede, por célula, a fração
// sem dados e a de nuvem/sombra (SCL quando houver; senão estimada se estimar=true).
func Avaliar(ctx context.Context, f Fonte, ref []*Amostras, paineis []Painel, estimar bool, lim LimiaresNuvem) (*Avaliacao, []*Amostras, error) {
	pint := make([]*Amostras, len(paineis))
	var pares []Par
	for i, p := range paineis {
		a, err := f.Pintar(ctx, p.Grade)
		if err != nil {
			return nil, nil, err
		}
		pint[i] = a
		if ref == nil {
			continue
		}
		r := ref[i]
		for j := 0; j < p.Grade.A; j += 2 {
			for x := (j / 2) % 2; x < p.Grade.L; x += 2 {
				k := j*p.Grade.L + x
				if a.Classe[k] == ClasseOK && r.Classe[k] == ClasseOK {
					pares = append(pares, Par{
						Fonte: [3]float64{float64(a.RGB[3*k]), float64(a.RGB[3*k+1]), float64(a.RGB[3*k+2])},
						Ref:   [3]float64{float64(r.RGB[3*k]), float64(r.RGB[3*k+1]), float64(r.RGB[3*k+2])},
					})
				}
			}
		}
	}
	av := &Avaliacao{Cor: Identidade(), NuvemEstimada: estimar && ref != nil}
	var mb modeloBrilho
	if ref != nil {
		av.Cor = AjustarCor(pares)
		if av.NuvemEstimada {
			mb = ajustarBrilho(pares, &av.Cor, lim)
		}
	}
	for i, p := range paineis {
		a := pint[i]
		nc := p.Cel.NX * p.Cel.NY
		sd, nv, tot := make([]float32, nc), make([]float32, nc), make([]float32, nc)
		for j := 0; j < p.Grade.A; j++ {
			for x := 0; x < p.Grade.L; x++ {
				k := j*p.Grade.L + x
				c := (j/p.K)*p.Cel.NX + x/p.K
				tot[c]++
				cl := a.Classe[k]
				if cl == ClasseOK && av.NuvemEstimada && ref[i].Classe[k] == ClasseOK {
					cl = estimarClasse(&av.Cor, mb, a.RGB[3*k:3*k+3], ref[i].RGB[3*k:3*k+3], lim)
				}
				switch cl {
				case ClasseSemDado:
					sd[c]++
				case ClasseNuvem, ClasseSombra:
					nv[c]++
				}
			}
		}
		for c := range tot {
			if tot[c] > 0 {
				sd[c] /= tot[c]
				nv[c] /= tot[c]
			}
		}
		av.SemDados = append(av.SemDados, sd)
		av.Nuvem = append(av.Nuvem, nv)
	}
	return av, pint, nil
}

func estimarClasse(cor *Cor, mb modeloBrilho, px, rf []uint8, lim LimiaresNuvem) uint8 {
	r, g, b := cor.Aplicar(px[0], px[1], px[2])
	lc := lum(float64(r), float64(g), float64(b))
	lr := lum(float64(rf[0]), float64(rf[1]), float64(rf[2]))
	mx := math.Max(float64(r), math.Max(float64(g), float64(b)))
	mn := math.Min(float64(r), math.Min(float64(g), float64(b)))
	sat := 0.0
	if mx > 0 {
		sat = (mx - mn) / mx
	}
	desvio := lc - (mb.A*lr + mb.B)
	switch {
	case desvio > mb.Lim && sat < lim.SaturacaoMax:
		return ClasseNuvem
	case desvio < -mb.Lim && lr > lim.RefMinSombra:
		return ClasseSombra
	}
	return ClasseOK
}

// ---------- tiles XYZ (EPSG:3857) ----------

// LadoTile em pixels.
const LadoTile = 256

const meioMundo = math.Pi * geo.Raio

// RetTile: retângulo 3857 do tile (z, x, y) no esquema XYZ.
func RetTile(z, x, y int) geo.Retangulo {
	tam := 2 * meioMundo / float64(int(1)<<z)
	return geo.Retangulo{MinX: -meioMundo + float64(x)*tam, MaxX: -meioMundo + float64(x+1)*tam,
		MaxY: meioMundo - float64(y)*tam, MinY: meioMundo - float64(y+1)*tam}
}

// TilesEm: faixa de tiles (inclusiva) que cobre o retângulo no zoom z.
func TilesEm(r geo.Retangulo, z int) (x0, y0, x1, y1 int) {
	tam := 2 * meioMundo / float64(int(1)<<z)
	n := int(1)<<z - 1
	cl := func(v int) int { return max(0, min(n, v)) }
	return cl(int(math.Floor((r.MinX + meioMundo) / tam))), cl(int(math.Floor((meioMundo - r.MaxY) / tam))),
		cl(int(math.Floor((r.MaxX + meioMundo - 1e-6) / tam))), cl(int(math.Floor((meioMundo - r.MinY - 1e-6) / tam)))
}

// ZoomNativo: menor zoom cujo pixel (m reais, na latitude) não é maior que mppReal.
func ZoomNativo(mppReal, lat float64) int {
	for z := 0; z <= 20; z++ {
		px := 2 * meioMundo / float64(int(1)<<z) / LadoTile * math.Cos(lat*math.Pi/180)
		if px <= mppReal*1.25 {
			return z
		}
	}
	return 20
}

// JPEG da imagem; pixels sem imagem recebem a cor de fundo do mapa.
func JPEG(img *image.NRGBA, qualidade int) ([]byte, error) {
	rgba := image.NewRGBA(img.Rect)
	for k := 0; k < len(img.Pix); k += 4 {
		if img.Pix[k+3] == 0 {
			rgba.Pix[k], rgba.Pix[k+1], rgba.Pix[k+2] = CorFundo.R, CorFundo.G, CorFundo.B
		} else {
			rgba.Pix[k], rgba.Pix[k+1], rgba.Pix[k+2] = img.Pix[k], img.Pix[k+1], img.Pix[k+2]
		}
		rgba.Pix[k+3] = 255
	}
	var b bytes.Buffer
	err := jpeg.Encode(&b, rgba, &jpeg.Options{Quality: qualidade})
	return b.Bytes(), err
}
