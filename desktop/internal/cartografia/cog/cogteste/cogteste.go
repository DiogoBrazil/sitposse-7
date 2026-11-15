// Package cogteste escreve COGs sintéticos para os testes (nunca usado em produção).
package cogteste

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"math"
)

// Imagem: pixels intercalados (L × A × Amostras bytes).
type Imagem struct {
	L, A, Amostras int
	Pix            []byte
}

// Opcoes do arquivo gerado.
type Opcoes struct {
	Big        bool
	Bloco      int // padrão 64
	EPSG       int // 4326 = geográfico; senão projetado
	OrigemX    float64
	OrigemY    float64
	ResX, ResY float64
	NoData     *int
	SemDeflate bool
	Preditor2  bool
	PixelPonto bool // RasterType = PixelIsPoint
}

type ent struct {
	tag, tipo uint16
	cont      uint64
	val       []byte // valores já codificados
}

var le = binary.LittleEndian

func shorts(v ...uint64) ent {
	b := make([]byte, 2*len(v))
	for i, x := range v {
		le.PutUint16(b[2*i:], uint16(x))
	}
	return ent{tipo: 3, cont: uint64(len(v)), val: b}
}

func longs(big bool, v ...uint64) ent {
	if big {
		b := make([]byte, 8*len(v))
		for i, x := range v {
			le.PutUint64(b[8*i:], x)
		}
		return ent{tipo: 16, cont: uint64(len(v)), val: b}
	}
	b := make([]byte, 4*len(v))
	for i, x := range v {
		le.PutUint32(b[4*i:], uint32(x))
	}
	return ent{tipo: 4, cont: uint64(len(v)), val: b}
}

func doubles(v ...float64) ent {
	b := make([]byte, 8*len(v))
	for i, x := range v {
		le.PutUint64(b[8*i:], math.Float64bits(x))
	}
	return ent{tipo: 12, cont: uint64(len(v)), val: b}
}

func ascii(s string) ent { return ent{tipo: 2, cont: uint64(len(s) + 1), val: append([]byte(s), 0)} }

func com(tag uint16, e ent) ent { e.tag = tag; return e }

// blocos corta e comprime os blocos de uma imagem.
func blocos(im Imagem, b int, op Opcoes) [][]byte {
	bx, by := (im.L+b-1)/b, (im.A+b-1)/b
	var out [][]byte
	for ty := 0; ty < by; ty++ {
		for tx := 0; tx < bx; tx++ {
			px := make([]byte, b*b*im.Amostras)
			for y := 0; y < b; y++ {
				for x := 0; x < b; x++ {
					gx, gy := tx*b+x, ty*b+y
					if gx < im.L && gy < im.A {
						copy(px[(y*b+x)*im.Amostras:], im.Pix[(gy*im.L+gx)*im.Amostras:(gy*im.L+gx+1)*im.Amostras])
					}
				}
			}
			if op.Preditor2 {
				linha := b * im.Amostras
				for y := 0; y < b; y++ {
					l := px[y*linha : (y+1)*linha]
					for x := linha - 1; x >= im.Amostras; x-- {
						l[x] -= l[x-im.Amostras]
					}
				}
			}
			if !op.SemDeflate {
				var z bytes.Buffer
				w := zlib.NewWriter(&z)
				w.Write(px)
				w.Close()
				px = z.Bytes()
			}
			out = append(out, px)
		}
	}
	return out
}

