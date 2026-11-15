// Package composicao orquestra a imagem das cartas (ADR-0027): busca as candidatas no INPE,
// avalia cada uma sobre a área das propriedades, escolhe a composição, ajusta o tom,
// pré-gera os tiles da prévia e as imagens das folhas, e guarda tudo na pasta de dados para
// reabrir sem internet. Se o INPE estiver inacessível, usa a composição guardada da mesma
// área ou, se configurada, a reserva do Copernicus (CDSE).
package composicao

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/cog"
	"sitposse7/desktop/internal/cartografia/geo"
	"sitposse7/desktop/internal/cartografia/raster"
	"sitposse7/desktop/internal/cartografia/selecao"
)

// Catalogo de candidatas (o INPE; substituível nos testes).
type Catalogo interface {
	Candidatas(ctx context.Context, bbox [4]float64, agora time.Time) ([]cartografia.Candidata, error)
}

// Reserva obtém uma imagem pronta em EPSG:3857 (CDSE). nil = sem reserva configurada.
type Reserva func(ctx context.Context, ret geo.Retangulo, larg, alt int) (*cartografia.Imagem, error)

// Progresso informa fases e contadores reais (nada de percentuais fictícios).
type Progresso struct {
	Fase       string `json:"fase"`
	Feito      int    `json:"feito"`
	Total      int    `json:"total"`
	BytesRede  int64  `json:"bytes_rede"`
	BytesCache int64  `json:"bytes_cache"`
}

// Parte da composição, para o quadro de proveniência da carta e o painel da tela.
type Parte struct {
	Rotulo        string  `json:"rotulo"`
	Periodo       string  `json:"periodo"`
	ResolucaoM    float64 `json:"resolucao_m"`
	Fracao        float64 `json:"fracao"` // fração das células atribuídas a esta cena
	NuvemEstimada bool    `json:"nuvem_estimada"`
	Atribuicao    string  `json:"atribuicao"`
	Fonte         string  `json:"fonte"`
}

// Composicao guardada.
type Composicao struct {
	ID         string          `json:"id"`
	Receita    raster.Receita  `json:"receita"`
	Folhas     geo.Setorizacao `json:"folhas"`
	ZGeral     int             `json:"z_geral"`
	ZNativo    int             `json:"z_nativo"`
	ZMin       int             `json:"z_min"`
	Partes     []Parte         `json:"partes"`
	Mosaico    bool            `json:"mosaico"` // mais de uma cena ou data
	ResolucaoM float64         `json:"resolucao_m"`
	Avisos     []string        `json:"avisos"`
	CriadaEm   time.Time       `json:"criada_em"`
	Reserva    bool            `json:"reserva"` // veio do CDSE
	Pontos     []geo.Ponto     `json:"-"`
	Guardada   bool            `json:"guardada"` // reaproveitada sem consultar a internet
}

// Servico de composição.
type Servico struct {
	Dir       string // pasta "cartografia" da Central
	Catalogo  Catalogo
	HTTP      *http.Client
	Reserva   Reserva
	Criterios selecao.Criterios
	Limiares  raster.LimiaresNuvem
	Agora     func() time.Time

	cache    *raster.CacheBlocos
	once     sync.Once
	mu       sync.Mutex
	arquivos map[string]*cog.Arquivo
	leitores []*cog.HTTP
	comps    map[string]*carregada
}

type carregada struct {
	comp   *Composicao
	fontes []raster.Fonte
}

func (s *Servico) iniciar() {
	s.once.Do(func() {
		s.cache = raster.NovoCacheBlocos(0)
		s.arquivos = map[string]*cog.Arquivo{}
		s.comps = map[string]*carregada{}
		if s.Criterios == (selecao.Criterios{}) {
			s.Criterios = selecao.Padrao()
		}
		if s.Limiares == (raster.LimiaresNuvem{}) {
			s.Limiares = raster.LimiaresPadrao()
		}
	})
}

func (s *Servico) agora() time.Time {
	if s.Agora != nil {
		return s.Agora()
	}
	return time.Now()
}

func (s *Servico) dirComp(id string) string { return filepath.Join(s.Dir, "composicoes", id) }

func (s *Servico) bytesRede() (rede, cache int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, h := range s.leitores {
		rede += h.Baixados.Load()
		cache += h.DoCache.Load()
	}
	return
}

