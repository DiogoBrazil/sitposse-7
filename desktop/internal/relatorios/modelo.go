// Package relatorios gera os relatórios da Central em PDF, DOCX e XLSX (A26, ADR-0021;
// docs/requirements/estatisticas.md §22-A; RF-C07).
//
// Os números vêm sempre do mesmo estatisticas.Resultado mostrado no Painel, com os filtros
// ativos (RN-81). PDF e DOCX são desenhados a partir de um único modelo de documento
// (Documento), para que os dois tenham o mesmo conteúdo; a planilha tem layout próprio,
// com células numéricas, filtros e totais que acompanham os filtros do Excel.
//
// Nada aqui acessa a rede: fontes e logotipo vão embutidos no executável.
package relatorios

import (
	_ "embed"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"sitposse7/desktop/internal/estatisticas"
)

// Cabeçalho institucional (texto do usuário, 14/11/2026).
const (
	Instituicao = "Polícia Militar de Rondônia"
	Unidade     = "7º Batalhão de Polícia Militar"
	Sistema     = "SITPOSSE-7"
)

//go:embed logo-cabecalho-docs.jpeg
var Logo []byte

// Logo: 150 × 182 px.
const logoLarguraPx, logoAlturaPx = 150, 182

// Dados de um relatório geral (panorama + famílias).
type Dados struct {
	Operacao  string
	GeradoEm  time.Time
	Criterios []string // filtros ativos, como na tela ("Equipe: Alfa", "Bovinos (total): mais de 5")
	R         *estatisticas.Resultado
	Fichas    bool // incluir uma ficha por família (PDF e DOCX)
}

// ---------- modelo de documento (PDF e DOCX) ----------

// Documento é o conteúdo independente de formato.
type Documento struct {
	Titulo    string
	Subtitulo string
	Meta      [][2]string // pares rótulo/valor sob o título
	Secoes    []Secao
	GeradoEm  time.Time
}

// Secao começa com um título; NovaPagina força a quebra antes dela.
type Secao struct {
	Titulo     string
	Subtitulo  string
	NovaPagina bool
	Blocos     []Bloco
}

// Bloco é um Paragrafo, Pares ou Tabela.
type Bloco interface{ bloco() }

type Paragrafo struct {
	Texto string
	Fraco bool
}

// Pares é uma lista rótulo: valor (dados de uma ficha).
type Pares struct{ Itens [][2]string }

// Tabela com cabeçalho, linhas e linha de total opcional. Peso define a largura relativa.
type Tabela struct {
	Titulo  string
	Colunas []Coluna
	Linhas  [][]string
	Total   []string
	Vazio   string
}

type Coluna struct {
	Rotulo string
	Peso   float64
}

func (Paragrafo) bloco() {}
func (Pares) bloco()     {}
func (Tabela) bloco()    {}

// ---------- formatação pt-BR ----------

// Inteiro com separador de milhar ("1.234").
func Int(n int64) string {
	s := strconv.FormatInt(int64(math.Abs(float64(n))), 10)
	var b strings.Builder
	if n < 0 {
		b.WriteByte('-')
	}
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// Ha formata hectares já em texto da Central ("1234,50" → "1.234,50").
func Ha(s string) string {
	inteiro, dec, ok := strings.Cut(s, ",")
	n, err := strconv.ParseInt(inteiro, 10, 64)
	if err != nil {
		return s
	}
	if !ok {
		return Int(n)
	}
	return Int(n) + "," + dec
}

// Quando formata data e hora no fuso local (ADR-0010).
func Quando(ms int64) string {
	if ms == 0 {
		return "—"
	}
	return time.UnixMilli(ms).Local().Format("02/01/2006 15:04")
}

func plural(n int, um, varios string) string {
	if n == 1 {
		return "1 " + um
	}
	return Int(int64(n)) + " " + varios
}

func val(m map[string]int64, k string) string { return Int(m[k]) }

func deref(s *string, padrao string) string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return padrao
	}
	return *s
}

// ---------- montagem ----------

var colunasResumo = []struct{ chave, rotulo string }{
	{"pessoas_total", "Pessoas"}, {"veic_total", "Veículos"}, {"bov_total", "Bovinos"}, {"equ_total", "Equinos"},
	{"aves_total", "Aves"}, {"eletro_total", "Eletrodom."}, {"moveis_total", "Móveis"},
}

