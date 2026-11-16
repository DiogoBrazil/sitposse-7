package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/carta"
	"sitposse7/desktop/internal/cartografia/cdse"
	"sitposse7/desktop/internal/cartografia/composicao"
	"sitposse7/desktop/internal/cartografia/geo"
	"sitposse7/desktop/internal/cartografia/inpe"
	"sitposse7/desktop/internal/credenciais"
	"sitposse7/desktop/internal/estatisticas"
	"sitposse7/desktop/internal/relatorios"
)

// AlvoCredencial no Gerenciador de Credenciais do Windows (Copernicus, fonte de reserva).
const AlvoCredencial = "SITPOSSE-7/Copernicus-CDSE"

// CartaService: cartas-imagem com seleção automática de imagens (A30, ADR-0027; A27, ADR-0022).
type CartaService struct {
	C *Central

	mu       sync.Mutex
	cofre    credenciais.Cofre
	provedor *cdse.Cliente
	// Provedor de reserva substituível nos testes (senão, o CDSE com as credenciais do cofre).
	Provedor cartografia.Provedor
	// Catalogo substituível nos testes (senão, o INPE).
	Catalogo composicao.Catalogo
	Agora    func() time.Time

	comp    *composicao.Servico
	tarefas map[string]*tarefa
}

func (s *CartaService) agora() time.Time {
	if s.Agora != nil {
		return s.Agora()
	}
	return time.Now()
}

func (s *CartaService) dirCartografia() string { return filepath.Join(s.C.PastaDados, "cartografia") }

func (s *CartaService) armazem() cartografia.Armazem {
	return cartografia.Armazem{Dir: s.dirCartografia()}
}

func (s *CartaService) cofreAtual() credenciais.Cofre {
	if s.cofre == nil {
		s.cofre = credenciais.Nativo()
	}
	return s.cofre
}

// prov devolve o provedor de reserva com as credenciais do cofre (nunca logadas).
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

// servicoComposicao (um por Central; guarda os COGs abertos e as composições carregadas).
func (s *CartaService) servicoComposicao() *composicao.Servico {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.comp == nil {
		cat := s.Catalogo
		if cat == nil {
			cat = inpe.Novo()
		}
		s.comp = &composicao.Servico{Dir: s.dirCartografia(), Catalogo: cat, Agora: s.agora, Reserva: s.reserva}
	}
	return s.comp
}

// reserva: Sentinel-2 pelo Copernicus (mosaico de menos nuvens dos últimos 90 dias), só se
// houver credenciais configuradas.
func (s *CartaService) reserva(ctx context.Context, ret geo.Retangulo, larg, alt int) (*cartografia.Imagem, error) {
	p, err := s.prov()
	if err != nil {
		return nil, err
	}
	fim := s.agora()
	return p.Imagem(ctx, cartografia.PedidoImagem{Ret: ret, Largura: larg, Altura: alt, De: fim.Add(-90 * 24 * time.Hour), Ate: fim})
}

// ---------- configuração ----------

// EstadoConexao: fonte de reserva configurada e uso do disco (sem expor o segredo).
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
	e := EstadoConexao{Uso: s.armazem().Uso(), Provedor: "INPE (CBERS-4A e Sentinel-2); reserva: Copernicus Data Space"}
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

// LimparImagens apaga as imagens guardadas (brutas, blocos e composições); as cartas ficam.
func (s *CartaService) LimparImagens() (int, error) {
	n, err := s.armazem().LimparImagens()
	if err != nil {
		return n, err
	}
	for _, d := range []string{"blocos", "composicoes"} {
		if err := os.RemoveAll(filepath.Join(s.dirCartografia(), d)); err != nil {
			return n, err
		}
	}
	s.mu.Lock()
	s.comp = nil // descarta o que estava aberto em memória
	s.mu.Unlock()
	return n, nil
}

// ---------- planejamento ----------

