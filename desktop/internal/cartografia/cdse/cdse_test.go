package cdse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/geo"
)

func pngTeste(t *testing.T, w, h int, transparentes int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := uint8(255)
			if n < transparentes {
				a = 0
			}
			n++
			img.Set(x, y, color.NRGBA{40, 90, 50, a})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

type falso struct {
	tokens    atomic.Int32
	corpos    []string
	process   func(w http.ResponseWriter, r *http.Request)
	tokenResp func(w http.ResponseWriter)
}

func novoFalso(t *testing.T) (*falso, *httptest.Server, *Cliente) {
	f := &falso{}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		f.tokens.Add(1)
		_ = r.ParseForm()
		if f.tokenResp != nil {
			f.tokenResp(w)
			return
		}
		if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_secret") != "segredo" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"tok-1","expires_in":600}`))
	})
	auth := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer tok-1" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			b, _ := io.ReadAll(r.Body)
			f.corpos = append(f.corpos, string(b))
			h(w, r)
		}
	}
	mux.HandleFunc("/catalog", auth(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"features":[
			{"properties":{"datetime":"2026-11-02T14:05:01Z","eo:cloud_cover":62.5}},
			{"properties":{"datetime":"2026-11-02T14:05:09Z","eo:cloud_cover":40.1}},
			{"properties":{"datetime":"2026-10-28T14:05:00Z","eo:cloud_cover":8.0}},
			{"properties":{"datetime":"2026-09-10T14:05:00Z","eo:cloud_cover":1.2}}],"context":{}}`))
	}))
	mux.HandleFunc("/process", auth(func(w http.ResponseWriter, r *http.Request) {
		if f.process != nil {
			f.process(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngTeste(t, 40, 30, 0))
	}))
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	c := &Cliente{ClientID: "cliente-sitposse", ClientSecret: "segredo", URLToken: s.URL + "/token", URLCatalogo: s.URL + "/catalog", URLProcesso: s.URL + "/process"}
	return f, s, c
}

func ret() geo.Retangulo {
	r, _ := geo.Enquadrar([]geo.Ponto{{Lat: -10.43, Lon: -62.47}}, 500, 1500, 40, 30)
	return r
}

func TestPassagensAgrupaPorDiaEReaproveitaToken(t *testing.T) {
	f, _, c := novoFalso(t)
	ctx := context.Background()
	ps, err := c.Passagens(ctx, ret(), time.Now().Add(-90*24*time.Hour), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 3 || ps[0].Data.Format("2006-01-02") != "2026-11-02" || ps[0].Nuvens != 40.1 || ps[0].Tiles != 2 {
		t.Fatalf("passagens = %+v", ps)
	}
	if _, err := c.Imagem(ctx, cartografia.PedidoImagem{Ret: ret(), Largura: 40, Altura: 30, Data: &ps[1].Data}); err != nil {
		t.Fatal(err)
	}
	if n := f.tokens.Load(); n != 1 {
		t.Errorf("pediu %d tokens; o token deve ser reaproveitado", n)
	}
	// Mais nuvens recentes não vencem: a sugestão é a de menos nuvens recente (28/10, 8%).
	s := Sugerir(ps, time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC))
	if s == nil || s.Data.Format("2006-01-02") != "2026-10-28" {
		t.Errorf("sugestão = %+v", s)
	}
	// Sem nada recente e limpo, a de menos nuvens de todas.
	s = Sugerir(ps, time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC))
	if s.Data.Format("2006-01-02") != "2026-09-10" {
		t.Errorf("sugestão antiga = %+v", s)
	}
}

// AGENTS.md §8: ao provedor vão só retângulo, período e parâmetros da imagem.
func TestNadaDeCadastroSaiParaAInternet(t *testing.T) {
	f, _, c := novoFalso(t)
	d := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	if _, err := c.Imagem(context.Background(), cartografia.PedidoImagem{Ret: ret(), Largura: 40, Altura: 30, Data: &d}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Passagens(context.Background(), ret(), d.Add(-24*time.Hour), d); err != nil {
		t.Fatal(err)
	}
	permitidas := map[string]bool{"input": true, "output": true, "evalscript": true, "bbox": true, "datetime": true, "collections": true, "limit": true, "fields": true}
	for _, corpo := range f.corpos {
		var m map[string]any
		if err := json.Unmarshal([]byte(corpo), &m); err != nil {
			t.Fatal(err)
		}
		for k := range m {
			if !permitidas[k] {
				t.Errorf("campo inesperado enviado ao provedor: %q", k)
			}
		}
	}
	// O pedido de imagem vai em EPSG:3857, com a mesma caixa da prévia.
	if !strings.Contains(f.corpos[0], "EPSG/0/3857") || !strings.Contains(f.corpos[0], `"leastCC"`) {
		t.Errorf("pedido de imagem: %s", f.corpos[0])
	}
}

func TestPassagemUnicaUsaOProprioDia(t *testing.T) {
	d := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	c := CorpoProcesso(cartografia.PedidoImagem{Ret: ret(), Largura: 10, Altura: 10, Data: &d})
	tr := c["input"].(map[string]any)["data"].([]any)[0].(map[string]any)["dataFilter"].(map[string]any)["timeRange"].(map[string]string)
	if tr["from"] != "2026-10-28T00:00:00Z" || tr["to"] != "2026-10-28T23:59:59Z" {
		t.Errorf("timeRange = %v", tr)
	}
}

func TestErros(t *testing.T) {
	ctx := context.Background()
	d := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	ped := cartografia.PedidoImagem{Ret: ret(), Largura: 40, Altura: 30, Data: &d}

	t.Run("sem credenciais", func(t *testing.T) {
		_, _, c := novoFalso(t)
		c.ClientSecret = ""
		if _, err := c.Imagem(ctx, ped); !errors.Is(err, cartografia.ErrSemCredenciais) {
			t.Errorf("erro = %v", err)
		}
	})
	t.Run("credenciais recusadas", func(t *testing.T) {
		_, _, c := novoFalso(t)
		c.ClientSecret = "errado"
		if _, err := c.Imagem(ctx, ped); !errors.Is(err, cartografia.ErrCredenciaisInvalidas) {
			t.Errorf("erro = %v", err)
		}
	})
	t.Run("cota", func(t *testing.T) {
		f, _, c := novoFalso(t)
		f.process = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "42")
			w.WriteHeader(http.StatusTooManyRequests)
		}
		_, err := c.Imagem(ctx, ped)
		var cota *cartografia.ErrCota
		if !errors.As(err, &cota) || cota.TentarApos != 42*time.Second {
			t.Errorf("erro = %v", err)
		}
	})
	t.Run("sem internet", func(t *testing.T) {
		_, s, c := novoFalso(t)
		s.Close()
		if _, err := c.Imagem(ctx, ped); !errors.Is(err, cartografia.ErrSemConexao) {
			t.Errorf("erro = %v", err)
		}
	})
	t.Run("queda durante o download", func(t *testing.T) {
		f, _, c := novoFalso(t)
		f.process = func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "100000")
			_, _ = w.Write([]byte("\x89PNG parcial"))
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
		}
		_, err := c.Imagem(ctx, ped)
		if err == nil {
			t.Fatal("download incompleto aceito")
		}
	})
	t.Run("resposta ilegível", func(t *testing.T) {
		f, _, c := novoFalso(t)
		f.process = func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("não é png")) }
		if _, err := c.Imagem(ctx, ped); !errors.Is(err, cartografia.ErrResposta) {
			t.Errorf("erro = %v", err)
		}
	})
	t.Run("token expira e é renovado", func(t *testing.T) {
		f, _, c := novoFalso(t)
		agora := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
		c.Agora = func() time.Time { return agora }
		_, _ = c.Token(ctx)
		agora = agora.Add(20 * time.Minute)
		_, _ = c.Token(ctx)
		if f.tokens.Load() != 2 {
			t.Errorf("tokens = %d; esperado renovar após expirar", f.tokens.Load())
		}
	})
	t.Run("imagem com área sem dados", func(t *testing.T) {
		f, _, c := novoFalso(t)
		f.process = func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(pngTeste(t, 40, 30, 300)) }
		img, err := c.Imagem(ctx, ped)
		if err != nil {
			t.Fatal(err)
		}
		if img.SemDados < 0.24 || img.SemDados > 0.26 {
			t.Errorf("sem dados = %.3f; esperado 0,25", img.SemDados)
		}
	})
	t.Run("tamanho acima do limite", func(t *testing.T) {
		_, _, c := novoFalso(t)
		p := ped
		p.Largura = 3000
		if _, err := c.Imagem(ctx, p); err == nil {
			t.Error("aceitou 3000 px")
		}
	})
}
