// Package cog lê Cloud Optimized GeoTIFF por faixas de bytes (ADR-0027), sem GDAL.
//
// Subconjunto suportado (o dos produtos do INPE verificados em 15/11/2026): TIFF clássico e
// BigTIFF, em blocos (tiles), 8 bits sem sinal, 1 a 4 amostras intercaladas, compressão
// nenhuma ou deflate (8/32946), preditor 1 ou 2, overviews (SubfileType 1) e GeoKeys com
// EPSG projetado (UTM) ou geográfico (4326). Qualquer outra coisa é recusada com erro claro.
package cog

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
)

// Faixas lê n bytes a partir de off (HTTP Range, arquivo local, memória…).
type Faixas interface {
	LerFaixa(ctx context.Context, off, n int64) ([]byte, error)
}

// ErrNaoSuportado: arquivo fora do subconjunto lido por este pacote.
var ErrNaoSuportado = errors.New("formato de imagem não suportado")

// Nivel é a resolução completa (0) ou um overview.
type Nivel struct {
	Largura, Altura int
	BlocoL, BlocoA  int
	ResX, ResY      float64       // unidades do SRC por pixel
	offsets, tams   []uint64      // tabelas carregadas (arquivos comuns)
	tabela          *tabelaAdiada // ou lidas sob demanda (mosaicos com milhões de blocos)
}

// tabelaAdiada: TileOffsets/TileByteCounts lidas por pedaços quando a tabela é grande.
type tabelaAdiada struct {
	posOff, posTam uint64
	tipoOff        uint16
	tipoTam        uint16
	mu             sync.Mutex
	pedacos        map[[2]int][]uint64 // (0 = offsets | 1 = tamanhos, pedaço) → valores
}

const tamPedaco = 4096 // entradas por leitura (32 KB em BigTIFF)

// BlocosX e BlocosY: quantos blocos há em cada direção.
func (n Nivel) BlocosX() int { return (n.Largura + n.BlocoL - 1) / n.BlocoL }
func (n Nivel) BlocosY() int { return (n.Altura + n.BlocoA - 1) / n.BlocoA }

// Arquivo COG aberto (só o cabeçalho é lido na abertura).
type Arquivo struct {
	fonte      Faixas
	ordem      binary.ByteOrder
	Niveis     []Nivel
	Amostras   int
	compressao int
	preditor   int
	EPSG       int
	OrigemX    float64 // canto superior esquerdo do pixel (0,0), no SRC
	OrigemY    float64
	NoData     *int
}

// cabecalho lê o início do arquivo e busca mais quando uma tag aponta além do que já veio.
type cabecalho struct {
	ctx   context.Context
	f     Faixas
	dados []byte
	ordem binary.ByteOrder
}

const limiteCabecalho = 64 << 20

func (c *cabecalho) ler(off, n uint64) ([]byte, error) {
	if n > limiteCabecalho || off > math.MaxInt64-n {
		return nil, fmt.Errorf("%w: cabeçalho grande demais", ErrNaoSuportado)
	}
	fim := off + n
	if fim <= uint64(len(c.dados)) {
		return c.dados[off:fim], nil
	}
	if fim <= limiteCabecalho {
		novo := uint64(len(c.dados)) * 2
		if novo < fim {
			novo = fim
		}
		if novo > limiteCabecalho {
			novo = limiteCabecalho
		}
		mais, err := c.f.LerFaixa(c.ctx, int64(len(c.dados)), int64(novo)-int64(len(c.dados)))
		if err != nil {
			return nil, err
		}
		c.dados = append(c.dados, mais...)
		if fim > uint64(len(c.dados)) {
			return nil, fmt.Errorf("%w: arquivo truncado", ErrNaoSuportado)
		}
		return c.dados[off:fim], nil
	}
	return c.f.LerFaixa(c.ctx, int64(off), int64(n)) // dado fora da área inicial (não é COG típico)
}

type entrada struct {
	tipo   uint16
	cont   uint64
	valor  []byte // bytes dos valores (já buscados)
	adiado bool   // valores grandes deixados no arquivo (tabelas de blocos)
	pos    uint64 // posição dos valores no arquivo
}