// Plano: propriedades representáveis (com o número fixo do estudo) e as sem localização.
type Plano struct {
	Marcadores     []carta.Marcador `json:"marcadores"`
	UUIDs          []string         `json:"uuids"`
	SemLocalizacao []string         `json:"sem_localizacao"`
	Suspeitas      []string         `json:"suspeitas"` // coordenadas fora do Brasil ou muito distantes das demais
	Retangulo      geo.Retangulo    `json:"retangulo"`
	SO             geo.Ponto        `json:"so"`
	NE             geo.Ponto        `json:"ne"`
	ExtensaoKm     float64          `json:"extensao_km"`
	Individual     bool             `json:"individual"`
	estudos        []string
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

// configFolhas: a geral e os setores (individual: cerca de 4 km de largura, sem setores).
func configFolhas(individual bool) composicao.ConfigFolhas {
	return func(res float64) geo.ConfigFolhas {
		c := geo.ConfigPadrao(res)
		if individual {
			c.MeiaLarguraMinM, c.MargemMinM, c.LimiteGeral = 2000, 0, 1e9
		}
		return c
	}
}

func montarPlano(fs []estatisticas.Familia, individual bool) (*Plano, error) {
	p := &Plano{Marcadores: []carta.Marcador{}, UUIDs: []string{}, SemLocalizacao: []string{}, Suspeitas: []string{}, Individual: individual}
	// Ordem estável pelo número fixo no estudo (RN-96); sem número, depois, pelo nome e UUID.
	fs = append([]estatisticas.Familia(nil), fs...)
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if (a.Numero == 0) != (b.Numero == 0) {
			return a.Numero != 0
		}
		if a.EstudoUUID != b.EstudoUUID {
			return a.EstudoUUID < b.EstudoUUID
		}
		if a.Numero != b.Numero {
			return a.Numero < b.Numero
		}
		return a.UUID < b.UUID
	})
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
		p.Marcadores = append(p.Marcadores, carta.Marcador{Numero: f.Numero, UUID: f.UUID, Nome: f.Nome, Responsavel: f.Responsavel, Ponto: pt,
			PrecisaoM: f.GpsPrecisaoM, Origem: origem(f), Equipe: f.Equipe, Jornada: f.Jornada})
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
	// Sem número (cadastro anterior à numeração) ou de mais de um estudo (números podem
	// repetir): numera em sequência, depois do maior número presente.
	maior := 0
	for _, m := range p.Marcadores {
		maior = max(maior, m.Numero)
	}
	for i := range p.Marcadores {
		if p.Marcadores[i].Numero == 0 {
			maior++
			p.Marcadores[i].Numero = maior
		}
	}
	if len(p.estudos) > 1 {
		for i := range p.Marcadores {
			p.Marcadores[i].Numero = i + 1
		}
	}
	for _, i := range geo.Suspeitas(pts, 50) {
		p.Suspeitas = append(p.Suspeitas, fmt.Sprintf("%d %s", p.Marcadores[i].Numero, p.Marcadores[i].Nome))
	}
	set, err := geo.Setorizar(pts, configFolhas(individual)(2))
	if err != nil {
		return p, err
	}
	r := set.Geral.Retangulo
	p.Retangulo, p.SO, p.NE = r, r.SO(), r.NE()
	p.ExtensaoKm = geo.DistanciaM(geo.LatLon(r.MinX, (r.MinY+r.MaxY)/2), geo.LatLon(r.MaxX, (r.MinY+r.MaxY)/2)) / 1000
	return p, nil
}

func (p *Plano) pontos() []geo.Ponto {
	pts := make([]geo.Ponto, len(p.Marcadores))
	for i, m := range p.Marcadores {
		pts[i] = m.Ponto
	}
	return pts
}

// PlanejarIndividual: a propriedade, com o seu número fixo, e o enquadramento inicial.
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

// PlanejarGeral: as propriedades dos filtros do Painel (RN-81), na ordem dos números fixos.
func (s *CartaService) PlanejarGeral(filtro estatisticas.Filtro) (*Plano, error) {
	fs, err := s.familias(filtro)
	if err != nil {
		return nil, err
	}
	return montarPlano(fs, false)
}

