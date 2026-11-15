package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/carta"
	"sitposse7/desktop/internal/cartografia/cdse"
	"sitposse7/desktop/internal/cartografia/geo"
	"sitposse7/desktop/internal/credenciais"
	"sitposse7/desktop/internal/estatisticas"
	"sitposse7/desktop/internal/relatorios"
)

// AlvoCredencial no Gerenciador de Credenciais do Windows.
const AlvoCredencial = "SITPOSSE-7/Copernicus-CDSE"

// Tamanho da imagem de uma carta (proporção 4:3; ≈ 11 unidades de processamento).
const (
	LarguraCarta = 2000
	AlturaCarta  = 1500
	// Lado máximo de um grupo de propriedades numa carta geral legível (≈ 20 m/px).
	LadoGrupoM = 40000
	// Largura mínima (m) de uma imagem pedida: abaixo disso os pixels de 10 m viram borrões.
	LarguraMinimaM = 1500
)

// CartaService: cartas-imagem com Sentinel-2/CDSE (A27, ADR-0022).
type CartaService struct {
	C *Central

	mu       sync.Mutex
	cofre    credenciais.Cofre
	provedor *cdse.Cliente
	// Provedor substituível nos testes.
	Provedor cartografia.Provedor
	Agora    func() time.Time
}

func (s *CartaService) agora() time.Time {
	if s.Agora != nil {
		return s.Agora()
	}
	return time.Now()
}

func (s *CartaService) armazem() cartografia.Armazem {
	return cartografia.Armazem{Dir: filepath.Join(s.C.PastaDados, "cartografia")}
}

func (s *CartaService) cofreAtual() credenciais.Cofre {
	if s.cofre == nil {
		s.cofre = credenciais.Nativo()
	}
	return s.cofre
}

// prov devolve o provedor com as credenciais do cofre (lidas a cada criação, nunca logadas).
func (s *CartaService) prov() (cartografia.Provedor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Provedor != nil {
		return s.Provedor, nil
	}
	if s.provedor == nil {
		cred, err := s.cofreAtual().Ler(AlvoCredencial)
		if errors.Is(err, credenciais.ErrNaoEncontrada) {
			return nil, cartografia.ErrSemCredenciais
		}
		if err != nil {
			return nil, fmt.Errorf("não foi possível ler as credenciais do Windows: %w", err)
		}
		s.provedor = &cdse.Cliente{ClientID: cred.Usuario, ClientSecret: cred.Segredo}
	}
	return s.provedor, nil
}

// ---------- configuração ----------

// EstadoConexao: credenciais configuradas e uso do disco (sem expor o segredo).
type EstadoConexao struct {
	Configurado bool            `json:"configurado"`
	ClientID    string          `json:"client_id"` // mascarado
	Uso         cartografia.Uso `json:"uso"`
	Provedor    string          `json:"provedor"`
}

func mascarar(s string) string {
	r := []rune(s)
	if len(r) <= 8 {
		return strings.Repeat("•", len(r))
	}
	return string(r[:4]) + strings.Repeat("•", 6) + string(r[len(r)-4:])
}

// Estado da configuração.
func (s *CartaService) Estado() EstadoConexao {
	e := EstadoConexao{Uso: s.armazem().Uso(), Provedor: "Copernicus Data Space Ecosystem — Sentinel-2 L2A"}
	if c, err := s.cofreAtual().Ler(AlvoCredencial); err == nil {
		e.Configurado, e.ClientID = true, mascarar(c.Usuario)
	}
	return e
}

// SalvarCredenciais grava no Gerenciador de Credenciais e testa a conexão.
func (s *CartaService) SalvarCredenciais(clientID, segredo string) error {
	clientID, segredo = strings.TrimSpace(clientID), strings.TrimSpace(segredo)
	if clientID == "" || segredo == "" {
		return errors.New("informe o identificador (client ID) e o segredo (client secret)")
	}
	teste := &cdse.Cliente{ClientID: clientID, ClientSecret: segredo}
	if _, err := teste.Token(context.Background()); err != nil {
		return err
	}
	if err := s.cofreAtual().Gravar(AlvoCredencial, credenciais.Credencial{Usuario: clientID, Segredo: segredo}); err != nil {
		return fmt.Errorf("não foi possível gravar no Gerenciador de Credenciais do Windows: %w", err)
	}
	s.mu.Lock()
	s.provedor = teste
	s.mu.Unlock()
	return nil
}

