package geo

import (
	"errors"
	"math"
	"sort"
)

// Setorização das cartas (RN-97, ADR-0027): uma folha geral com todas as propriedades e,
// quando ela fica com muito menos detalhe que a imagem ou com pinos sobrepostos, folhas de
// setor de tamanho fixo (mesma escala em todas), calculadas de forma determinística.

// ConfigFolhas define as dimensões e os limites da setorização.
type ConfigFolhas struct {
	LarguraPx, AlturaPx int     // mapa de cada folha (proporção da folha)
	ResolucaoM          float64 // resolução nominal da imagem (2 ou 10 m)
	FatorSetor          float64 // metros por pixel do setor = ResolucaoM × FatorSetor
	LimiteGeral         float64 // setoriza se metros/pixel da geral > ResolucaoM × LimiteGeral
	MargemFracao        float64 // margem em cada lado, fração da largura do retângulo
	MargemMinM          float64 // margem mínima (m reais)
	MeiaLarguraMinM     float64 // meia-largura mínima da carta geral (m reais)
	RaioPinoPx          float64 // raio do pino no mapa da folha
}

// ConfigPadrao para o mapa 2000 × 1500 da folha atual.
func ConfigPadrao(resolucaoM float64) ConfigFolhas {
	return ConfigFolhas{LarguraPx: 2000, AlturaPx: 1500, ResolucaoM: resolucaoM, FatorSetor: 1.5, LimiteGeral: 2.5,
		MargemFracao: 0.08, MargemMinM: 300, MeiaLarguraMinM: 1500, RaioPinoPx: math.Max(11, 2000.0/115)}
}

// Folha é a área de uma página da carta e os pontos atribuídos a ela.
type Folha struct {
	Rotulo    string    `json:"rotulo"` // "" na geral; "A", "B"… nos setores
	Retangulo Retangulo `json:"retangulo"`
	Indices   []int     `json:"indices"` // pontos atribuídos (cada ponto em um único setor)
}

// Setorizacao: a folha geral e os setores (vazio quando a geral basta).
type Setorizacao struct {
	Geral         Folha   `json:"geral"`
	Setores       []Folha `json:"setores"`
	MetrosPxGeral float64 `json:"metros_px_geral"`
	MetrosPxSetor float64 `json:"metros_px_setor"`
	Motivo        string  `json:"motivo"` // por que houve setores ("" sem setores)
}

// metrosReais converte uma distância projetada (3857) em metros reais na latitude dada.
func metrosReais(dProj, lat float64) float64 { return dProj / fatorEscala(lat) }

func latMedia(pts []Ponto) float64 {
	min, max := math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		min, max = math.Min(min, p.Lat), math.Max(max, p.Lat)
	}
	return (min + max) / 2
}

// Setorizar calcula a folha geral e, se preciso, os setores. Os índices referem-se a pts.
func Setorizar(pts []Ponto, c ConfigFolhas) (*Setorizacao, error) {
	if len(pts) == 0 {
		return nil, errors.New("nenhum ponto com coordenada válida")
	}
	for _, p := range pts {
		if err := Validar(p); err != nil {
			return nil, err
		}
	}
	todos := make([]int, len(pts))
	for i := range todos {
		todos[i] = i
	}
	// Margem proporcional à extensão (no maior lado), com mínimo absoluto.
	ext := extensaoM(pts)
	margem := math.Max(c.MargemMinM, c.MargemFracao*ext)
	rg, err := Enquadrar(pts, margem, c.MeiaLarguraMinM, c.LarguraPx, c.AlturaPx)
	if err != nil {
		return nil, err
	}
	s := &Setorizacao{Geral: Folha{Retangulo: rg, Indices: todos}, Setores: []Folha{}}
	s.MetrosPxGeral = CalcularEscala(rg, c.LarguraPx).MetrosPorPixel
	s.MetrosPxSetor = c.ResolucaoM * c.FatorSetor
	if s.MetrosPxGeral <= s.MetrosPxSetor {
		return s, nil // a geral já tem o detalhe de um setor
	}
	switch {
	case s.MetrosPxGeral > c.ResolucaoM*c.LimiteGeral:
		s.Motivo = "detalhe"
	case sobrepoemResolvivel(pts, s.MetrosPxGeral, s.MetrosPxSetor, c.RaioPinoPx):
		s.Motivo = "sobreposicao"
	default:
		return s, nil
	}
	s.Setores = setores(pts, c, s.MetrosPxSetor)
	if len(s.Setores) <= 1 {
		// Um setor só cobriria o mesmo que a geral, com mais detalhe: vira a própria geral.
		if len(s.Setores) == 1 {
			s.Geral.Retangulo = s.Setores[0].Retangulo
			s.MetrosPxGeral = s.MetrosPxSetor
		}
		s.Setores, s.Motivo = []Folha{}, ""
	}
	return s, nil
}

// extensaoM: maior lado (m reais) do retângulo dos pontos.
func extensaoM(pts []Ponto) float64 {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		x, y := XY(p)
		minX, maxX, minY, maxY = math.Min(minX, x), math.Max(maxX, x), math.Min(minY, y), math.Max(maxY, y)
	}
	return metrosReais(math.Max(maxX-minX, maxY-minY), latMedia(pts))
}

