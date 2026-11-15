// Package geo é a matemática das cartas-imagem (ADR-0022): validação de coordenadas,
// projeção Web Mercator (EPSG:3857), enquadramento, posição dos pinos em pixel, escala
// e agrupamento de pontos distantes.
//
// A imagem é pedida ao provedor em EPSG:3857 e a prévia (Leaflet) usa a mesma projeção.
// Por isso a posição de um ponto na imagem é linear nas coordenadas projetadas (metros),
// e nunca nas coordenadas geográficas (graus): é a mesma função na tela e no arquivo.
package geo

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Raio da esfera do Web Mercator (EPSG:3857), em metros.
const Raio = 6378137.0

// LatMax é o limite de latitude do Web Mercator.
const LatMax = 85.05112877980659

// ErrCoordenada indica coordenada ausente ou fora da faixa válida.
var ErrCoordenada = errors.New("coordenada inválida")

// Ponto geográfico em graus decimais, WGS 84 (EPSG:4326).
type Ponto struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Validar confere a faixa (latitude −90..90, longitude −180..180), a projetabilidade e
// rejeita (0, 0), que no Brasil é sempre um valor não preenchido.
func Validar(p Ponto) error {
	switch {
	case math.IsNaN(p.Lat) || math.IsNaN(p.Lon) || math.IsInf(p.Lat, 0) || math.IsInf(p.Lon, 0):
		return fmt.Errorf("%w: não numérica", ErrCoordenada)
	case p.Lat < -90 || p.Lat > 90:
		return fmt.Errorf("%w: latitude %.6f fora de −90..90", ErrCoordenada, p.Lat)
	case p.Lon < -180 || p.Lon > 180:
		return fmt.Errorf("%w: longitude %.6f fora de −180..180", ErrCoordenada, p.Lon)
	case math.Abs(p.Lat) > LatMax:
		return fmt.Errorf("%w: latitude %.6f fora da projeção Web Mercator", ErrCoordenada, p.Lat)
	case p.Lat == 0 && p.Lon == 0:
		return fmt.Errorf("%w: (0, 0)", ErrCoordenada)
	}
	return nil
}

// XY projeta para EPSG:3857 (metros).
func XY(p Ponto) (x, y float64) {
	x = Raio * p.Lon * math.Pi / 180
	y = Raio * math.Log(math.Tan(math.Pi/4+p.Lat*math.Pi/360))
	return
}

// LatLon é a inversa de XY.
func LatLon(x, y float64) Ponto {
	return Ponto{Lat: (2*math.Atan(math.Exp(y/Raio)) - math.Pi/2) * 180 / math.Pi, Lon: x / Raio * 180 / math.Pi}
}

// Retangulo em EPSG:3857 (metros).
type Retangulo struct {
	MinX float64 `json:"min_x"`
	MinY float64 `json:"min_y"`
	MaxX float64 `json:"max_x"`
	MaxY float64 `json:"max_y"`
}

func (r Retangulo) Largura() float64 { return r.MaxX - r.MinX }
func (r Retangulo) Altura() float64  { return r.MaxY - r.MinY }

// Centro geográfico do retângulo.
func (r Retangulo) Centro() Ponto { return LatLon((r.MinX+r.MaxX)/2, (r.MinY+r.MaxY)/2) }

// SO e NE: cantos sudoeste e nordeste em graus (para o catálogo e o Leaflet).
func (r Retangulo) SO() Ponto { return LatLon(r.MinX, r.MinY) }
func (r Retangulo) NE() Ponto { return LatLon(r.MaxX, r.MaxY) }

// Contem diz se o ponto está dentro do retângulo.
func (r Retangulo) Contem(p Ponto) bool {
	x, y := XY(p)
	return x >= r.MinX && x <= r.MaxX && y >= r.MinY && y <= r.MaxY
}

// fatorEscala é 1/cos(lat): quantos metros projetados há em um metro real nessa latitude.
func fatorEscala(lat float64) float64 { return 1 / math.Cos(lat*math.Pi/180) }

