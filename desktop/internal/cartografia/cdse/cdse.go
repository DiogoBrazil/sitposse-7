// Package cdse é o provedor Sentinel-2 L2A do Copernicus Data Space Ecosystem
// (ADR-0022): OAuth2 client_credentials, Catalog API e Process API.
// Documentação: https://documentation.dataspace.copernicus.eu/APIs/SentinelHub/
package cdse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"sitposse7/desktop/internal/cartografia"
	"sitposse7/desktop/internal/cartografia/geo"
)

// Endereços oficiais (consultados em 14/11/2026).
const (
	URLToken    = "https://identity.dataspace.copernicus.eu/auth/realms/CDSE/protocol/openid-connect/token"
	URLCatalogo = "https://sh.dataspace.copernicus.eu/catalog/v1/search"
	URLProcesso = "https://sh.dataspace.copernicus.eu/process/v1"
	Colecao     = "sentinel-2-l2a"
	MaxPixels   = 2500 // limite por lado do Process API
)

// Evalscript: cores reais (B04, B03, B02) com ganho 2,5 sobre a reflectância e a
// dataMask no canal alfa (0 = sem dados), para detectar imagem incompleta (RN-93).
// Validado contra os exemplos oficiais de S2L2A (true color, ganho 2,5).
const Evalscript = `//VERSION=3
function setup() {
  return { input: ["B02", "B03", "B04", "dataMask"], output: { bands: 4, sampleType: "AUTO" } };
}
function evaluatePixel(s) {
  return [2.5 * s.B04, 2.5 * s.B03, 2.5 * s.B02, s.dataMask];
}`

// VersaoEvalscript entra na chave do cache: mudar o script invalida as imagens guardadas.
const VersaoEvalscript = "tc-2.5-dm-v1"

// Cliente do CDSE. Seguro para uso concorrente.
type Cliente struct {
	HTTP         *http.Client
	ClientID     string
	ClientSecret string
	URLToken     string // vazio = oficial (os testes apontam para um servidor falso)
	URLCatalogo  string
	URLProcesso  string
	Agora        func() time.Time

	mu       sync.Mutex
	token    string
	expiraEm time.Time
}

func (c *Cliente) agora() time.Time {
	if c.Agora != nil {
		return c.Agora()
	}
	return time.Now()
}

func ou(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (c *Cliente) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 90 * time.Second}
}

func (c *Cliente) Nome() string { return "Copernicus Data Space Ecosystem — Sentinel-2 L2A" }

func (c *Cliente) Atribuicao(ano int) string {
	return fmt.Sprintf("Contains modified Copernicus Sentinel data %d (imagem processada: composição em cores reais B04/B03/B02).", ano)
}

// erroDeRede traduz falhas de transporte em ErrSemConexao (RN-91).
func erroDeRede(err error) error {
	var ne net.Error
	var ue *url.Error
	if errors.As(err, &ne) || errors.As(err, &ue) {
		return fmt.Errorf("%w (%v)", cartografia.ErrSemConexao, err)
	}
	return err
}

// erroHTTP traduz o status em erros com significado.
func erroHTTP(r *http.Response, corpo []byte) error {
	switch r.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return cartografia.ErrCredenciaisInvalidas
	case http.StatusTooManyRequests:
		e := &cartografia.ErrCota{}
		if s, err := strconv.Atoi(r.Header.Get("Retry-After")); err == nil {
			e.TentarApos = time.Duration(s) * time.Second
		} else if ms, err := strconv.Atoi(r.Header.Get("Retry-After-Ms")); err == nil {
			e.TentarApos = time.Duration(ms) * time.Millisecond
		}
		return e
	}
	msg := strings.TrimSpace(string(corpo))
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return fmt.Errorf("%w: HTTP %d %s", cartografia.ErrResposta, r.StatusCode, msg)
}

