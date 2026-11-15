package relatorios

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

// DOCX gerado só com a biblioteca padrão (ADR-0021): um .docx é um pacote ZIP com
// partes XML (ECMA-376, WordprocessingML). Página A4 retrato; o cabeçalho de todas as
// páginas traz o brasão e, abaixo dele, os textos institucionais.

const (
	twipsLargUtil = 9638 // A4 (11906) − margens de 2 cm (2 × 1134)
	emuPorCm      = 360000
)

func esc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// run com texto (quebras de linha viram <w:br/>).
func run(s string, props string) string {
	var b strings.Builder
	for i, parte := range strings.Split(s, "\n") {
		if i > 0 {
			b.WriteString(`<w:r>` + rpr(props) + `<w:br/></w:r>`)
		}
		b.WriteString(`<w:r>` + rpr(props) + `<w:t xml:space="preserve">` + esc(parte) + `</w:t></w:r>`)
	}
	return b.String()
}

func rpr(props string) string {
	if props == "" {
		return ""
	}
	return "<w:rPr>" + props + "</w:rPr>"
}

func par(conteudo, ppr string) string {
	return `<w:p>` + func() string {
		if ppr == "" {
			return ""
		}
		return "<w:pPr>" + ppr + "</w:pPr>"
	}() + conteudo + `</w:p>`
}

const (
	negrito   = `<w:b/>`
	corBranca = `<w:color w:val="FFFFFF"/>`
	corFraca  = `<w:color w:val="4A584F"/>`
)

func tam(meioPontos int) string {
	return fmt.Sprintf(`<w:sz w:val="%d"/><w:szCs w:val="%d"/>`, meioPontos, meioPontos)
}

type docx struct{ b strings.Builder }

func (d *docx) tabela(t Tabela) {
	if t.Titulo != "" {
		d.b.WriteString(par(run(t.Titulo, negrito+tam(21)), `<w:pStyle w:val="TituloTabela"/><w:keepNext/>`))
	}
	// Calibri 8,5 pt: largura média ≈ 0,52 em (negrito ≈ 0,56); 1 pt = 20 twips.
	fl := larguras(t, twipsLargUtil, 2*85+40, func(txt string, cab bool) float64 {
		k := 0.52
		if cab {
			k = 0.56
		}
		return float64(len([]rune(txt))) * 8.5 * k * 20
	})
	ws := make([]int, len(fl))
	for i, w := range fl {
		ws[i] = int(w)
	}
	d.b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="5000" w:type="pct"/><w:jc w:val="center"/>` +
		`<w:tblBorders><w:top w:val="single" w:sz="4" w:color="C4CCC0"/><w:left w:val="single" w:sz="4" w:color="C4CCC0"/>` +
		`<w:bottom w:val="single" w:sz="4" w:color="C4CCC0"/><w:right w:val="single" w:sz="4" w:color="C4CCC0"/>` +
		`<w:insideH w:val="single" w:sz="4" w:color="C4CCC0"/><w:insideV w:val="single" w:sz="4" w:color="C4CCC0"/></w:tblBorders>` +
		`<w:tblLayout w:type="fixed"/><w:tblCellMar><w:top w:w="57" w:type="dxa"/><w:left w:w="85" w:type="dxa"/><w:bottom w:w="57" w:type="dxa"/><w:right w:w="85" w:type="dxa"/></w:tblCellMar>` +
		`<w:tblLook w:val="0420" w:firstRow="1" w:lastRow="0" w:firstColumn="0" w:lastColumn="0" w:noHBand="0" w:noVBand="1"/></w:tblPr><w:tblGrid>`)
	for _, w := range ws {
		fmt.Fprintf(&d.b, `<w:gridCol w:w="%d"/>`, w)
	}
	d.b.WriteString(`</w:tblGrid>`)
	linha := func(cels []string, cabecalho bool, fundo, props string) {
		d.b.WriteString(`<w:tr>`)
		if cabecalho {
			d.b.WriteString(`<w:trPr><w:tblHeader/><w:cantSplit/></w:trPr>`)
		} else {
			d.b.WriteString(`<w:trPr><w:cantSplit/></w:trPr>`)
		}
		for i, c := range cels {
			sombra := ""
			if fundo != "" {
				sombra = `<w:shd w:val="clear" w:color="auto" w:fill="` + fundo + `"/>`
			}
			fmt.Fprintf(&d.b, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/>%s<w:vAlign w:val="center"/></w:tcPr>`, ws[i], sombra)
			d.b.WriteString(par(run(c, props+tam(17)), `<w:jc w:val="center"/><w:spacing w:before="0" w:after="0"/>`))
			d.b.WriteString(`</w:tc>`)
		}
		d.b.WriteString(`</w:tr>`)
	}
	rot := make([]string, len(t.Colunas))
	for i, c := range t.Colunas {
		rot[i] = c.Rotulo
	}
	linha(rot, true, "1F5F3B", negrito+corBranca)
	if len(t.Linhas) == 0 {
		v := make([]string, len(t.Colunas))
		v[0] = t.Vazio
		linha(v, false, "", corFraca)
	}
	for i, l := range t.Linhas {
		fundo := ""
		if i%2 == 1 {
			fundo = "F3F6F1"
		}
		linha(l, false, fundo, "")
	}
	if t.Total != nil {
		linha(t.Total, false, "E3EEE5", negrito)
	}
	d.b.WriteString(`</w:tbl>`)
	d.b.WriteString(par("", `<w:spacing w:before="0" w:after="120"/>`))
}

