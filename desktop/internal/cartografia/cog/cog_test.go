package cog_test

import (
	"bytes"
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/cog"
	"sitposse7/desktop/internal/cartografia/cog/cogteste"
)

func pixel(t *testing.T, a *cog.Arquivo, nivel, x, y int) []byte {
	t.Helper()
	n := a.Niveis[nivel]
	b, err := a.Bloco(context.Background(), nivel, x/n.BlocoL, y/n.BlocoA)
	if err != nil {
		t.Fatal(err)
	}
	i := ((y%n.BlocoA)*n.BlocoL + x%n.BlocoL) * a.Amostras
	return b[i : i+a.Amostras]
}

func TestLerCOGClassicoEBig(t *testing.T) {
	base := cogteste.Gradiente(300, 200)
	ov := cogteste.Reduzir(base)
	zero := 0
	for _, op := range []cogteste.Opcoes{
		{Big: false, EPSG: 32720, OrigemX: 431990, OrigemY: 8915048, ResX: 2, ResY: 2, NoData: &zero},
		{Big: true, EPSG: 32720, OrigemX: 431990, OrigemY: 8915048, ResX: 2, ResY: 2, Preditor2: true},
		{Big: true, EPSG: 4326, OrigemX: -76, OrigemY: 6, ResX: 0.0001, ResY: 0.0001, SemDeflate: true, Bloco: 128},
	} {
		arq := cogteste.Escrever([]cogteste.Imagem{base, ov}, op)
		a, err := cog.Abrir(context.Background(), cogteste.Memoria(arq))
		if err != nil {
			t.Fatal(err)
		}
		if len(a.Niveis) != 2 || a.Amostras != 3 || a.EPSG != op.EPSG || a.Niveis[0].Largura != 300 || a.Niveis[1].Largura != 150 {
			t.Fatalf("cabeçalho: %+v", a)
		}
		if a.Niveis[1].ResX != 2*op.ResX {
			t.Fatalf("resolução do overview %v", a.Niveis[1].ResX)
		}
		for _, xy := range [][2]int{{0, 0}, {299, 199}, {65, 130}, {128, 64}} {
			got := pixel(t, a, 0, xy[0], xy[1])
			i := (xy[1]*300 + xy[0]) * 3
			if !bytes.Equal(got, base.Pix[i:i+3]) {
				t.Fatalf("pixel %v = %v, esperado %v", xy, got, base.Pix[i:i+3])
			}
		}
		x, y := a.PixelParaSRC(0, 10, 20)
		if x != op.OrigemX+10*op.ResX || y != op.OrigemY-20*op.ResY {
			t.Fatal("georreferência", x, y)
		}
		c, l := a.SRCParaPixel(0, x, y)
		if math.Abs(c-10) > 1e-6 || math.Abs(l-20) > 1e-6 {
			t.Fatal("inversa", c, l)
		}
		if a.NivelPara(op.ResX*3) != 1 || a.NivelPara(op.ResX) != 0 || a.NivelPara(op.ResX/2) != 0 {
			t.Fatal("escolha do nível")
		}
		if (op.NoData == nil) != (a.NoData == nil) {
			t.Fatal("nodata")
		}
	}
}

func TestPixelIsPoint(t *testing.T) {
	arq := cogteste.Escrever([]cogteste.Imagem{cogteste.Gradiente(64, 64)}, cogteste.Opcoes{EPSG: 32720, OrigemX: 1000, OrigemY: 2000, ResX: 10, ResY: 10, PixelPonto: true})
	a, err := cog.Abrir(context.Background(), cogteste.Memoria(arq))
	if err != nil {
		t.Fatal(err)
	}
	if a.OrigemX != 995 || a.OrigemY != 2005 {
		t.Fatal("PixelIsPoint", a.OrigemX, a.OrigemY)
	}
}

func TestRecusas(t *testing.T) {
	ctx := context.Background()
	if _, err := cog.Abrir(ctx, cogteste.Memoria([]byte("isto não é um tiff, só texto qualquer"))); !errors.Is(err, cog.ErrNaoSuportado) {
		t.Fatal("aceitou não-TIFF", err)
	}
	arq := cogteste.Escrever([]cogteste.Imagem{cogteste.Gradiente(64, 64)}, cogteste.Opcoes{EPSG: 32720, ResX: 1, ResY: 1})
	// Compressão JPEG (7) não é suportada: troca o valor da tag 259 no arquivo.
	ruim := append([]byte(nil), arq...)
	for i := 10; i+12 < len(ruim); i += 12 {
		if ruim[i] == 0x03 && ruim[i+1] == 0x01 { // tag 259 em little-endian
			ruim[i+8] = 7
			break
		}
	}
	if _, err := cog.Abrir(ctx, cogteste.Memoria(ruim)); !errors.Is(err, cog.ErrNaoSuportado) {
		t.Fatal("aceitou JPEG", err)
	}
	// Bloco corrompido.
	a, err := cog.Abrir(ctx, cogteste.Memoria(arq[:len(arq)-40]))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Bloco(ctx, 0, 0, 0); err == nil {
		t.Fatal("bloco truncado aceito")
	}
}