// abrir um COG (cabeçalho uma vez por execução; blocos pelo cache em disco).
func (s *Servico) abrir(ctx context.Context, url string) (*cog.Arquivo, error) {
	s.mu.Lock()
	if a, ok := s.arquivos[url]; ok {
		s.mu.Unlock()
		return a, nil
	}
	h := &cog.HTTP{URL: url, Cliente: s.HTTP, Cache: &cog.Cache{Dir: filepath.Join(s.Dir, "blocos")}}
	s.leitores = append(s.leitores, h)
	s.mu.Unlock()
	a, err := cog.Abrir(ctx, h)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.arquivos[url] = a
	s.mu.Unlock()
	return a, nil
}

func (s *Servico) fonte(ctx context.Context, c cartografia.Candidata) (raster.Fonte, error) {
	rgb, err := s.abrir(ctx, c.URLRGB)
	if err != nil {
		return nil, err
	}
	f := &raster.FonteCOG{RGB: rgb, Cache: s.cache}
	if c.URLSCL != "" {
		if f.SCL, err = s.abrir(ctx, c.URLSCL); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// ---------- preparo ----------

// ConfigFolhas devolve a configuração das folhas para a resolução da imagem.
type ConfigFolhas func(resolucaoM float64) geo.ConfigFolhas

// ChaveArea identifica a área (os pontos e a configuração) para reaproveitar composições.
func ChaveArea(pts []geo.Ponto, cfg geo.ConfigFolhas) string {
	ps := append([]geo.Ponto(nil), pts...)
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].Lat != ps[j].Lat {
			return ps[i].Lat < ps[j].Lat
		}
		return ps[i].Lon < ps[j].Lon
	})
	var b strings.Builder
	fmt.Fprintf(&b, "v%d|%+v|", raster.VersaoReceita, cfg)
	for _, p := range ps {
		fmt.Fprintf(&b, "%.6f,%.6f;", p.Lat, p.Lon)
	}
	h := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(h[:16])
}

func bboxGraus(r geo.Retangulo) [4]float64 {
	so, ne := r.SO(), r.NE()
	return [4]float64{so.Lon, so.Lat, ne.Lon, ne.Lat}
}

// paineis: geral (células de 250 m ou maiores, até ~48 na largura) e um por setor (250 m).
func paineis(set *geo.Setorizacao) []raster.Painel {
	lat := set.Geral.Retangulo.Centro().Lat
	k := 1 / math.Cos(lat*math.Pi/180)
	base := 250 * k
	tamGeral := math.Max(base, set.Geral.Retangulo.Largura()/48)
	out := []raster.Painel{raster.NovoPainel(raster.NovasCelulas(set.Geral.Retangulo, tamGeral), 12, 1600)}
	for _, f := range set.Setores {
		out = append(out, raster.NovoPainel(raster.NovasCelulas(f.Retangulo, base), 12, 1600))
	}
	return out
}

// Preparar compõe a imagem para os pontos (já validados). Com reusar=true, a composição
// guardada da mesma área é devolvida sem consultar a internet (resultado estável); com
// reusar=false ("Atualizar melhor imagem") a busca é refeita. Sem internet, devolve a
// composição guardada (Guardada = true) ou o erro de conexão (RN-91).
func (s *Servico) Preparar(ctx context.Context, pts []geo.Ponto, cfg ConfigFolhas, prog func(Progresso), reusar bool) (*Composicao, error) {
	s.iniciar()
	if prog == nil {
		prog = func(Progresso) {}
	}
	chave := ChaveArea(pts, cfg(2))
	if reusar {
		if c, ok := s.guardada(chave); ok {
			c.Guardada = true
			return c, nil
		}
	}
	set, err := geo.Setorizar(pts, cfg(2))
	if err != nil {
		return nil, err
	}
	prog(Progresso{Fase: "Buscando imagens disponíveis"})
	cands, err := s.Catalogo.Candidatas(ctx, bboxGraus(set.Geral.Retangulo), s.agora())
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if c, ok := s.guardada(chave); ok {
			c.Guardada = true
			return c, nil
		}
		if s.Reserva != nil {
			c, errR := s.prepararReserva(ctx, pts, set, chave, prog)
			if errors.Is(errR, cartografia.ErrSemCredenciais) {
				return nil, err // sem reserva configurada: vale o erro do INPE (RN-91)
			}
			return c, errR
		}
		return nil, err
	}
	comp, err := s.compor(ctx, pts, set, cfg, cands, prog)
	if err != nil {
		if errors.Is(err, cartografia.ErrSemConexao) && ctx.Err() == nil {
			if c, ok := s.guardada(chave); ok {
				c.Guardada = true
				return c, nil
			}
		}
		return nil, err
	}
	comp.Pontos = pts
	if err := s.gravar(comp, chave); err != nil {
		return nil, err
	}
	if err := s.preRenderizar(ctx, comp, prog); err != nil {
		return nil, err
	}
	return comp, nil
}

