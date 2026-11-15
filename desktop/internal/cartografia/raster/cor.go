package raster

import (
	"math"
	"sort"
)

// Cor transforma o RGB de uma cena para as cores da referência (mosaico Sentinel-2 em cores
// reais). "afim": transformação afim 3×3 + deslocamento ajustada por mínimos quadrados
// robustos sobre pixels pareados com a referência (corrige a névoa e o tom arroxeado do
// CBERS-4A fusionado). "lut": casamento de percentis por canal, quando o ajuste afim não é
// confiável. Nenhuma das duas inventa feições: só remapeia cores pixel a pixel.
type Cor struct {
	Tipo string        `json:"tipo"` // identidade | afim | lut
	M    [3][4]float64 `json:"m,omitempty"`
	LUT  [3][]uint8    `json:"lut,omitempty"`
	R2   float64       `json:"r2"` // qualidade do ajuste (0..1)
}

// Identidade não altera as cores.
func Identidade() Cor { return Cor{Tipo: "identidade", R2: 1} }

func clamp8(v float64) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 255:
		return 255
	}
	return uint8(v + 0.5)
}

// Aplicar a cor a um pixel.
func (c *Cor) Aplicar(r, g, b uint8) (uint8, uint8, uint8) {
	switch c.Tipo {
	case "afim":
		x := [4]float64{float64(r), float64(g), float64(b), 1}
		var o [3]uint8
		for k := 0; k < 3; k++ {
			o[k] = clamp8(c.M[k][0]*x[0] + c.M[k][1]*x[1] + c.M[k][2]*x[2] + c.M[k][3])
		}
		return o[0], o[1], o[2]
	case "lut":
		if len(c.LUT[0]) == 256 {
			return c.LUT[0][r], c.LUT[1][g], c.LUT[2][b]
		}
	}
	return r, g, b
}

// Par de amostras: RGB da cena e da referência no mesmo lugar.
type Par struct{ Fonte, Ref [3]float64 }

// resolver4 resolve o sistema 4×4 (eliminação de Gauss com pivô parcial).
func resolver4(a [4][4]float64, b [4]float64) ([4]float64, bool) {
	for i := 0; i < 4; i++ {
		p := i
		for k := i + 1; k < 4; k++ {
			if math.Abs(a[k][i]) > math.Abs(a[p][i]) {
				p = k
			}
		}
		if math.Abs(a[p][i]) < 1e-9 {
			return [4]float64{}, false
		}
		a[i], a[p] = a[p], a[i]
		b[i], b[p] = b[p], b[i]
		for k := i + 1; k < 4; k++ {
			f := a[k][i] / a[i][i]
			for j := i; j < 4; j++ {
				a[k][j] -= f * a[i][j]
			}
			b[k] -= f * b[i]
		}
	}
	var x [4]float64
	for i := 3; i >= 0; i-- {
		s := b[i]
		for j := i + 1; j < 4; j++ {
			s -= a[i][j] * x[j]
		}
		x[i] = s / a[i][i]
	}
	return x, true
}

func ajusteAfim(ps []Par) ([3][4]float64, bool) {
	var ata [4][4]float64
	var atb [3][4]float64
	for _, p := range ps {
		x := [4]float64{p.Fonte[0], p.Fonte[1], p.Fonte[2], 1}
		for i := 0; i < 4; i++ {
			for j := 0; j < 4; j++ {
				ata[i][j] += x[i] * x[j]
			}
			for k := 0; k < 3; k++ {
				atb[k][i] += x[i] * p.Ref[k]
			}
		}
	}
	var m [3][4]float64
	for k := 0; k < 3; k++ {
		s, ok := resolver4(ata, atb[k])
		if !ok {
			return m, false
		}
		m[k] = s
	}
	return m, true
}

func residuo(m [3][4]float64, p Par) float64 {
	s := 0.0
	for k := 0; k < 3; k++ {
		v := m[k][0]*p.Fonte[0] + m[k][1]*p.Fonte[1] + m[k][2]*p.Fonte[2] + m[k][3] - p.Ref[k]
		s += v * v
	}
	return math.Sqrt(s)
}

// AjustarCor calcula a transformação da cena para a referência. Exige ao menos 200 pares.
// Iterações robustas descartam pares com resíduo acima de 2,5 × a mediana (mudanças no
// terreno, nuvens que escaparam). Se o ajuste for ruim ou extremo, usa percentis.
func AjustarCor(ps []Par) Cor {
	if len(ps) < 200 {
		return Identidade()
	}
	atual := ps
	var m [3][4]float64
	ok := false
	for it := 0; it < 4; it++ {
		m, ok = ajusteAfim(atual)
		if !ok {
			break
		}
		res := make([]float64, len(atual))
		for i, p := range atual {
			res[i] = residuo(m, p)
		}
		ord := append([]float64(nil), res...)
		sort.Float64s(ord)
		lim := math.Max(2.5*ord[len(ord)/2], 6)
		var prox []Par
		for i, p := range atual {
			if res[i] <= lim {
				prox = append(prox, p)
			}
		}
		if len(prox) < len(ps)/2 || len(prox) == len(atual) {
			break
		}
		atual = prox
	}
	if ok {
		r2 := r2Afim(m, atual)
		if r2 >= 0.35 && afimRazoavel(m) {
			return Cor{Tipo: "afim", M: m, R2: r2}
		}
	}
	return ajustePercentis(ps)
}

