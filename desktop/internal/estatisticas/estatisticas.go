// Package estatisticas implementa o FiltroEstudo da Central (Fase 4, A25;
// docs/requirements/estatisticas.md §22-A).
//
// Um único cálculo (Calcular) produz, a partir das propriedades vigentes (RN-80) e de um
// Filtro, todos os números que a tela mostra: totais, grupos por equipe e por área
// identificada, tipos de item, faixas de área e a relação de famílias. Por isso cartões,
// gráficos e tabelas são sempre coerentes entre si (RN-81, 22-A.5).
//
// Para incluir um filtro novo sobre uma quantidade basta acrescentar uma entrada em
// camposFixos (ou um tipo de item no catálogo): as condições, as colunas e os totais
// passam a tratá-la sem outra mudança.
package estatisticas

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"sitposse7/desktop/internal/dominio"
)

// ErrFiltroInvalido indica um filtro com campo ou operador desconhecido.
var ErrFiltroInvalido = errors.New("filtro inválido")

// Condicao restringe as famílias por uma quantidade (22-A.2 "presença ou ausência",
// "faixas numéricas"): por exemplo, bovinos > 5 ou "não tem geladeira".
type Condicao struct {
	Campo    string `json:"campo"`
	Operador string `json:"operador"` // tem | nao_tem | maior | pelo_menos | menor | igual
	Valor    int64  `json:"valor"`
}

// Filtro é o FiltroEstudo: todos os critérios são combinados com "e"; listas vazias
// e ponteiros nulos não filtram.
type Filtro struct {
	Estudo         string     `json:"estudo"`   // estudo em foco ("" = todos); define a base (A28)
	Jornadas       []string   `json:"jornadas"` // jornadas do estudo
	Equipes        []string   `json:"equipes"`
	Busca          string     `json:"busca"`         // nome da propriedade ou responsável, sem acento
	PeriodoCampo   string     `json:"periodo_campo"` // cadastro | recebimento
	De             *int64     `json:"de"`            // ms UTC, inclusive
	Ate            *int64     `json:"ate"`           // ms UTC, inclusive
	ComLocalizacao *bool      `json:"com_localizacao"`
	ComFoto        *bool      `json:"com_foto"`
	ComArea        *bool      `json:"com_area"`
	FaixasArea     []string   `json:"faixas_area"`
	AreaMinHa      *float64   `json:"area_min_ha"` // inclusive
	AreaMaxHa      *float64   `json:"area_max_ha"` // inclusive
	Condicoes      []Condicao `json:"condicoes"`
}

// Campo é uma quantidade que pode ser somada, filtrada e mostrada em coluna.
type Campo struct {
	Chave     string `json:"chave"`
	Rotulo    string `json:"rotulo"`
	Grupo     string `json:"grupo"`
	Categoria string `json:"categoria,omitempty"` // só itens: eletrodomestico | movel | ave
	Outros    bool   `json:"outros,omitempty"`    // item personalizado ("Outros")
}

// Familia é uma propriedade vigente (cada propriedade corresponde a uma família).
type Familia struct {
	UUID           string           `json:"uuid"`
	Nome           string           `json:"nome"`
	Responsavel    string           `json:"responsavel"`
	Observacoes    *string          `json:"observacoes"`
	EquipeUUID     string           `json:"equipe_uuid"`
	Equipe         string           `json:"equipe"`
	EquipeCodigo   string           `json:"equipe_codigo"`
	JornadaUUID    string           `json:"jornada_uuid"`
	Jornada        string           `json:"jornada"` // "Jornada 2 · 11/10/2026"
	JornadaNumero  int              `json:"jornada_numero"`
	JornadaData    string           `json:"jornada_data"`
	EstudoUUID     string           `json:"estudo_uuid"`
	Estudo         string           `json:"estudo"`
	Valores        map[string]int64 `json:"valores"`
	AreaDm2        *int64           `json:"area_dm2"`
	AreaHa         *string          `json:"area_ha"`
	AreaUnidade    *string          `json:"area_unidade"`
	ComLocalizacao bool             `json:"com_localizacao"`
	Lat            *float64         `json:"lat"`
	Lon            *float64         `json:"lon"`
	GpsOrigem      *string          `json:"gps_origem"` // gnss | manual
	GpsPrecisaoM   *float64         `json:"gps_precisao_m"`
	CriadoEm       int64            `json:"criado_em"`
	RecebidoEm     int64            `json:"recebido_em"`
}