func (s *Servico) compor(ctx context.Context, pts []geo.Ponto, set *geo.Setorizacao, cfg ConfigFolhas, cands []cartografia.Candidata, prog func(Progresso)) (*Composicao, error) {
	// Abre os cabeçalhos (em paralelo); a referência de cor é o mosaico mais recente.
	type aberta struct {
		c cartografia.Candidata
		f raster.Fonte
	}
	abertas := make([]*aberta, len(cands))
	var avisos []string
	var wg sync.WaitGroup
	var muAv sync.Mutex
	sem := make(chan struct{}, 4)
	for i, c := range cands {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			f, err := s.fonte(ctx, c)
			if err != nil {
				muAv.Lock()
				avisos = append(avisos, fmt.Sprintf("%s de %s não pôde ser lida (%v)", c.Rotulo(), c.Periodo(), err))
				muAv.Unlock()
				return
			}
			abertas[i] = &aberta{c, f}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var lista []*aberta
	var ref *aberta
	for _, a := range abertas {
		if a == nil {
			continue
		}
		if a.c.Fonte == cartografia.FonteMosaico && (ref == nil || a.c.Data.After(ref.c.Data)) {
			ref = a
		}
		lista = append(lista, a)
	}
	if len(lista) == 0 {
		if len(avisos) > 0 {
			return nil, fmt.Errorf("%w: nenhuma imagem pôde ser lida (%s)", cartografia.ErrSemConexao, avisos[0])
		}
		return nil, errors.New("não há imagens de satélite para esta área")
	}
	sort.SliceStable(avisos, func(i, j int) bool { return avisos[i] < avisos[j] })

	for tentativa := 0; tentativa < 2; tentativa++ {
		pns := paineis(set)
		total := len(lista)
		feito := atomic.Int32{}
		emitir := func() {
			r, c := s.bytesRede()
			prog(Progresso{Fase: "Avaliando a qualidade das imagens", Feito: int(feito.Load()), Total: total, BytesRede: r, BytesCache: c})
		}
		emitir()
		var refPint []*raster.Amostras
		var refAv *raster.Avaliacao
		if ref != nil {
			av, pint, err := raster.Avaliar(ctx, ref.f, nil, pns, false, s.Limiares)
			if err != nil {
				return nil, err
			}
			refPint, refAv = pint, av
			feito.Add(1)
			emitir()
		}
		ents := make([]*selecao.Entrada, len(lista))
		var errs []error
		var muE sync.Mutex
		sem := make(chan struct{}, 3)
		var wg sync.WaitGroup
		for i, a := range lista {
			if a == ref {
				ents[i] = &selecao.Entrada{Candidata: a.c, Av: refAv}
				continue
			}
			if ref == nil && a.c.Fonte == cartografia.FonteCBERS {
				continue // sem referência não há como harmonizar a cor nem estimar nuvens do CBERS
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				av, _, err := raster.Avaliar(ctx, a.f, refPint, pns, a.c.Fonte == cartografia.FonteCBERS, s.Limiares)
				feito.Add(1)
				emitir()
				muE.Lock()
				defer muE.Unlock()
				if err != nil {
					errs = append(errs, fmt.Errorf("%s de %s: %w", a.c.Rotulo(), a.c.Periodo(), err))
					return
				}
				ents[i] = &selecao.Entrada{Candidata: a.c, Av: av}
			}()
		}
		wg.Wait()
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var es []selecao.Entrada
		for _, e := range ents {
			if e != nil {
				es = append(es, *e)
			}
		}
		if len(es) == 0 {
			if len(errs) > 0 {
				return nil, errs[0]
			}
			return nil, errors.New("não há imagens de satélite utilizáveis para esta área")
		}
		for _, e := range errs {
			avisos = append(avisos, e.Error())
		}
		escolha := selecao.Escolher(es, pns, s.agora(), s.Criterios)
		// Sem 2 m na maior parte dos setores: refaz os setores na escala de 10 m (menos folhas).
		if tentativa == 0 && len(set.Setores) > 0 {
			soma := 0.0
			for p := 1; p < len(pns); p++ {
				soma += escolha.FracaoNivel(p, 2)
			}
			if soma/float64(len(pns)-1) < 0.5 {
				novo, err := geo.Setorizar(pts, cfg(10))
				if err == nil && (len(novo.Setores) != len(set.Setores) || novo.MetrosPxSetor != set.MetrosPxSetor) {
					set = novo
					continue
				}
			}
		}
		return s.montar(ctx, set, escolha, avisos)
	}
	return nil, errors.New("composição não convergiu")
}

func (s *Servico) montar(ctx context.Context, set *geo.Setorizacao, es *selecao.Escolha, avisos []string) (*Composicao, error) {
	comp := &Composicao{Receita: es.Receita, Folhas: *set, CriadaEm: s.agora(), Avisos: avisos}
	fontes, err := s.fontesDe(ctx, comp, "")
	if err != nil {
		return nil, err
	}
	// Tom único da composição: calculado sobre a geral (cores já harmonizadas).
	g := raster.Grade{Ret: set.Geral.Retangulo, L: 400, A: 300}
	r, err := raster.Renderizar(ctx, &comp.Receita, fontes, g)
	if err != nil {
		return nil, err
	}
	var amostras [][3]uint8
	for k := 0; k+3 < len(r.Img.Pix); k += 4 {
		if r.Img.Pix[k+3] > 0 {
			amostras = append(amostras, [3]uint8{r.Img.Pix[k], r.Img.Pix[k+1], r.Img.Pix[k+2]})
		}
	}
	comp.Receita.Tom = raster.CalcularTom(amostras, 0.92, 1.08)
	comp.ResolucaoM = 10
	datas := map[string]bool{}
	for i, c := range comp.Receita.Cenas {
		fr := 0.0
		if es.Total > 0 {
			fr = float64(es.Celulas[i]) / float64(es.Total)
		}
		if es.Celulas[i] > 0 {
			datas[c.Candidata.Periodo()] = true
			comp.ResolucaoM = math.Min(comp.ResolucaoM, c.Candidata.ResolucaoM)
		}
		comp.Partes = append(comp.Partes, Parte{Rotulo: c.Candidata.Rotulo(), Periodo: c.Candidata.Periodo(), ResolucaoM: c.Candidata.ResolucaoM,
			Fracao: fr, NuvemEstimada: c.NuvemEstimada, Atribuicao: c.Candidata.Atribuicao(), Fonte: c.Candidata.Fonte})
	}
	comp.Mosaico = len(datas) > 1
	if es.SemImagem > 0 {
		comp.Avisos = append(comp.Avisos, fmt.Sprintf("%d de %d células da área não têm nenhuma imagem disponível", es.SemImagem, es.Total))
	}
	if es.Limitadas > 0 {
		comp.Avisos = append(comp.Avisos, fmt.Sprintf("%d de %d células só têm imagem com nuvens ou falhas acima do limite", es.Limitadas, es.Total))
	}
	comp.ZNativo = raster.ZoomNativo(comp.ResolucaoM, set.Geral.Retangulo.Centro().Lat)
	comp.ZGeral = min(comp.ZNativo, raster.ZoomNativo(set.MetrosPxGeral, set.Geral.Retangulo.Centro().Lat))
	comp.ZMin = max(3, comp.ZGeral-3)
	b, _ := json.Marshal(struct {
		R raster.Receita
		F geo.Setorizacao
	}{comp.Receita, comp.Folhas})
	h := sha256.Sum256(b)
	comp.ID = hex.EncodeToString(h[:16])
	return comp, nil
}

// fontesDe abre as fontes da receita (COG pelo cache; reserva CDSE pelas imagens guardadas).
func (s *Servico) fontesDe(ctx context.Context, comp *Composicao, dir string) ([]raster.Fonte, error) {
	var out []raster.Fonte
	for _, c := range comp.Receita.Cenas {
		if c.Candidata.Fonte == cartografia.FonteCDSE {
			f, err := s.fonteReserva(dir, comp)
			if err != nil {
				return nil, err
			}
			out = append(out, f)
			continue
		}
		f, err := s.fonte(ctx, c.Candidata)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// ---------- reserva CDSE ----------

type fonteLista []*raster.FonteImagem3857

// Pintar com a imagem mais detalhada que cobre cada pixel (setores antes da geral).
func (l fonteLista) Pintar(ctx context.Context, g raster.Grade) (*raster.Amostras, error) {
	var out *raster.Amostras
	for _, f := range l {
		a, err := f.Pintar(ctx, g)
		if err != nil {
			return nil, err
		}
		if out == nil {
			out = a
			continue
		}
		for k, c := range out.Classe {
			if c != raster.ClasseOK && a.Classe[k] == raster.ClasseOK {
				out.Classe[k] = raster.ClasseOK
				copy(out.RGB[3*k:3*k+3], a.RGB[3*k:3*k+3])
			}
		}
	}
	return out, nil
}

func (s *Servico) fonteReserva(dir string, comp *Composicao) (raster.Fonte, error) {
	if dir == "" {
		dir = s.dirComp(comp.ID)
	}
	var l fonteLista
	folhas := append(append([]geo.Folha{}, comp.Folhas.Setores...), comp.Folhas.Geral)
	for i, f := range folhas {
		b, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("reserva-%02d.png", i)))
		if err != nil {
			return nil, fmt.Errorf("imagem da reserva ausente: %w", err)
		}
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		l = append(l, &raster.FonteImagem3857{Img: img, Ret: f.Retangulo})
	}
	return l, nil
}

func (s *Servico) prepararReserva(ctx context.Context, pts []geo.Ponto, set *geo.Setorizacao, chave string, prog func(Progresso)) (*Composicao, error) {
	folhas := append(append([]geo.Folha{}, set.Setores...), set.Geral)
	imgs := make([][]byte, len(folhas))
	var ini, fim time.Time
	for i, f := range folhas {
		prog(Progresso{Fase: "Obtendo imagem da fonte de reserva (Copernicus)", Feito: i, Total: len(folhas)})
		im, err := s.Reserva(ctx, f.Retangulo, 2000, 1500)
		if err != nil {
			return nil, err
		}
		if im.SemDados > 0.0005 {
			return nil, cartografia.ErrImagemIncompleta
		}
		imgs[i] = im.PNG
		ini, fim = im.De, im.Ate
	}
	c := cartografia.Candidata{ID: "cdse", Fonte: cartografia.FonteCDSE, Data: ini, Ate: &fim, ResolucaoM: 10}
	cel := raster.NovasCelulas(set.Geral.Retangulo, set.Geral.Retangulo.Largura())
	for i := range cel.Cena {
		cel.Cena[i] = 0
	}
	comp := &Composicao{Receita: raster.Receita{Versao: raster.VersaoReceita, Cenas: []raster.CenaReceita{{Candidata: c, Cor: raster.Identidade()}}, Geral: cel, Tom: raster.TomNeutro()},
		Folhas: *set, CriadaEm: s.agora(), Reserva: true, ResolucaoM: 10, Pontos: pts,
		Partes: []Parte{{Rotulo: c.Rotulo(), Periodo: c.Periodo(), ResolucaoM: 10, Fracao: 1, Atribuicao: c.Atribuicao(), Fonte: c.Fonte}},
		Avisos: []string{"O catálogo do INPE não respondeu; foi usada a fonte de reserva (Copernicus, 10 m)."}}
	lat := set.Geral.Retangulo.Centro().Lat
	comp.ZNativo = raster.ZoomNativo(set.MetrosPxSetor, lat)
	comp.ZGeral = min(comp.ZNativo, raster.ZoomNativo(set.MetrosPxGeral, lat))
	if len(set.Setores) == 0 {
		comp.ZNativo = comp.ZGeral
	}
	comp.ZMin = max(3, comp.ZGeral-3)
	h := sha256.Sum256([]byte(fmt.Sprintf("reserva|%s|%d", chave, comp.CriadaEm.UnixNano())))
	comp.ID = hex.EncodeToString(h[:16])
	dir := s.dirComp(comp.ID)
	for i, b := range imgs {
		if err := gravarAtomico(filepath.Join(dir, fmt.Sprintf("reserva-%02d.png", i)), b); err != nil {
			return nil, err
		}
	}
	if err := s.gravar(comp, chave); err != nil {
		return nil, err
	}
	if err := s.preRenderizar(ctx, comp, prog); err != nil {
		return nil, err
	}
	return comp, nil
}

// ---------- armazenamento ----------

var reID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func gravarAtomico(destino string, dados []byte) error {
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(destino), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(dados); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), destino)
}