func tabelaGrupos(titulo, rotulo string, gs []estatisticas.Grupo, t estatisticas.Totais) Tabela {
	cols := []Coluna{{rotulo, 2.2}, {"Famílias", 1}}
	for _, c := range colunasResumo {
		peso := 1.0
		if len([]rune(c.rotulo)) > 8 {
			peso = 1.4 // "Eletrodom." sem quebrar a palavra
		}
		cols = append(cols, Coluna{c.rotulo, peso})
	}
	cols = append(cols, Coluna{"Área (ha)", 1.9})
	linha := func(nome string, x estatisticas.Totais) []string {
		l := []string{nome, Int(int64(x.Familias))}
		for _, c := range colunasResumo {
			l = append(l, val(x.Valores, c.chave))
		}
		a := Ha(x.AreaHa)
		if x.SemArea > 0 {
			a += fmt.Sprintf(" (%d sem)", x.SemArea)
		}
		return append(l, a)
	}
	tb := Tabela{Titulo: titulo, Colunas: cols, Vazio: "Nenhuma família com os filtros aplicados."}
	for _, g := range gs {
		tb.Linhas = append(tb.Linhas, linha(g.Nome, g.Totais))
	}
	if len(gs) > 0 {
		tb.Total = linha("Total geral", t)
	}
	return tb
}

