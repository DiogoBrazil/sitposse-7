package relatorios

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"sitposse7/desktop/internal/estatisticas"
)

// Planilha (A26): uma aba por visão, cada uma com o cabeçalho institucional em texto,
// tabela do Excel (filtro, ordenação e faixas), linha de total com SUBTOTAL (acompanha os
// filtros aplicados no próprio Excel), painéis congelados e gráficos nas abas de resumo.

const linhaCab = 6 // linha do cabeçalho das tabelas (1–4: cabeçalho institucional)

type planilha struct {
	f      *excelize.File
	d      Dados
	st     map[string]int
	tabSeq int
}

func (p *planilha) estilo(nome string, s *excelize.Style) {
	id, err := p.f.NewStyle(s)
	if err == nil {
		p.st[nome] = id
	}
}

func borda(cor string) []excelize.Border {
	return []excelize.Border{{Type: "left", Color: cor, Style: 1}, {Type: "right", Color: cor, Style: 1}, {Type: "top", Color: cor, Style: 1}, {Type: "bottom", Color: cor, Style: 1}}
}

func (p *planilha) estilos() {
	centro := &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}
	inteiro := "#,##0"
	area := "#,##0.00"
	data := "dd/mm/yyyy hh:mm"
	p.estilo("inst", &excelize.Style{Font: &excelize.Font{Bold: true, Size: 15, Color: "1F5F3B", Family: "Calibri"}})
	p.estilo("unidade", &excelize.Style{Font: &excelize.Font{Bold: true, Size: 11, Color: "1B2620", Family: "Calibri"}})
	p.estilo("titulo", &excelize.Style{Font: &excelize.Font{Bold: true, Size: 13, Color: "1B2620", Family: "Calibri"}})
	p.estilo("meta", &excelize.Style{Font: &excelize.Font{Italic: true, Size: 9, Color: "4A584F", Family: "Calibri"}})
	p.estilo("cab", &excelize.Style{Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 10, Family: "Calibri"}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1F5F3B"}},
		Alignment: centro, Border: borda("FFFFFF")})
	p.estilo("texto", &excelize.Style{Alignment: centro, Border: borda("C4CCC0"), Font: &excelize.Font{Size: 10, Family: "Calibri"}})
	p.estilo("int", &excelize.Style{Alignment: centro, Border: borda("C4CCC0"), CustomNumFmt: &inteiro, Font: &excelize.Font{Size: 10, Family: "Calibri"}})
	p.estilo("area", &excelize.Style{Alignment: centro, Border: borda("C4CCC0"), CustomNumFmt: &area, Font: &excelize.Font{Size: 10, Family: "Calibri"}})
	p.estilo("data", &excelize.Style{Alignment: centro, Border: borda("C4CCC0"), CustomNumFmt: &data, Font: &excelize.Font{Size: 10, Family: "Calibri"}})
	p.estilo("coord", &excelize.Style{Alignment: centro, Border: borda("C4CCC0"), CustomNumFmt: strPtr("0.000000"), Font: &excelize.Font{Size: 10, Family: "Calibri"}})
	tot := func(fmt *string) *excelize.Style {
		return &excelize.Style{Font: &excelize.Font{Bold: true, Size: 10, Family: "Calibri"}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E3EEE5"}},
			Alignment: centro, CustomNumFmt: fmt, Border: append(borda("C4CCC0"), excelize.Border{Type: "top", Color: "1F5F3B", Style: 2})}
	}
	p.estilo("tot", tot(nil))
	p.estilo("tot_int", tot(&inteiro))
	p.estilo("tot_area", tot(&area))
	p.estilo("link", &excelize.Style{Font: &excelize.Font{Color: "2B6FC0", Underline: "single", Size: 11, Family: "Calibri"}})
	p.estilo("nota", &excelize.Style{Font: &excelize.Font{Size: 10, Color: "4A584F", Family: "Calibri"}, Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}})
}

func strPtr(s string) *string { return &s }