// Token obtém (ou reaproveita) o token de acesso. Fica só em memória.
func (c *Cliente) Token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && c.agora().Before(c.expiraEm) {
		return c.token, nil
	}
	if strings.TrimSpace(c.ClientID) == "" || strings.TrimSpace(c.ClientSecret) == "" {
		return "", cartografia.ErrSemCredenciais
	}
	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {c.ClientID}, "client_secret": {c.ClientSecret}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ou(c.URLToken, URLToken), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http().Do(req)
	if err != nil {
		return "", erroDeRede(err)
	}
	defer resp.Body.Close()
	corpo, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusBadRequest && strings.Contains(string(corpo), "invalid_client") {
		return "", cartografia.ErrCredenciaisInvalidas
	}
	if resp.StatusCode != http.StatusOK {
		return "", erroHTTP(resp, corpo)
	}
	var t struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(corpo, &t); err != nil || t.AccessToken == "" {
		return "", fmt.Errorf("%w: token ilegível", cartografia.ErrResposta)
	}
	if t.ExpiresIn <= 0 {
		t.ExpiresIn = 300
	}
	c.token = t.AccessToken
	// Renova um minuto antes do vencimento (o CDSE limita pedidos de token).
	c.expiraEm = c.agora().Add(time.Duration(t.ExpiresIn)*time.Second - time.Minute)
	return c.token, nil
}

// Esquecer descarta o token (ao trocar ou apagar as credenciais).
func (c *Cliente) Esquecer() {
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
}

func (c *Cliente) post(ctx context.Context, urlDestino string, corpo any, aceitar string) ([]byte, error) {
	tok, err := c.Token(ctx)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(corpo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlDestino, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", aceitar)
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, erroDeRede(err)
	}
	defer resp.Body.Close()
	dados, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, erroDeRede(err) // queda durante o download
	}
	if resp.StatusCode == http.StatusUnauthorized {
		c.Esquecer()
	}
	if resp.StatusCode != http.StatusOK {
		return nil, erroHTTP(resp, dados)
	}
	return dados, nil
}

// bboxGraus: [oeste, sul, leste, norte] em CRS84 (o catálogo usa graus).
func bboxGraus(r geo.Retangulo) []float64 {
	so, ne := r.SO(), r.NE()
	return []float64{so.Lon, so.Lat, ne.Lon, ne.Lat}
}

// Passagens lista os dias com aquisição sobre a área, com a menor nebulosidade do dia,
// do mais recente para o mais antigo.
func (c *Cliente) Passagens(ctx context.Context, r geo.Retangulo, de, ate time.Time) ([]cartografia.Passagem, error) {
	porDia := map[string]*cartografia.Passagem{}
	var next any
	for pagina := 0; pagina < 10; pagina++ {
		corpo := map[string]any{
			"bbox":        bboxGraus(r),
			"datetime":    de.UTC().Format(time.RFC3339) + "/" + ate.UTC().Format(time.RFC3339),
			"collections": []string{Colecao},
			"limit":       100,
			"fields":      map[string]any{"include": []string{"properties.datetime", "properties.eo:cloud_cover"}, "exclude": []string{}},
		}
		if next != nil {
			corpo["next"] = next
		}
		dados, err := c.post(ctx, ou(c.URLCatalogo, URLCatalogo), corpo, "application/geo+json")
		if err != nil {
			return nil, err
		}
		var resp struct {
			Features []struct {
				Properties struct {
					Datetime   time.Time `json:"datetime"`
					CloudCover *float64  `json:"eo:cloud_cover"`
				} `json:"properties"`
			} `json:"features"`
			Context struct {
				Next any `json:"next"`
			} `json:"context"`
		}
		if err := json.Unmarshal(dados, &resp); err != nil {
			return nil, fmt.Errorf("%w: catálogo ilegível", cartografia.ErrResposta)
		}
		for _, f := range resp.Features {
			d := f.Properties.Datetime.UTC()
			chave := d.Format("2006-01-02")
			nuv := 100.0
			if f.Properties.CloudCover != nil {
				nuv = *f.Properties.CloudCover
			}
			p, ok := porDia[chave]
			if !ok {
				dia := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
				p = &cartografia.Passagem{Data: dia, Nuvens: nuv}
				porDia[chave] = p
			}
			p.Tiles++
			if nuv < p.Nuvens {
				p.Nuvens = nuv
			}
		}
		if resp.Context.Next == nil || len(resp.Features) == 0 {
			break
		}
		next = resp.Context.Next
	}
	out := make([]cartografia.Passagem, 0, len(porDia))
	for _, p := range porDia {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Data.After(out[j].Data) })
	return out, nil
}

