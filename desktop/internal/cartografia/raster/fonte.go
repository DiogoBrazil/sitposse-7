// Package raster lê as imagens candidatas numa grade Web Mercator (EPSG:3857), harmoniza
// as cores, compõe mosaicos e gera tiles (ADR-0027). Go puro, sem GDAL.
package raster

import (
	"context"
	"image"
	"image/color"
	"math"
	"sync"

	"sitposse7/desktop/internal/cartografia/cog"
	"sitposse7/desktop/internal/cartografia/geo"
)

// Grade de pixels em EPSG:3857: L × A pixels cobrindo Ret (norte para cima).
type Grade struct {
	Ret  geo.Retangulo
	L, A int
}

// Centro do pixel (i, j) em EPSG:3857.
func (g Grade) Centro(i, j float64) (x, y float64) {
	return g.Ret.MinX + (i+0.5)*g.Ret.Largura()/float64(g.L), g.Ret.MaxY - (j+0.5)*g.Ret.Altura()/float64(g.A)
}

// MetrosPorPixel reais na latitude central.
func (g Grade) MetrosPorPixel() float64 { return geo.ResolucaoEfetiva(g.Ret, g.L) }

// Classes de cada amostra.
const (
	ClasseOK      uint8 = 0
	ClasseSemDado uint8 = 1
	ClasseNuvem   uint8 = 2
	ClasseSombra  uint8 = 3
)

// Amostras: RGB e classe de cada pixel da grade.
type Amostras struct {
	L, A   int
	RGB    []uint8 // 3 por pixel
	Classe []uint8
}

func novasAmostras(g Grade) *Amostras {
	n := g.L * g.A
	a := &Amostras{L: g.L, A: g.A, RGB: make([]uint8, 3*n), Classe: make([]uint8, n)}
	for i := range a.Classe {
		a.Classe[i] = ClasseSemDado
	}
	return a
}

// Fonte pinta uma grade (sem cor ajustada: RGB como no arquivo).
type Fonte interface {
	Pintar(ctx context.Context, g Grade) (*Amostras, error)
}

// ---------- cache de blocos decodificados ----------

type chaveBloco struct {
	arq         *cog.Arquivo
	nivel, x, y int
}

type entradaBloco struct {
	pronto chan struct{}
	dados  []byte
	err    error
	uso    int64
}

// CacheBlocos guarda blocos decodificados em memória (LRU aproximado por orçamento de bytes)
// e evita baixar o mesmo bloco duas vezes ao mesmo tempo.
type CacheBlocos struct {
	MaxBytes int64
	mu       sync.Mutex
	m        map[chaveBloco]*entradaBloco
	bytes    int64
	relogio  int64
}

// NovoCacheBlocos com orçamento em bytes (padrão 192 MB).
func NovoCacheBlocos(max int64) *CacheBlocos {
	if max <= 0 {
		max = 192 << 20
	}
	return &CacheBlocos{MaxBytes: max, m: map[chaveBloco]*entradaBloco{}}
}