// cabecalho: textos institucionais (sem logo), título da aba e operação/filtros/data.
func (p *planilha) cabecalho(aba, titulo string) {
	f := p.f
	_ = f.SetCellValue(aba, "A1", Instituicao)
	_ = f.SetCellValue(aba, "A2", Unidade)
	_ = f.SetCellValue(aba, "A3", titulo)
	filtros := "nenhum (todas as famílias recebidas)"
	if len(p.d.Criterios) > 0 {
		filtros = strings.Join(p.d.Criterios, "; ")
	}
	_ = f.SetCellValue(aba, "A4", fmt.Sprintf("Operação: %s   ·   Filtros: %s   ·   Gerado em %s pelo %s Central", deref(&p.d.Operacao, "—"), filtros, p.d.GeradoEm.Local().Format("02/01/2006 15:04"), Sistema))
	for c, s := range map[string]string{"A1": "inst", "A2": "unidade", "A3": "titulo", "A4": "meta"} {
		_ = f.SetCellStyle(aba, c, c, p.st[s])
	}
	_ = f.SetRowHeight(aba, 1, 22)
	_ = f.SetRowHeight(aba, 3, 19)
	if aba != "Panorama" {
		_ = f.SetCellHyperLink(aba, "A5", "'Panorama'!A1", "Location")
		_ = f.SetCellValue(aba, "A5", "← Voltar ao Panorama")
		_ = f.SetCellStyle(aba, "A5", "A5", p.st["link"])
	}
	horiz := "landscape"
	um, zero := 1, 0
	_ = f.SetPageLayout(aba, &excelize.PageLayoutOptions{Orientation: &horiz, FitToWidth: &um, FitToHeight: &zero})
	sim := true
	_ = f.SetSheetProps(aba, &excelize.SheetPropsOptions{FitToPage: &sim, TabColorRGB: strPtr("1F5F3B")})
	// Na impressão, só o rodapé: os textos institucionais já estão nas linhas 1 e 2.
	_ = f.SetHeaderFooter(aba, &excelize.HeaderFooterOptions{OddFooter: "&L&8" + Sistema + " Central&R&8Página &P de &N"})
}

// col descreve uma coluna da tabela: rótulo, tipo de célula, largura e total.
type col struct {
	rotulo string
	tipo   string // texto | int | area | data | coord
	larg   float64
	total  bool
}

// tabela escreve cabeçalho + linhas a partir de linhaCab, cria a Tabela do Excel e a linha de total.
// Devolve a última linha de dados.
func (p *planilha) tabela(aba, nome string, cols []col, linhas [][]any, rotuloTotal string) int {
	f := p.f
	for i, c := range cols {
		cel, _ := excelize.CoordinatesToCellName(i+1, linhaCab)
		_ = f.SetCellValue(aba, cel, c.rotulo)
		nomeCol, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(aba, nomeCol, nomeCol, c.larg)
	}
	ini, _ := excelize.CoordinatesToCellName(1, linhaCab)
	fimCab, _ := excelize.CoordinatesToCellName(len(cols), linhaCab)
	_ = f.SetCellStyle(aba, ini, fimCab, p.st["cab"])
	_ = f.SetRowHeight(aba, linhaCab, 32)
	if len(linhas) == 0 {
		vazio := make([]any, len(cols))
		vazio[0] = "Nenhum registro com os filtros aplicados."
		linhas = [][]any{vazio}
		rotuloTotal = ""
	}
	for r, l := range linhas {
		row := linhaCab + 1 + r
		cel, _ := excelize.CoordinatesToCellName(1, row)
		_ = f.SetSheetRow(aba, cel, &l)
		for i, c := range cols {
			x, _ := excelize.CoordinatesToCellName(i+1, row)
			_ = f.SetCellStyle(aba, x, x, p.st[c.tipo])
		}
	}
	ultima := linhaCab + len(linhas)
	fim, _ := excelize.CoordinatesToCellName(len(cols), ultima)
	listras := true
	p.tabSeq++
	_ = f.AddTable(aba, &excelize.Table{Range: ini + ":" + fim, Name: nome, StyleName: "TableStyleLight15", ShowRowStripes: &listras})
	_ = f.SetPanes(aba, &excelize.Panes{Freeze: true, XSplit: 1, YSplit: linhaCab, TopLeftCell: "B" + fmt.Sprint(linhaCab+1), ActivePane: "bottomRight"})
	if rotuloTotal != "" {
		rt := ultima + 1
		for i, c := range cols {
			x, _ := excelize.CoordinatesToCellName(i+1, rt)
			switch {
			case i == 0:
				_ = f.SetCellValue(aba, x, rotuloTotal)
				_ = f.SetCellStyle(aba, x, x, p.st["tot"])
			case c.total:
				nc, _ := excelize.ColumnNumberToName(i + 1)
				// SUBTOTAL(109, …) soma só as linhas visíveis: acompanha os filtros do Excel.
				_ = f.SetCellFormula(aba, x, fmt.Sprintf("SUBTOTAL(109,%s%d:%s%d)", nc, linhaCab+1, nc, ultima))
				_ = f.SetCellStyle(aba, x, x, p.st["tot_"+c.tipo])
			default:
				_ = f.SetCellStyle(aba, x, x, p.st["tot"])
			}
		}
	}
	return ultima
}