func (s *Servico) gravar(c *Composicao, chave string) error {
	b, _ := json.MarshalIndent(c, "", " ")
	if err := gravarAtomico(filepath.Join(s.dirComp(c.ID), "composicao.json"), b); err != nil {
		return err
	}
	return gravarAtomico(filepath.Join(s.Dir, "composicoes", "areas", chave), []byte(c.ID))
}

// Carregar uma composição guardada.
func (s *Servico) Carregar(id string) (*Composicao, error) {
	if !reID.MatchString(id) {
		return nil, errors.New("composição inválida")
	}
	b, err := os.ReadFile(filepath.Join(s.dirComp(id), "composicao.json"))
	if err != nil {
		return nil, errors.New("composição não encontrada")
	}
	var c Composicao
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Servico) guardada(chave string) (*Composicao, bool) {
	b, err := os.ReadFile(filepath.Join(s.Dir, "composicoes", "areas", chave))
	if err != nil {
		return nil, false
	}
	c, err := s.Carregar(strings.TrimSpace(string(b)))
	return c, err == nil
}

func (s *Servico) carregada(ctx context.Context, id string) (*carregada, error) {
	s.iniciar()
	s.mu.Lock()
	if c, ok := s.comps[id]; ok {
		s.mu.Unlock()
		return c, nil
	}
	s.mu.Unlock()
	comp, err := s.Carregar(id)
	if err != nil {
		return nil, err
	}
	fontes, err := s.fontesDe(ctx, comp, "")
	if err != nil {
		return nil, err
	}
	c := &carregada{comp: comp, fontes: fontes}
	s.mu.Lock()
	s.comps[id] = c
	s.mu.Unlock()
	return c, nil
}