func (d *docx) pares(itens [][2]string) {
	d.b.WriteString(`<w:tbl><w:tblPr><w:tblW w:w="5000" w:type="pct"/><w:tblBorders><w:insideH w:val="single" w:sz="4" w:color="DDE3DA"/><w:bottom w:val="single" w:sz="4" w:color="DDE3DA"/></w:tblBorders>` +
		`<w:tblLayout w:type="fixed"/><w:tblCellMar><w:top w:w="40" w:type="dxa"/><w:left w:w="0" w:type="dxa"/><w:bottom w:w="40" w:type="dxa"/><w:right w:w="85" w:type="dxa"/></w:tblCellMar></w:tblPr>` +
		`<w:tblGrid><w:gridCol w:w="2900"/><w:gridCol w:w="6738"/></w:tblGrid>`)
	for _, it := range itens {
		d.b.WriteString(`<w:tr><w:trPr><w:cantSplit/></w:trPr><w:tc><w:tcPr><w:tcW w:w="2900" w:type="dxa"/></w:tcPr>` +
			par(run(it[0], negrito+corFraca+tam(19)), `<w:spacing w:before="0" w:after="0"/>`) +
			`</w:tc><w:tc><w:tcPr><w:tcW w:w="6738" w:type="dxa"/></w:tcPr>` +
			par(run(it[1], tam(19)), `<w:spacing w:before="0" w:after="0"/>`) + `</w:tc></w:tr>`)
	}
	d.b.WriteString(`</w:tbl>`)
	d.b.WriteString(par("", `<w:spacing w:before="0" w:after="160"/>`))
}

func (d *docx) corpo(doc Documento) string {
	d.b.WriteString(par(run(doc.Titulo, ""), `<w:pStyle w:val="Title"/>`))
	if doc.Subtitulo != "" {
		d.b.WriteString(par(run(doc.Subtitulo, ""), `<w:pStyle w:val="Subtitle"/>`))
	}
	d.pares(doc.Meta)
	for _, s := range doc.Secoes {
		ppr := `<w:pStyle w:val="Heading1"/>`
		if s.NovaPagina {
			ppr += `<w:pageBreakBefore/>`
		}
		d.b.WriteString(par(run(s.Titulo, ""), ppr))
		if s.Subtitulo != "" {
			d.b.WriteString(par(run(s.Subtitulo, corFraca), ""))
		}
		for _, b := range s.Blocos {
			switch x := b.(type) {
			case Paragrafo:
				props := ""
				if x.Fraco {
					props = corFraca + tam(18)
				}
				d.b.WriteString(par(run(x.Texto, props), ""))
			case Pares:
				d.pares(x.Itens)
			case Tabela:
				d.tabela(x)
			}
		}
	}
	d.b.WriteString(`<w:sectPr><w:headerReference w:type="default" r:id="rIdCab"/><w:footerReference w:type="default" r:id="rIdRod"/>` +
		`<w:pgSz w:w="11906" w:h="16838"/><w:pgMar w:top="2665" w:right="1134" w:bottom="1134" w:left="1134" w:header="454" w:footer="454" w:gutter="0"/></w:sectPr>`)
	return d.b.String()
}

const nsW = `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"`

const cabecalhoXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:hdr ` + nsW + `>%s</w:hdr>`

func cabecalhoDocx() string {
	h := int64(1.75 * emuPorCm)
	w := h * logoLarguraPx / logoAlturaPx
	img := fmt.Sprintf(`<w:r><w:drawing><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:docPr id="1" name="Brasão"/>`+
		`<wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr>`+
		`<a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:nvPicPr><pic:cNvPr id="0" name="logo-cabecalho-docs.jpeg"/><pic:cNvPicPr/></pic:nvPicPr>`+
		`<pic:blipFill><a:blip r:embed="rIdLogo"/><a:stretch><a:fillRect/></a:stretch></pic:blipFill>`+
		`<pic:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`, w, h, w, h)
	return fmt.Sprintf(cabecalhoXML,
		par(img, `<w:jc w:val="center"/><w:spacing w:before="0" w:after="40"/>`)+
			par(run(Instituicao, negrito+tam(22)), `<w:jc w:val="center"/><w:spacing w:before="0" w:after="0"/>`)+
			par(run(Unidade, tam(19)+corFraca), `<w:jc w:val="center"/><w:spacing w:before="0" w:after="0"/><w:pBdr><w:bottom w:val="single" w:sz="8" w:space="6" w:color="1F5F3B"/></w:pBdr>`))
}

func rodapeDocx(doc Documento) string {
	campo := func(instr string) string {
		return `<w:r>` + rpr(corFraca+tam(16)) + `<w:fldChar w:fldCharType="begin"/></w:r><w:r>` + rpr(corFraca+tam(16)) + `<w:instrText xml:space="preserve"> ` + instr + ` </w:instrText></w:r>` +
			`<w:r>` + rpr(corFraca+tam(16)) + `<w:fldChar w:fldCharType="separate"/></w:r><w:r>` + rpr(corFraca+tam(16)) + `<w:t>1</w:t></w:r><w:r>` + rpr(corFraca+tam(16)) + `<w:fldChar w:fldCharType="end"/></w:r>`
	}
	txt := fmt.Sprintf("%s Central · %s · gerado em %s", Sistema, doc.Titulo, doc.GeradoEm.Local().Format("02/01/2006 15:04"))
	conteudo := run(txt, corFraca+tam(16)) + `<w:r><w:tab/></w:r>` + run("Página ", corFraca+tam(16)) + campo("PAGE") + run(" de ", corFraca+tam(16)) + campo("NUMPAGES")
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:ftr ` + nsW + `>` +
		par(conteudo, fmt.Sprintf(`<w:tabs><w:tab w:val="right" w:pos="%d"/></w:tabs><w:pBdr><w:top w:val="single" w:sz="4" w:space="4" w:color="C4CCC0"/></w:pBdr>`, twipsLargUtil)) + `</w:ftr>`
}

