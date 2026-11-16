package relatorios

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"sitposse7/desktop/internal/estatisticas"
)

func ptr[T any](v T) *T { return &v }

// base com n famílias, acentos, campos vazios e um item de "Outros".
func dados(t *testing.T, n int, fichas bool) Dados {
	t.Helper()
	cat := map[string]string{"item:eletro.geladeira": "eletrodomestico", "item:ave.outros:codornas": "ave"}
	b := &estatisticas.Base{Itens: []estatisticas.Campo{
		{Chave: "item:eletro.geladeira", Rotulo: "Geladeira", Grupo: "Eletrodomésticos", Categoria: "eletrodomestico"},
		{Chave: "item:ave.outros:codornas", Rotulo: "Codornas", Grupo: "Aves", Categoria: "ave", Outros: true},
	}}
	for i := 0; i < n; i++ {
		f := estatisticas.Familia{
			UUID: fmt.Sprintf("u%03d", i), Nome: fmt.Sprintf("Sítio São João Nº %d — Linha C-25, Ramal do Igarapé Água Branca", i),
			EquipeUUID: fmt.Sprint("e", i%3), Equipe: []string{"Alfa", "Bravo", "Charlie"}[i%3], EstudoUUID: "o", Estudo: "Levantamento Linha C-25",
			JornadaUUID: fmt.Sprint("j", i%2), JornadaNumero: i%2 + 1, JornadaData: "2026-10-1" + fmt.Sprint(i%2), Responsavel: "Responsável " + fmt.Sprint(i),
			Valores:  map[string]int64{"pessoas_adultos": 2, "pessoas_jovens": int64(i % 3), "bov_adultos": int64(i * 3), "item:eletro.geladeira": 1, estatisticas.Fotos: int64(i % 2)},
			CriadoEm: 1_763_100_000_000, RecebidoEm: 1_763_200_000_000,
		}
		if i%4 != 0 {
			f.Responsavel = "José Antônio da Conceição"
			f.AreaDm2, f.AreaHa, f.AreaUnidade = ptr(int64(24_200_000*(i+1))), ptr(fmt.Sprintf("%d,20", 24*(i+1))), ptr("alq_paulista")
			f.Lat, f.Lon, f.GpsOrigem, f.GpsPrecisaoM, f.ComLocalizacao = ptr(-10.43), ptr(-62.47), ptr("gnss"), ptr(6.0), true
			f.Valores["item:ave.outros:codornas"] = 12
		}
		if i%3 == 1 {
			f.Observacoes = ptr("Família pede prazo para retirar o gado.\nHá um idoso acamado.")
		}
		f.Jornada = estatisticas.RotuloJornada(f.JornadaNumero, f.JornadaData)
		b.Familias = append(b.Familias, estatisticas.NovaFamilia(f, cat))
	}
	r, err := estatisticas.Calcular(b, estatisticas.Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	return Dados{Estudo: "Levantamento Linha C-25 (SITPOSSE7-A82F-91C6)", GeradoEm: time.Date(2026, 11, 14, 22, 0, 0, 0, time.Local),
		Criterios: []string{"Equipe: Alfa", "Bovinos (total): mais de 5"}, R: r, Fichas: fichas}
}

func TestFormatacao(t *testing.T) {
	for in, want := range map[int64]string{0: "0", 999: "999", 1000: "1.000", 1234567: "1.234.567", -1500: "-1.500"} {
		if got := Int(in); got != want {
			t.Errorf("Int(%d) = %q, esperado %q", in, got, want)
		}
	}
	if got := Ha("1234,50"); got != "1.234,50" {
		t.Errorf("Ha = %q", got)
	}
	if got := NomeArquivo("SITPOSSE-7", `Ficha: Sítio "A/B"?`); got != "SITPOSSE-7 - Ficha- Sítio A-B" {
		t.Errorf("NomeArquivo = %q", got)
	}
}

func TestPDF(t *testing.T) {
	d := dados(t, 60, true)
	b, err := PDF(Montar(d))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatal("não começa com %PDF-")
	}
	// O brasão é PNG com fundo transparente: o PDF precisa levar a máscara de transparência.
	if !bytes.Contains(b, []byte("/SMask")) {
		t.Error("brasão sem máscara de transparência no PDF")
	}
	m := regexp.MustCompile(`/Type /Pages[^>]*?/Count (\d+)`).FindSubmatch(b)
	if m == nil {
		m = regexp.MustCompile(`/Count (\d+)`).FindSubmatch(b)
	}
	if m == nil {
		t.Fatal("contagem de páginas não encontrada")
	}
	n, _ := strconv.Atoi(string(m[1]))
	// panorama + tabela por família + 60 fichas (uma por página)
	if n < 62 {
		t.Errorf("%d páginas; esperado ao menos 62 (60 fichas)", n)
	}
	if !bytes.Contains(b, []byte("/FontFile2")) {
		t.Error("fonte não embutida")
	}
	t.Logf("PDF: %d páginas, %d KB", n, len(b)/1024)

	ficha, err := PDF(MontarFicha(d.R.Familias[1], d.R.Campos, d.Estudo, d.GeradoEm))
	if err != nil || !bytes.HasPrefix(ficha, []byte("%PDF-")) {
		t.Fatalf("ficha: %v", err)
	}
}