// Sugerir escolhe a passagem: a de menos nuvens entre as dos últimos 45 dias com até
// 20% de nuvens; senão, a de menos nuvens de todas. A mais recente não é presumida a melhor.
func Sugerir(ps []cartografia.Passagem, agora time.Time) *cartografia.Passagem {
	var melhor *cartografia.Passagem
	for i := range ps {
		p := &ps[i]
		if agora.Sub(p.Data) <= 45*24*time.Hour && p.Nuvens <= 20 && (melhor == nil || p.Nuvens < melhor.Nuvens) {
			melhor = p
		}
	}
	if melhor != nil {
		return melhor
	}
	for i := range ps {
		if melhor == nil || ps[i].Nuvens < melhor.Nuvens || (ps[i].Nuvens == melhor.Nuvens && ps[i].Data.After(melhor.Data)) {
			melhor = &ps[i]
		}
	}
	return melhor
}

// CorpoProcesso monta o pedido do Process API (exportado para teste).
func CorpoProcesso(p cartografia.PedidoImagem) map[string]any {
	de, ate := p.De, p.Ate
	if p.Data != nil {
		d := p.Data.UTC()
		de = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		ate = de.Add(24*time.Hour - time.Second)
	}
	filtro := map[string]any{
		"timeRange":       map[string]string{"from": de.UTC().Format(time.RFC3339), "to": ate.UTC().Format(time.RFC3339)},
		"mosaickingOrder": "leastCC",
	}
	if p.MaxNuvens != nil {
		filtro["maxCloudCoverage"] = *p.MaxNuvens
	}
	return map[string]any{
		"input": map[string]any{
			"bounds": map[string]any{
				"bbox":       []float64{p.Ret.MinX, p.Ret.MinY, p.Ret.MaxX, p.Ret.MaxY},
				"properties": map[string]string{"crs": "http://www.opengis.net/def/crs/EPSG/0/3857"},
			},
			"data": []any{map[string]any{
				"type":       Colecao,
				"dataFilter": filtro,
				"processing": map[string]string{"upsampling": "BICUBIC", "downsampling": "BICUBIC"},
			}},
		},
		"output": map[string]any{
			"width": p.Largura, "height": p.Altura,
			"responses": []any{map[string]any{"identifier": "default", "format": map[string]string{"type": "image/png"}}},
		},
		"evalscript": Evalscript,
	}
}

// FracaoSemDados conta os pixels transparentes (dataMask = 0) de um PNG.
func FracaoSemDados(pngBytes []byte) (float64, image.Image, error) {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("%w: imagem ilegível", cartografia.ErrResposta)
	}
	b := img.Bounds()
	vazios := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a == 0 {
				vazios++
			}
		}
	}
	return float64(vazios) / float64(b.Dx()*b.Dy()), img, nil
}

// Imagem pede a imagem ao Process API e confere tamanho e cobertura.
func (c *Cliente) Imagem(ctx context.Context, p cartografia.PedidoImagem) (*cartografia.Imagem, error) {
	if p.Largura <= 0 || p.Altura <= 0 || p.Largura > MaxPixels || p.Altura > MaxPixels {
		return nil, fmt.Errorf("tamanho de imagem inválido (%d × %d; máximo %d por lado)", p.Largura, p.Altura, MaxPixels)
	}
	dados, err := c.post(ctx, ou(c.URLProcesso, URLProcesso), CorpoProcesso(p), "image/png")
	if err != nil {
		return nil, err
	}
	semDados, img, err := FracaoSemDados(dados)
	if err != nil {
		return nil, err
	}
	if img.Bounds().Dx() != p.Largura || img.Bounds().Dy() != p.Altura {
		return nil, fmt.Errorf("%w: imagem de %d × %d (pedido %d × %d)", cartografia.ErrResposta, img.Bounds().Dx(), img.Bounds().Dy(), p.Largura, p.Altura)
	}
	return &cartografia.Imagem{
		PNG: dados, Ret: p.Ret, Largura: p.Largura, Altura: p.Altura, Data: p.Data, De: p.De, Ate: p.Ate, SemDados: semDados,
		Fonte: c.Nome(), Colecao: Colecao, ResolucaoNativaM: 10, ObtidaEm: c.agora(),
	}, nil
}