// ---------- tiles e folhas ----------

func intersecta(a, b geo.Retangulo) bool {
	return a.MinX < b.MaxX && b.MinX < a.MaxX && a.MinY < b.MaxY && b.MinY < a.MaxY
}

// cobre: o tile existe nesta composição (geral até ZGeral; setores até ZNativo).
func (c *Composicao) cobre(z, x, y int) bool {
	if z < c.ZMin || z > c.ZNativo {
		return false
	}
	r := raster.RetTile(z, x, y)
	if z <= c.ZGeral && intersecta(r, c.Folhas.Geral.Retangulo) {
		return true
	}
	for _, f := range c.Folhas.Setores {
		if intersecta(r, f.Retangulo) {
			return true
		}
	}
	return z <= c.ZNativo && len(c.Folhas.Setores) == 0 && intersecta(r, c.Folhas.Geral.Retangulo)
}

func (s *Servico) caminhoTile(id string, z, x, y int) string {
	return filepath.Join(s.dirComp(id), "tiles", fmt.Sprint(z), fmt.Sprint(x), fmt.Sprintf("%d.jpg", y))
}

// Tile devolve o JPEG do tile (gerado e guardado na primeira vez). nil = fora da composição.
func (s *Servico) Tile(ctx context.Context, id string, z, x, y int) ([]byte, error) {
	if !reID.MatchString(id) || z < 0 || z > 22 || x < 0 || y < 0 || x >= 1<<z || y >= 1<<z {
		return nil, errors.New("tile inválido")
	}
	if b, err := os.ReadFile(s.caminhoTile(id, z, x, y)); err == nil {
		return b, nil
	}
	cc, err := s.carregada(ctx, id)
	if err != nil {
		return nil, err
	}
	if !cc.comp.cobre(z, x, y) {
		return nil, nil
	}
	return s.gerarTile(ctx, cc, z, x, y)
}