// Montar produz o relatório geral: panorama, grupos, itens, tamanhos, famílias e fichas.
func Montar(d Dados) Documento {
	r := d.R
	t := r.Totais
	v := t.Valores
	filtros := "Nenhum: todas as famílias recebidas"
	if len(d.Criterios) > 0 {
		filtros = strings.Join(d.Criterios, "; ")
	}
	doc := Documento{
		Titulo:    "Levantamento de propriedades rurais",
		Subtitulo: "Panorama geral e por família",
		GeradoEm:  d.GeradoEm,
		Meta: [][2]string{
			{"Operação", deref(&d.Operacao, "—")},
			{"Filtros aplicados", filtros},
			{"Famílias no relatório", fmt.Sprintf("%s de %s recebidas", Int(int64(t.Familias)), Int(int64(r.FamiliasBase)))},
			{"Gerado em", d.GeradoEm.Local().Format("02/01/2006 15:04")},
		},
	}

	media := "—"
	if t.AreaMediaHa != nil {
		media = Ha(*t.AreaMediaHa) + " ha"
	}
	geral := Tabela{Titulo: "Totais", Colunas: []Coluna{{"Assunto", 1.2}, {"Indicador", 2}, {"Quantidade", 1}}, Linhas: [][]string{
		{"Famílias", "Propriedades (famílias)", Int(int64(t.Familias))},
		{"Famílias", "Com localização (GPS ou digitada)", Int(int64(t.ComLocalizacao))},
		{"Famílias", "Sem localização", Int(int64(t.SemLocalizacao))},
		{"Famílias", "Com pelo menos uma foto", Int(int64(t.ComFoto))},
		{"Pessoas", "Total de pessoas", val(v, "pessoas_total")},
		{"Pessoas", "Adultos (18+)", val(v, "pessoas_adultos")},
		{"Pessoas", "Jovens (12–17)", val(v, "pessoas_jovens")},
		{"Pessoas", "Crianças (0–11)", val(v, "pessoas_criancas")},
		{"Veículos", "Total de veículos", val(v, "veic_total")},
		{"Veículos", "Caminhões", val(v, "veic_caminhoes")},
		{"Veículos", "Automóveis, caminhonetes e utilitários", val(v, "veic_automoveis_utilitarios")},
		{"Veículos", "Motocicletas e ciclomotores", val(v, "veic_motos_ciclomotores")},
		{"Rebanho", "Bovinos (total)", val(v, "bov_total")},
		{"Rebanho", "Bovinos adultos", val(v, "bov_adultos")},
		{"Rebanho", "Bezerros e bezerras", val(v, "bov_bezerros")},
		{"Rebanho", "Equinos (total)", val(v, "equ_total")},
		{"Rebanho", "Equinos adultos", val(v, "equ_adultos")},
		{"Rebanho", "Potros", val(v, "equ_potros")},
		{"Bens e criações", "Aves de criação", val(v, "aves_total")},
		{"Bens e criações", "Eletrodomésticos", val(v, "eletro_total")},
		{"Bens e criações", "Móveis", val(v, "moveis_total")},
		{"Área", "Área total informada (ha)", Ha(t.AreaHa)},
		{"Área", "Média por família com área", media},
		{"Área", "Famílias com área informada", Int(int64(t.ComArea))},
		{"Área", "Famílias sem área informada", Int(int64(t.SemArea))},
	}}
	doc.Secoes = append(doc.Secoes, Secao{Titulo: "1. Panorama geral", Blocos: []Bloco{
		Paragrafo{Texto: "Totais das famílias que atendem aos filtros aplicados. Consideram só os cadastros vigentes, sem os excluídos e sem versões anteriores. Área convertida para hectares (1 alqueire paulista = 2,42 ha); propriedades sem área informada não valem zero.", Fraco: true},
		geral,
	}})

	doc.Secoes = append(doc.Secoes, Secao{Titulo: "2. Por equipe e por área identificada", Blocos: []Bloco{
		tabelaGrupos("Por equipe", "Equipe", r.PorEquipe, t),
		tabelaGrupos("Por área identificada", "Área identificada", r.PorArea, t),
	}})

	itens := Tabela{Titulo: "Itens por tipo", Colunas: []Coluna{{"Categoria", 1.3}, {"Tipo", 1.8}, {"Origem", 1.3}, {"Quantidade", 1}, {"Famílias que possuem", 1.2}}, Vazio: "Nenhum item registrado."}
	cats := map[string]string{"ave": "Aves", "eletrodomestico": "Eletrodomésticos", "movel": "Móveis"}
	lista := append([]estatisticas.TipoItem{}, r.Itens...)
	sort.SliceStable(lista, func(i, j int) bool {
		if lista[i].Campo.Categoria != lista[j].Campo.Categoria {
			return cats[lista[i].Campo.Categoria] < cats[lista[j].Campo.Categoria]
		}
		return lista[i].Total > lista[j].Total
	})
	for _, it := range lista {
		if it.Total == 0 {
			continue
		}
		origem := "Catálogo"
		if it.Campo.Outros {
			origem = "Digitado em “Outros”"
		}
		itens.Linhas = append(itens.Linhas, []string{cats[it.Campo.Categoria], it.Campo.Rotulo, origem, Int(it.Total), Int(int64(it.Familias))})
	}
	faixas := Tabela{Titulo: "Tamanho das propriedades", Colunas: []Coluna{{"Tamanho", 2}, {"Famílias", 1}, {"Área somada (ha)", 1.2}}}
	for _, f := range r.FaixasArea {
		a := Ha(f.AreaHa)
		if f.Chave == "sem_area" {
			a = "—"
		}
		faixas.Linhas = append(faixas.Linhas, []string{f.Rotulo, Int(int64(f.Familias)), a})
	}
	faixas.Total = []string{"Total", Int(int64(t.Familias)), Ha(t.AreaHa)}
	doc.Secoes = append(doc.Secoes, Secao{Titulo: "3. Bens, criações e tamanho das propriedades", Blocos: []Bloco{
		Paragrafo{Texto: "Itens digitados em “Outros” são agrupados pelo nome, sem diferenciar acentos e maiúsculas; o texto original não é alterado.", Fraco: true},
		itens, faixas,
	}})

	fam := Tabela{Titulo: "", Colunas: []Coluna{{"Família / propriedade", 2.6}, {"Equipe", 1.1}, {"Pessoas", 1}, {"Veículos", 1}, {"Bovinos", 1}, {"Equinos", 1}, {"Aves", 0.85}, {"Área (ha)", 1.15}, {"Localização", 1.45}},
		Vazio: "Nenhuma família com os filtros aplicados."}
	for _, p := range r.Familias {
		nome := p.Nome
		if p.Responsavel != nil && strings.TrimSpace(*p.Responsavel) != "" {
			nome += "\n" + *p.Responsavel
		}
		loc := "Não"
		if p.ComLocalizacao {
			loc = "Sim"
		}
		fam.Linhas = append(fam.Linhas, []string{nome, p.Equipe, val(p.Valores, "pessoas_total"), val(p.Valores, "veic_total"),
			val(p.Valores, "bov_total"), val(p.Valores, "equ_total"), val(p.Valores, "aves_total"), Ha(deref(p.AreaHa, "—")), loc})
	}
	if len(r.Familias) > 0 {
		fam.Total = []string{"Total geral", "", val(v, "pessoas_total"), val(v, "veic_total"), val(v, "bov_total"), val(v, "equ_total"), val(v, "aves_total"), Ha(t.AreaHa), Int(int64(t.ComLocalizacao)) + " com"}
	}
	doc.Secoes = append(doc.Secoes, Secao{Titulo: "4. Por família", NovaPagina: true, Blocos: []Bloco{
		Paragrafo{Texto: "Cada propriedade corresponde a uma família. Ordem alfabética.", Fraco: true}, fam,
	}})

	if d.Fichas {
		rotulos := rotulosItens(r.Campos)
		for i, p := range r.Familias {
			s := FichaSecao(p, rotulos)
			s.Titulo = fmt.Sprintf("5.%d %s", i+1, p.Nome)
			if i == 0 {
				s.Subtitulo = "Fichas das famílias. " + s.Subtitulo
			}
			doc.Secoes = append(doc.Secoes, s)
		}
	}
	return doc
}