// Totais de um conjunto de famílias.
type Totais struct {
	Familias       int              `json:"familias"`
	Valores        map[string]int64 `json:"valores"`
	AreaDm2        int64            `json:"area_dm2"`
	AreaHa         string           `json:"area_ha"`
	AreaMediaHa    *string          `json:"area_media_ha"` // média entre as famílias com área
	ComArea        int              `json:"com_area"`
	SemArea        int              `json:"sem_area"`
	ComLocalizacao int              `json:"com_localizacao"`
	SemLocalizacao int              `json:"sem_localizacao"`
	ComFoto        int              `json:"com_foto"`
	SemFoto        int              `json:"sem_foto"`
}

// Grupo são os totais de uma equipe ou de uma jornada.
type Grupo struct {
	Chave  string `json:"chave"`
	Nome   string `json:"nome"`
	Totais Totais `json:"totais"`
}

// TipoItem é o total de um tipo de item (predefinido ou grupo de "Outros").
type TipoItem struct {
	Campo    Campo `json:"campo"`
	Total    int64 `json:"total"`
	Familias int   `json:"familias"` // famílias com quantidade > 0
}

// Faixa de área (22-A.2 "faixas de tamanho").
type Faixa struct {
	Chave    string `json:"chave"`
	Rotulo   string `json:"rotulo"`
	Familias int    `json:"familias"`
	AreaDm2  int64  `json:"area_dm2"`
	AreaHa   string `json:"area_ha"`
}

// Opcao para listas de seleção da tela.
type Opcao struct {
	Valor  string `json:"valor"`
	Rotulo string `json:"rotulo"`
}

// Resultado de um cálculo.
type Resultado struct {
	FamiliasBase int        `json:"familias_base"` // total sem filtros, para "N de M famílias"
	Totais       Totais     `json:"totais"`
	PorEquipe    []Grupo    `json:"por_equipe"`
	PorJornada   []Grupo    `json:"por_jornada"`
	Itens        []TipoItem `json:"itens"`
	FaixasArea   []Faixa    `json:"faixas_area"`
	Familias     []Familia  `json:"familias"`
	Campos       []Campo    `json:"campos"`
	Equipes      []Opcao    `json:"equipes"`
	Jornadas     []Opcao    `json:"jornadas"`
	Estudos      []Opcao    `json:"estudos"`
}

// ---------- campos ----------

// Chaves dos totais de cada categoria de itens.
const (
	AvesTotal   = "aves_total"
	EletroTotal = "eletro_total"
	MoveisTotal = "moveis_total"
	Fotos       = "fotos"
	prefixoItem = "item:"
)

// Quantidades fixas da propriedade, na ordem de exibição.
var camposFixos = []Campo{
	{Chave: "pessoas_total", Rotulo: "Pessoas (total)", Grupo: "Pessoas"},
	{Chave: "pessoas_adultos", Rotulo: "Adultos (18+)", Grupo: "Pessoas"},
	{Chave: "pessoas_jovens", Rotulo: "Jovens (12–17)", Grupo: "Pessoas"},
	{Chave: "pessoas_criancas", Rotulo: "Crianças (0–11)", Grupo: "Pessoas"},
	{Chave: "veic_total", Rotulo: "Veículos (total)", Grupo: "Veículos"},
	{Chave: "veic_caminhoes", Rotulo: "Caminhões", Grupo: "Veículos"},
	{Chave: "veic_automoveis_utilitarios", Rotulo: "Automóveis, caminhonetes e utilitários", Grupo: "Veículos"},
	{Chave: "veic_motos_ciclomotores", Rotulo: "Motocicletas e ciclomotores", Grupo: "Veículos"},
	{Chave: "bov_total", Rotulo: "Bovinos (total)", Grupo: "Bovinos"},
	{Chave: "bov_adultos", Rotulo: "Bovinos adultos", Grupo: "Bovinos"},
	{Chave: "bov_bezerros", Rotulo: "Bezerros e bezerras", Grupo: "Bovinos"},
	{Chave: "equ_total", Rotulo: "Equinos (total)", Grupo: "Equinos"},
	{Chave: "equ_adultos", Rotulo: "Equinos adultos", Grupo: "Equinos"},
	{Chave: "equ_potros", Rotulo: "Potros", Grupo: "Equinos"},
	{Chave: AvesTotal, Rotulo: "Aves (total)", Grupo: "Aves"},
	{Chave: EletroTotal, Rotulo: "Eletrodomésticos (total)", Grupo: "Eletrodomésticos"},
	{Chave: MoveisTotal, Rotulo: "Móveis (total)", Grupo: "Móveis"},
	{Chave: Fotos, Rotulo: "Fotos", Grupo: "Fotos"},
}