func haNum(dm2 int64) float64 { return float64((dm2+5000)/10000) / 100 } // HALF_UP como dominio.FormatarHa

func grafico(f *excelize.File, aba, cel, titulo, cor, cat, val string, n int) {
	if n == 0 {
		return
	}
	altura := uint(120 + 26*n)
	if altura > 520 {
		altura = 520
	}
	_ = f.AddChart(aba, cel, &excelize.Chart{
		Type:      excelize.Bar,
		Series:    []excelize.ChartSeries{{Name: titulo, Categories: cat, Values: val, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{cor}}}},
		Title:     excelize.ChartTitle{Paragraph: []excelize.RichTextRun{{Text: titulo, Font: &excelize.Font{Bold: true, Size: 11, Color: "1B2620"}}}},
		Legend:    excelize.ChartLegend{Position: "none"},
		PlotArea:  excelize.ChartPlotArea{ShowVal: true},
		XAxis:     excelize.ChartAxis{ReverseOrder: true},
		YAxis:     excelize.ChartAxis{MajorGridLines: true},
		Dimension: excelize.ChartDimension{Width: 520, Height: altura},
	})
}

var colsGrupo = []struct {
	chave, rotulo string
}{
	{"pessoas_total", "Pessoas"}, {"pessoas_adultos", "Adultos (18+)"}, {"pessoas_jovens", "Jovens (12–17)"}, {"pessoas_criancas", "Crianças (0–11)"},
	{"veic_total", "Veículos"}, {"veic_caminhoes", "Caminhões"}, {"veic_automoveis_utilitarios", "Automóveis e utilitários"}, {"veic_motos_ciclomotores", "Motos e ciclomotores"},
	{"bov_total", "Bovinos"}, {"bov_adultos", "Bovinos adultos"}, {"bov_bezerros", "Bezerros"},
	{"equ_total", "Equinos"}, {"equ_adultos", "Equinos adultos"}, {"equ_potros", "Potros"},
	{"aves_total", "Aves"}, {"eletro_total", "Eletrodomésticos"}, {"moveis_total", "Móveis"}, {"fotos", "Fotos"},
}

func (p *planilha) abaGrupos(aba, nome, rotulo string, gs []estatisticas.Grupo) {
	_, _ = p.f.NewSheet(aba)
	p.cabecalho(aba, rotulo+": totais")
	cols := []col{{rotulo, "texto", 26, false}, {"Famílias", "int", 10, true}}
	for _, c := range colsGrupo {
		cols = append(cols, col{c.rotulo, "int", 12, true})
	}
	cols = append(cols, col{"Área (ha)", "area", 13, true}, col{"Sem área informada", "int", 12, true}, col{"Com localização", "int", 12, true})
	var linhas [][]any
	for _, g := range gs {
		l := []any{g.Nome, g.Totais.Familias}
		for _, c := range colsGrupo {
			l = append(l, g.Totais.Valores[c.chave])
		}
		l = append(l, haNum(g.Totais.AreaDm2), g.Totais.SemArea, g.Totais.ComLocalizacao)
		linhas = append(linhas, l)
	}
	ultima := p.tabela(aba, nome, cols, linhas, "Total (linhas visíveis)")
	n := len(gs)
	grafico(p.f, aba, fmt.Sprintf("A%d", ultima+4), "Famílias por "+strings.ToLower(rotulo), "1F5F3B",
		fmt.Sprintf("'%s'!$A$%d:$A$%d", aba, linhaCab+1, ultima), fmt.Sprintf("'%s'!$B$%d:$B$%d", aba, linhaCab+1, ultima), n)
	grafico(p.f, aba, fmt.Sprintf("H%d", ultima+4), "Pessoas por "+strings.ToLower(rotulo), "2B6FC0",
		fmt.Sprintf("'%s'!$A$%d:$A$%d", aba, linhaCab+1, ultima), fmt.Sprintf("'%s'!$C$%d:$C$%d", aba, linhaCab+1, ultima), n)
}