type rotuloItem struct{ nome, categoria string }

func rotulosItens(campos []estatisticas.Campo) map[string]rotuloItem {
	m := map[string]rotuloItem{}
	for _, c := range campos {
		if c.Categoria != "" {
			m[c.Chave] = rotuloItem{c.Rotulo, c.Categoria}
		}
	}
	return m
}

// MontarFicha produz o documento de uma única propriedade.
func MontarFicha(p estatisticas.Familia, campos []estatisticas.Campo, operacao string, geradoEm time.Time) Documento {
	s := FichaSecao(p, rotulosItens(campos))
	s.NovaPagina = false
	return Documento{
		Titulo: "Ficha da propriedade", GeradoEm: geradoEm,
		Meta:   [][2]string{{"Operação", deref(&operacao, "—")}, {"Gerado em", geradoEm.Local().Format("02/01/2006 15:04")}},
		Secoes: []Secao{s},
	}
}

// FichaSecao é a ficha de uma família: dados, pessoas, veículos, rebanho e bens.
func FichaSecao(p estatisticas.Familia, rotulos map[string]rotuloItem) Secao {
	v := p.Valores
	area := "Não informada"
	if p.AreaHa != nil {
		area = Ha(*p.AreaHa) + " ha"
		if p.AreaUnidade != nil && *p.AreaUnidade == "alq_paulista" {
			area += " (informada em alqueire paulista)"
		}
	}
	loc := "Não informada"
	if p.Lat != nil && p.Lon != nil {
		loc = fmt.Sprintf("%.6f, %.6f", *p.Lat, *p.Lon)
		switch {
		case p.GpsOrigem != nil && *p.GpsOrigem == "manual":
			loc += " (digitada no celular)"
		case p.GpsPrecisaoM != nil:
			loc += fmt.Sprintf(" (GPS, precisão de %.0f m)", *p.GpsPrecisaoM)
		default:
			loc += " (GPS)"
		}
	}
	dados := Pares{Itens: [][2]string{
		{"Responsável", deref(p.Responsavel, "Não informado")},
		{"Equipe", p.Equipe},
		{"Área identificada", deref(p.AreaIdentificada, "—")},
		{"Área da propriedade", area},
		{"Localização", loc},
		{"Fotos", Int(v[estatisticas.Fotos])},
		{"Cadastrado no celular", Quando(p.CriadoEm)},
		{"Recebido pela Central", Quando(p.RecebidoEm)},
	}}
	qtd := Tabela{Titulo: "Pessoas, veículos e rebanho", Colunas: []Coluna{{"Assunto", 1.2}, {"Categoria", 2}, {"Quantidade", 1}}, Linhas: [][]string{
		{"Pessoas", "Adultos (18+)", val(v, "pessoas_adultos")},
		{"Pessoas", "Jovens (12–17)", val(v, "pessoas_jovens")},
		{"Pessoas", "Crianças (0–11)", val(v, "pessoas_criancas")},
		{"Pessoas", "Total", val(v, "pessoas_total")},
		{"Veículos", "Caminhões", val(v, "veic_caminhoes")},
		{"Veículos", "Automóveis, caminhonetes e utilitários", val(v, "veic_automoveis_utilitarios")},
		{"Veículos", "Motocicletas e ciclomotores", val(v, "veic_motos_ciclomotores")},
		{"Bovinos", "Adultos", val(v, "bov_adultos")},
		{"Bovinos", "Bezerros e bezerras", val(v, "bov_bezerros")},
		{"Equinos", "Adultos", val(v, "equ_adultos")},
		{"Equinos", "Potros", val(v, "equ_potros")},
	}}
	cats := map[string]string{"ave": "Aves", "eletrodomestico": "Eletrodomésticos", "movel": "Móveis"}
	bens := Tabela{Titulo: "Bens e criações", Colunas: []Coluna{{"Categoria", 1.2}, {"Item", 2}, {"Quantidade", 1}}, Vazio: "Nenhum item registrado."}
	type linhaItem struct {
		cat, nome string
		q         int64
	}
	var ls []linhaItem
	for k, q := range v {
		if r, ok := rotulos[k]; ok && q > 0 {
			ls = append(ls, linhaItem{cats[r.categoria], r.nome, q})
		}
	}
	sort.Slice(ls, func(i, j int) bool {
		if ls[i].cat != ls[j].cat {
			return ls[i].cat < ls[j].cat
		}
		return ls[i].nome < ls[j].nome
	})
	for _, l := range ls {
		bens.Linhas = append(bens.Linhas, []string{l.cat, l.nome, Int(l.q)})
	}
	return Secao{
		Titulo: p.Nome, NovaPagina: true,
		Subtitulo: fmt.Sprintf("%s · Equipe %s", deref(p.Responsavel, "Responsável não informado"), p.Equipe),
		Blocos:    []Bloco{dados, qtd, bens},
	}
}