// planoDe refaz o plano a partir dos UUIDs (para preparar e gerar com os dados atuais).
func (s *CartaService) planoDe(tipo string, uuids []string) (*Plano, error) {
	if len(uuids) == 0 {
		return nil, errors.New("nenhuma propriedade selecionada")
	}
	fs, err := s.familias(estatisticas.Filtro{})
	if err != nil {
		return nil, err
	}
	quer := map[string]bool{}
	for _, u := range uuids {
		quer[u] = true
	}
	var sel []estatisticas.Familia
	for _, f := range fs {
		if quer[f.UUID] {
			sel = append(sel, f)
		}
	}
	return montarPlano(sel, tipo == "individual")
}

// ---------- preparo da imagem (tarefa em segundo plano) ----------

type tarefa struct {
	cancelar context.CancelFunc
	mu       sync.Mutex
	estado   Andamento
	fim      time.Time
}

// Andamento de uma tarefa de preparo (consultado pela tela).
type Andamento struct {
	Fase       string            `json:"fase"`
	Feito      int               `json:"feito"`
	Total      int               `json:"total"`
	BytesRede  int64             `json:"bytes_rede"`
	Concluida  bool              `json:"concluida"`
	Erro       string            `json:"erro"`
	SemConexao bool              `json:"sem_conexao"`
	Resultado  *ResultadoPreparo `json:"resultado"`
}

// FolhaPrevia: uma folha da carta (para desenhar o contorno na prévia).
type FolhaPrevia struct {
	Rotulo       string    `json:"rotulo"`
	SO           geo.Ponto `json:"so"`
	NE           geo.Ponto `json:"ne"`
	Propriedades int       `json:"propriedades"`
}

// ResultadoPreparo: a composição pronta para a prévia e a carta.
type ResultadoPreparo struct {
	ComposicaoID string             `json:"composicao_id"`
	Tiles        string             `json:"tiles"` // modelo de URL dos tiles
	ZMin         int                `json:"z_min"`
	ZGeral       int                `json:"z_geral"`
	ZNativo      int                `json:"z_nativo"`
	Geral        FolhaPrevia        `json:"geral"`
	Setores      []FolhaPrevia      `json:"setores"`
	Partes       []composicao.Parte `json:"partes"`
	Mosaico      bool               `json:"mosaico"`
	ResolucaoM   float64            `json:"resolucao_m"`
	Guardada     bool               `json:"guardada"`
	Reserva      bool               `json:"reserva"`
	CriadaEm     time.Time          `json:"criada_em"`
	Avisos       []string           `json:"avisos"`
	// Anterior: sem internet, as coordenadas mudaram e foi usada a imagem preparada antes
	// para estas propriedades, que ainda cobre todas elas (RN-98).
	Anterior bool `json:"anterior"`
}

// PedidoPreparo: as propriedades (UUIDs do plano) e se a busca deve ser refeita.
type PedidoPreparo struct {
	Tipo      string   `json:"tipo"` // individual | geral
	UUIDs     []string `json:"uuids"`
	Atualizar bool     `json:"atualizar"` // "Atualizar melhor imagem": ignora a composição guardada
}

// ErrReferenciasDesatualizadas: a imagem foi preparada para outras coordenadas (RN-98).
var ErrReferenciasDesatualizadas = errors.New("a localização de propriedade(s) mudou depois que a imagem foi preparada; use \"Atualizar referências\" antes de gerar a carta")

// chaveSelecao identifica a seleção (tipo + propriedades), independentemente das coordenadas.
func chaveSelecao(tipo string, uuids []string) string {
	us := append([]string(nil), uuids...)
	sort.Strings(us)
	h := sha256.Sum256([]byte(tipo + "|" + strings.Join(us, ",")))
	return hex.EncodeToString(h[:16])
}

func novoID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func folhaPrevia(f geo.Folha) FolhaPrevia {
	return FolhaPrevia{Rotulo: f.Rotulo, SO: f.Retangulo.SO(), NE: f.Retangulo.NE(), Propriedades: len(f.Indices)}
}