func (c *CacheBlocos) bloco(ctx context.Context, a *cog.Arquivo, nivel, bx, by int) ([]byte, error) {
	k := chaveBloco{a, nivel, bx, by}
	c.mu.Lock()
	c.relogio++
	if e, ok := c.m[k]; ok {
		e.uso = c.relogio
		c.mu.Unlock()
		select {
		case <-e.pronto:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return e.dados, e.err
	}
	e := &entradaBloco{pronto: make(chan struct{}), uso: c.relogio}
	c.m[k] = e
	c.mu.Unlock()
	e.dados, e.err = a.Bloco(ctx, nivel, bx, by)
	close(e.pronto)
	c.mu.Lock()
	if e.err != nil {
		delete(c.m, k) // erro não fica guardado (pode ter sido a rede)
	} else {
		c.bytes += int64(len(e.dados))
		for c.bytes > c.MaxBytes && len(c.m) > 1 {
			var velha chaveBloco
			var vUso int64 = math.MaxInt64
			for kk, ee := range c.m {
				select {
				case <-ee.pronto:
				default:
					continue
				}
				if ee.uso < vUso && kk != k {
					velha, vUso = kk, ee.uso
				}
			}
			if vUso == math.MaxInt64 {
				break
			}
			c.bytes -= int64(len(c.m[velha].dados))
			delete(c.m, velha)
		}
	}
	c.mu.Unlock()
	return e.dados, e.err
}

// Paralelo: quantos blocos são buscados ao mesmo tempo (a latência da rede domina).
const Paralelo = 8

// antecipar busca em paralelo os blocos que a grade vai ler (sem duplicar pedidos).
func (c *CacheBlocos) antecipar(ctx context.Context, a *cog.Arquivo, nivel int, cols, lins []float32) error {
	n := a.Niveis[nivel]
	vistos := map[[2]int]bool{}
	var lista [][2]int
	for k := range cols {
		c0, r0 := int(math.Floor(float64(cols[k]))), int(math.Floor(float64(lins[k])))
		for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
			cc, rr := c0+d[0], r0+d[1]
			if cc < 0 || rr < 0 || cc >= n.Largura || rr >= n.Altura {
				continue
			}
			b := [2]int{cc / n.BlocoL, rr / n.BlocoA}
			if !vistos[b] {
				vistos[b] = true
				lista = append(lista, b)
			}
		}
	}
	if len(lista) <= 1 {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	fila := make(chan [2]int)
	erros := make(chan error, Paralelo)
	var wg sync.WaitGroup
	for w := 0; w < Paralelo; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for b := range fila {
				if _, err := c.bloco(ctx, a, nivel, b[0], b[1]); err != nil {
					select {
					case erros <- err:
					default:
					}
					cancel()
					return
				}
			}
		}()
	}
envio:
	for _, b := range lista {
		select {
		case fila <- b:
		case <-ctx.Done():
			break envio
		}
	}
	close(fila)
	wg.Wait()
	select {
	case err := <-erros:
		return err
	default:
		return nil
	}
}

// ---------- fonte COG (CBERS, Sentinel-2 TCI + SCL, mosaico) ----------

// FonteCOG lê um COG RGB e, opcionalmente, a classificação de cena (SCL) do Sentinel-2.
type FonteCOG struct {
	RGB   *cog.Arquivo
	SCL   *cog.Arquivo // nil se não houver
	Cache *CacheBlocos
}

// Classes da SCL do Sentinel-2 tratadas como inválidas: 0 sem dados, 1 saturado/defeituoso,
// 3 sombra de nuvem, 8/9 nuvem (média/alta probabilidade), 10 cirro.
func classeSCL(v uint8) uint8 {
	switch v {
	case 0, 1:
		return ClasseSemDado
	case 3:
		return ClasseSombra
	case 8, 9, 10:
		return ClasseNuvem
	}
	return ClasseOK
}

// resSRC converte metros reais para as unidades do SRC do arquivo, na latitude dada.
func resSRC(epsg int, metros, lat float64) float64 {
	if epsg == 4326 {
		return metros / (111320 * math.Cos(lat*math.Pi/180))
	}
	return metros
}