// limiteTabela: tabelas de blocos maiores que isso são lidas sob demanda.
var limiteTabela uint64 = 1 << 20

var tamTipo = map[uint16]uint64{1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 6: 1, 7: 1, 8: 2, 9: 4, 10: 8, 11: 4, 12: 8, 16: 8, 17: 8, 18: 8}

func (c *cabecalho) inteiros(e entrada) []uint64 {
	var out []uint64
	switch e.tipo {
	case 1, 7:
		for _, b := range e.valor {
			out = append(out, uint64(b))
		}
	case 3:
		for i := 0; i+2 <= len(e.valor); i += 2 {
			out = append(out, uint64(c.ordem.Uint16(e.valor[i:])))
		}
	case 4:
		for i := 0; i+4 <= len(e.valor); i += 4 {
			out = append(out, uint64(c.ordem.Uint32(e.valor[i:])))
		}
	case 16:
		for i := 0; i+8 <= len(e.valor); i += 8 {
			out = append(out, c.ordem.Uint64(e.valor[i:]))
		}
	}
	return out
}

func (c *cabecalho) reais(e entrada) []float64 {
	var out []float64
	if e.tipo == 12 {
		for i := 0; i+8 <= len(e.valor); i += 8 {
			out = append(out, math.Float64frombits(c.ordem.Uint64(e.valor[i:])))
		}
	}
	return out
}

// Abrir lê e valida o cabeçalho.
func Abrir(ctx context.Context, f Faixas) (*Arquivo, error) {
	ini, err := f.LerFaixa(ctx, 0, 64<<10)
	if err != nil {
		return nil, err
	}
	if len(ini) < 16 {
		return nil, fmt.Errorf("%w: arquivo curto demais", ErrNaoSuportado)
	}
	c := &cabecalho{ctx: ctx, f: f, dados: ini}
	switch string(ini[:2]) {
	case "II":
		c.ordem = binary.LittleEndian
	case "MM":
		c.ordem = binary.BigEndian
	default:
		return nil, fmt.Errorf("%w: não é TIFF", ErrNaoSuportado)
	}
	big := false
	var prox uint64
	switch c.ordem.Uint16(ini[2:]) {
	case 42:
		prox = uint64(c.ordem.Uint32(ini[4:]))
	case 43:
		big = true
		prox = c.ordem.Uint64(ini[8:])
	default:
		return nil, fmt.Errorf("%w: versão TIFF desconhecida", ErrNaoSuportado)
	}
	a := &Arquivo{fonte: f, ordem: c.ordem}
	for i := 0; prox != 0 && i < 64; i++ {
		tags, seguinte, err := c.ifd(prox, big)
		if err != nil {
			return nil, err
		}
		prox = seguinte
		if err := a.acrescentar(c, tags, i == 0); err != nil {
			return nil, err
		}
	}
	if len(a.Niveis) == 0 {
		return nil, fmt.Errorf("%w: sem imagem", ErrNaoSuportado)
	}
	return a, nil
}