// Enquadrar devolve o retângulo que contém todos os pontos, com margem (metros reais,
// em cada lado) e meia-largura mínima (metros reais), ajustado à proporção larg:alt
// da imagem para que os pixels sejam quadrados.
func Enquadrar(pts []Ponto, margemM, minMeiaLarguraM float64, larg, alt int) (Retangulo, error) {
	if len(pts) == 0 {
		return Retangulo{}, errors.New("nenhum ponto com coordenada válida")
	}
	r := Retangulo{MinX: math.Inf(1), MinY: math.Inf(1), MaxX: math.Inf(-1), MaxY: math.Inf(-1)}
	for _, p := range pts {
		if err := Validar(p); err != nil {
			return Retangulo{}, err
		}
		x, y := XY(p)
		r.MinX, r.MaxX = math.Min(r.MinX, x), math.Max(r.MaxX, x)
		r.MinY, r.MaxY = math.Min(r.MinY, y), math.Max(r.MaxY, y)
	}
	k := fatorEscala(r.Centro().Lat)
	m := margemM * k
	r.MinX, r.MaxX, r.MinY, r.MaxY = r.MinX-m, r.MaxX+m, r.MinY-m, r.MaxY+m
	minW := 2 * minMeiaLarguraM * k
	cx, cy := (r.MinX+r.MaxX)/2, (r.MinY+r.MaxY)/2
	w, h := math.Max(r.Largura(), minW), math.Max(r.Altura(), minW*float64(alt)/float64(larg))
	// Proporção da imagem: cresce a dimensão que faltar.
	if w/h > float64(larg)/float64(alt) {
		h = w * float64(alt) / float64(larg)
	} else {
		w = h * float64(larg) / float64(alt)
	}
	return Retangulo{MinX: cx - w/2, MaxX: cx + w/2, MinY: cy - h/2, MaxY: cy + h/2}, nil
}

// AjustarProporcao centraliza o retângulo r na proporção larg:alt, sem cortar nada.
func AjustarProporcao(r Retangulo, larg, alt int) Retangulo {
	cx, cy := (r.MinX+r.MaxX)/2, (r.MinY+r.MaxY)/2
	w, h := r.Largura(), r.Altura()
	if w/h > float64(larg)/float64(alt) {
		h = w * float64(alt) / float64(larg)
	} else {
		w = h * float64(larg) / float64(alt)
	}
	return Retangulo{MinX: cx - w/2, MaxX: cx + w/2, MinY: cy - h/2, MaxY: cy + h/2}
}

// GarantirLargura amplia o retângulo, mantendo o centro e a proporção, até ter pelo menos
// minM metros reais de largura. Evita pedir imagem de uma área minúscula (10 m por pixel
// esticados viram borrões e gastam cota à toa).
func GarantirLargura(r Retangulo, minM float64) Retangulo {
	k := fatorEscala(r.Centro().Lat)
	minW := minM * k
	if r.Largura() >= minW || r.Largura() <= 0 {
		return r
	}
	f := minW / r.Largura()
	cx, cy := (r.MinX+r.MaxX)/2, (r.MinY+r.MaxY)/2
	w, h := r.Largura()*f, r.Altura()*f
	return Retangulo{MinX: cx - w/2, MaxX: cx + w/2, MinY: cy - h/2, MaxY: cy + h/2}
}

// Pixel devolve a posição do ponto numa imagem larg × alt que cobre r (origem no canto
// superior esquerdo; y cresce para baixo). Exato para imagem em EPSG:3857.
func Pixel(p Ponto, r Retangulo, larg, alt int) (px, py float64) {
	x, y := XY(p)
	return (x - r.MinX) / r.Largura() * float64(larg), (r.MaxY - y) / r.Altura() * float64(alt)
}

// PontoDoPixel é a inversa de Pixel.
func PontoDoPixel(px, py float64, r Retangulo, larg, alt int) Ponto {
	return LatLon(r.MinX+px/float64(larg)*r.Largura(), r.MaxY-py/float64(alt)*r.Altura())
}