// coordenadas calcula a posição (col, lin) no nível de cada pixel da grade: exata a cada 16
// pixels e interpolada entre eles (a transformação é suave nessas distâncias).
func coordenadas(a *cog.Arquivo, nivel int, g Grade) (cols, lins []float32, err error) {
	const passo = 16
	nx, ny := (g.L+passo-1)/passo+1, (g.A+passo-1)/passo+1
	cx, cy := make([]float64, nx*ny), make([]float64, nx*ny)
	for j := 0; j < ny; j++ {
		for i := 0; i < nx; i++ {
			x, y := g.Centro(float64(i*passo), float64(j*passo))
			p := geo.LatLon(x, y)
			sx, sy, e := geo.ParaSRC(a.EPSG, p)
			if e != nil {
				return nil, nil, e
			}
			c, l := a.SRCParaPixel(nivel, sx, sy)
			cx[j*nx+i], cy[j*nx+i] = c-0.5, l-0.5
		}
	}
	cols, lins = make([]float32, g.L*g.A), make([]float32, g.L*g.A)
	for j := 0; j < g.A; j++ {
		gj := j / passo
		fj := float64(j%passo) / passo
		for i := 0; i < g.L; i++ {
			gi := i / passo
			fi := float64(i%passo) / passo
			k := gj*nx + gi
			c := (cx[k]*(1-fi)+cx[k+1]*fi)*(1-fj) + (cx[k+nx]*(1-fi)+cx[k+nx+1]*fi)*fj
			l := (cy[k]*(1-fi)+cy[k+1]*fi)*(1-fj) + (cy[k+nx]*(1-fi)+cy[k+nx+1]*fi)*fj
			cols[j*g.L+i], lins[j*g.L+i] = float32(c), float32(l)
		}
	}
	return
}

// leitorNivel acessa pixels de um nível com os blocos já buscados.
type leitorNivel struct {
	ctx    context.Context
	a      *cog.Arquivo
	cache  *CacheBlocos
	nivel  int
	n      cog.Nivel
	blocos map[[2]int][]byte
	nodata int
}

func (l *leitorNivel) px(c, r int) ([]byte, bool, error) {
	if c < 0 || r < 0 || c >= l.n.Largura || r >= l.n.Altura {
		return nil, false, nil
	}
	bx, by := c/l.n.BlocoL, r/l.n.BlocoA
	b, ok := l.blocos[[2]int{bx, by}]
	if !ok {
		var err error
		b, err = l.cache.bloco(l.ctx, l.a, l.nivel, bx, by)
		if err != nil {
			return nil, false, err
		}
		l.blocos[[2]int{bx, by}] = b
	}
	s := l.a.Amostras
	i := ((r%l.n.BlocoA)*l.n.BlocoL + c%l.n.BlocoL) * s
	v := b[i : i+s]
	todosNodata := true
	for _, x := range v[:min(s, 3)] {
		if int(x) != l.nodata {
			todosNodata = false
		}
	}
	return v, !todosNodata, nil
}

func novoLeitor(ctx context.Context, a *cog.Arquivo, c *CacheBlocos, nivel int) *leitorNivel {
	nd := 0
	if a.NoData != nil {
		nd = *a.NoData
	}
	return &leitorNivel{ctx: ctx, a: a, cache: c, nivel: nivel, n: a.Niveis[nivel], blocos: map[[2]int][]byte{}, nodata: nd}
}