// TestarConexao pede um token com as credenciais guardadas.
func (s *CartaService) TestarConexao() error {
	p, err := s.prov()
	if err != nil {
		return err
	}
	if c, ok := p.(*cdse.Cliente); ok {
		c.Esquecer()
		_, err = c.Token(context.Background())
	}
	return err
}

// RemoverCredenciais apaga as credenciais do Windows.
func (s *CartaService) RemoverCredenciais() error {
	s.mu.Lock()
	s.provedor = nil
	s.mu.Unlock()
	return s.cofreAtual().Apagar(AlvoCredencial)
}

// LimparImagens apaga só as imagens brutas guardadas (as cartas ficam).
func (s *CartaService) LimparImagens() (int, error) { return s.armazem().LimparImagens() }

// ---------- planejamento ----------

// Plano: propriedades representáveis (numeradas), as sem localização e o enquadramento.
type Plano struct {
	Marcadores     []carta.Marcador `json:"marcadores"`
	UUIDs          []string         `json:"uuids"`
	SemLocalizacao []string         `json:"sem_localizacao"`
	Retangulo      geo.Retangulo    `json:"retangulo"`
	SO             geo.Ponto        `json:"so"`
	NE             geo.Ponto        `json:"ne"`
	ExtensaoKm     float64          `json:"extensao_km"`
	ResolucaoM     float64          `json:"resolucao_m"`
	Grupos         [][]int          `json:"grupos"` // índices em Marcadores, quando convém dividir
	Largura        int              `json:"largura"`
	Altura         int              `json:"altura"`
	estudos        []string         // estudos das propriedades representadas
}