func TestHTTPFaixasCacheEOffline(t *testing.T) {
	arq := cogteste.Escrever([]cogteste.Imagem{cogteste.Gradiente(256, 256)}, cogteste.Opcoes{Big: true, EPSG: 32720, ResX: 2, ResY: 2})
	var pedidos atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pedidos.Add(1)
		w.Header().Set("ETag", `"v1"`)
		http.ServeContent(w, r, "a.tif", time.Unix(0, 0), bytes.NewReader(arq))
	}))
	cache := &cog.Cache{Dir: t.TempDir()}
	h := &cog.HTTP{URL: srv.URL + "/a.tif", Cache: cache, Esperar: func(time.Duration) {}}
	a, err := cog.Abrir(context.Background(), h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Bloco(context.Background(), 0, 3, 3); err != nil {
		t.Fatal(err)
	}
	if h.Baixados.Load() == 0 || h.Baixados.Load() > int64(len(arq)) {
		t.Fatal("bytes baixados", h.Baixados.Load())
	}
	antes := pedidos.Load()
	if _, err := a.Bloco(context.Background(), 0, 3, 3); err != nil || pedidos.Load() != antes {
		t.Fatal("bloco não veio do cache", err)
	}
	// Sem rede: um leitor novo (outra sessão) abre pelo cache.
	srv.Close()
	h2 := &cog.HTTP{URL: h.URL, Cache: cache, Esperar: func(time.Duration) {}}
	a2, err := cog.Abrir(context.Background(), h2)
	if err != nil {
		t.Fatal("offline com cache:", err)
	}
	if _, err := a2.Bloco(context.Background(), 0, 3, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := a2.Bloco(context.Background(), 0, 0, 0); !errors.Is(err, cartografia.ErrSemConexao) {
		t.Fatal("bloco não guardado deveria exigir rede", err)
	}
}

func TestHTTPRecusaServidorSemRangeERepete5xx(t *testing.T) {
	arq := cogteste.Escrever([]cogteste.Imagem{cogteste.Gradiente(64, 64)}, cogteste.Opcoes{EPSG: 32720, ResX: 1, ResY: 1})
	semRange := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(arq) }))
	defer semRange.Close()
	if _, err := cog.Abrir(context.Background(), &cog.HTTP{URL: semRange.URL}); !errors.Is(err, cartografia.ErrResposta) {
		t.Fatal("aceitou servidor sem Range", err)
	}
	var n atomic.Int64
	instavel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) <= 2 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		http.ServeContent(w, r, "a.tif", time.Unix(0, 0), bytes.NewReader(arq))
	}))
	defer instavel.Close()
	if _, err := cog.Abrir(context.Background(), &cog.HTTP{URL: instavel.URL, Esperar: func(time.Duration) {}}); err != nil {
		t.Fatal("não repetiu após 502", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cog.Abrir(ctx, &cog.HTTP{URL: instavel.URL}); err == nil {
		t.Fatal("ignorou o cancelamento")
	}
}

// Tabelas de blocos enormes (mosaico do Brasil: 13,7 milhões de blocos) são lidas por pedaços.
func TestTabelaDeBlocosSobDemanda(t *testing.T) {
	defer cog.LimiteTabela(16)()
	base := cogteste.Gradiente(300, 200)
	for _, big := range []bool{false, true} {
		arq := cogteste.Escrever([]cogteste.Imagem{base, cogteste.Reduzir(base)}, cogteste.Opcoes{Big: big, EPSG: 32720, ResX: 2, ResY: 2, Bloco: 32})
		a, err := cog.Abrir(context.Background(), cogteste.Memoria(arq))
		if err != nil {
			t.Fatal(err)
		}
		for _, xy := range [][2]int{{0, 0}, {299, 199}, {150, 77}} {
			got := pixel(t, a, 0, xy[0], xy[1])
			i := (xy[1]*300 + xy[0]) * 3
			if !bytes.Equal(got, base.Pix[i:i+3]) {
				t.Fatalf("big=%v pixel %v = %v", big, xy, got)
			}
		}
	}
}