func resultado(c *composicao.Composicao) *ResultadoPreparo {
	r := &ResultadoPreparo{ComposicaoID: c.ID, Tiles: "/cartografia/composicoes/" + c.ID + "/{z}/{x}/{y}.jpg", ZMin: c.ZMin, ZGeral: c.ZGeral, ZNativo: c.ZNativo,
		Geral: folhaPrevia(c.Folhas.Geral), Setores: []FolhaPrevia{}, Partes: c.Partes, Mosaico: c.Mosaico, ResolucaoM: c.ResolucaoM,
		Guardada: c.Guardada, Reserva: c.Reserva, CriadaEm: c.CriadaEm, Avisos: c.Avisos}
	for _, f := range c.Folhas.Setores {
		r.Setores = append(r.Setores, folhaPrevia(f))
	}
	if r.Avisos == nil {
		r.Avisos = []string{}
	}
	return r
}

// Preparar inicia a busca e a composição da imagem em segundo plano; devolve o id da tarefa.
func (s *CartaService) Preparar(p PedidoPreparo) (string, error) {
	plano, err := s.planoDe(p.Tipo, p.UUIDs)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithCancel(context.Background())
	t := &tarefa{cancelar: cancel, estado: Andamento{Fase: "Iniciando"}}
	id := novoID()
	s.mu.Lock()
	if s.tarefas == nil {
		s.tarefas = map[string]*tarefa{}
	}
	for k, v := range s.tarefas { // esquece tarefas concluídas há mais de 10 minutos
		if !v.fim.IsZero() && time.Since(v.fim) > 10*time.Minute {
			delete(s.tarefas, k)
		}
	}
	s.tarefas[id] = t
	s.mu.Unlock()
	cs := s.servicoComposicao()
	go func() {
		defer cancel()
		comp, err := cs.Preparar(ctx, plano.pontos(), configFolhas(plano.Individual), func(pr composicao.Progresso) {
			t.mu.Lock()
			t.estado.Fase, t.estado.Feito, t.estado.Total, t.estado.BytesRede = pr.Fase, pr.Feito, pr.Total, pr.BytesRede
			t.mu.Unlock()
		}, !p.Atualizar)
		anterior := false
		sel := chaveSelecao(p.Tipo, plano.UUIDs)
		if err != nil && errors.Is(err, cartografia.ErrSemConexao) && ctx.Err() == nil {
			// Sem internet e sem imagem para as coordenadas atuais: a imagem preparada antes
			// para estas mesmas propriedades serve se ainda cobrir todos os pinos (RN-98).
			if ant, ok := cs.DaSelecao(sel); ok && ant.Cobre(plano.pontos()) {
				ant.Guardada = true
				comp, err, anterior = ant, nil, true
			}
		}
		if err == nil && !anterior {
			_ = cs.LembrarSelecao(sel, comp.ID) // só um atalho para o uso sem internet
		}
		t.mu.Lock()
		defer t.mu.Unlock()
		t.estado.Concluida, t.fim = true, time.Now()
		switch {
		case errors.Is(err, context.Canceled):
			t.estado.Erro = "Operação cancelada."
		case err != nil:
			t.estado.Erro = err.Error()
			t.estado.SemConexao = errors.Is(err, cartografia.ErrSemConexao)
			if t.estado.SemConexao {
				t.estado.Erro = cartografia.ErrSemConexao.Error()
			}
		default:
			t.estado.Resultado = resultado(comp)
			t.estado.Resultado.Anterior = anterior // a tela explica e oferece "Atualizar referências"
		}
	}()
	return id, nil
}