func contem(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

func origem(p estatisticas.Familia) string {
	if p.GpsOrigem == nil {
		return ""
	}
	return *p.GpsOrigem
}

func (s *CartaService) familias(filtro estatisticas.Filtro) ([]estatisticas.Familia, error) {
	base, err := estatisticas.Carregar(contexto(), s.C.c.DB)
	if err != nil {
		return nil, err
	}
	r, err := estatisticas.Calcular(base, filtro)
	if err != nil {
		return nil, err
	}
	return r.Familias, nil
}

func montarPlano(fs []estatisticas.Familia, individual bool) (*Plano, error) {
	p := &Plano{Largura: LarguraCarta, Altura: AlturaCarta, Marcadores: []carta.Marcador{}, UUIDs: []string{}, SemLocalizacao: []string{}, Grupos: [][]int{}}
	var pts []geo.Ponto
	for _, f := range fs {
		if f.Lat == nil || f.Lon == nil {
			p.SemLocalizacao = append(p.SemLocalizacao, f.Nome)
			continue
		}
		pt := geo.Ponto{Lat: *f.Lat, Lon: *f.Lon}
		if geo.Validar(pt) != nil {
			p.SemLocalizacao = append(p.SemLocalizacao, f.Nome+" (coordenada inválida)")
			continue
		}
		p.Marcadores = append(p.Marcadores, carta.Marcador{Numero: len(p.Marcadores) + 1, Nome: f.Nome, Ponto: pt, PrecisaoM: f.GpsPrecisaoM, Origem: origem(f), Equipe: f.Equipe, Jornada: f.Jornada})
		if !contem(p.estudos, f.EstudoUUID) {
			p.estudos = append(p.estudos, f.EstudoUUID)
		}
		p.UUIDs = append(p.UUIDs, f.UUID)
		pts = append(pts, pt)
	}
	if len(pts) == 0 {
		if individual {
			return p, errors.New("esta propriedade não tem coordenada e não pode ser representada na carta até que tenha localização")
		}
		return p, errors.New("nenhuma das propriedades selecionadas tem coordenada")
	}
	margem, meia := 600.0, 1500.0
	if individual {
		margem, meia = 0, 2000 // ≈ 4 km de largura: 2 m/px na carta de 2000 px
	}
	r, err := geo.Enquadrar(pts, margem, meia, LarguraCarta, AlturaCarta)
	if err != nil {
		return p, err
	}
	p.Retangulo, p.SO, p.NE = r, r.SO(), r.NE()
	p.ResolucaoM = geo.ResolucaoEfetiva(r, LarguraCarta)
	p.ExtensaoKm = geo.DistanciaM(geo.LatLon(r.MinX, (r.MinY+r.MaxY)/2), geo.LatLon(r.MaxX, (r.MinY+r.MaxY)/2)) / 1000
	if !individual && p.ResolucaoM > 20 {
		p.Grupos = geo.Agrupar(pts, LadoGrupoM)
	}
	return p, nil
}

// PlanejarIndividual: a propriedade, numerada como 1, e o enquadramento inicial.
func (s *CartaService) PlanejarIndividual(uuid string) (*Plano, error) {
	fs, err := s.familias(estatisticas.Filtro{})
	if err != nil {
		return nil, err
	}
	for _, f := range fs {
		if f.UUID == uuid {
			return montarPlano([]estatisticas.Familia{f}, true)
		}
	}
	return nil, errors.New("propriedade não encontrada")
}

// PlanejarGeral: as propriedades dos filtros do Painel (RN-81), numeradas em ordem alfabética.
func (s *CartaService) PlanejarGeral(filtro estatisticas.Filtro) (*Plano, error) {
	fs, err := s.familias(filtro)
	if err != nil {
		return nil, err
	}
	return montarPlano(fs, false)
}

// EnquadrarGrupo: retângulo de um grupo de marcadores (carta dividida).
func (s *CartaService) EnquadrarGrupo(pontos []geo.Ponto) (geo.Retangulo, error) {
	return geo.Enquadrar(pontos, 600, 1500, LarguraCarta, AlturaCarta)
}

// ---------- imagens ----------

// PassagensResposta: dias com imagem sobre a área e a sugestão.
type PassagensResposta struct {
	Passagens []cartografia.Passagem `json:"passagens"`
	Sugerida  *time.Time             `json:"sugerida"`
}

// Passagens sobre o retângulo no período (padrão: últimos 90 dias).
func (s *CartaService) Passagens(r geo.Retangulo, de, ate *time.Time) (*PassagensResposta, error) {
	p, err := s.prov()
	if err != nil {
		return nil, err
	}
	fim := s.agora()
	if ate != nil {
		fim = *ate
	}
	ini := fim.Add(-90 * 24 * time.Hour)
	if de != nil {
		ini = *de
	}
	ps, err := p.Passagens(context.Background(), r, ini, fim)
	if err != nil {
		return nil, err
	}
	resp := &PassagensResposta{Passagens: ps}
	if sug := cdse.Sugerir(ps, s.agora()); sug != nil {
		d := sug.Data
		resp.Sugerida = &d
	}
	return resp, nil
}

// PedidoTela: a área e a data escolhidas na tela (Data ou período De–Ate).
type PedidoTela struct {
	Retangulo geo.Retangulo `json:"retangulo"`
	Data      *time.Time    `json:"data"`
	De        *time.Time    `json:"de"`
	Ate       *time.Time    `json:"ate"`
}

func (s *CartaService) pedido(t PedidoTela) (cartografia.PedidoImagem, error) {
	r := geo.GarantirLargura(geo.AjustarProporcao(t.Retangulo, LarguraCarta, AlturaCarta), LarguraMinimaM)
	if r.Largura() <= 0 || r.Altura() <= 0 {
		return cartografia.PedidoImagem{}, errors.New("enquadramento inválido")
	}
	p := cartografia.PedidoImagem{Ret: r, Largura: LarguraCarta, Altura: AlturaCarta, Data: t.Data}
	if t.Data == nil {
		if t.De == nil || t.Ate == nil {
			return p, errors.New("escolha uma data de imagem ou um período")
		}
		p.De, p.Ate = *t.De, *t.Ate
	}
	return p, nil
}

// obter busca no armazém e, se não houver, no provedor (e guarda, sem validade).
func (s *CartaService) obter(t PedidoTela) (*cartografia.Imagem, string, bool, error) {
	ped, err := s.pedido(t)
	if err != nil {
		return nil, "", false, err
	}
	chave := cartografia.ChaveImagem("cdse", cdse.VersaoEvalscript, ped)
	a := s.armazem()
	if img, ok := a.LerImagem(chave); ok {
		return img, chave, true, nil
	}
	p, err := s.prov()
	if err != nil {
		return nil, "", false, err
	}
	img, err := p.Imagem(context.Background(), ped)
	if err != nil {
		return nil, "", false, err
	}
	if err := a.GravarImagem(chave, img); err != nil {
		return nil, "", false, fmt.Errorf("não foi possível guardar a imagem: %w", err)
	}
	return img, chave, false, nil
}

// ImagemPrevia: imagem para a prévia (servida em /cartografia/imagens/{chave}.png).
type ImagemPrevia struct {
	URL       string        `json:"url"`
	Retangulo geo.Retangulo `json:"retangulo"`
	SO        geo.Ponto     `json:"so"`
	NE        geo.Ponto     `json:"ne"`
	Periodo   string        `json:"periodo"`
	SemDados  float64       `json:"sem_dados"`
	Guardada  bool          `json:"guardada"` // veio do armazém local (não gastou cota)
}

// Imagem obtém a imagem da área para a prévia.
func (s *CartaService) Imagem(t PedidoTela) (*ImagemPrevia, error) {
	img, chave, guardada, err := s.obter(t)
	if err != nil {
		return nil, err
	}
	return &ImagemPrevia{URL: "/cartografia/imagens/" + chave + ".png", Retangulo: img.Ret, SO: img.Ret.SO(), NE: img.Ret.NE(), Periodo: img.Periodo(), SemDados: img.SemDados, Guardada: guardada}, nil
}

// ---------- cartas ----------

// PedidoCarta: tipo, propriedades (UUIDs na ordem dos números), área, data e filtros.
type PedidoCarta struct {
	PedidoTela
	Tipo      string   `json:"tipo"` // individual | geral
	UUIDs     []string `json:"uuids"`
	Criterios []string `json:"criterios"`
}

// Tolerância de pixels sem dados (bordas): acima disso a carta não é gerada (RN-93).
const toleranciaSemDados = 0.0005

// GerarCarta monta e guarda a carta (PDF e PNG) e devolve o registro.
func (s *CartaService) GerarCarta(p PedidoCarta) (*cartografia.CartaGuardada, error) {
	if len(p.UUIDs) == 0 {
		return nil, errors.New("nenhuma propriedade selecionada")
	}
	fs, err := s.familias(estatisticas.Filtro{})
	if err != nil {
		return nil, err
	}
	porUUID := map[string]estatisticas.Familia{}
	for _, f := range fs {
		porUUID[f.UUID] = f
	}
	var sel []estatisticas.Familia
	for _, u := range p.UUIDs {
		if f, ok := porUUID[u]; ok {
			sel = append(sel, f)
		}
	}
	plano, err := montarPlano(sel, p.Tipo == "individual")
	if err != nil {
		return nil, err
	}
	img, _, _, err := s.obter(p.PedidoTela)
	if err != nil {
		return nil, err
	}
	if img.SemDados > toleranciaSemDados {
		return nil, cartografia.ErrImagemIncompleta
	}
	var fora []string
	for _, m := range plano.Marcadores {
		if !img.Ret.Contem(m.Ponto) {
			fora = append(fora, fmt.Sprintf("%d %s", m.Numero, m.Nome))
		}
	}
	if len(fora) > 0 {
		return nil, fmt.Errorf("ajuste o enquadramento: %d propriedade(s) ficaram fora da carta (%s)", len(fora), strings.Join(fora, "; "))
	}
	ano := s.agora().Year()
	if img.Data != nil {
		ano = img.Data.Year()
	} else if !img.Ate.IsZero() {
		ano = img.Ate.Year()
	}
	pv, _ := s.prov()
	atrib := (&cdse.Cliente{}).Atribuicao(ano)
	if pv != nil {
		atrib = pv.Atribuicao(ano)
	}
	subt := plano.Marcadores[0].Nome
	if p.Tipo != "individual" {
		subt = fmt.Sprintf("%d propriedade(s)", len(plano.Marcadores))
	}
	d := carta.Dados{Tipo: p.Tipo, Subtitulo: subt, Imagem: img, Atribuicao: atrib, Marcadores: plano.Marcadores, SemLocalizacao: plano.SemLocalizacao,
		Criterios: p.Criterios, Estudo: s.estudo(plano.estudos), GeradoEm: s.agora(), NomesNoMapa: len(plano.Marcadores) <= 12}
	_, mapa, err := carta.MapaPNG(d)
	if err != nil {
		return nil, err
	}
	folhaPNG, err := carta.FolhaPNG(d, mapa)
	if err != nil {
		return nil, err
	}
	pdf, err := carta.FolhaPDF(d, mapa)
	if err != nil {
		return nil, err
	}
	reg, err := s.armazem().GravarCarta(cartografia.CartaGuardada{Titulo: carta.Titulo + " — " + subt, Tipo: p.Tipo, Propriedades: len(plano.Marcadores), Periodo: img.Periodo(), Criterios: p.Criterios, GeradaEm: d.GeradoEm},
		map[string][]byte{"carta.pdf": pdf, "carta.png": folhaPNG})
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

// estudo nomeia o(s) estudo(s) das propriedades da carta.
func (s *CartaService) estudo(uuids []string) string {
	var nomes []string
	for _, u := range uuids {
		if n := s.C.nomeEstudo(u); n != "" {
			nomes = append(nomes, n)
		}
	}
	return strings.Join(nomes, "; ")
}

// Cartas guardadas (abrem sem internet).
func (s *CartaService) Cartas() []cartografia.CartaGuardada { return s.armazem().ListarCartas() }

// AbrirCarta abre o PDF ou PNG no programa padrão.
func (s *CartaService) AbrirCarta(id, arquivo string) error {
	p, err := s.armazem().CaminhoCarta(id, arquivo)
	if err != nil {
		return err
	}
	return s.C.Arquivos.Abrir(p)
}

// MostrarCarta abre o Explorador na pasta da carta.
func (s *CartaService) MostrarCarta(id, arquivo string) error {
	p, err := s.armazem().CaminhoCarta(id, arquivo)
	if err != nil {
		return err
	}
	return s.C.Arquivos.MostrarNaPasta(p)
}

// SalvarCopia grava uma cópia da carta onde o operador escolher; "" se cancelar.
func (s *CartaService) SalvarCopia(id, arquivo string) (string, error) {
	p, err := s.armazem().CaminhoCarta(id, arquivo)
	if err != nil {
		return "", err
	}
	dados, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(arquivo)
	descr := map[string]string{".pdf": "Documento PDF", ".png": "Imagem PNG"}[ext]
	var titulo string
	for _, c := range s.Cartas() {
		if c.ID == id {
			titulo = c.Titulo
		}
	}
	destino, err := s.C.Arquivos.EscolherDestino(relatorios.NomeArquivo("SITPOSSE-7", titulo, id[:8])+ext, descr, ext)
	if err != nil || destino == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(destino), ext) {
		destino += ext
	}
	return destino, GravarAtomico(destino, dados)
}

// ---------- rotas locais (prévia) ----------

var (
	reImagemCarto = regexp.MustCompile(`^/cartografia/imagens/([0-9a-f]{32})\.png$`)
	reArqCarta    = regexp.MustCompile(`^/cartografia/cartas/([0-9]{8}-[0-9]{6}-[0-9a-f]{8})/(carta\.png)$`)
)

// HandlerCartografia serve à tela as imagens guardadas e o PNG das cartas (só leitura).
// É função do pacote (não método) para não ser exposta pelo Wails como chamada da tela.
func HandlerCartografia(s *CartaService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		var caminho string
		if m := reImagemCarto.FindStringSubmatch(r.URL.Path); m != nil {
			p, ok := s.armazem().CaminhoImagem(m[1])
			if !ok {
				http.NotFound(w, r)
				return
			}
			caminho = p
		} else if m := reArqCarta.FindStringSubmatch(r.URL.Path); m != nil {
			p, err := s.armazem().CaminhoCarta(m[1], m[2])
			if err != nil {
				http.NotFound(w, r)
				return
			}
			caminho = p
		} else {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
		http.ServeFile(w, r, caminho)
	})
}