// Escala descreve a escala de uma imagem que cobre r com larg pixels.
type Escala struct {
	MetrosPorPixel float64 `json:"metros_por_pixel"` // metros reais por pixel na latitude central
	Variacao       float64 `json:"variacao"`         // variação relativa da escala entre o norte e o sul da imagem
	Valida         bool    `json:"valida"`           // barra de escala tecnicamente correta (variação < 1%; RN-92)
}

// CalcularEscala usa a latitude central; a escala do Web Mercator varia com 1/cos(lat).
func CalcularEscala(r Retangulo, larg int) Escala {
	latC := r.Centro().Lat
	mpp := r.Largura() / float64(larg) / fatorEscala(latC)
	n, s := LatLon(r.MinX, r.MaxY).Lat, LatLon(r.MinX, r.MinY).Lat
	kn, ks := fatorEscala(n), fatorEscala(s)
	v := math.Abs(kn-ks) / math.Min(kn, ks)
	return Escala{MetrosPorPixel: mpp, Variacao: v, Valida: v < 0.01}
}

// Barra escolhe uma barra de escala "redonda" (1, 2 ou 5 × 10ⁿ metros) de no máximo
// maxPx pixels. Devolve o comprimento em metros e em pixels.
func Barra(metrosPorPixel, maxPx float64) (metros, px float64) {
	alvo := metrosPorPixel * maxPx
	pot := math.Pow(10, math.Floor(math.Log10(alvo)))
	for _, f := range []float64{5, 2, 1} {
		if f*pot <= alvo {
			return f * pot, f * pot / metrosPorPixel
		}
	}
	return pot / 2, pot / 2 / metrosPorPixel
}

// DistanciaM é a distância geodésica aproximada (esfera, haversine) em metros.
func DistanciaM(a, b Ponto) float64 {
	const r = 6371008.8
	la1, la2 := a.Lat*math.Pi/180, b.Lat*math.Pi/180
	dla, dlo := la2-la1, (b.Lon-a.Lon)*math.Pi/180
	h := math.Sin(dla/2)*math.Sin(dla/2) + math.Cos(la1)*math.Cos(la2)*math.Sin(dlo/2)*math.Sin(dlo/2)
	return 2 * r * math.Asin(math.Min(1, math.Sqrt(h)))
}

// Agrupar divide os pontos em grupos cuja extensão não passa de ladoM metros reais
// (para cartas com resolução legível quando as propriedades estão muito espalhadas).
// Determinístico: varre os pontos ordenados de oeste para leste e de norte para sul.
// Devolve os índices dos pontos de cada grupo.
func Agrupar(pts []Ponto, ladoM float64) [][]int {
	idx := make([]int, len(pts))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		if pts[idx[a]].Lon != pts[idx[b]].Lon {
			return pts[idx[a]].Lon < pts[idx[b]].Lon
		}
		return pts[idx[a]].Lat > pts[idx[b]].Lat
	})
	var grupos [][]int
	usado := make([]bool, len(pts))
	for _, i := range idx {
		if usado[i] {
			continue
		}
		// Semente: o ponto mais a oeste ainda livre; o grupo cobre um quadrado de lado ladoM a partir dele.
		g := []int{}
		minLat, maxLat := pts[i].Lat, pts[i].Lat
		for _, j := range idx {
			if usado[j] {
				continue
			}
			dx := DistanciaM(Ponto{pts[i].Lat, pts[i].Lon}, Ponto{pts[i].Lat, pts[j].Lon})
			if pts[j].Lon < pts[i].Lon || dx > ladoM {
				continue
			}
			nmin, nmax := math.Min(minLat, pts[j].Lat), math.Max(maxLat, pts[j].Lat)
			if DistanciaM(Ponto{nmin, pts[j].Lon}, Ponto{nmax, pts[j].Lon}) > ladoM {
				continue
			}
			minLat, maxLat = nmin, nmax
			usado[j] = true
			g = append(g, j)
		}
		sort.Ints(g)
		grupos = append(grupos, g)
	}
	return grupos
}

// ResolucaoEfetiva: metros reais por pixel de uma imagem de larg pixels cobrindo r.
func ResolucaoEfetiva(r Retangulo, larg int) float64 { return CalcularEscala(r, larg).MetrosPorPixel }
