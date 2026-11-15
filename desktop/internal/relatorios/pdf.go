package relatorios

import (
	"fmt"
	"strings"

	"github.com/signintech/gopdf"

	"sitposse7/desktop/internal/fontes"
)

type rgb struct{ r, g, b uint8 }

// Cores da identidade visual (desktop/docs/telas-e-fluxos.md §Identidade visual).
var (
	corTinta  = rgb{0x1b, 0x26, 0x20}
	corTinta2 = rgb{0x4a, 0x58, 0x4f}
	corMata   = rgb{0x1f, 0x5f, 0x3b}
	corRegua  = rgb{0xc4, 0xcc, 0xc0}
	corFaixa  = rgb{0xf3, 0xf6, 0xf1}
	corTotal  = rgb{0xe3, 0xee, 0xe5}
	corBranco = rgb{0xff, 0xff, 0xff}
)

const (
	pdfMargemX     = 42.0
	pdfTopoConteud = 132.0 // abaixo do cabeçalho institucional
	pdfRodape      = 48.0
	pdfLinhaCelula = 11.0
	pdfPadCelula   = 4.0
)

type pdf struct {
	gp      *gopdf.GoPdf
	larg    float64
	alt     float64
	y       float64
	doc     *Documento
	cabecTb *Tabela // cabeçalho a repetir quando a tabela quebra de página
}

func (p *pdf) fonte(familia string, tam float64) {
	_ = p.gp.SetFont(familia, "", tam)
}
func (p *pdf) texto(c rgb)   { p.gp.SetTextColor(c.r, c.g, c.b) }
func (p *pdf) preench(c rgb) { p.gp.SetFillColor(c.r, c.g, c.b) }
func (p *pdf) traco(c rgb)   { p.gp.SetStrokeColor(c.r, c.g, c.b) }

func (p *pdf) largUtil() float64 { return p.larg - 2*pdfMargemX }
func (p *pdf) limite() float64   { return p.alt - pdfRodape - 14 }

// cabecalho: brasão centralizado e, abaixo dele, os dois textos institucionais.
func (p *pdf) cabecalho() {
	h := 58.0
	w := h * logoLarguraPx / logoAlturaPx
	if img, err := gopdf.ImageHolderByBytes(Logo); err == nil {
		_ = p.gp.ImageByHolder(img, (p.larg-w)/2, 22, &gopdf.Rect{W: w, H: h})
	}
	p.texto(corTinta)
	p.fonte("negrito", 11)
	p.centro(Instituicao, 22+h+6, 14)
	p.fonte("semi", 9.5)
	p.texto(corTinta2)
	p.centro(Unidade, 22+h+21, 12)
	p.traco(corMata)
	p.gp.SetLineWidth(1.2)
	p.gp.Line(pdfMargemX, 120, p.larg-pdfMargemX, 120)
	p.y = pdfTopoConteud
}