func xmlValido(t *testing.T, nome string, r io.Reader) {
	t.Helper()
	dec := xml.NewDecoder(r)
	for {
		if _, err := dec.Token(); err == io.EOF {
			return
		} else if err != nil {
			t.Fatalf("%s: XML inválido: %v", nome, err)
		}
	}
}

func TestDOCX(t *testing.T) {
	d := dados(t, 25, true)
	b, err := DOCX(Montar(d))
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	partes := map[string]string{}
	for _, f := range z.File {
		rc, _ := f.Open()
		conteudo, _ := io.ReadAll(rc)
		rc.Close()
		partes[f.Name] = string(conteudo)
		if strings.HasSuffix(f.Name, ".xml") || strings.HasSuffix(f.Name, ".rels") {
			xmlValido(t, f.Name, bytes.NewReader(conteudo))
		}
	}
	for _, p := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml", "word/styles.xml", "word/header1.xml", "word/footer1.xml", "word/_rels/header1.xml.rels", "word/media/logo-cabecalho-docs.png"} {
		if _, ok := partes[p]; !ok {
			t.Errorf("parte %s ausente", p)
		}
	}
	cab := partes["word/header1.xml"]
	if !strings.Contains(cab, Instituicao) || !strings.Contains(cab, Unidade) || !strings.Contains(cab, `r:embed="rIdLogo"`) {
		t.Error("cabeçalho sem brasão ou sem os textos institucionais")
	}
	// O texto institucional vem abaixo do brasão.
	if strings.Index(cab, "rIdLogo") > strings.Index(cab, Instituicao) {
		t.Error("textos institucionais antes do brasão")
	}
	doc := partes["word/document.xml"]
	if strings.Count(doc, "<w:pageBreakBefore/>") < 25 {
		t.Error("esperada uma ficha por página")
	}
	if !strings.Contains(doc, "<w:tblHeader/>") || !strings.Contains(doc, `w:insideV w:val="single"`) {
		t.Error("tabela sem cabeçalho repetido ou sem divisória entre colunas")
	}
	if !strings.Contains(doc, esc("Sítio São João Nº 3 — Linha C-25, Ramal do Igarapé Água Branca")) || !strings.Contains(doc, "Equipe: Alfa") {
		t.Error("conteúdo esperado ausente")
	}
	if !bytes.Equal([]byte(partes["word/media/logo-cabecalho-docs.png"]), Logo) {
		t.Error("logo diferente do embutido")
	}
}

func TestXLSX(t *testing.T) {
	d := dados(t, 30, false)
	b, err := XLSX(d)
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	want := []string{"Panorama", "Por jornada", "Por equipe", "Por família", "Itens por tipo", "Tamanho", "Sobre"}
	if got := f.GetSheetList(); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("abas %v, esperado %v", got, want)
	}
	for _, aba := range want {
		if v, _ := f.GetCellValue(aba, "A1"); v != Instituicao {
			t.Errorf("%s!A1 = %q", aba, v)
		}
		if v, _ := f.GetCellValue(aba, "A2"); v != Unidade {
			t.Errorf("%s!A2 = %q", aba, v)
		}
	}
	// Por família: cabeçalho na linha 6, 30 linhas, total com SUBTOTAL.
	if v, _ := f.GetCellValue("Por família", "A6"); v != "Família / propriedade" {
		t.Errorf("cabeçalho = %q", v)
	}
	if v, _ := f.GetCellValue("Por família", "A36"); !strings.HasPrefix(v, "Sítio São João") {
		t.Errorf("última família = %q", v)
	}
	fm, _ := f.GetCellFormula("Por família", "E37")
	if !strings.HasPrefix(fm, "SUBTOTAL(109,E7:E36)") {
		t.Errorf("total = %q", fm)
	}
	tabs, _ := f.GetTables("Por família")
	if len(tabs) != 1 || tabs[0].Name != "PorFamilia" {
		t.Errorf("tabelas = %+v", tabs)
	}
	// Valor numérico (não texto) para poder somar e filtrar no Excel.
	tipo, _ := f.GetCellType("Por família", "E7")
	if tipo == excelize.CellTypeSharedString || tipo == excelize.CellTypeInlineString {
		t.Error("quantidade gravada como texto")
	}
	t.Logf("XLSX: %d KB", len(b)/1024)
}

// Amostras para revisão visual: SITPOSSE7_AMOSTRAS=<pasta> go test ./internal/relatorios -run Amostras
func TestAmostras(t *testing.T) {
	dir := os.Getenv("SITPOSSE7_AMOSTRAS")
	if dir == "" {
		t.Skip("defina SITPOSSE7_AMOSTRAS para gravar amostras")
	}
	d := dados(t, 14, true)
	doc := Montar(d)
	p, _ := PDF(doc)
	w, _ := DOCX(doc)
	x, _ := XLSX(d)
	f, _ := PDF(MontarFicha(d.R.Familias[1], d.R.Campos, d.Estudo, d.GeradoEm))
	for nome, b := range map[string][]byte{"relatorio.pdf": p, "relatorio.docx": w, "relatorio.xlsx": x, "ficha.pdf": f} {
		if err := os.WriteFile(filepath.Join(dir, nome), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
