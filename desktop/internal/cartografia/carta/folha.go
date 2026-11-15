package carta

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"strings"

	"github.com/signintech/gopdf"

	"sitposse7/desktop/internal/cartografia/geo"
	"sitposse7/desktop/internal/fontes"
	"sitposse7/desktop/internal/relatorios"
)

// Textos fixos (RN-92 e §11 da especificação).
const (
	Titulo   = "Carta-imagem de localização das propriedades"
	Ressalva = "Ponto(s) coletado(s) no levantamento; não representa(m) limites fundiários nem documento cartográfico oficial."
	NotaPino = "O pino indica a coordenada registrada em campo, com a precisão do GPS do celular; não confirma visualmente a construção. A imagem tem resolução espacial de 10 m: ampliá-la não aumenta a resolução."
	SRC      = "Coordenadas: WGS 84 (EPSG:4326). Projeção da carta: WGS 84 / Pseudo-Mercator (EPSG:3857). Norte para cima."
)

// Coordenada formatada (6 casas ≈ 0,1 m: sem perder a precisão coletada).
func Coordenada(p geo.Ponto) string { return fmt.Sprintf("%.6f, %.6f", p.Lat, p.Lon) }

func precisao(mk Marcador) string {
	switch {
	case mk.Origem == "manual":
		return "digitada"
	case mk.PrecisaoM != nil:
		return fmt.Sprintf("± %.0f m", *mk.PrecisaoM)
	default:
		return "—"
	}
}

// Informacoes do quadro da carta (pares rótulo/valor), iguais no PNG e no PDF.
func Informacoes(d Dados) [][2]string {
	esc := geo.CalcularEscala(d.Imagem.Ret, d.Imagem.Largura)
	equipes, areas := unicos(d.Marcadores, func(m Marcador) string { return m.Equipe }), unicos(d.Marcadores, func(m Marcador) string { return m.AreaIdentificada })
	out := [][2]string{
		{"Operação", padrao(d.Operacao, "—")},
		{"Equipe(s)", padrao(strings.Join(equipes, ", "), "—")},
		{"Área(s) de estudo", padrao(strings.Join(areas, ", "), "—")},
		{"Imagem", "Copernicus Sentinel-2 L2A (Copernicus Data Space Ecosystem), cores reais (B04, B03, B02)"},
		{"Data da imagem", d.Imagem.Periodo()},
		{"Resolução", fmt.Sprintf("10 m (imagem); apresentação ≈ %.1f m por pixel", esc.MetrosPorPixel)},
		{"Referência", SRC},
		{"Gerado em", d.GeradoEm.Local().Format("02/01/2006 15:04") + " pelo SITPOSSE-7 Central"},
	}
	if !esc.Valida {
		out = append(out, [2]string{"Escala", "Sem barra de escala: a área é extensa e a escala varia mais de 1% na carta."})
	}
	if len(d.Criterios) > 0 {
		out = append(out, [2]string{"Filtros", strings.Join(d.Criterios, "; ")})
	}
	if len(d.SemLocalizacao) > 0 {
		out = append(out, [2]string{"Sem localização", fmt.Sprintf("%d propriedade(s) não representada(s) por não ter coordenada: %s", len(d.SemLocalizacao), strings.Join(d.SemLocalizacao, "; "))})
	}
	return append(out, [2]string{"Fonte", d.Atribuicao})
}

func unicos(ms []Marcador, f func(Marcador) string) []string {
	visto := map[string]bool{}
	var out []string
	for _, m := range ms {
		if v := strings.TrimSpace(f(m)); v != "" && !visto[v] {
			visto[v] = true
			out = append(out, v)
		}
	}
	return out
}

func padrao(s, p string) string {
	if strings.TrimSpace(s) == "" {
		return p
	}
	return s
}

// ---------- PNG ----------