func (p *pdf) centro(s string, y, h float64) {
	p.gp.SetXY(pdfMargemX, y)
	_ = p.gp.CellWithOption(&gopdf.Rect{W: p.largUtil(), H: h}, s, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
}

func (p *pdf) novaPagina() {
	p.gp.AddPage()
	p.cabecalho()
}

// garantir quebra a página se não couber h pontos.
func (p *pdf) garantir(h float64) bool {
	if p.y+h > p.limite() {
		p.novaPagina()
		return true
	}
	return false
}

func (p *pdf) linhas(s string, larg float64) []string {
	var out []string
	for _, parte := range strings.Split(s, "\n") {
		ls, err := p.gp.SplitTextWithWordWrap(parte, larg)
		if err != nil || len(ls) == 0 {
			ls = []string{parte}
		}
		out = append(out, ls...)
	}
	return out
}

func (p *pdf) paragrafo(s string, familia string, tam float64, c rgb, entre float64) {
	p.fonte(familia, tam)
	p.texto(c)
	for _, l := range p.linhas(s, p.largUtil()) {
		p.garantir(tam + 3)
		p.gp.SetXY(pdfMargemX, p.y)
		_ = p.gp.Cell(nil, l)
		p.y += tam + 3
	}
	p.y += entre
}

func (p *pdf) tituloDocumento() {
	d := p.doc
	p.fonte("condensada", 20)
	p.texto(corTinta)
	p.gp.SetXY(pdfMargemX, p.y)
	_ = p.gp.Cell(nil, d.Titulo)
	p.y += 24
	if d.Subtitulo != "" {
		p.paragrafo(d.Subtitulo, "semi", 12, corMata, 4)
	}
	p.pares(d.Meta, 0.28)
	p.y += 6
}

func (p *pdf) pares(itens [][2]string, fracRotulo float64) {
	wr := p.largUtil() * fracRotulo
	wv := p.largUtil() - wr
	for _, it := range itens {
		p.fonte("regular", 9.5)
		lv := p.linhas(it[1], wv-6)
		h := float64(len(lv))*12 + 4
		p.garantir(h)
		p.fonte("semi", 9.5)
		p.texto(corTinta2)
		p.gp.SetXY(pdfMargemX, p.y+2)
		_ = p.gp.Cell(nil, it[0])
		p.fonte("regular", 9.5)
		p.texto(corTinta)
		for i, l := range lv {
			p.gp.SetXY(pdfMargemX+wr, p.y+2+float64(i)*12)
			_ = p.gp.Cell(nil, l)
		}
		p.traco(corRegua)
		p.gp.SetLineWidth(0.4)
		p.gp.Line(pdfMargemX, p.y+h, pdfMargemX+p.largUtil(), p.y+h)
		p.y += h
	}
	p.y += 8
}

func (p *pdf) larguras(t Tabela) []float64 {
	return larguras(t, p.largUtil(), 2*pdfPadCelula+1, func(txt string, cab bool) float64 {
		if cab {
			p.fonte("negrito", 8.5)
		} else {
			p.fonte("regular", 8.5)
		}
		w, _ := p.gp.MeasureTextWidth(txt)
		return w
	})
}

// linhaTabela desenha uma linha com texto centralizado, divisórias entre colunas e quebra de texto.
func (p *pdf) linhaTabela(cels []string, ws []float64, familia string, tam float64, fundo *rgb, cor rgb) {
	p.fonte(familia, tam)
	conteudo := make([][]string, len(cels))
	maxL := 1
	for i, c := range cels {
		conteudo[i] = p.linhas(c, ws[i]-2*pdfPadCelula)
		if len(conteudo[i]) > maxL {
			maxL = len(conteudo[i])
		}
	}
	h := float64(maxL)*pdfLinhaCelula + 2*pdfPadCelula
	if p.garantir(h) && p.cabecTb != nil {
		p.cabecalhoTabela(*p.cabecTb, ws)
		p.fonte(familia, tam)
	}
	x := pdfMargemX
	if fundo != nil {
		p.preench(*fundo)
		p.gp.RectFromUpperLeftWithStyle(x, p.y, p.largUtil(), h, "F")
	}
	p.texto(cor)
	for i, ls := range conteudo {
		topo := p.y + (h-float64(len(ls))*pdfLinhaCelula)/2
		for j, l := range ls {
			p.gp.SetXY(x+pdfPadCelula, topo+float64(j)*pdfLinhaCelula)
			_ = p.gp.CellWithOption(&gopdf.Rect{W: ws[i] - 2*pdfPadCelula, H: pdfLinhaCelula}, l, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
		}
		x += ws[i]
	}
	// Bordas: contorno, divisórias verticais e linha inferior.
	p.traco(corRegua)
	p.gp.SetLineWidth(0.5)
	x = pdfMargemX
	p.gp.Line(x, p.y, x, p.y+h)
	for _, w := range ws {
		x += w
		p.gp.Line(x, p.y, x, p.y+h)
	}
	p.gp.Line(pdfMargemX, p.y+h, pdfMargemX+p.largUtil(), p.y+h)
	p.y += h
}

func (p *pdf) cabecalhoTabela(t Tabela, ws []float64) {
	rot := make([]string, len(t.Colunas))
	for i, c := range t.Colunas {
		rot[i] = c.Rotulo
	}
	fundo := corMata
	p.linhaTabela(rot, ws, "negrito", 8.5, &fundo, corBranco)
}

func (p *pdf) tabela(t Tabela) {
	ws := p.larguras(t)
	// Título + cabeçalho + ao menos uma linha na mesma página.
	p.garantir(18 + 3*(pdfLinhaCelula+2*pdfPadCelula))
	if t.Titulo != "" {
		p.paragrafo(t.Titulo, "semi", 10.5, corTinta, 1)
	}
	p.cabecalhoTabela(t, ws)
	if len(t.Linhas) == 0 {
		vazio := make([]string, len(t.Colunas))
		vazio[0] = t.Vazio
		p.linhaTabela(vazio, ws, "regular", 8.5, nil, corTinta2)
	}
	cp := t
	p.cabecTb = &cp
	for i, l := range t.Linhas {
		var fundo *rgb
		if i%2 == 1 {
			f := corFaixa
			fundo = &f
		}
		p.linhaTabela(l, ws, "regular", 8.5, fundo, corTinta)
	}
	if t.Total != nil {
		f := corTotal
		p.linhaTabela(t.Total, ws, "negrito", 8.5, &f, corTinta)
	}
	p.cabecTb = nil
	p.y += 12
}

func (p *pdf) secao(s Secao, primeira bool) {
	if s.NovaPagina && !primeira {
		p.novaPagina()
	}
	p.garantir(60)
	p.fonte("condensada", 14)
	p.texto(corMata)
	p.gp.SetXY(pdfMargemX, p.y)
	_ = p.gp.Cell(nil, s.Titulo)
	p.y += 17
	if s.Subtitulo != "" {
		p.paragrafo(s.Subtitulo, "regular", 9.5, corTinta2, 2)
	}
	p.y += 4
	for _, b := range s.Blocos {
		switch x := b.(type) {
		case Paragrafo:
			c := corTinta
			if x.Fraco {
				c = corTinta2
			}
			p.paragrafo(x.Texto, "regular", 9, c, 6)
		case Pares:
			p.pares(x.Itens, 0.3)
		case Tabela:
			p.tabela(x)
		}
	}
}

// PDF gera o documento em PDF (A4 retrato).
func PDF(d Documento) ([]byte, error) {
	gp := &gopdf.GoPdf{}
	gp.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: *gopdf.PageSizeA4})
	for nome, dados := range map[string][]byte{"regular": fontes.Regular, "semi": fontes.SemiBold, "negrito": fontes.Bold, "condensada": fontes.CondensedBold} {
		if err := gp.AddTTFFontData(nome, dados); err != nil {
			return nil, fmt.Errorf("fonte %s: %w", nome, err)
		}
	}
	gp.SetInfo(gopdf.PdfInfo{Title: d.Titulo + " — " + d.Subtitulo, Author: Instituicao + " — " + Unidade, Creator: Sistema + " Central", CreationDate: d.GeradoEm})
	p := &pdf{gp: gp, larg: gopdf.PageSizeA4.W, alt: gopdf.PageSizeA4.H, doc: &d}
	p.novaPagina()
	p.tituloDocumento()
	for i, s := range d.Secoes {
		p.secao(s, i == 0)
	}
	// Rodapé em todas as páginas, com o total já conhecido.
	n := gp.GetNumberOfPages()
	for i := 1; i <= n; i++ {
		if err := gp.SetPage(i); err != nil {
			return nil, err
		}
		p.traco(corRegua)
		gp.SetLineWidth(0.5)
		gp.Line(pdfMargemX, p.alt-pdfRodape+6, p.larg-pdfMargemX, p.alt-pdfRodape+6)
		p.fonte("regular", 8)
		p.texto(corTinta2)
		gp.SetXY(pdfMargemX, p.alt-pdfRodape+12)
		_ = gp.CellWithOption(&gopdf.Rect{W: p.largUtil() / 2, H: 10}, fmt.Sprintf("%s Central · %s · gerado em %s", Sistema, d.Titulo, d.GeradoEm.Local().Format("02/01/2006 15:04")), gopdf.CellOption{Align: gopdf.Left})
		gp.SetXY(pdfMargemX+p.largUtil()/2, p.alt-pdfRodape+12)
		_ = gp.CellWithOption(&gopdf.Rect{W: p.largUtil() / 2, H: 10}, fmt.Sprintf("Página %d de %d", i, n), gopdf.CellOption{Align: gopdf.Right})
	}
	return gp.GetBytesPdf(), nil
}