func (c *cabecalho) ifd(off uint64, big bool) (map[uint16]entrada, uint64, error) {
	tamCont, tamEnt, tamVal := uint64(2), uint64(12), uint64(4)
	if big {
		tamCont, tamEnt, tamVal = 8, 20, 8
	}
	b, err := c.ler(off, tamCont)
	if err != nil {
		return nil, 0, err
	}
	var n uint64
	if big {
		n = c.ordem.Uint64(b)
	} else {
		n = uint64(c.ordem.Uint16(b))
	}
	if n == 0 || n > 4096 {
		return nil, 0, fmt.Errorf("%w: diretório TIFF inválido", ErrNaoSuportado)
	}
	tab, err := c.ler(off+tamCont, n*tamEnt+tamVal)
	if err != nil {
		return nil, 0, err
	}
	tab = append([]byte(nil), tab...) // c.dados pode crescer (e mudar de endereço) abaixo
	tags := map[uint16]entrada{}
	for i := uint64(0); i < n; i++ {
		e := tab[i*tamEnt : (i+1)*tamEnt]
		tag, tipo := c.ordem.Uint16(e), c.ordem.Uint16(e[2:])
		var cont uint64
		var campo []byte
		if big {
			cont, campo = c.ordem.Uint64(e[4:]), e[12:20]
		} else {
			cont, campo = uint64(c.ordem.Uint32(e[4:])), e[8:12]
		}
		t, ok := tamTipo[tipo]
		if !ok {
			continue
		}
		total := t * cont
		var val []byte
		if total <= tamVal {
			val = campo[:total]
		} else {
			var pos uint64
			if big {
				pos = c.ordem.Uint64(campo)
			} else {
				pos = uint64(c.ordem.Uint32(campo))
			}
			if (tag == 324 || tag == 325) && cont*8 > limiteTabela && (tipo == 4 || tipo == 16) {
				tags[tag] = entrada{tipo: tipo, cont: cont, adiado: true, pos: pos}
				continue
			}
			v, err := c.ler(pos, total)
			if err != nil {
				return nil, 0, err
			}
			val = append([]byte(nil), v...)
		}
		tags[tag] = entrada{tipo: tipo, cont: cont, valor: val}
	}
	resto := tab[n*tamEnt:]
	var prox uint64
	if big {
		prox = c.ordem.Uint64(resto)
	} else {
		prox = uint64(c.ordem.Uint32(resto))
	}
	return tags, prox, nil
}

func (c *cabecalho) um(tags map[uint16]entrada, tag uint16, padrao uint64) uint64 {
	if e, ok := tags[tag]; ok {
		if v := c.inteiros(e); len(v) > 0 {
			return v[0]
		}
	}
	return padrao
}

func (a *Arquivo) acrescentar(c *cabecalho, tags map[uint16]entrada, primeiro bool) error {
	subtipo := c.um(tags, 254, 0)
	if !primeiro && subtipo&4 != 0 {
		return nil // máscara: ignorada (nodata vem do valor 0)
	}
	if !primeiro && subtipo&1 == 0 {
		return nil // outra página que não é overview
	}
	if _, ok := tags[322]; !ok {
		return fmt.Errorf("%w: imagem sem blocos (não é COG)", ErrNaoSuportado)
	}
	for _, b := range c.inteiros(tags[258]) {
		if b != 8 {
			return fmt.Errorf("%w: %d bits por amostra", ErrNaoSuportado, b)
		}
	}
	if c.um(tags, 339, 1) != 1 {
		return fmt.Errorf("%w: amostras com sinal ou em ponto flutuante", ErrNaoSuportado)
	}
	if c.um(tags, 284, 1) != 1 {
		return fmt.Errorf("%w: bandas em planos separados", ErrNaoSuportado)
	}
	comp, pred := int(c.um(tags, 259, 1)), int(c.um(tags, 317, 1))
	if comp != 1 && comp != 8 && comp != 32946 {
		return fmt.Errorf("%w: compressão %d", ErrNaoSuportado, comp)
	}
	if pred != 1 && pred != 2 {
		return fmt.Errorf("%w: preditor %d", ErrNaoSuportado, pred)
	}
	spp := int(c.um(tags, 277, 1))
	if spp < 1 || spp > 4 {
		return fmt.Errorf("%w: %d amostras por pixel", ErrNaoSuportado, spp)
	}
	n := Nivel{Largura: int(c.um(tags, 256, 0)), Altura: int(c.um(tags, 257, 0)), BlocoL: int(c.um(tags, 322, 0)), BlocoA: int(c.um(tags, 323, 0))}
	if n.Largura <= 0 || n.Altura <= 0 || n.BlocoL <= 0 || n.BlocoA <= 0 || n.BlocoL > 4096 || n.BlocoA > 4096 {
		return fmt.Errorf("%w: dimensões inválidas", ErrNaoSuportado)
	}
	eo, et := tags[324], tags[325]
	if eo.adiado || et.adiado {
		if eo.cont != uint64(n.BlocosX()*n.BlocosY()) || et.cont != eo.cont {
			return fmt.Errorf("%w: tabela de blocos inconsistente", ErrNaoSuportado)
		}
		if !eo.adiado || !et.adiado {
			return fmt.Errorf("%w: tabela de blocos mista", ErrNaoSuportado)
		}
		n.tabela = &tabelaAdiada{posOff: eo.pos, posTam: et.pos, tipoOff: eo.tipo, tipoTam: et.tipo, pedacos: map[[2]int][]uint64{}}
	} else {
		n.offsets, n.tams = c.inteiros(eo), c.inteiros(et)
		if len(n.offsets) != n.BlocosX()*n.BlocosY() || len(n.tams) != len(n.offsets) {
			return fmt.Errorf("%w: tabela de blocos inconsistente", ErrNaoSuportado)
		}
	}
	if primeiro {
		a.Amostras, a.compressao, a.preditor = spp, comp, pred
		if err := a.georreferencia(c, tags, &n); err != nil {
			return err
		}
		if e, ok := tags[42113]; ok {
			s := strings.TrimRight(string(e.valor), "\x00 ")
			if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				a.NoData = &v
			}
		}
	} else {
		if spp != a.Amostras || comp != a.compressao || pred != a.preditor {
			return fmt.Errorf("%w: overview com formato diferente", ErrNaoSuportado)
		}
		base := a.Niveis[0]
		n.ResX = base.ResX * float64(base.Largura) / float64(n.Largura)
		n.ResY = base.ResY * float64(base.Altura) / float64(n.Altura)
	}
	a.Niveis = append(a.Niveis, n)
	return nil
}