// Grupo de exibição e total de cada categoria de item.
var categorias = map[string]struct{ grupo, total string }{
	"ave":             {"Aves", AvesTotal},
	"eletrodomestico": {"Eletrodomésticos", EletroTotal},
	"movel":           {"Móveis", MoveisTotal},
}

var ordemCategoria = map[string]int{"ave": 0, "eletrodomestico": 1, "movel": 2}

// ChaveItem é a chave de um tipo de item: o código do catálogo ou, para "Outros",
// a categoria e o nome normalizado (RN-35: o texto original nunca é alterado).
func ChaveItem(codigo, nomeNormalizado string) string {
	if strings.HasSuffix(codigo, ".outros") && nomeNormalizado != "" {
		return prefixoItem + codigo + ":" + nomeNormalizado
	}
	return prefixoItem + codigo
}

// ---------- faixas de área ----------

const dm2PorHa = 1_000_000 // 1 ha = 10.000 m² = 1.000.000 dm²

var faixas = []struct {
	chave, rotulo string
	min, max      int64 // dm², [min, max); max 0 = sem limite
}{
	{"ate10", "Até 10 ha", 0, 10 * dm2PorHa},
	{"10a25", "De 10 a 25 ha", 10 * dm2PorHa, 25 * dm2PorHa},
	{"25a50", "De 25 a 50 ha", 25 * dm2PorHa, 50 * dm2PorHa},
	{"50a100", "De 50 a 100 ha", 50 * dm2PorHa, 100 * dm2PorHa},
	{"acima100", "100 ha ou mais", 100 * dm2PorHa, 0},
}

const faixaSemArea = "sem_area"

func faixaDe(dm2 *int64) string {
	if dm2 == nil {
		return faixaSemArea
	}
	for _, f := range faixas {
		if *dm2 >= f.min && (f.max == 0 || *dm2 < f.max) {
			return f.chave
		}
	}
	return faixaSemArea // inalcançável: áreas são >= 0
}

// ---------- cálculo ----------

// Base são as famílias vigentes e os tipos de item conhecidos (catálogo + "Outros" recebidos).
type Base struct {
	Familias []Familia
	Itens    []Campo // tipos de item (Chave com prefixo "item:")
	Estudos  []Opcao
}

func validar(f Filtro, conhecidos map[string]bool) error {
	for _, c := range f.Condicoes {
		if !conhecidos[c.Campo] && !strings.HasPrefix(c.Campo, prefixoItem) {
			return fmt.Errorf("%w: campo %q desconhecido", ErrFiltroInvalido, c.Campo)
		}
		switch c.Operador {
		case "tem", "nao_tem", "maior", "pelo_menos", "menor", "igual":
		default:
			return fmt.Errorf("%w: operador %q desconhecido", ErrFiltroInvalido, c.Operador)
		}
		if c.Valor < 0 {
			return fmt.Errorf("%w: valor negativo em %q", ErrFiltroInvalido, c.Campo)
		}
	}
	switch f.PeriodoCampo {
	case "", "cadastro", "recebimento":
	default:
		return fmt.Errorf("%w: período %q desconhecido", ErrFiltroInvalido, f.PeriodoCampo)
	}
	for _, x := range []*float64{f.AreaMinHa, f.AreaMaxHa} {
		if x != nil && (math.IsNaN(*x) || *x < 0) {
			return fmt.Errorf("%w: área deve ser um número maior ou igual a zero", ErrFiltroInvalido)
		}
	}
	return nil
}