// Andamento da tarefa.
func (s *CartaService) Andamento(id string) (*Andamento, error) {
	s.mu.Lock()
	t, ok := s.tarefas[id]
	s.mu.Unlock()
	if !ok {
		return nil, errors.New("tarefa não encontrada")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	a := t.estado
	return &a, nil
}

// Cancelar a tarefa (a composição incompleta não é guardada).
func (s *CartaService) Cancelar(id string) {
	s.mu.Lock()
	t, ok := s.tarefas[id]
	s.mu.Unlock()
	if ok {
		t.cancelar()
	}
}

// ---------- cartas ----------

// PedidoCarta: tipo, propriedades (UUIDs), composição preparada e filtros.
type PedidoCarta struct {
	Tipo         string   `json:"tipo"` // individual | geral
	UUIDs        []string `json:"uuids"`
	ComposicaoID string   `json:"composicao_id"`
	Criterios    []string `json:"criterios"`
	// EnquadramentoAnterior: a tela aceitou a imagem preparada antes (sem internet, RN-98).
	EnquadramentoAnterior bool `json:"enquadramento_anterior"`
}

// Tolerância de pixels sem nenhuma imagem (bordas): acima disso a folha não é gerada (RN-93).
const toleranciaSemDados = 0.0005

// marcadoresEm: os pinos que caem dentro do retângulo (com folga do raio do pino).
func marcadoresEm(ms []carta.Marcador, r geo.Retangulo) []carta.Marcador {
	var out []carta.Marcador
	for _, m := range ms {
		if r.Contem(m.Ponto) {
			out = append(out, m)
		}
	}
	return out
}

// GerarCarta monta a carta (folha geral + setores) em PDF e um PNG por folha.
func (s *CartaService) GerarCarta(p PedidoCarta) (*cartografia.CartaGuardada, error) {
	plano, err := s.planoDe(p.Tipo, p.UUIDs)
	if err != nil {
		return nil, err
	}
	cs := s.servicoComposicao()
	comp, err := cs.Carregar(p.ComposicaoID)
	if err != nil {
		return nil, errors.New("prepare a imagem antes de gerar a carta")
	}
	// A imagem precisa ter sido preparada para as coordenadas atuais; a anterior só vale
	// quando aceita sem internet e se ainda cobre todos os pinos (RN-98).
	pts := plano.pontos()
	if !cs.Corresponde(comp, pts, configFolhas(plano.Individual)) && !(p.EnquadramentoAnterior && comp.Cobre(pts)) {
		return nil, ErrReferenciasDesatualizadas
	}
	var fora []string
	for _, m := range plano.Marcadores {
		if !comp.Folhas.Geral.Retangulo.Contem(m.Ponto) {
			fora = append(fora, fmt.Sprintf("%d %s", m.Numero, m.Nome))
		}
	}
	if len(fora) > 0 {
		return nil, fmt.Errorf("a imagem preparada não cobre %d propriedade(s) (%s); use \"Atualizar melhor imagem\"", len(fora), strings.Join(fora, "; "))
	}
	partes := make([]carta.Proveniencia, 0, len(comp.Partes))
	atrib := []string{}
	for _, pt := range comp.Partes {
		partes = append(partes, carta.Proveniencia{Rotulo: pt.Rotulo, Periodo: pt.Periodo, ResolucaoM: pt.ResolucaoM, Fracao: pt.Fracao, Estimada: pt.NuvemEstimada})
		if pt.Fracao > 0 && !contem(atrib, pt.Atribuicao) {
			atrib = append(atrib, pt.Atribuicao)
		}
	}
	subt := plano.Marcadores[0].Nome
	if p.Tipo != "individual" {
		subt = fmt.Sprintf("%d propriedade(s)", len(plano.Marcadores))
	}
	agora := s.agora()
	base := carta.Dados{Tipo: p.Tipo, Subtitulo: subt, Partes: partes, Mosaico: comp.Mosaico, Atribuicao: strings.Join(atrib, " · "),
		SemLocalizacao: plano.SemLocalizacao, Criterios: p.Criterios, Estudo: s.estudo(plano.estudos), GeradoEm: agora}
	folhas := append([]geo.Folha{comp.Folhas.Geral}, comp.Folhas.Setores...)
	var paginas []carta.Pagina
	arquivos := map[string][]byte{}
	for i, f := range folhas {
		img, cob, err := cs.Folha(context.Background(), comp.ID, i)
		if err != nil {
			return nil, err
		}
		if 1-cob > toleranciaSemDados {
			return nil, cartografia.ErrImagemIncompleta
		}
		d := base
		d.Imagem = &cartografia.Imagem{Ret: f.Retangulo, Largura: img.Bounds().Dx(), Altura: img.Bounds().Dy(), ResolucaoNativaM: comp.ResolucaoM}
		d.Base = img
		if i == 0 {
			d.Marcadores = plano.Marcadores
			for _, st := range comp.Folhas.Setores {
				d.Setores = append(d.Setores, carta.SetorIndice{Rotulo: st.Rotulo, Retangulo: st.Retangulo})
			}
			if len(folhas) > 1 {
				d.Referencia = fmt.Sprintf("Carta geral (folha 1 de %d); os setores estão nas folhas seguintes", len(folhas))
			}
		} else {
			d.Marcadores = marcadoresEm(plano.Marcadores, f.Retangulo)
			d.Folha = "Setor " + f.Rotulo
			d.Subtitulo = fmt.Sprintf("%s — Setor %s (%d propriedade(s))", subt, f.Rotulo, len(d.Marcadores))
			d.Referencia = fmt.Sprintf("Setor %s (folha %d de %d); localização dos setores na carta geral (folha 1)", f.Rotulo, i+1, len(folhas))
		}
		d.NomesNoMapa = len(d.Marcadores) <= 12
		_, mapa, err := carta.MapaPNG(d)
		if err != nil {
			return nil, err
		}
		folhaPNG, err := carta.FolhaPNG(d, mapa)
		if err != nil {
			return nil, err
		}
		nome := "carta-01-geral.png"
		if i > 0 {
			nome = fmt.Sprintf("carta-%02d-setor-%s.png", i+1, f.Rotulo)
		}
		arquivos[nome] = folhaPNG
		paginas = append(paginas, carta.Pagina{Dados: d, Mapa: mapa})
	}
	pdf, err := carta.CartaPDF(paginas)
	if err != nil {
		return nil, err
	}
	arquivos["carta.pdf"] = pdf
	periodos := []string{}
	for _, pt := range comp.Partes {
		if pt.Fracao > 0 && !contem(periodos, pt.Periodo) {
			periodos = append(periodos, pt.Periodo)
		}
	}
	reg, err := s.armazem().GravarCarta(cartografia.CartaGuardada{Titulo: carta.Titulo + " — " + subt, Tipo: p.Tipo, Propriedades: len(plano.Marcadores),
		Periodo: strings.Join(periodos, "; "), Criterios: p.Criterios, GeradaEm: agora}, arquivos)
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

// AbrirCarta abre o PDF ou um PNG no programa padrão.
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

// SalvarCopia grava uma cópia de um arquivo da carta onde o operador escolher; "" se cancelar.
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
	sufixo := strings.TrimSuffix(strings.TrimPrefix(arquivo, "carta"), ext)
	destino, err := s.C.Arquivos.EscolherDestino(relatorios.NomeArquivo("SITPOSSE-7", titulo, id[:8])+sufixo+ext, descr, ext)
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
	reArqCarta    = regexp.MustCompile(`^/cartografia/cartas/([0-9]{8}-[0-9]{6}-[0-9a-f]{8})/(carta(?:-[0-9]{2}-(?:geral|setor-[A-Z]{1,3}))?\.png)$`)
	reTile        = regexp.MustCompile(`^/cartografia/composicoes/([0-9a-f]{32})/([0-9]{1,2})/([0-9]{1,7})/([0-9]{1,7})\.jpg$`)
)

// pngVazio: tile transparente de 1 × 1 (fora da composição).
var pngVazio = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01,
	0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c,
	0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82}

// HandlerCartografia serve à tela as imagens guardadas, os tiles das composições e os PNG das
// cartas (só leitura). É função do pacote (não método) para não ser exposta pelo Wails.
func HandlerCartografia(s *CartaService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		if m := reTile.FindStringSubmatch(r.URL.Path); m != nil {
			z, _ := strconv.Atoi(m[2])
			x, _ := strconv.Atoi(m[3])
			y, _ := strconv.Atoi(m[4])
			b, err := s.servicoComposicao().Tile(r.Context(), m[1], z, x, y)
			w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
			if err != nil || b == nil {
				w.Header().Set("Content-Type", "image/png")
				if err != nil {
					w.Header().Set("Cache-Control", "no-store") // tenta de novo (pode ter sido a rede)
				}
				_, _ = w.Write(pngVazio)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(b)
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