// XLSX gera a planilha do relatório geral.
func XLSX(d Dados) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	p := &planilha{f: f, d: d, st: map[string]int{}}
	p.estilos()
	r := d.R
	t := r.Totais
	v := t.Valores
	_ = f.SetDocProps(&excelize.DocProperties{Title: "Levantamento de propriedades rurais — panorama e famílias", Creator: Instituicao + " — " + Unidade, Language: "pt-BR", Created: d.GeradoEm.UTC().Format(time.RFC3339)})

	// ---------- Panorama ----------
	_ = f.SetSheetName("Sheet1", "Panorama")
	p.cabecalho("Panorama", "Levantamento de propriedades rurais: panorama geral")
	totais := [][]any{
		{"Famílias", "Propriedades (famílias) no relatório", t.Familias},
		{"Famílias", "Famílias recebidas (sem filtros)", r.FamiliasBase},
		{"Famílias", "Com localização (GPS ou digitada)", t.ComLocalizacao},
		{"Famílias", "Sem localização", t.SemLocalizacao},
		{"Famílias", "Com pelo menos uma foto", t.ComFoto},
		{"Pessoas", "Adultos (18+)", v["pessoas_adultos"]},
		{"Pessoas", "Jovens (12–17)", v["pessoas_jovens"]},
		{"Pessoas", "Crianças (0–11)", v["pessoas_criancas"]},
		{"Pessoas", "Total de pessoas", v["pessoas_total"]},
		{"Veículos", "Caminhões", v["veic_caminhoes"]},
		{"Veículos", "Automóveis, caminhonetes e utilitários", v["veic_automoveis_utilitarios"]},
		{"Veículos", "Motocicletas e ciclomotores", v["veic_motos_ciclomotores"]},
		{"Veículos", "Total de veículos", v["veic_total"]},
		{"Rebanho", "Bovinos adultos", v["bov_adultos"]},
		{"Rebanho", "Bezerros e bezerras", v["bov_bezerros"]},
		{"Rebanho", "Bovinos (total)", v["bov_total"]},
		{"Rebanho", "Equinos adultos", v["equ_adultos"]},
		{"Rebanho", "Potros", v["equ_potros"]},
		{"Rebanho", "Equinos (total)", v["equ_total"]},
		{"Bens e criações", "Aves de criação", v["aves_total"]},
		{"Bens e criações", "Eletrodomésticos", v["eletro_total"]},
		{"Bens e criações", "Móveis", v["moveis_total"]},
		{"Área", "Famílias com área informada", t.ComArea},
		{"Área", "Famílias sem área informada", t.SemArea},
	}
	ultima := p.tabela("Panorama", "Totais", []col{{"Assunto", "texto", 18, false}, {"Indicador", "texto", 40, false}, {"Quantidade", "int", 14, false}}, totais, "")
	// Área total e média (hectares, 2 casas).
	for i, l := range [][]any{{"Área", "Área total informada (ha)", haNum(t.AreaDm2)}, {"Área", "Média por família com área (ha)", func() any {
		if t.ComArea == 0 {
			return "—"
		}
		return haNum(t.AreaDm2 / int64(t.ComArea))
	}()}} {
		row := ultima + 1 + i
		cel := fmt.Sprintf("A%d", row)
		_ = f.SetSheetRow("Panorama", cel, &l)
		_ = f.SetCellStyle("Panorama", fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), p.st["tot"])
		_ = f.SetCellStyle("Panorama", fmt.Sprintf("C%d", row), fmt.Sprintf("C%d", row), p.st["tot_area"])
	}
	// Índice com links para as abas.
	abas := []struct{ nome, descr string }{
		{"Por equipe", "Totais por equipe, com gráficos"},
		{"Por área", "Totais por área identificada"},
		{"Por família", "Uma linha por família, com todas as quantidades e cada tipo de item"},
		{"Itens por tipo", "Eletrodomésticos, móveis e aves por tipo (inclui “Outros”)"},
		{"Tamanho", "Famílias por faixa de tamanho da propriedade"},
		{"Sobre", "Filtros aplicados e regras de cálculo"},
	}
	_ = f.SetCellValue("Panorama", fmt.Sprintf("E%d", linhaCab), "Conteúdo da planilha")
	_ = f.SetCellStyle("Panorama", fmt.Sprintf("E%d", linhaCab), fmt.Sprintf("F%d", linhaCab), p.st["cab"])
	_ = f.MergeCell("Panorama", fmt.Sprintf("E%d", linhaCab), fmt.Sprintf("F%d", linhaCab))
	for i, a := range abas {
		row := linhaCab + 1 + i
		_ = f.SetCellValue("Panorama", fmt.Sprintf("E%d", row), a.nome)
		_ = f.SetCellHyperLink("Panorama", fmt.Sprintf("E%d", row), fmt.Sprintf("'%s'!A1", a.nome), "Location")
		_ = f.SetCellStyle("Panorama", fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), p.st["link"])
		_ = f.SetCellValue("Panorama", fmt.Sprintf("F%d", row), a.descr)
		_ = f.SetCellStyle("Panorama", fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), p.st["nota"])
	}
	_ = f.SetColWidth("Panorama", "D", "D", 3)
	_ = f.SetColWidth("Panorama", "E", "E", 18)
	_ = f.SetColWidth("Panorama", "F", "F", 58)
	_ = f.SetPanes("Panorama", &excelize.Panes{Freeze: true, YSplit: linhaCab, TopLeftCell: fmt.Sprintf("A%d", linhaCab+1), ActivePane: "bottomLeft"})
	grafico(f, "Panorama", fmt.Sprintf("E%d", linhaCab+9), "Pessoas por faixa etária", "2B6FC0",
		fmt.Sprintf("'Panorama'!$B$%d:$B$%d", linhaCab+6, linhaCab+8), fmt.Sprintf("'Panorama'!$C$%d:$C$%d", linhaCab+6, linhaCab+8), 3)
	grafico(f, "Panorama", fmt.Sprintf("E%d", linhaCab+24), "Veículos por tipo", "7A4FA8",
		fmt.Sprintf("'Panorama'!$B$%d:$B$%d", linhaCab+10, linhaCab+12), fmt.Sprintf("'Panorama'!$C$%d:$C$%d", linhaCab+10, linhaCab+12), 3)

	p.abaGrupos("Por equipe", "PorEquipe", "Equipe", r.PorEquipe)
	p.abaGrupos("Por área", "PorArea", "Área identificada", r.PorArea)

	// ---------- Por família ----------
	_, _ = f.NewSheet("Por família")
	p.cabecalho("Por família", "Famílias (cada propriedade corresponde a uma família)")
	cols := []col{{"Família / propriedade", "texto", 30, false}, {"Responsável", "texto", 24, false}, {"Equipe", "texto", 16, false}, {"Área identificada", "texto", 18, false}}
	for _, c := range colsGrupo {
		cols = append(cols, col{c.rotulo, "int", 12, true})
	}
	itens := []estatisticas.Campo{}
	for _, c := range r.Campos {
		if c.Categoria != "" {
			itens = append(itens, c)
		}
	}
	cats := map[string]string{"ave": "ave", "eletrodomestico": "eletrodoméstico", "movel": "móvel"}
	for _, c := range itens {
		rot := fmt.Sprintf("%s (%s)", c.Rotulo, cats[c.Categoria])
		if c.Outros {
			rot = fmt.Sprintf("%s (%s, “Outros”)", c.Rotulo, cats[c.Categoria])
		}
		cols = append(cols, col{rot, "int", 13, true})
	}
	cols = append(cols, col{"Área (ha)", "area", 12, true}, col{"Unidade informada", "texto", 14, false}, col{"Latitude", "coord", 13, false}, col{"Longitude", "coord", 13, false},
		col{"Origem da localização", "texto", 16, false}, col{"Cadastrado no celular", "data", 17, false}, col{"Recebido pela Central", "data", 17, false})
	var linhas [][]any
	for _, x := range r.Familias {
		l := []any{x.Nome, deref(x.Responsavel, ""), x.Equipe, deref(x.AreaIdentificada, "")}
		for _, c := range colsGrupo {
			l = append(l, x.Valores[c.chave])
		}
		for _, c := range itens {
			l = append(l, x.Valores[c.Chave])
		}
		var area, lat, lon any = "", "", ""
		unidade, origem := "", ""
		if x.AreaDm2 != nil {
			area = haNum(*x.AreaDm2)
			unidade = map[string]string{"ha": "hectare", "alq_paulista": "alqueire paulista"}[deref(x.AreaUnidade, "")]
		}
		if x.Lat != nil && x.Lon != nil {
			lat, lon = *x.Lat, *x.Lon
			origem = map[string]string{"gnss": "GPS do celular", "manual": "Digitada"}[deref(x.GpsOrigem, "")]
		}
		l = append(l, area, unidade, lat, lon, origem, time.UnixMilli(x.CriadoEm).Local(), time.UnixMilli(x.RecebidoEm).Local())
		linhas = append(linhas, l)
	}
	p.tabela("Por família", "PorFamilia", cols, linhas, "Total (linhas visíveis)")

	// ---------- Itens por tipo ----------
	_, _ = f.NewSheet("Itens por tipo")
	p.cabecalho("Itens por tipo", "Bens e criações por tipo")
	lista := append([]estatisticas.TipoItem{}, r.Itens...)
	nomesCat := map[string]string{"ave": "Aves", "eletrodomestico": "Eletrodomésticos", "movel": "Móveis"}
	sort.SliceStable(lista, func(i, j int) bool {
		if lista[i].Campo.Categoria != lista[j].Campo.Categoria {
			return nomesCat[lista[i].Campo.Categoria] < nomesCat[lista[j].Campo.Categoria]
		}
		return lista[i].Total > lista[j].Total
	})
	var li [][]any
	for _, it := range lista {
		if it.Total == 0 {
			continue
		}
		origem := "Catálogo"
		if it.Campo.Outros {
			origem = "Digitado em “Outros”"
		}
		li = append(li, []any{nomesCat[it.Campo.Categoria], it.Campo.Rotulo, origem, it.Total, it.Familias})
	}
	ult := p.tabela("Itens por tipo", "ItensPorTipo", []col{{"Categoria", "texto", 18, false}, {"Tipo", "texto", 26, false}, {"Origem", "texto", 20, false}, {"Quantidade", "int", 13, true}, {"Famílias que possuem", "int", 15, false}}, li, "Total (linhas visíveis)")
	grafico(f, "Itens por tipo", "G6", "Quantidade por tipo", "2F7D4A", fmt.Sprintf("'Itens por tipo'!$B$%d:$B$%d", linhaCab+1, ult), fmt.Sprintf("'Itens por tipo'!$D$%d:$D$%d", linhaCab+1, ult), len(li))

	// ---------- Tamanho ----------
	_, _ = f.NewSheet("Tamanho")
	p.cabecalho("Tamanho", "Famílias por tamanho da propriedade")
	var lf [][]any
	for _, x := range r.FaixasArea {
		var a any = haNum(x.AreaDm2)
		if x.Chave == "sem_area" {
			a = "—"
		}
		lf = append(lf, []any{x.Rotulo, x.Familias, a})
	}
	ult = p.tabela("Tamanho", "Tamanho", []col{{"Tamanho", "texto", 24, false}, {"Famílias", "int", 12, true}, {"Área somada (ha)", "area", 16, true}}, lf, "Total")
	grafico(f, "Tamanho", "E6", "Famílias por tamanho", "1F5F3B", fmt.Sprintf("'Tamanho'!$A$%d:$A$%d", linhaCab+1, ult), fmt.Sprintf("'Tamanho'!$B$%d:$B$%d", linhaCab+1, ult), len(lf))

	// ---------- Sobre ----------
	_, _ = f.NewSheet("Sobre")
	p.cabecalho("Sobre", "Filtros aplicados e regras de cálculo")
	notas := []string{"Filtros aplicados no SITPOSSE-7 Central ao gerar esta planilha:"}
	if len(d.Criterios) == 0 {
		notas = append(notas, "  • nenhum (todas as famílias recebidas)")
	}
	for _, c := range d.Criterios {
		notas = append(notas, "  • "+c)
	}
	notas = append(notas, "",
		"Regras de cálculo:",
		"  • Só cadastros vigentes, na última versão recebida; registros excluídos e versões anteriores não entram (RN-80).",
		"  • Cada propriedade corresponde a uma família.",
		"  • Área convertida para hectares (1 alqueire paulista = 2,42 ha). Propriedade sem área informada não vale zero: é contada à parte.",
		"  • Itens digitados em “Outros” são agrupados pelo nome, sem diferenciar acentos e maiúsculas; o texto original não é alterado.",
		"  • As linhas de total usam SUBTOTAL: ao filtrar uma tabela no Excel, o total passa a somar só as linhas visíveis.",
	)
	for i, n := range notas {
		cel := fmt.Sprintf("A%d", linhaCab+i)
		_ = f.SetCellValue("Sobre", cel, n)
		_ = f.SetCellStyle("Sobre", cel, cel, p.st["nota"])
	}
	_ = f.SetColWidth("Sobre", "A", "A", 120)

	f.SetActiveSheet(0)
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