// sobrepoemResolvivel: há pinos que se sobrepõem na geral mas não num setor.
func sobrepoemResolvivel(pts []Ponto, mpxGeral, mpxSetor, raio float64) bool {
	lim := 2.2 * raio
	for i := range pts {
		for j := i + 1; j < len(pts); j++ {
			d := DistanciaM(pts[i], pts[j])
			if d/mpxGeral < lim && d/mpxSetor >= lim {
				return true
			}
		}
	}
	return false
}

// setores cobre os pontos com janelas de tamanho fixo (a escala do setor), de forma gulosa e
// determinística: a semente é o ponto livre mais a oeste (depois o mais ao norte); entre
// posições candidatas fixas, fica a que cobre mais pontos livres; depois a janela é
// recentrada nos seus pontos. Grupos distantes viram setores próprios.
func setores(pts []Ponto, c ConfigFolhas, mpxSetor float64) []Folha {
	lat := latMedia(pts)
	k := fatorEscala(lat)
	w := float64(c.LarguraPx) * mpxSetor * k // largura projetada
	h := float64(c.AlturaPx) * mpxSetor * k
	m := math.Max(c.MargemMinM, c.MargemFracao*float64(c.LarguraPx)*mpxSetor) * k
	if m > w/4 {
		m = w / 4
	}
	type xy struct{ x, y float64 }
	p := make([]xy, len(pts))
	for i, q := range pts {
		x, y := XY(q)
		p[i] = xy{x, y}
	}
	ordem := make([]int, len(pts))
	for i := range ordem {
		ordem[i] = i
	}
	sort.SliceStable(ordem, func(a, b int) bool {
		if p[ordem[a]].x != p[ordem[b]].x {
			return p[ordem[a]].x < p[ordem[b]].x
		}
		return p[ordem[a]].y > p[ordem[b]].y
	})
	livre := make([]bool, len(pts))
	for i := range livre {
		livre[i] = true
	}
	dentro := func(r Retangulo, i int) bool {
		const eps = 1e-6
		return p[i].x >= r.MinX+m-eps && p[i].x <= r.MaxX-m+eps && p[i].y >= r.MinY+m-eps && p[i].y <= r.MaxY-m+eps
	}
	var out []Folha
	for _, s := range ordem {
		if !livre[s] {
			continue
		}
		var melhorIdx []int
		x0 := p[s].x - m
		for _, f := range []float64{0.5, 0.25, 0.75, 0, 1} { // posição vertical da semente na janela
			top := p[s].y + m + f*(h-2*m)
			r := Retangulo{MinX: x0, MaxX: x0 + w, MaxY: top, MinY: top - h}
			var idx []int
			for _, i := range ordem {
				if livre[i] && dentro(r, i) {
					idx = append(idx, i)
				}
			}
			if len(idx) > len(melhorIdx) {
				melhorIdx = idx
			}
		}
		// Recentra a janela nos pontos (mesmo tamanho: mesma escala em todos os setores).
		minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
		for _, i := range melhorIdx {
			livre[i] = false
			minX, maxX = math.Min(minX, p[i].x), math.Max(maxX, p[i].x)
			minY, maxY = math.Min(minY, p[i].y), math.Max(maxY, p[i].y)
		}
		cx, cy := (minX+maxX)/2, (minY+maxY)/2
		sort.Ints(melhorIdx)
		out = append(out, Folha{Retangulo: Retangulo{MinX: cx - w/2, MaxX: cx + w/2, MinY: cy - h/2, MaxY: cy + h/2}, Indices: melhorIdx})
	}
	// Rótulos em ordem de leitura: norte → sul, oeste → leste (faixas da altura de um setor).
	sort.SliceStable(out, func(a, b int) bool {
		fa, fb := math.Floor(-out[a].Retangulo.MaxY/h), math.Floor(-out[b].Retangulo.MaxY/h)
		if fa != fb {
			return fa < fb
		}
		return out[a].Retangulo.MinX < out[b].Retangulo.MinX
	})
	for i := range out {
		out[i].Rotulo = rotuloSetor(i)
	}
	return out
}

// rotuloSetor: A…Z, AA, AB…
func rotuloSetor(i int) string {
	s := ""
	for {
		s = string(rune('A'+i%26)) + s
		i = i/26 - 1
		if i < 0 {
			return s
		}
	}
}

// Suspeitas devolve os índices de coordenadas fora do retângulo do Brasil (com folga) ou a
// mais de limiteKm da mediana dos pontos. Não são descartadas: só geram aviso.
func Suspeitas(pts []Ponto, limiteKm float64) []int {
	if len(pts) == 0 {
		return nil
	}
	lats, lons := make([]float64, len(pts)), make([]float64, len(pts))
	for i, p := range pts {
		lats[i], lons[i] = p.Lat, p.Lon
	}
	sort.Float64s(lats)
	sort.Float64s(lons)
	med := Ponto{Lat: lats[len(lats)/2], Lon: lons[len(lons)/2]}
	var out []int
	for i, p := range pts {
		foraBrasil := p.Lat < -34.5 || p.Lat > 6 || p.Lon < -75 || p.Lon > -28
		if foraBrasil || DistanciaM(p, med) > limiteKm*1000 {
			out = append(out, i)
		}
	}
	return out
}