// larguras distribui a largura útil pelas colunas: cada coluna recebe pelo menos a largura
// da sua palavra mais longa (cabeçalho e células), para nunca quebrar uma palavra ao meio;
// o restante segue os pesos. medir devolve a largura de um texto (cabecalho = negrito).
func larguras(t Tabela, util, folga float64, medir func(s string, cabecalho bool) float64) []float64 {
	n := len(t.Colunas)
	min := make([]float64, n)
	palavras := func(i int, s string, cab bool) {
		for _, w := range strings.Fields(s) {
			if x := medir(w, cab) + folga; x > min[i] {
				min[i] = x
			}
		}
	}
	for i, c := range t.Colunas {
		palavras(i, c.Rotulo, true)
	}
	for _, l := range t.Linhas {
		for i := 0; i < n && i < len(l); i++ {
			palavras(i, l[i], false)
		}
	}
	for i := 0; i < n && i < len(t.Total); i++ {
		palavras(i, t.Total[i], true) // a linha de total é em negrito
	}
	soma := 0.0
	for _, c := range t.Colunas {
		soma += c.Peso
	}
	ws := make([]float64, n)
	fixo := make([]bool, n)
	// Iterativo: colunas cujo quinhão ficaria abaixo do mínimo ficam no mínimo; as demais dividem o resto.
	for {
		resto, pesos := util, 0.0
		for i, c := range t.Colunas {
			if fixo[i] {
				resto -= min[i]
			} else {
				pesos += c.Peso
			}
		}
		mudou := false
		for i, c := range t.Colunas {
			if fixo[i] {
				ws[i] = min[i]
				continue
			}
			ws[i] = resto * c.Peso / pesos
			if ws[i] < min[i] {
				fixo[i], mudou = true, true
			}
		}
		if !mudou || pesos == 0 {
			break
		}
	}
	return ws
}

// NomeArquivo sugere um nome de arquivo seguro para o Windows.
func NomeArquivo(partes ...string) string {
	s := strings.Join(partes, " - ")
	troca := strings.NewReplacer(`\`, "-", "/", "-", ":", "-", "*", "", "?", "", `"`, "", "<", "", ">", "", "|", "-")
	s = strings.TrimSpace(troca.Replace(s))
	if r := []rune(s); len(r) > 120 {
		s = string(r[:120])
	}
	return s
}