// Pintar lê a fonte na grade (bilinear no RGB, vizinho mais próximo na SCL).
func (f *FonteCOG) Pintar(ctx context.Context, g Grade) (*Amostras, error) {
	out := novasAmostras(g)
	lat := g.Ret.Centro().Lat
	mpp := g.MetrosPorPixel()
	nivel := f.RGB.NivelPara(resSRC(f.RGB.EPSG, mpp, lat))
	cols, lins, err := coordenadas(f.RGB, nivel, g)
	if err != nil {
		return nil, err
	}
	if err := f.Cache.antecipar(ctx, f.RGB, nivel, cols, lins); err != nil {
		return nil, err
	}
	lr := novoLeitor(ctx, f.RGB, f.Cache, nivel)
	for k := range cols {
		if k%4096 == 0 && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		c, r := float64(cols[k]), float64(lins[k])
		c0, r0 := int(math.Floor(c)), int(math.Floor(r))
		fx, fy := c-float64(c0), r-float64(r0)
		var vs [4][]byte
		var oks [4]bool
		todos := true
		for q := 0; q < 4; q++ {
			v, ok, err := lr.px(c0+q%2, r0+q/2)
			if err != nil {
				return nil, err
			}
			vs[q], oks[q] = v, ok
			todos = todos && ok
		}
		if todos {
			for ch := 0; ch < 3 && ch < f.RGB.Amostras; ch++ {
				v := (float64(vs[0][ch])*(1-fx)+float64(vs[1][ch])*fx)*(1-fy) + (float64(vs[2][ch])*(1-fx)+float64(vs[3][ch])*fx)*fy
				out.RGB[3*k+ch] = uint8(math.Round(v))
			}
			if f.RGB.Amostras == 1 {
				out.RGB[3*k+1], out.RGB[3*k+2] = out.RGB[3*k], out.RGB[3*k]
			}
			out.Classe[k] = ClasseOK
			continue
		}
		// Borda dos dados: vizinho mais próximo, se válido.
		q := int(math.Round(fy))*2 + int(math.Round(fx))
		if oks[q] {
			for ch := 0; ch < 3; ch++ {
				out.RGB[3*k+ch] = vs[q][min(ch, f.RGB.Amostras-1)]
			}
			out.Classe[k] = ClasseOK
		}
	}
	if f.SCL != nil {
		nivelS := f.SCL.NivelPara(resSRC(f.SCL.EPSG, mpp, lat))
		cs, ls, err := coordenadas(f.SCL, nivelS, g)
		if err != nil {
			return nil, err
		}
		if err := f.Cache.antecipar(ctx, f.SCL, nivelS, cs, ls); err != nil {
			return nil, err
		}
		ls2 := novoLeitor(ctx, f.SCL, f.Cache, nivelS)
		ls2.nodata = -1 // classe 0 é tratada abaixo
		for k := range cs {
			if out.Classe[k] != ClasseOK {
				continue
			}
			v, _, err := ls2.px(int(math.Floor(float64(cs[k])+0.5)), int(math.Floor(float64(ls[k])+0.5)))
			if err != nil {
				return nil, err
			}
			if v == nil {
				out.Classe[k] = ClasseSemDado
				continue
			}
			out.Classe[k] = classeSCL(v[0])
		}
	}
	return out, nil
}

// ---------- fonte de imagem já em EPSG:3857 (reserva CDSE) ----------

// FonteImagem3857 é uma imagem que cobre Ret em EPSG:3857 (alfa 0 = sem dados).
type FonteImagem3857 struct {
	Img image.Image
	Ret geo.Retangulo
}

// Pintar por interpolação bilinear.
func (f *FonteImagem3857) Pintar(ctx context.Context, g Grade) (*Amostras, error) {
	out := novasAmostras(g)
	b := f.Img.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	at := func(x, y int) (color.NRGBA, bool) {
		if x < 0 || y < 0 || x >= b.Dx() || y >= b.Dy() {
			return color.NRGBA{}, false
		}
		c := color.NRGBAModel.Convert(f.Img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
		return c, c.A > 0
	}
	for j := 0; j < g.A; j++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		for i := 0; i < g.L; i++ {
			x, y := g.Centro(float64(i), float64(j))
			c := (x-f.Ret.MinX)/f.Ret.Largura()*w - 0.5
			r := (f.Ret.MaxY-y)/f.Ret.Altura()*h - 0.5
			c0, r0 := int(math.Floor(c)), int(math.Floor(r))
			fx, fy := c-float64(c0), r-float64(r0)
			p00, ok00 := at(c0, r0)
			p10, ok10 := at(c0+1, r0)
			p01, ok01 := at(c0, r0+1)
			p11, ok11 := at(c0+1, r0+1)
			k := j*g.L + i
			if !(ok00 && ok10 && ok01 && ok11) {
				continue
			}
			for ch, v := range [][4]uint8{{p00.R, p10.R, p01.R, p11.R}, {p00.G, p10.G, p01.G, p11.G}, {p00.B, p10.B, p01.B, p11.B}} {
				val := (float64(v[0])*(1-fx)+float64(v[1])*fx)*(1-fy) + (float64(v[2])*(1-fx)+float64(v[3])*fx)*fy
				out.RGB[3*k+ch] = uint8(math.Round(val))
			}
			out.Classe[k] = ClasseOK
		}
	}
	return out, nil
}