func contem(lista []string, v string) bool {
	for _, x := range lista {
		if x == v {
			return true
		}
	}
	return false
}

func atende(c Condicao, v int64) bool {
	switch c.Operador {
	case "tem":
		return v > 0
	case "nao_tem":
		return v == 0
	case "maior":
		return v > c.Valor
	case "pelo_menos":
		return v >= c.Valor
	case "menor":
		return v < c.Valor
	default: // igual
		return v == c.Valor
	}
}

func haParaDm2(ha float64) int64 { return int64(math.Round(ha * dm2PorHa)) }

// passa diz se a família atende ao filtro (busca já normalizada).
func passa(p *Familia, f *Filtro, termo string) bool {
	if f.Estudo != "" && p.EstudoUUID != f.Estudo {
		return false
	}
	if len(f.Jornadas) > 0 && !contem(f.Jornadas, p.JornadaUUID) {
		return false
	}
	if len(f.Equipes) > 0 && !contem(f.Equipes, p.EquipeUUID) {
		return false
	}
	if termo != "" {
		nome, _ := dominio.NormalizarNome(p.Nome)
		resp, _ := dominio.NormalizarNome(p.Responsavel)
		if !strings.Contains(nome, termo) && !strings.Contains(resp, termo) {
			return false
		}
	}
	if f.De != nil || f.Ate != nil {
		t := p.CriadoEm
		if f.PeriodoCampo == "recebimento" {
			t = p.RecebidoEm
		}
		if (f.De != nil && t < *f.De) || (f.Ate != nil && t > *f.Ate) {
			return false
		}
	}
	if f.ComLocalizacao != nil && p.ComLocalizacao != *f.ComLocalizacao {
		return false
	}
	if f.ComFoto != nil && (p.Valores[Fotos] > 0) != *f.ComFoto {
		return false
	}
	if f.ComArea != nil && (p.AreaDm2 != nil) != *f.ComArea {
		return false
	}
	if len(f.FaixasArea) > 0 && !contem(f.FaixasArea, faixaDe(p.AreaDm2)) {
		return false
	}
	if f.AreaMinHa != nil || f.AreaMaxHa != nil {
		// Ausência de área não é zero (AGENTS.md §3): sem área nunca entra numa faixa numérica.
		if p.AreaDm2 == nil {
			return false
		}
		if (f.AreaMinHa != nil && *p.AreaDm2 < haParaDm2(*f.AreaMinHa)) || (f.AreaMaxHa != nil && *p.AreaDm2 > haParaDm2(*f.AreaMaxHa)) {
			return false
		}
	}
	for _, c := range f.Condicoes {
		if !atende(c, p.Valores[c.Campo]) {
			return false
		}
	}
	return true
}

func novoTotais() Totais { return Totais{Valores: map[string]int64{}} }

func (t *Totais) somar(p *Familia) {
	t.Familias++
	for k, v := range p.Valores {
		t.Valores[k] += v
	}
	if p.AreaDm2 != nil {
		t.ComArea++
		t.AreaDm2 += *p.AreaDm2
	} else {
		t.SemArea++
	}
	if p.ComLocalizacao {
		t.ComLocalizacao++
	} else {
		t.SemLocalizacao++
	}
	if p.Valores[Fotos] > 0 {
		t.ComFoto++
	} else {
		t.SemFoto++
	}
}

func (t *Totais) fechar() {
	t.AreaHa = dominio.FormatarHa(t.AreaDm2)
	if t.ComArea > 0 {
		m := dominio.FormatarHa(int64(math.Round(float64(t.AreaDm2) / float64(t.ComArea))))
		t.AreaMediaHa = &m
	}
}