func (a *Arquivo) georreferencia(c *cabecalho, tags map[uint16]entrada, n *Nivel) error {
	if _, ok := tags[34264]; ok {
		return fmt.Errorf("%w: imagem rotacionada (ModelTransformation)", ErrNaoSuportado)
	}
	escala, tie := c.reais(tags[33550]), c.reais(tags[33922])
	if len(escala) < 2 || len(tie) < 6 || escala[0] <= 0 || escala[1] <= 0 {
		return fmt.Errorf("%w: sem georreferência", ErrNaoSuportado)
	}
	n.ResX, n.ResY = escala[0], escala[1]
	a.OrigemX = tie[3] - tie[0]*n.ResX
	a.OrigemY = tie[4] + tie[1]*n.ResY
	chaves := c.inteiros(tags[34735])
	if len(chaves) < 4 {
		return fmt.Errorf("%w: sem GeoKeys", ErrNaoSuportado)
	}
	val := map[uint64]uint64{}
	for i := 4; i+3 < len(chaves) && i < 4+4*int(chaves[3]); i += 4 {
		if chaves[i+1] == 0 { // valor no próprio diretório
			val[chaves[i]] = chaves[i+3]
		}
	}
	switch val[1024] {
	case 1:
		a.EPSG = int(val[3072])
	case 2:
		a.EPSG = int(val[2048])
	}
	if a.EPSG == 0 || a.EPSG == 32767 {
		return fmt.Errorf("%w: sistema de referência sem código EPSG", ErrNaoSuportado)
	}
	if val[1025] == 2 { // PixelIsPoint: o tiepoint é o centro do pixel
		a.OrigemX -= n.ResX / 2
		a.OrigemY += n.ResY / 2
	}
	return nil
}