// Escrever gera o COG: o nível 0 e os overviews (do mais fino ao mais grosseiro).
func Escrever(niveis []Imagem, op Opcoes) []byte {
	b := op.Bloco
	if b == 0 {
		b = 64
	}
	dados := make([][][]byte, len(niveis))
	for i, im := range niveis {
		dados[i] = blocos(im, b, op)
	}
	monta := func(i int, offs []uint64) []ent {
		im := niveis[i]
		var tams []uint64
		for _, d := range dados[i] {
			tams = append(tams, uint64(len(d)))
		}
		bits := make([]uint64, im.Amostras)
		fmts := make([]uint64, im.Amostras)
		for k := range bits {
			bits[k], fmts[k] = 8, 1
		}
		foto := uint64(1)
		if im.Amostras >= 3 {
			foto = 2
		}
		comp := uint64(8)
		if op.SemDeflate {
			comp = 1
		}
		pred := uint64(1)
		if op.Preditor2 {
			pred = 2
		}
		es := []ent{}
		if i > 0 {
			es = append(es, com(254, longs(false, 1)))
		}
		es = append(es,
			com(256, longs(false, uint64(im.L))), com(257, longs(false, uint64(im.A))),
			com(258, shorts(bits...)), com(259, shorts(comp)), com(262, shorts(foto)),
			com(277, shorts(uint64(im.Amostras))), com(284, shorts(1)), com(317, shorts(pred)),
			com(322, shorts(uint64(b))), com(323, shorts(uint64(b))),
			com(324, longs(op.Big, offs...)), com(325, longs(op.Big, tams...)), com(339, shorts(fmts...)))
		if i == 0 {
			tipo := uint64(1)
			if op.PixelPonto {
				tipo = 2
			}
			var geo ent
			if op.EPSG == 4326 {
				geo = shorts(1, 1, 0, 3, 1024, 0, 1, 2, 1025, 0, 1, tipo, 2048, 0, 1, 4326)
			} else {
				geo = shorts(1, 1, 0, 3, 1024, 0, 1, 1, 1025, 0, 1, tipo, 3072, 0, 1, uint64(op.EPSG))
			}
			es = append(es, com(33550, doubles(op.ResX, op.ResY, 0)), com(33922, doubles(0, 0, 0, op.OrigemX, op.OrigemY, 0)), com(34735, geo))
			if op.NoData != nil {
				es = append(es, com(42113, ascii(itoa(*op.NoData))))
			}
		}
		return es
	}
	tamCab, tamCont, tamEnt, tamVal := 8, 2, 12, 4
	if op.Big {
		tamCab, tamCont, tamEnt, tamVal = 16, 8, 20, 8
	}
	tamIFD := func(es []ent) int {
		t := tamCont + len(es)*tamEnt + tamVal
		for _, e := range es {
			if len(e.val) > tamVal {
				t += len(e.val) + len(e.val)%2
			}
		}
		return t
	}
	// 1ª passada: tamanhos (independem dos valores dos offsets).
	pos := tamCab
	inicioIFD := make([]int, len(niveis))
	for i := range niveis {
		inicioIFD[i] = pos
		pos += tamIFD(monta(i, make([]uint64, len(dados[i]))))
	}
	offs := make([][]uint64, len(niveis))
	for i := range niveis {
		for _, d := range dados[i] {
			offs[i] = append(offs[i], uint64(pos))
			pos += len(d)
		}
	}
	out := make([]byte, 0, pos)
	if op.Big {
		out = append(out, 'I', 'I', 43, 0, 8, 0, 0, 0)
		out = le.AppendUint64(out, uint64(inicioIFD[0]))
	} else {
		out = append(out, 'I', 'I', 42, 0)
		out = le.AppendUint32(out, uint32(inicioIFD[0]))
	}
	for i := range niveis {
		es := monta(i, offs[i])
		base := len(out)
		extra := base + tamCont + len(es)*tamEnt + tamVal
		var tab, ext []byte
		if op.Big {
			tab = le.AppendUint64(tab, uint64(len(es)))
		} else {
			tab = le.AppendUint16(tab, uint16(len(es)))
		}
		for _, e := range es {
			tab = le.AppendUint16(tab, e.tag)
			tab = le.AppendUint16(tab, e.tipo)
			if op.Big {
				tab = le.AppendUint64(tab, e.cont)
			} else {
				tab = le.AppendUint32(tab, uint32(e.cont))
			}
			campo := make([]byte, tamVal)
			if len(e.val) <= tamVal {
				copy(campo, e.val)
			} else {
				p := extra + len(ext)
				if op.Big {
					le.PutUint64(campo, uint64(p))
				} else {
					le.PutUint32(campo, uint32(p))
				}
				ext = append(ext, e.val...)
				if len(e.val)%2 == 1 {
					ext = append(ext, 0)
				}
			}
			tab = append(tab, campo...)
		}
		prox := 0
		if i+1 < len(niveis) {
			prox = inicioIFD[i+1]
		}
		if op.Big {
			tab = le.AppendUint64(tab, uint64(prox))
		} else {
			tab = le.AppendUint32(tab, uint32(prox))
		}
		out = append(out, tab...)
		out = append(out, ext...)
	}
	for i := range niveis {
		for _, d := range dados[i] {
			out = append(out, d...)
		}
	}
	return out
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var s []byte
	for v > 0 {
		s = append([]byte{byte('0' + v%10)}, s...)
		v /= 10
	}
	if neg {
		s = append([]byte{'-'}, s...)
	}
	return string(s)
}

// Memoria implementa cog.Faixas sobre um arquivo em memória.
type Memoria []byte

func (m Memoria) LerFaixa(_ context.Context, off, n int64) ([]byte, error) {
	if off >= int64(len(m)) {
		return nil, nil
	}
	fim := off + n
	if fim > int64(len(m)) {
		fim = int64(len(m))
	}
	return m[off:fim], nil
}

// Gradiente cria uma imagem RGB determinística (para conferir posições de pixels).
func Gradiente(l, a int) Imagem {
	im := Imagem{L: l, A: a, Amostras: 3, Pix: make([]byte, l*a*3)}
	for y := 0; y < a; y++ {
		for x := 0; x < l; x++ {
			i := (y*l + x) * 3
			im.Pix[i], im.Pix[i+1], im.Pix[i+2] = byte(x%251+1), byte(y%251+1), byte((x+y)%200+20)
		}
	}
	return im
}

// Reduzir faz um overview 2× por média simples.
func Reduzir(im Imagem) Imagem {
	l, a := (im.L+1)/2, (im.A+1)/2
	o := Imagem{L: l, A: a, Amostras: im.Amostras, Pix: make([]byte, l*a*im.Amostras)}
	for y := 0; y < a; y++ {
		for x := 0; x < l; x++ {
			for k := 0; k < im.Amostras; k++ {
				s, n := 0, 0
				for dy := 0; dy < 2; dy++ {
					for dx := 0; dx < 2; dx++ {
						gx, gy := 2*x+dx, 2*y+dy
						if gx < im.L && gy < im.A {
							s += int(im.Pix[(gy*im.L+gx)*im.Amostras+k])
							n++
						}
					}
				}
				o.Pix[(y*l+x)*im.Amostras+k] = byte(s / n)
			}
		}
	}
	return o
}