// Calcular aplica o filtro à base e produz todos os números da tela a partir do mesmo
// conjunto de famílias (RN-81).
func Calcular(b *Base, f Filtro) (*Resultado, error) {
	conhecidos := map[string]bool{}
	campos := append([]Campo{}, camposFixos...)
	for _, c := range campos {
		conhecidos[c.Chave] = true
	}
	itens := append([]Campo{}, b.Itens...)
	sort.SliceStable(itens, func(i, j int) bool {
		if ordemCategoria[itens[i].Categoria] != ordemCategoria[itens[j].Categoria] {
			return ordemCategoria[itens[i].Categoria] < ordemCategoria[itens[j].Categoria]
		}
		if itens[i].Outros != itens[j].Outros {
			return !itens[i].Outros
		}
		return false
	})
	campos = append(campos, itens...)
	if err := validar(f, conhecidos); err != nil {
		return nil, err
	}
	termo, _ := dominio.NormalizarNome(f.Busca)

	r := &Resultado{Totais: novoTotais(), Campos: campos, Estudos: b.Estudos, Familias: []Familia{}}
	porEquipe := map[string]*Grupo{}
	porJornada := map[string]*Grupo{}
	porFaixa := map[string]*Faixa{}
	type opcaoOrdem struct {
		Opcao
		numero int
	}
	equipes := map[string]opcaoOrdem{}
	jornadas := map[string]opcaoOrdem{}
	ordemEq, ordemJor := []string{}, []string{}

	for i := range b.Familias {
		p := &b.Familias[i]
		if f.Estudo != "" && p.EstudoUUID != f.Estudo {
			continue
		}
		r.FamiliasBase++
		// Opções de seleção vêm de toda a base do estudo (para não sumirem ao filtrar).
		if _, ok := equipes[p.EquipeUUID]; !ok {
			equipes[p.EquipeUUID] = opcaoOrdem{Opcao{p.EquipeUUID, RotuloEquipe(p)}, p.JornadaNumero}
		}
		if _, ok := jornadas[p.JornadaUUID]; !ok {
			jornadas[p.JornadaUUID] = opcaoOrdem{Opcao{p.JornadaUUID, p.Jornada}, p.JornadaNumero}
		}
		if !passa(p, &f, termo) {
			continue
		}
		r.Familias = append(r.Familias, *p)
		r.Totais.somar(p)

		g, ok := porEquipe[p.EquipeUUID]
		if !ok {
			g = &Grupo{Chave: p.EquipeUUID, Nome: RotuloEquipe(p), Totais: novoTotais()}
			porEquipe[p.EquipeUUID] = g
			ordemEq = append(ordemEq, p.EquipeUUID)
		}
		g.Totais.somar(p)

		gj, ok := porJornada[p.JornadaUUID]
		if !ok {
			gj = &Grupo{Chave: p.JornadaUUID, Nome: p.Jornada, Totais: novoTotais()}
			porJornada[p.JornadaUUID] = gj
			ordemJor = append(ordemJor, p.JornadaUUID)
		}
		gj.Totais.somar(p)

		fx := faixaDe(p.AreaDm2)
		if porFaixa[fx] == nil {
			porFaixa[fx] = &Faixa{Chave: fx}
		}
		porFaixa[fx].Familias++
		if p.AreaDm2 != nil {
			porFaixa[fx].AreaDm2 += *p.AreaDm2
		}
	}
	r.Totais.fechar()

	// Equipes e jornadas na ordem das jornadas (e, dentro delas, pelo nome).
	ordenar := func(ks []string, op map[string]opcaoOrdem) {
		sort.Slice(ks, func(i, j int) bool {
			a, b := op[ks[i]], op[ks[j]]
			if a.numero != b.numero {
				return a.numero < b.numero
			}
			return comparar(a.Rotulo, b.Rotulo) < 0
		})
	}
	ordenar(ordemEq, equipes)
	for _, k := range ordemEq {
		g := porEquipe[k]
		g.Totais.fechar()
		r.PorEquipe = append(r.PorEquipe, *g)
	}
	ordenar(ordemJor, jornadas)
	for _, k := range ordemJor {
		g := porJornada[k]
		g.Totais.fechar()
		r.PorJornada = append(r.PorJornada, *g)
	}

	for _, fx := range faixas {
		x := Faixa{Chave: fx.chave, Rotulo: fx.rotulo}
		if p := porFaixa[fx.chave]; p != nil {
			x.Familias, x.AreaDm2 = p.Familias, p.AreaDm2
		}
		x.AreaHa = dominio.FormatarHa(x.AreaDm2)
		r.FaixasArea = append(r.FaixasArea, x)
	}
	sem := Faixa{Chave: faixaSemArea, Rotulo: "Sem área informada", AreaHa: dominio.FormatarHa(0)}
	if p := porFaixa[faixaSemArea]; p != nil {
		sem.Familias = p.Familias
	}
	r.FaixasArea = append(r.FaixasArea, sem)

	for _, c := range itens {
		t := TipoItem{Campo: c, Total: r.Totais.Valores[c.Chave]}
		for i := range r.Familias {
			if r.Familias[i].Valores[c.Chave] > 0 {
				t.Familias++
			}
		}
		if t.Total > 0 || !c.Outros {
			r.Itens = append(r.Itens, t)
		}
	}

	opcoes := func(op map[string]opcaoOrdem) []Opcao {
		ks := make([]string, 0, len(op))
		for k := range op {
			ks = append(ks, k)
		}
		ordenar(ks, op)
		out := make([]Opcao, 0, len(ks))
		for _, k := range ks {
			out = append(out, op[k].Opcao)
		}
		return out
	}
	r.Equipes, r.Jornadas = opcoes(equipes), opcoes(jornadas)
	if r.PorEquipe == nil {
		r.PorEquipe = []Grupo{}
	}
	if r.PorJornada == nil {
		r.PorJornada = []Grupo{}
	}
	if r.Estudos == nil {
		r.Estudos = []Opcao{}
	}
	if r.Itens == nil {
		r.Itens = []TipoItem{}
	}
	return r, nil
}