// FolhaPNG: cabeçalho, mapa e faixa com legenda, informações e ressalva.
func FolhaPNG(d Dados, mapa *image.NRGBA) ([]byte, error) {
	w := mapa.Bounds().Dx()
	u := float64(w) / 2000
	fTit := face(fontesCarta.negrito, 40*u)
	fSub := face(fontesCarta.normal, 24*u)
	fRot := face(fontesCarta.negrito, 20*u)
	fTxt := face(fontesCarta.normal, 20*u)
	lh := 28 * u

	quebrar := func(s string, larg float64) []string {
		var linhas []string
		atual := ""
		for _, p := range strings.Fields(s) {
			t := strings.TrimSpace(atual + " " + p)
			if largura(fTxt, t) > larg && atual != "" {
				linhas = append(linhas, atual)
				atual = p
			} else {
				atual = t
			}
		}
		return append(linhas, atual)
	}
	colEsq, colDir := float64(w)*0.55, float64(w)*0.45
	// Legenda (coluna direita): número, nome e coordenada.
	var legenda []string
	for _, mk := range d.Marcadores {
		legenda = append(legenda, fmt.Sprintf("%d  %s  (%s; %s)", mk.Numero, mk.Nome, Coordenada(mk.Ponto), precisao(mk)))
	}
	maxLeg := 24
	extra := 0
	if len(legenda) > maxLeg {
		extra = len(legenda) - maxLeg
		legenda = legenda[:maxLeg]
	}
	var linhasInfo [][2]string
	for _, p := range Informacoes(d) {
		for i, l := range quebrar(p[1], colEsq-260*u) {
			r := ""
			if i == 0 {
				r = p[0]
			}
			linhasInfo = append(linhasInfo, [2]string{r, l})
		}
	}
	nLeg := len(legenda) + 1
	if extra > 0 {
		nLeg++
	}
	ressalva := quebrar(Ressalva+" "+NotaPino, float64(w)-80*u)
	topo := 150 * u
	rodape := float64(max(len(linhasInfo), nLeg))*lh + float64(len(ressalva))*lh + 110*u
	H := int(topo) + mapa.Bounds().Dy() + int(rodape)
	f := image.NewNRGBA(image.Rect(0, 0, w, H))
	draw.Draw(f, f.Bounds(), image.NewUniform(corBranca), image.Point{}, draw.Src)

	texto(f, fRot, relatorios.Instituicao+"  ·  "+relatorios.Unidade+"  ·  "+relatorios.Sistema, 40*u, 46*u, corTinta, nil, 0)
	texto(f, fTit, Titulo, 40*u, 100*u, corTinta, nil, 0)
	texto(f, fSub, d.Subtitulo, 40*u, 134*u, corTinta, nil, 0)
	draw.Draw(f, mapa.Bounds().Add(image.Pt(0, int(topo))), mapa, image.Point{}, draw.Src)

	y := topo + float64(mapa.Bounds().Dy()) + 50*u
	for i, l := range linhasInfo {
		texto(f, fRot, l[0], 40*u, y+float64(i)*lh, corTinta, nil, 0)
		texto(f, fTxt, l[1], 260*u, y+float64(i)*lh, corTinta, nil, 0)
	}
	xL := colEsq + 20*u
	texto(f, fRot, "Legenda (nº, propriedade, coordenada, precisão)", xL, y, corTinta, nil, 0)
	for i, l := range legenda {
		for len([]rune(l)) > 10 && largura(fTxt, l) > colDir-60*u {
			l = string([]rune(l)[:len([]rune(l))-2]) + "…"
		}
		texto(f, fTxt, l, xL, y+float64(i+1)*lh, corTinta, nil, 0)
	}
	if extra > 0 {
		texto(f, fTxt, fmt.Sprintf("… e mais %d (lista completa no PDF)", extra), xL, y+float64(len(legenda)+1)*lh, corTinta, nil, 0)
	}
	yr := y + float64(max(len(linhasInfo), nLeg))*lh + 30*u
	for i, l := range ressalva {
		texto(f, fTxt, l, 40*u, yr+float64(i)*lh, corTinta, nil, 0)
	}
	var b bytes.Buffer
	if err := png.Encode(&b, f); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ---------- PDF ----------

type folhaPDF struct {
	gp         *gopdf.GoPdf
	larg, alt  float64
	margem     float64
	cabecalhoY float64
}

func (p *folhaPDF) fonte(f string, t float64) { _ = p.gp.SetFont(f, "", t) }
func (p *folhaPDF) cor(r, g, b uint8)         { p.gp.SetTextColor(r, g, b) }

func (p *folhaPDF) cabecalho(d Dados) {
	gp := p.gp
	m := p.margem
	h := 46.0
	if img, err := gopdf.ImageHolderByBytes(relatorios.Logo); err == nil {
		_ = gp.ImageByHolder(img, m, 16, &gopdf.Rect{W: h * 150 / 182, H: h})
	}
	x := m + h*150/182 + 10
	p.cor(0x1b, 0x26, 0x20)
	p.fonte("negrito", 11)
	gp.SetXY(x, 20)
	_ = gp.Cell(nil, relatorios.Instituicao)
	p.fonte("semi", 9)
	gp.SetXY(x, 34)
	_ = gp.Cell(nil, relatorios.Unidade)
	gp.SetXY(x, 46)
	_ = gp.Cell(nil, relatorios.Sistema)
	p.fonte("condensada", 16)
	gp.SetXY(m, 20)
	_ = gp.CellWithOption(&gopdf.Rect{W: p.larg - 2*m, H: 18}, Titulo, gopdf.CellOption{Align: gopdf.Right})
	p.fonte("semi", 10)
	p.cor(0x1f, 0x5f, 0x3b)
	gp.SetXY(m, 42)
	_ = gp.CellWithOption(&gopdf.Rect{W: p.larg - 2*m, H: 12}, d.Subtitulo, gopdf.CellOption{Align: gopdf.Right})
	gp.SetStrokeColor(0x1f, 0x5f, 0x3b)
	gp.SetLineWidth(1.1)
	gp.Line(m, 68, p.larg-m, 68)
	p.cabecalhoY = 76
}

func (p *folhaPDF) linhas(s string, larg float64) []string {
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

// FolhaPDF: A4 paisagem (A3 com muitas propriedades). Página 1: mapa + legenda +
// informações; páginas seguintes (carta geral): tabela de coordenadas.
func FolhaPDF(d Dados, mapa *image.NRGBA) ([]byte, error) {
	gp := &gopdf.GoPdf{}
	tam := *gopdf.PageSizeA4Landscape
	if len(d.Marcadores) > 25 {
		tam = gopdf.Rect{W: 1190.55, H: 841.89} // A3 paisagem
	}
	gp.Start(gopdf.Config{Unit: gopdf.UnitPT, PageSize: tam})
	for nome, dados := range map[string][]byte{"regular": fontes.Regular, "semi": fontes.SemiBold, "negrito": fontes.Bold, "condensada": fontes.CondensedBold} {
		if err := gp.AddTTFFontData(nome, dados); err != nil {
			return nil, err
		}
	}
	gp.SetInfo(gopdf.PdfInfo{Title: Titulo + " — " + d.Subtitulo, Author: relatorios.Instituicao + " — " + relatorios.Unidade, Creator: relatorios.Sistema + " Central", CreationDate: d.GeradoEm})
	p := &folhaPDF{gp: gp, larg: tam.W, alt: tam.H, margem: 28}
	gp.AddPage()
	p.cabecalho(d)

	// Mapa à esquerda; painel à direita.
	painel := 236.0
	if tam.W > 1000 {
		painel = 320
	}
	disp := tam.W - 2*p.margem - painel - 14
	altDisp := tam.H - p.cabecalhoY - 40
	mw := disp
	mh := mw * float64(mapa.Bounds().Dy()) / float64(mapa.Bounds().Dx())
	if mh > altDisp {
		mh = altDisp
		mw = mh * float64(mapa.Bounds().Dx()) / float64(mapa.Bounds().Dy())
	}
	var jb bytes.Buffer
	if err := jpeg.Encode(&jb, mapa, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	holder, err := gopdf.ImageHolderByBytes(jb.Bytes())
	if err != nil {
		return nil, err
	}
	if err := gp.ImageByHolder(holder, p.margem, p.cabecalhoY, &gopdf.Rect{W: mw, H: mh}); err != nil {
		return nil, err
	}
	gp.SetStrokeColor(0x44, 0x55, 0x49)
	gp.SetLineWidth(0.6)
	gp.RectFromUpperLeftWithStyle(p.margem, p.cabecalhoY, mw, mh, "D")

	x := p.margem + mw + 14
	pw := tam.W - x - p.margem
	y := p.cabecalhoY
	limite := tam.H - 34
	// Legenda.
	p.fonte("negrito", 9.5)
	p.cor(0x1f, 0x5f, 0x3b)
	gp.SetXY(x, y)
	_ = gp.Cell(nil, "Legenda")
	y += 14
	cabem := len(d.Marcadores)
	if lim := int((limite - y - 250) / 11.5); cabem > lim {
		cabem = max(lim, 0)
	}
	p.fonte("regular", 7.5)
	for _, mk := range d.Marcadores[:cabem] {
		gp.SetFillColor(0xF2, 0xB7, 0x05)
		gp.SetStrokeColor(0x10, 0x14, 0x12)
		gp.SetLineWidth(0.6)
		_ = gp.Rectangle(x, y-1, x+11, y+10, "FD", 5.5, 8)
		p.fonte("negrito", 6.5)
		p.cor(0x10, 0x14, 0x12)
		num := fmt.Sprint(mk.Numero)
		gp.SetXY(x, y+0.5)
		_ = gp.CellWithOption(&gopdf.Rect{W: 11, H: 9}, num, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
		p.fonte("regular", 7.5)
		p.cor(0x1b, 0x26, 0x20)
		l := p.linhas(mk.Nome, pw-16)[0]
		gp.SetXY(x+15, y+1)
		_ = gp.Cell(nil, l)
		y += 11.5
	}
	if cabem < len(d.Marcadores) {
		gp.SetXY(x, y+1)
		_ = gp.Cell(nil, fmt.Sprintf("… e mais %d: ver a tabela de coordenadas.", len(d.Marcadores)-cabem))
		y += 12
	}
	y += 8
	// Informações.
	p.fonte("negrito", 9.5)
	p.cor(0x1f, 0x5f, 0x3b)
	gp.SetXY(x, y)
	_ = gp.Cell(nil, "Informações")
	y += 13
	infos := Informacoes(d)
	if d.Tipo == "individual" && len(d.Marcadores) == 1 {
		mk := d.Marcadores[0]
		infos = append([][2]string{{"Coordenada", Coordenada(mk.Ponto) + " (" + precisao(mk) + ")"}}, infos...)
	}
	for _, it := range infos {
		p.fonte("negrito", 6.8)
		p.cor(0x4a, 0x58, 0x4f)
		gp.SetXY(x, y)
		_ = gp.Cell(nil, it[0])
		y += 8.5
		p.fonte("regular", 6.8)
		p.cor(0x1b, 0x26, 0x20)
		for _, l := range p.linhas(it[1], pw) {
			if y > limite-38 {
				break
			}
			gp.SetXY(x, y)
			_ = gp.Cell(nil, l)
			y += 8.3
		}
		y += 2.5
	}
	// Ressalva (RN-92) e nota sobre o pino, sob o mapa.
	yr := p.cabecalhoY + mh + 8
	p.fonte("regular", 7)
	p.cor(0x4a, 0x58, 0x4f)
	for _, l := range p.linhas(Ressalva+" "+NotaPino, mw) {
		gp.SetXY(p.margem, yr)
		_ = gp.Cell(nil, l)
		yr += 8.6
	}

	// Tabela de coordenadas (carta geral ou legenda que não coube).
	if d.Tipo == "geral" || cabem < len(d.Marcadores) {
		cols := []relatorios.Coluna{{Rotulo: "Nº", Peso: 0.5}, {Rotulo: "Propriedade", Peso: 3}, {Rotulo: "Equipe", Peso: 1.3}, {Rotulo: "Latitude", Peso: 1.2}, {Rotulo: "Longitude", Peso: 1.2}, {Rotulo: "Precisão do GPS", Peso: 1.1}}
		t := relatorios.Tabela{Colunas: cols}
		for _, mk := range d.Marcadores {
			t.Linhas = append(t.Linhas, []string{fmt.Sprint(mk.Numero), mk.Nome, mk.Equipe, fmt.Sprintf("%.6f", mk.Ponto.Lat), fmt.Sprintf("%.6f", mk.Ponto.Lon), precisao(mk)})
		}
		p.tabela(d, "Coordenadas das propriedades (WGS 84, graus decimais)", t)
	}

	n := gp.GetNumberOfPages()
	for i := 1; i <= n; i++ {
		_ = gp.SetPage(i)
		p.fonte("regular", 7)
		p.cor(0x4a, 0x58, 0x4f)
		gp.SetXY(p.margem, tam.H-22)
		_ = gp.Cell(nil, fmt.Sprintf("%s Central · %s · gerado em %s", relatorios.Sistema, Titulo, d.GeradoEm.Local().Format("02/01/2006 15:04")))
		gp.SetXY(p.margem, tam.H-22)
		_ = gp.CellWithOption(&gopdf.Rect{W: tam.W - 2*p.margem, H: 9}, fmt.Sprintf("Página %d de %d", i, n), gopdf.CellOption{Align: gopdf.Right})
	}
	return gp.GetBytesPdf(), nil
}

// tabela simples em páginas novas, com cabeçalho repetido, texto centralizado e divisórias.
func (p *folhaPDF) tabela(d Dados, titulo string, t relatorios.Tabela) {
	gp := p.gp
	soma := 0.0
	for _, c := range t.Colunas {
		soma += c.Peso
	}
	util := p.larg - 2*p.margem
	ws := make([]float64, len(t.Colunas))
	for i, c := range t.Colunas {
		ws[i] = util * c.Peso / soma
	}
	y := 0.0
	nova := func() {
		gp.AddPage()
		p.cabecalho(d)
		y = p.cabecalhoY
		p.fonte("condensada", 12)
		p.cor(0x1f, 0x5f, 0x3b)
		gp.SetXY(p.margem, y)
		_ = gp.Cell(nil, titulo)
		y += 18
	}
	linha := func(cels []string, cab bool, zebra bool) {
		h := 15.0
		if y+h > p.alt-36 {
			nova()
			linha2(p, ws, t, &y, true)
		}
		desenharLinha(p, ws, cels, &y, cab, zebra)
	}
	nova()
	linha2(p, ws, t, &y, true)
	for i, l := range t.Linhas {
		linha(l, false, i%2 == 1)
	}
	if len(d.SemLocalizacao) > 0 {
		y += 12
		p.fonte("regular", 8)
		p.cor(0x4a, 0x58, 0x4f)
		for _, l := range p.linhas(fmt.Sprintf("Não representadas por não ter coordenada (%d): %s.", len(d.SemLocalizacao), strings.Join(d.SemLocalizacao, "; ")), util) {
			if y > p.alt-40 {
				nova()
			}
			gp.SetXY(p.margem, y)
			_ = gp.Cell(nil, l)
			y += 10
		}
	}
}

func linha2(p *folhaPDF, ws []float64, t relatorios.Tabela, y *float64, _ bool) {
	rot := make([]string, len(t.Colunas))
	for i, c := range t.Colunas {
		rot[i] = c.Rotulo
	}
	desenharLinha(p, ws, rot, y, true, false)
}

func desenharLinha(p *folhaPDF, ws []float64, cels []string, y *float64, cab, zebra bool) {
	gp := p.gp
	h := 15.0
	x := p.margem
	total := 0.0
	for _, w := range ws {
		total += w
	}
	switch {
	case cab:
		gp.SetFillColor(0x1f, 0x5f, 0x3b)
		gp.RectFromUpperLeftWithStyle(x, *y, total, h, "F")
		p.fonte("negrito", 7.5)
		p.cor(0xff, 0xff, 0xff)
	case zebra:
		gp.SetFillColor(0xf3, 0xf6, 0xf1)
		gp.RectFromUpperLeftWithStyle(x, *y, total, h, "F")
		fallthrough
	default:
		p.fonte("regular", 7.5)
		p.cor(0x1b, 0x26, 0x20)
	}
	for i, c := range cels {
		s := c
		for len([]rune(s)) > 3 {
			if w, _ := gp.MeasureTextWidth(s); w <= ws[i]-6 {
				break
			}
			s = string([]rune(s)[:len([]rune(s))-2]) + "…"
		}
		gp.SetXY(x+3, *y+3)
		_ = gp.CellWithOption(&gopdf.Rect{W: ws[i] - 6, H: 9}, s, gopdf.CellOption{Align: gopdf.Center | gopdf.Middle})
		x += ws[i]
	}
	gp.SetStrokeColor(0xc4, 0xcc, 0xc0)
	gp.SetLineWidth(0.5)
	x = p.margem
	gp.Line(x, *y, x, *y+h)
	for _, w := range ws {
		x += w
		gp.Line(x, *y, x, *y+h)
	}
	gp.Line(p.margem, *y+h, p.margem+total, *y+h)
	*y += h
}
