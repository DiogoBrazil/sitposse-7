// Package stac é o cliente mínimo de busca STAC (API "item search") usado para encontrar
// cenas de satélite sobre a área das propriedades (ADR-0027). Só envia o retângulo, as
// coleções e o período: nenhum dado de cadastro (AGENTS.md §8).
package stac

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"sitposse7/desktop/internal/cartografia"
)

// Busca: parâmetros da pesquisa.
type Busca struct {
	Colecoes []string
	BBox     [4]float64 // oeste, sul, leste, norte (graus)
	De, Ate  time.Time
	Limite   int // itens por página (padrão 100)
	MaxItens int // teto total (padrão 500)
}

// Ativo de um item (arquivo).
type Ativo struct {
	Href string `json:"href"`
	Tipo string `json:"type"`
}

// Item é uma cena do catálogo.
type Item struct {
	ID       string
	Colecao  string
	Data     time.Time
	Inicio   time.Time // start_datetime (mosaicos)
	Fim      time.Time // end_datetime (mosaicos)
	Nuvens   *float64  // eo:cloud_cover da cena inteira (pode faltar)
	BBox     [4]float64
	Poligono [][2]float64 // anel externo (lon, lat), se a geometria for polígono
	Ativos   map[string]Ativo
}

// Cliente STAC.
type Cliente struct {
	URL        string // endpoint raiz, ex.: https://data.inpe.br/bdc/stac/v1
	HTTP       *http.Client
	Tentativas int
	Esperar    func(time.Duration)
}

func (c *Cliente) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

type link struct {
	Rel    string         `json:"rel"`
	Href   string         `json:"href"`
	Method string         `json:"method"`
	Body   map[string]any `json:"body"`
	Merge  bool           `json:"merge"`
}

type resposta struct {
	Features []struct {
		ID         string                     `json:"id"`
		Collection string                     `json:"collection"`
		BBox       []float64                  `json:"bbox"`
		Geometry   json.RawMessage            `json:"geometry"`
		Properties map[string]json.RawMessage `json:"properties"`
		Assets     map[string]Ativo           `json:"assets"`
	} `json:"features"`
	Links []link `json:"links"`
}

// Buscar pesquisa e percorre as páginas (links rel=next, GET ou POST).
func (c *Cliente) Buscar(ctx context.Context, b Busca) ([]Item, error) {
	if b.Limite <= 0 {
		b.Limite = 100
	}
	if b.MaxItens <= 0 {
		b.MaxItens = 500
	}
	corpo := map[string]any{
		"collections": b.Colecoes,
		"bbox":        b.BBox[:],
		"limit":       b.Limite,
	}
	if !b.De.IsZero() || !b.Ate.IsZero() {
		de, ate := "..", ".."
		if !b.De.IsZero() {
			de = b.De.UTC().Format(time.RFC3339)
		}
		if !b.Ate.IsZero() {
			ate = b.Ate.UTC().Format(time.RFC3339)
		}
		corpo["datetime"] = de + "/" + ate
	}
	metodo, endereco := http.MethodPost, c.URL+"/search"
	var out []Item
	for pagina := 0; pagina < 50 && len(out) < b.MaxItens; pagina++ {
		r, err := c.pedir(ctx, metodo, endereco, corpo)
		if err != nil {
			return nil, err
		}
		for _, f := range r.Features {
			it := Item{ID: f.ID, Colecao: f.Collection, Ativos: f.Assets}
			if len(f.BBox) >= 4 {
				copy(it.BBox[:], f.BBox[:4])
			}
			it.Data = tempo(f.Properties["datetime"])
			it.Inicio, it.Fim = tempo(f.Properties["start_datetime"]), tempo(f.Properties["end_datetime"])
			for _, k := range []string{"eo:cloud_cover", "cloud_cover"} {
				var v *float64
				if raw, ok := f.Properties[k]; ok && json.Unmarshal(raw, &v) == nil && v != nil {
					it.Nuvens = v
					break
				}
			}
			it.Poligono = anel(f.Geometry)
			out = append(out, it)
		}
		var prox *link
		for i := range r.Links {
			if r.Links[i].Rel == "next" {
				prox = &r.Links[i]
			}
		}
		if prox == nil || len(r.Features) == 0 {
			break
		}
		endereco = prox.Href
		if prox.Method == http.MethodPost || (prox.Method == "" && prox.Body != nil) {
			metodo = http.MethodPost
			if prox.Merge {
				for k, v := range prox.Body {
					corpo[k] = v
				}
			} else if prox.Body != nil {
				corpo = prox.Body
			}
		} else {
			metodo, corpo = http.MethodGet, nil
		}
	}
	if len(out) > b.MaxItens {
		out = out[:b.MaxItens]
	}
	return out, nil
}

func tempo(raw json.RawMessage) time.Time {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}

func anel(raw json.RawMessage) [][2]float64 {
	var g struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}
	if json.Unmarshal(raw, &g) != nil {
		return nil
	}
	switch g.Type {
	case "Polygon":
		var c [][][2]float64
		if json.Unmarshal(g.Coordinates, &c) == nil && len(c) > 0 {
			return c[0]
		}
	case "MultiPolygon":
		var c [][][][2]float64
		if json.Unmarshal(g.Coordinates, &c) == nil && len(c) > 0 && len(c[0]) > 0 {
			return c[0][0] // o primeiro polígono basta para a pré-seleção; a leitura confere os dados
		}
	}
	return nil
}

func (c *Cliente) pedir(ctx context.Context, metodo, endereco string, corpo map[string]any) (*resposta, error) {
	tent := c.Tentativas
	if tent <= 0 {
		tent = 3
	}
	var ultimo error
	espera := time.Second
	for i := 0; i < tent; i++ {
		if i > 0 {
			if c.Esperar != nil {
				c.Esperar(espera)
			} else {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(espera):
				}
			}
			espera *= 2
		}
		r, repetir, err := c.umaVez(ctx, metodo, endereco, corpo)
		if err == nil {
			return r, nil
		}
		ultimo = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !repetir {
			break
		}
	}
	return nil, ultimo
}

func (c *Cliente) umaVez(ctx context.Context, metodo, endereco string, corpo map[string]any) (*resposta, bool, error) {
	var leitor io.Reader
	if metodo == http.MethodPost {
		b, _ := json.Marshal(corpo)
		leitor = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, metodo, endereco, leitor)
	if err != nil {
		return nil, false, err
	}
	if metodo == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/geo+json, application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		var ne net.Error
		var ue *url.Error
		if errors.As(err, &ne) || errors.As(err, &ue) {
			return nil, true, fmt.Errorf("%w (%v)", cartografia.ErrSemConexao, err)
		}
		return nil, true, err
	}
	defer resp.Body.Close()
	dados, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, true, fmt.Errorf("%w (%v)", cartografia.ErrSemConexao, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("%w: catálogo de imagens respondeu HTTP %d", cartografia.ErrResposta, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("%w: catálogo de imagens respondeu HTTP %d", cartografia.ErrResposta, resp.StatusCode)
	}
	var r resposta
	if err := json.Unmarshal(dados, &r); err != nil {
		return nil, false, fmt.Errorf("%w: catálogo ilegível", cartografia.ErrResposta)
	}
	return &r, false, nil
}