// RotuloEquipe distingue equipes de mesmo nome em jornadas diferentes: "Alfa · J2 (11/10/2026)".
func RotuloEquipe(p *Familia) string {
	return fmt.Sprintf("%s · J%d (%s)", p.Equipe, p.JornadaNumero, dataBR(p.JornadaData))
}

// RotuloJornada é o texto padrão de uma jornada: "Jornada 2 · 11/10/2026".
func RotuloJornada(numero int, data string) string {
	return fmt.Sprintf("Jornada %d · %s", numero, dataBR(data))
}

func dataBR(d string) string {
	if len(d) != 10 {
		return d
	}
	return d[8:] + "/" + d[5:7] + "/" + d[:4]
}

// NovaFamilia completa os valores derivados (totais por grupo) de uma família.
// valores deve trazer as quantidades fixas e os itens ("item:..."); categoriaItem
// diz a categoria de cada chave de item.
func NovaFamilia(p Familia, categoriaItem map[string]string) Familia {
	v := p.Valores
	if v == nil {
		v = map[string]int64{}
	}
	v["pessoas_total"] = v["pessoas_adultos"] + v["pessoas_jovens"] + v["pessoas_criancas"]
	v["veic_total"] = v["veic_caminhoes"] + v["veic_automoveis_utilitarios"] + v["veic_motos_ciclomotores"]
	v["bov_total"] = v["bov_adultos"] + v["bov_bezerros"]
	v["equ_total"] = v["equ_adultos"] + v["equ_potros"]
	v[AvesTotal], v[EletroTotal], v[MoveisTotal] = 0, 0, 0
	for k, q := range v {
		if cat, ok := categoriaItem[k]; ok {
			v[categorias[cat].total] += q
		}
	}
	p.Valores = v
	return p
}

// comparar ordena nomes sem diferenciar acentos e maiúsculas (pt-BR).
func comparar(a, b string) int {
	na, _ := dominio.NormalizarNome(a)
	nb, _ := dominio.NormalizarNome(b)
	if c := strings.Compare(na, nb); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}