// afimRazoavel: ganhos positivos e limitados. A soma dos ganhos de cada canal é o quanto o
// detalhe comum às bandas (a pancromática, no CBERS fusionado) passa para a saída: abaixo
// de 0,6 o ajuste apagaria o detalhe e esconderia nuvens (visto em Ariquemes, 15/11/2026).
func afimRazoavel(m [3][4]float64) bool {
	for k := 0; k < 3; k++ {
		soma := m[k][0] + m[k][1] + m[k][2]
		if m[k][k] <= 0 || soma < 0.6 || soma > 4 || math.Abs(m[k][3]) > 120 {
			return false
		}
	}
	return true
}

func r2Afim(m [3][4]float64, ps []Par) float64 {
	var media [3]float64
	for _, p := range ps {
		for k := 0; k < 3; k++ {
			media[k] += p.Ref[k]
		}
	}
	for k := range media {
		media[k] /= float64(len(ps))
	}
	var sr, st float64
	for _, p := range ps {
		for k := 0; k < 3; k++ {
			v := m[k][0]*p.Fonte[0] + m[k][1]*p.Fonte[1] + m[k][2]*p.Fonte[2] + m[k][3]
			sr += (v - p.Ref[k]) * (v - p.Ref[k])
			st += (p.Ref[k] - media[k]) * (p.Ref[k] - media[k])
		}
	}
	if st == 0 {
		return 0
	}
	return math.Max(0, 1-sr/st)
}

// ajustePercentis casa os percentis 2, 25, 50, 75 e 98 de cada canal (interpolação linear).
func ajustePercentis(ps []Par) Cor {
	c := Cor{Tipo: "lut", R2: 0}
	qs := []float64{0.02, 0.25, 0.5, 0.75, 0.98}
	for k := 0; k < 3; k++ {
		f, r := make([]float64, len(ps)), make([]float64, len(ps))
		for i, p := range ps {
			f[i], r[i] = p.Fonte[k], p.Ref[k]
		}
		sort.Float64s(f)
		sort.Float64s(r)
		xs, ys := []float64{0}, []float64{0}
		for _, q := range qs {
			i := int(q * float64(len(f)-1))
			if f[i] > xs[len(xs)-1] {
				xs, ys = append(xs, f[i]), append(ys, math.Max(r[i], ys[len(ys)-1]))
			}
		}
		xs, ys = append(xs, 255), append(ys, 255)
		lut := make([]uint8, 256)
		for v := 0; v < 256; v++ {
			j := sort.SearchFloat64s(xs, float64(v))
			if j == 0 {
				lut[v] = clamp8(ys[0])
				continue
			}
			if j >= len(xs) {
				j = len(xs) - 1
			}
			x0, x1, y0, y1 := xs[j-1], xs[j], ys[j-1], ys[j]
			t := 0.0
			if x1 > x0 {
				t = (float64(v) - x0) / (x1 - x0)
			}
			lut[v] = clamp8(y0 + t*(y1-y0))
		}
		c.LUT[k] = lut
	}
	return c
}

// Tom é o ajuste final da composição inteira (o mesmo em todas as cenas e tiles):
// esticamento pelo brilho (mesmo ganho nos 3 canais, preservando a matiz), gama e saturação.
type Tom struct {
	Preto     float64 `json:"preto"`
	Branco    float64 `json:"branco"`
	Gama      float64 `json:"gama"`
	Saturacao float64 `json:"saturacao"`
}

// TomNeutro não muda nada.
func TomNeutro() Tom { return Tom{Preto: 0, Branco: 255, Gama: 1, Saturacao: 1} }

// CalcularTom a partir de amostras RGB válidas (já harmonizadas): brilho p0,5 → preto e
// p99,5 → branco, com limites para não exagerar o contraste.
func CalcularTom(rgb [][3]uint8, gama, saturacao float64) Tom {
	t := TomNeutro()
	t.Gama, t.Saturacao = gama, saturacao
	if len(rgb) < 100 {
		return t
	}
	lum := make([]float64, len(rgb))
	for i, p := range rgb {
		lum[i] = 0.2126*float64(p[0]) + 0.7152*float64(p[1]) + 0.0722*float64(p[2])
	}
	sort.Float64s(lum)
	t.Preto = math.Min(lum[len(lum)*5/1000], 60)
	t.Branco = math.Max(lum[len(lum)*995/1000], t.Preto+120)
	if t.Branco > 255 {
		t.Branco = 255
	}
	return t
}

// Aplicar o tom a um pixel.
func (t *Tom) Aplicar(r, g, b uint8) (uint8, uint8, uint8) {
	esc := 255 / math.Max(1, t.Branco-t.Preto)
	var c [3]float64
	for k, v := range [3]uint8{r, g, b} {
		x := math.Max(0, math.Min(1, (float64(v)-t.Preto)*esc/255))
		c[k] = 255 * math.Pow(x, t.Gama)
	}
	if t.Saturacao != 1 {
		l := 0.2126*c[0] + 0.7152*c[1] + 0.0722*c[2]
		for k := range c {
			c[k] = l + (c[k]-l)*t.Saturacao
		}
	}
	return clamp8(c[0]), clamp8(c[1]), clamp8(c[2])
}
