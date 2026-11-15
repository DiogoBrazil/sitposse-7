package stac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"sitposse7/desktop/internal/cartografia"
)

// Servidor falso no formato do STAC do INPE (paginação POST com body + merge, verificada em 15/11/2026).
func servidor(t *testing.T, total int, falhas int32) (*httptest.Server, *[]map[string]any) {
	var pedidos []map[string]any
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) <= falhas {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var corpo map[string]any
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &corpo)
		pedidos = append(pedidos, corpo)
		pagina := 1
		if p, ok := corpo["page"].(float64); ok {
			pagina = int(p)
		}
		lim := int(corpo["limit"].(float64))
		var feats []any
		for i := (pagina - 1) * lim; i < pagina*lim && i < total; i++ {
			props := map[string]any{"datetime": fmt.Sprintf("2026-%02d-01T00:00:00.000000Z", i%12+1)}
			if i%2 == 0 {
				props["eo:cloud_cover"] = float64(i)
			} else {
				props["eo:cloud_cover"] = nil
			}
			feats = append(feats, map[string]any{"id": fmt.Sprintf("cena-%03d", i), "collection": "CB4A-WPM-PCA-FUSED-1", "bbox": []float64{-63.6, -10.8, -62.6, -9.8},
				"geometry":   map[string]any{"type": "Polygon", "coordinates": [][][]float64{{{-63.6, -10.8}, {-63.6, -9.8}, {-62.6, -9.8}, {-62.6, -10.8}, {-63.6, -10.8}}}},
				"properties": props, "assets": map[string]any{"tci": map[string]any{"href": "https://x/c.tif", "type": "image/tiff"}}})
		}
		links := []any{}
		if pagina*lim < total {
			nb := map[string]any{"page": pagina + 1}
			links = append(links, map[string]any{"rel": "next", "href": "http://" + r.Host + "/search", "method": "POST", "body": nb, "merge": true})
		}
		json.NewEncoder(w).Encode(map[string]any{"features": feats, "links": links})
	}))
	t.Cleanup(srv.Close)
	return srv, &pedidos
}

func TestBuscarPaginacaoEMetadados(t *testing.T) {
	srv, pedidos := servidor(t, 23, 0)
	c := &Cliente{URL: srv.URL}
	its, err := c.Buscar(context.Background(), Busca{Colecoes: []string{"CB4A-WPM-PCA-FUSED-1"}, BBox: [4]float64{-63.2, -10.1, -63.1, -10.0},
		De: time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC), Limite: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(its) != 23 || len(*pedidos) != 3 {
		t.Fatalf("%d itens em %d páginas", len(its), len(*pedidos))
	}
	// O merge mantém coleções, bbox e período nas páginas seguintes.
	p3 := (*pedidos)[2]
	if p3["datetime"] != "2025-11-01T00:00:00Z/.." || p3["collections"] == nil || p3["bbox"] == nil {
		t.Fatalf("pedido da 3ª página: %v", p3)
	}
	if its[0].Nuvens == nil || its[1].Nuvens != nil || its[0].Data.Month() != 1 || len(its[0].Poligono) != 5 || its[0].Ativos["tci"].Href == "" {
		t.Fatalf("item: %+v", its[0])
	}
}

func TestBuscarSemCenasFalhaERepeticao(t *testing.T) {
	srv, _ := servidor(t, 0, 0)
	its, err := (&Cliente{URL: srv.URL}).Buscar(context.Background(), Busca{Colecoes: []string{"x"}})
	if err != nil || len(its) != 0 {
		t.Fatal(its, err)
	}
	instavel, _ := servidor(t, 3, 2)
	its, err = (&Cliente{URL: instavel.URL, Esperar: func(time.Duration) {}}).Buscar(context.Background(), Busca{Colecoes: []string{"x"}})
	if err != nil || len(its) != 3 {
		t.Fatal("não repetiu após 503", err)
	}
	fora, _ := servidor(t, 3, 100)
	if _, err := (&Cliente{URL: fora.URL, Esperar: func(time.Duration) {}}).Buscar(context.Background(), Busca{Colecoes: []string{"x"}}); !errors.Is(err, cartografia.ErrResposta) {
		t.Fatal(err)
	}
	if _, err := (&Cliente{URL: "http://127.0.0.1:1", Esperar: func(time.Duration) {}}).Buscar(context.Background(), Busca{}); !errors.Is(err, cartografia.ErrSemConexao) {
		t.Fatal("sem conexão:", err)
	}
}