// Bloco devolve os pixels do bloco (bx, by) do nível, com BlocoL × BlocoA × Amostras bytes.
// Bloco ausente (offset 0) vem zerado (sem dados).
func (a *Arquivo) Bloco(ctx context.Context, nivel, bx, by int) ([]byte, error) {
	if nivel < 0 || nivel >= len(a.Niveis) {
		return nil, errors.New("nível inexistente")
	}
	n := a.Niveis[nivel]
	if bx < 0 || by < 0 || bx >= n.BlocosX() || by >= n.BlocosY() {
		return nil, errors.New("bloco fora da imagem")
	}
	i := by*n.BlocosX() + bx
	tam := n.BlocoL * n.BlocoA * a.Amostras
	off, tb, err := a.posBloco(ctx, n, i)
	if err != nil {
		return nil, err
	}
	if off == 0 || tb == 0 {
		return make([]byte, tam), nil
	}
	if tb > 64<<20 {
		return nil, fmt.Errorf("%w: bloco grande demais", ErrNaoSuportado)
	}
	bruto, err := a.fonte.LerFaixa(ctx, int64(off), int64(tb))
	if err != nil {
		return nil, err
	}
	var px []byte
	if a.compressao == 1 {
		px = bruto
	} else {
		zr, err := zlib.NewReader(bytes.NewReader(bruto))
		if err != nil {
			return nil, fmt.Errorf("bloco corrompido: %w", err)
		}
		px = make([]byte, tam)
		if _, err := io.ReadFull(zr, px); err != nil {
			return nil, fmt.Errorf("bloco corrompido: %w", err)
		}
	}
	if len(px) < tam {
		return nil, errors.New("bloco corrompido: tamanho menor que o esperado")
	}
	px = px[:tam]
	if a.preditor == 2 {
		linha := n.BlocoL * a.Amostras
		for y := 0; y < n.BlocoA; y++ {
			l := px[y*linha : (y+1)*linha]
			for x := a.Amostras; x < linha; x++ {
				l[x] += l[x-a.Amostras]
			}
		}
	}
	return px, nil
}

// posBloco devolve a posição e o tamanho do bloco i (das tabelas carregadas ou por pedaços).
func (a *Arquivo) posBloco(ctx context.Context, n Nivel, i int) (uint64, uint64, error) {
	if n.tabela == nil {
		return n.offsets[i], n.tams[i], nil
	}
	off, err := a.valorTabela(ctx, n.tabela, 0, i)
	if err != nil {
		return 0, 0, err
	}
	tb, err := a.valorTabela(ctx, n.tabela, 1, i)
	return off, tb, err
}

func (a *Arquivo) valorTabela(ctx context.Context, t *tabelaAdiada, qual, i int) (uint64, error) {
	p := i / tamPedaco
	t.mu.Lock()
	v, ok := t.pedacos[[2]int{qual, p}]
	t.mu.Unlock()
	if !ok {
		pos, tipo := t.posOff, t.tipoOff
		if qual == 1 {
			pos, tipo = t.posTam, t.tipoTam
		}
		lado := uint64(4)
		if tipo == 16 {
			lado = 8
		}
		b, err := a.fonte.LerFaixa(ctx, int64(pos+uint64(p*tamPedaco)*lado), int64(tamPedaco*lado))
		if err != nil {
			return 0, err
		}
		v = make([]uint64, len(b)/int(lado))
		for k := range v {
			if lado == 8 {
				v[k] = a.ordem.Uint64(b[k*8:])
			} else {
				v[k] = uint64(a.ordem.Uint32(b[k*4:]))
			}
		}
		t.mu.Lock()
		t.pedacos[[2]int{qual, p}] = v
		t.mu.Unlock()
	}
	if i%tamPedaco >= len(v) {
		return 0, fmt.Errorf("%w: tabela de blocos truncada", ErrNaoSuportado)
	}
	return v[i%tamPedaco], nil
}

// PixelParaSRC: coordenada (no SRC do arquivo) do canto superior esquerdo do pixel (col, lin).
func (a *Arquivo) PixelParaSRC(nivel int, col, lin float64) (x, y float64) {
	n := a.Niveis[nivel]
	return a.OrigemX + col*n.ResX, a.OrigemY - lin*n.ResY
}

// SRCParaPixel é a inversa (col/lin fracionários; centro do pixel 0 = 0,5).
func (a *Arquivo) SRCParaPixel(nivel int, x, y float64) (col, lin float64) {
	n := a.Niveis[nivel]
	return (x - a.OrigemX) / n.ResX, (a.OrigemY - y) / n.ResY
}

// NivelPara escolhe o nível mais grosseiro cuja resolução não passa de res (unidades do SRC);
// se nenhum for fino o bastante, o nível 0.
func (a *Arquivo) NivelPara(res float64) int {
	escolhido := 0
	for i, n := range a.Niveis {
		if n.ResX <= res*1.0001 {
			escolhido = i
		}
	}
	return escolhido
}
