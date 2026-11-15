package cog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sitposse7/desktop/internal/cartografia"
)

// Cache guarda no disco as faixas já baixadas, pela URL, pela versão do arquivo (ETag) e
// pela posição. Os arquivos do INPE são imutáveis; se o servidor trocar o arquivo, o ETag
// muda e nada antigo é reaproveitado (ADR-0027).
type Cache struct{ Dir string }

func hashHex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (c *Cache) caminho(chave string) string {
	h := hashHex(chave)
	return filepath.Join(c.Dir, h[:2], h[2:]+".bin")
}

func (c *Cache) ler(chave string) ([]byte, bool) {
	if c == nil || c.Dir == "" {
		return nil, false
	}
	b, err := os.ReadFile(c.caminho(chave))
	return b, err == nil
}

func (c *Cache) gravar(chave string, dados []byte) {
	if c == nil || c.Dir == "" {
		return
	}
	destino := c.caminho(chave)
	if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
		return
	}
	tmp, err := os.CreateTemp(filepath.Dir(destino), ".tmp-*")
	if err != nil {
		return
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(dados); err != nil {
		tmp.Close()
		return
	}
	if tmp.Sync() != nil || tmp.Close() != nil {
		return
	}
	_ = os.Rename(tmp.Name(), destino)
}

// HTTP lê faixas de um arquivo público por HTTP Range, com cache em disco e novas
// tentativas. Recusa servidor que ignora Range (nunca baixa o arquivo inteiro).
type HTTP struct {
	URL        string
	Cliente    *http.Client
	Cache      *Cache
	Tentativas int                 // padrão 3
	Esperar    func(time.Duration) // substituível nos testes

	mu   sync.Mutex
	etag string

	Baixados atomic.Int64 // bytes trazidos da rede
	DoCache  atomic.Int64 // bytes servidos pelo cache
	Pedidos  atomic.Int64 // requisições feitas
	SemRede  atomic.Bool  // a última tentativa de rede falhou por conexão
}

func (h *HTTP) cliente() *http.Client {
	if h.Cliente != nil {
		return h.Cliente
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (h *HTTP) chave(etag string, off, n int64) string {
	return fmt.Sprintf("%s|%s|%d|%d", h.URL, etag, off, n)
}

func (h *HTTP) chaveVersao() string { return h.URL + "|versao" }

func (h *HTTP) versao() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.etag
}

// ErrArquivoMudou: o servidor passou a servir outra versão do arquivo durante a leitura.
var ErrArquivoMudou = errors.New("o arquivo de imagem foi atualizado no servidor durante a leitura; tente de novo")

// LerFaixa implementa Faixas.
func (h *HTTP) LerFaixa(ctx context.Context, off, n int64) ([]byte, error) {
	if n <= 0 {
		return nil, errors.New("faixa vazia")
	}
	if v := h.versao(); v != "" {
		if b, ok := h.Cache.ler(h.chave(v, off, n)); ok {
			h.DoCache.Add(int64(len(b)))
			return b, nil
		}
	}
	dados, etag, err := h.baixar(ctx, off, n)
	if err != nil {
		// Sem rede: usa a última versão conhecida do arquivo, se as faixas estiverem guardadas.
		if errors.Is(err, cartografia.ErrSemConexao) && h.versao() == "" {
			if v, ok := h.Cache.ler(h.chaveVersao()); ok {
				if b, ok := h.Cache.ler(h.chave(string(v), off, n)); ok {
					h.mu.Lock()
					h.etag = string(v)
					h.mu.Unlock()
					h.DoCache.Add(int64(len(b)))
					return b, nil
				}
			}
		}
		return nil, err
	}
	h.mu.Lock()
	if h.etag == "" {
		h.etag = etag
		h.mu.Unlock()
		h.Cache.gravar(h.chaveVersao(), []byte(etag))
	} else {
		mudou := etag != h.etag
		h.mu.Unlock()
		if mudou {
			return nil, ErrArquivoMudou
		}
	}
	h.Cache.gravar(h.chave(etag, off, n), dados)
	return dados, nil
}

func (h *HTTP) esperar(ctx context.Context, d time.Duration) error {
	if h.Esperar != nil {
		h.Esperar(d)
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (h *HTTP) baixar(ctx context.Context, off, n int64) ([]byte, string, error) {
	tent := h.Tentativas
	if tent <= 0 {
		tent = 3
	}
	var ultimo error
	espera := 500 * time.Millisecond
	for i := 0; i < tent; i++ {
		if i > 0 {
			if err := h.esperar(ctx, espera); err != nil {
				return nil, "", err
			}
			espera *= 2
		}
		dados, etag, repetir, err := h.umaVez(ctx, off, n)
		if err == nil {
			h.SemRede.Store(false)
			return dados, etag, nil
		}
		ultimo = err
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		if !repetir {
			break
		}
	}
	if errors.Is(ultimo, cartografia.ErrSemConexao) {
		h.SemRede.Store(true)
	}
	return nil, "", ultimo
}

func (h *HTTP) umaVez(ctx context.Context, off, n int64) (dados []byte, etag string, repetir bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.URL, nil)
	if err != nil {
		return nil, "", false, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, off+n-1))
	h.Pedidos.Add(1)
	resp, err := h.cliente().Do(req)
	if err != nil {
		var ne net.Error
		var ue *url.Error
		if errors.As(err, &ne) || errors.As(err, &ue) {
			return nil, "", true, fmt.Errorf("%w (%v)", cartografia.ErrSemConexao, err)
		}
		return nil, "", true, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusPartialContent:
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500:
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		return nil, "", true, fmt.Errorf("%w: HTTP %d ao ler a imagem", cartografia.ErrResposta, resp.StatusCode)
	case resp.StatusCode == http.StatusOK:
		return nil, "", false, fmt.Errorf("%w: o servidor de imagens não aceita leitura por faixa", cartografia.ErrResposta)
	default:
		return nil, "", false, fmt.Errorf("%w: HTTP %d ao ler a imagem", cartografia.ErrResposta, resp.StatusCode)
	}
	// Content-Range: bytes ini-fim/total
	cr := resp.Header.Get("Content-Range")
	var ini, fim int64
	if _, err := fmt.Sscanf(strings.TrimPrefix(cr, "bytes "), "%d-%d", &ini, &fim); err != nil || ini != off || fim > off+n-1 {
		return nil, "", false, fmt.Errorf("%w: faixa devolvida diferente da pedida (%q)", cartografia.ErrResposta, cr)
	}
	esperado := fim - ini + 1
	dados = make([]byte, esperado)
	if _, err := io.ReadFull(resp.Body, dados); err != nil {
		return nil, "", true, fmt.Errorf("%w (download interrompido: %v)", cartografia.ErrSemConexao, err)
	}
	h.Baixados.Add(esperado)
	etag = resp.Header.Get("ETag")
	if etag == "" {
		etag = resp.Header.Get("Last-Modified")
	}
	if etag == "" {
		etag = "sem-versao"
	}
	return dados, etag, false, nil // pode ser menor que n no fim do arquivo
}