func (s *Servico) gerarTile(ctx context.Context, cc *carregada, z, x, y int) ([]byte, error) {
	r, err := raster.Renderizar(ctx, &cc.comp.Receita, cc.fontes, raster.Grade{Ret: raster.RetTile(z, x, y), L: raster.LadoTile, A: raster.LadoTile})
	if err != nil {
		return nil, err
	}
	b, err := raster.JPEG(r.Img, 88)
	if err != nil {
		return nil, err
	}
	if err := gravarAtomico(s.caminhoTile(cc.comp.ID, z, x, y), b); err != nil {
		return nil, err
	}
	return b, nil
}

// LarguraFolha × AlturaFolha: o mapa de cada folha (proporção 4:3).
const (
	LarguraFolha = 2000
	AlturaFolha  = 1500
)

func (s *Servico) caminhoFolha(id string, i int) string {
	return filepath.Join(s.dirComp(id), fmt.Sprintf("folha-%02d.png", i))
}

// Folha devolve a imagem do mapa da folha i (0 = geral, 1.. = setores) e a cobertura.
func (s *Servico) Folha(ctx context.Context, id string, i int) (*image.NRGBA, float64, error) {
	cc, err := s.carregada(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	if b, err := os.ReadFile(s.caminhoFolha(id, i)); err == nil {
		if img, err := png.Decode(bytes.NewReader(b)); err == nil {
			if n, ok := img.(*image.NRGBA); ok {
				return n, cobertura(n), nil
			}
		}
	}
	ret := cc.comp.Folhas.Geral.Retangulo
	if i > 0 {
		if i > len(cc.comp.Folhas.Setores) {
			return nil, 0, errors.New("folha inexistente")
		}
		ret = cc.comp.Folhas.Setores[i-1].Retangulo
	}
	r, err := raster.Renderizar(ctx, &cc.comp.Receita, cc.fontes, raster.Grade{Ret: ret, L: LarguraFolha, A: AlturaFolha})
	if err != nil {
		return nil, 0, err
	}
	var b bytes.Buffer
	if err := png.Encode(&b, r.Img); err == nil {
		_ = gravarAtomico(s.caminhoFolha(id, i), b.Bytes())
	}
	return r.Img, r.Cobertura, nil
}

func cobertura(img *image.NRGBA) float64 {
	n, v := 0, 0
	for k := 3; k < len(img.Pix); k += 4 {
		n++
		if img.Pix[k] > 0 {
			v++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(v) / float64(n)
}

// preRenderizar gera as folhas e os tiles da prévia (para reabrir sem internet).
func (s *Servico) preRenderizar(ctx context.Context, comp *Composicao, prog func(Progresso)) error {
	type tile struct{ z, x, y int }
	var tiles []tile
	for z := comp.ZMin; z <= comp.ZNativo; z++ {
		var areas []geo.Retangulo
		if z <= comp.ZGeral || len(comp.Folhas.Setores) == 0 {
			areas = append(areas, comp.Folhas.Geral.Retangulo)
		} else {
			for _, f := range comp.Folhas.Setores {
				areas = append(areas, f.Retangulo)
			}
		}
		vistos := map[tile]bool{}
		for _, a := range areas {
			x0, y0, x1, y1 := raster.TilesEm(a, z)
			for x := x0; x <= x1; x++ {
				for y := y0; y <= y1; y++ {
					t := tile{z, x, y}
					if !vistos[t] {
						vistos[t] = true
						tiles = append(tiles, t)
					}
				}
			}
		}
	}
	folhas := 1 + len(comp.Folhas.Setores)
	total := folhas + len(tiles)
	feito := atomic.Int32{}
	emitir := func() {
		r, c := s.bytesRede()
		prog(Progresso{Fase: "Preparando a imagem da região", Feito: int(feito.Load()), Total: total, BytesRede: r, BytesCache: c})
	}
	emitir()
	for i := 0; i < folhas; i++ {
		if _, _, err := s.Folha(ctx, comp.ID, i); err != nil {
			return err
		}
		feito.Add(1)
		emitir()
	}
	cc, err := s.carregada(ctx, comp.ID)
	if err != nil {
		return err
	}
	fila := make(chan tile)
	erros := make(chan error, 1)
	var wg sync.WaitGroup
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()
	for w := 0; w < 6; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range fila {
				if _, err := os.Stat(s.caminhoTile(comp.ID, t.z, t.x, t.y)); err != nil {
					if _, err := s.gerarTile(ctx2, cc, t.z, t.x, t.y); err != nil {
						select {
						case erros <- err:
						default:
						}
						cancel()
						return
					}
				}
				if n := feito.Add(1); n%16 == 0 {
					emitir()
				}
			}
		}()
	}
envio:
	for _, t := range tiles {
		select {
		case fila <- t:
		case <-ctx2.Done():
			break envio
		}
	}
	close(fila)
	wg.Wait()
	select {
	case err := <-erros:
		return err
	default:
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	emitir()
	return nil
}