const estilosDocx = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Calibri" w:hAnsi="Calibri" w:eastAsia="Calibri" w:cs="Calibri"/><w:color w:val="1B2620"/><w:sz w:val="20"/><w:szCs w:val="20"/><w:lang w:val="pt-BR"/></w:rPr></w:rPrDefault>
<w:pPrDefault><w:pPr><w:spacing w:before="0" w:after="100" w:line="264" w:lineRule="auto"/></w:pPr></w:pPrDefault></w:docDefaults>
<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:qFormat/></w:style>
<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="0" w:after="40"/></w:pPr><w:rPr><w:b/><w:color w:val="1B2620"/><w:sz w:val="40"/><w:szCs w:val="40"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Subtitle"><w:name w:val="Subtitle"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:spacing w:before="0" w:after="200"/></w:pPr><w:rPr><w:color w:val="1F5F3B"/><w:sz w:val="26"/><w:szCs w:val="26"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:spacing w:before="280" w:after="120"/><w:pBdr><w:bottom w:val="single" w:sz="6" w:space="3" w:color="1F5F3B"/></w:pBdr><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:b/><w:color w:val="1F5F3B"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:style>
<w:style w:type="paragraph" w:styleId="TituloTabela"><w:name w:val="Título de tabela"/><w:basedOn w:val="Normal"/><w:next w:val="Normal"/><w:pPr><w:keepNext/><w:spacing w:before="160" w:after="60"/></w:pPr></w:style>
<w:style w:type="table" w:default="1" w:styleId="TableNormal"><w:name w:val="Normal Table"/><w:tblPr><w:tblInd w:w="0" w:type="dxa"/><w:tblCellMar><w:top w:w="0" w:type="dxa"/><w:left w:w="108" w:type="dxa"/><w:bottom w:w="0" w:type="dxa"/><w:right w:w="108" w:type="dxa"/></w:tblCellMar></w:tblPr></w:style>
</w:styles>`

// DOCX gera o documento do Word.
func DOCX(doc Documento) ([]byte, error) {
	var corpo docx
	partes := []struct{ nome, conteudo string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Default Extension="jpeg" ContentType="image/jpeg"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
<Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>
<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>
<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
<Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`},
		{"docProps/core.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
<dc:title>` + esc(doc.Titulo+" — "+doc.Subtitulo) + `</dc:title><dc:creator>` + esc(Instituicao+" — "+Unidade) + `</dc:creator><dc:language>pt-BR</dc:language>
<dcterms:created xsi:type="dcterms:W3CDTF">` + doc.GeradoEm.UTC().Format("2006-01-02T15:04:05Z") + `</dcterms:created></cp:coreProperties>`},
		{"docProps/app.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>` + Sistema + ` Central</Application></Properties>`},
		{"word/_rels/document.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rIdEstilos" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
<Relationship Id="rIdCab" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>
<Relationship Id="rIdRod" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>
</Relationships>`},
		{"word/_rels/header1.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rIdLogo" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="media/logo-cabecalho-docs.jpeg"/>
</Relationships>`},
		{"word/styles.xml", estilosDocx},
		{"word/header1.xml", cabecalhoDocx()},
		{"word/footer1.xml", rodapeDocx(doc)},
		{"word/document.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document ` + nsW + `><w:body>` + corpo.corpo(doc) + `</w:body></w:document>`},
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for _, p := range partes {
		w, err := z.Create(p.nome)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(p.conteudo)); err != nil {
			return nil, err
		}
	}
	w, err := z.CreateHeader(&zip.FileHeader{Name: "word/media/logo-cabecalho-docs.jpeg", Method: zip.Store})
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(Logo); err != nil {
		return nil, err
	}
	if err := z.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
