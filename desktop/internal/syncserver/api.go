package syncserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/importacao"
	"sitposse7/desktop/internal/pareamento"
)

// Limites de corpo das requisições.
const (
	maxCorpoPareamento = 64 << 10
	maxCorpoLote       = 64 << 20
)

// ErroAPI é o corpo de toda resposta de erro. "codigo" é estável (o app decide por ele);
// "erro" é a mensagem para a pessoa.
type ErroAPI struct {
	Codigo string `json:"codigo"`
	Erro   string `json:"erro"`
	Offset *int64 `json:"offset,omitempty"`
}

func (s *Servidor) registrarAPI(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/pareamento", s.parear)
	mux.HandleFunc("PUT /v1/lotes/{lote}", s.autenticado(s.putLote))
	mux.HandleFunc("GET /v1/lotes/{lote}", s.autenticado(s.getLote))
	mux.HandleFunc("POST /v1/lotes/{lote}/importar", s.autenticado(s.importar))
	mux.HandleFunc("POST /v1/lotes/{lote}/ack", s.autenticado(s.ack))
	mux.HandleFunc("PUT /v1/fotos/{foto}", s.autenticado(s.putFoto))
	mux.HandleFunc("GET /v1/fotos/{foto}", s.autenticado(s.getFoto))
}

func (s *Servidor) avisar(evento string) {
	if s.AoReceber != nil {
		go s.AoReceber(evento)
	}
}

// responderErro traduz os erros do núcleo para HTTP.
func responderErro(w http.ResponseWriter, err error) {
	type mapa struct {
		alvo   error
		status int
		codigo string
	}
	for _, m := range []mapa{
		{importacao.ErrInvalido, http.StatusBadRequest, "invalido"},
		{importacao.ErrVersao, http.StatusUnprocessableEntity, "versao_incompativel"},
		{pareamento.ErrNaoAutorizado, http.StatusUnauthorized, "nao_autorizado"},
		{importacao.ErrProibido, http.StatusForbidden, "proibido"},
		{importacao.ErrNaoEncontrado, http.StatusNotFound, "nao_encontrado"},
		{importacao.ErrConflitoLote, http.StatusConflict, "lote_divergente"},
		{pareamento.ErrTokenInvalido, http.StatusGone, "token_invalido"},
		{importacao.ErrLoteCancelado, http.StatusConflict, "lote_cancelado"},
		{importacao.ErrJaDecidido, http.StatusConflict, "ja_decidido"},
		{pareamento.ErrRecusado, http.StatusForbidden, "recusado"},
		// 409 (e não 408): o OkHttp repete sozinho requisições que recebem 408.
		{pareamento.ErrTempoEsgotado, http.StatusConflict, "tempo_esgotado"},
	} {
		if errors.Is(err, m.alvo) {
			escreverJSON(w, m.status, ErroAPI{Codigo: m.codigo, Erro: err.Error()})
			return
		}
	}
	if errors.Is(err, context.Canceled) {
		return
	}
	escreverJSON(w, http.StatusInternalServerError, ErroAPI{Codigo: "erro_interno", Erro: "erro interno na Central: " + err.Error()})
}

func (s *Servidor) autenticado(h func(http.ResponseWriter, *http.Request, importacao.Dispositivo)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cred, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			responderErro(w, pareamento.ErrNaoAutorizado)
			return
		}
		d, err := s.Pareamento.Autenticar(r.Context(), strings.TrimSpace(cred))
		if err != nil {
			responderErro(w, err)
			return
		}
		h(w, r, d)
	}
}

func (s *Servidor) parear(w http.ResponseWriter, r *http.Request) {
	var req contratos.PedidoPareamento
	dec := json.NewDecoder(io.LimitReader(r.Body, maxCorpoPareamento))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		responderErro(w, errors.Join(importacao.ErrInvalido, err))
		return
	}
	s.avisar("pareamento")
	resp, err := s.Pareamento.Pedir(r.Context(), req)
	if err != nil {
		responderErro(w, err)
		return
	}
	s.avisar("pareamento")
	escreverJSON(w, http.StatusOK, resp)
}

func (s *Servidor) putLote(w http.ResponseWriter, r *http.Request, d importacao.Dispositivo) {
	bruto, err := io.ReadAll(io.LimitReader(r.Body, maxCorpoLote+1))
	if err != nil {
		responderErro(w, err)
		return
	}
	if len(bruto) > maxCorpoLote {
		responderErro(w, errors.Join(importacao.ErrInvalido, errors.New("lote maior que 64 MiB")))
		return
	}
	est, err := s.Importador.ReceberLote(r.Context(), d, r.PathValue("lote"), bruto)
	if err != nil {
		responderErro(w, err)
		return
	}
	s.avisar("lote")
	escreverJSON(w, http.StatusOK, est)
}

func (s *Servidor) getLote(w http.ResponseWriter, r *http.Request, d importacao.Dispositivo) {
	est, err := s.Importador.EstadoLote(r.Context(), d, r.PathValue("lote"))
	if err != nil {
		responderErro(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, est)
}

func (s *Servidor) importar(w http.ResponseWriter, r *http.Request, d importacao.Dispositivo) {
	est, err := s.Importador.Importar(r.Context(), d, r.PathValue("lote"))
	if errors.Is(err, importacao.ErrFotosFaltantes) {
		escreverJSON(w, http.StatusConflict, est) // corpo = estado do lote com fotos_faltantes
		return
	}
	if err != nil {
		responderErro(w, err)
		return
	}
	s.avisar("importacao")
	escreverJSON(w, http.StatusOK, est)
}

func (s *Servidor) ack(w http.ResponseWriter, r *http.Request, d importacao.Dispositivo) {
	if err := s.Importador.Ack(r.Context(), d, r.PathValue("lote")); err != nil {
		responderErro(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// EstadoFoto é a resposta de GET/PUT /v1/fotos/{foto}.
type EstadoFoto struct {
	UUID   string `json:"uuid"`
	Offset int64  `json:"offset"`
	Existe bool   `json:"existe"`
}

func (s *Servidor) putFoto(w http.ResponseWriter, r *http.Request, d importacao.Dispositivo) {
	off, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if err != nil || off < 0 {
		responderErro(w, errors.Join(importacao.ErrInvalido, errors.New("offset inválido")))
		return
	}
	foto := r.PathValue("foto")
	novo, err := s.Importador.ReceberBlocoFoto(r.Context(), d, foto, off, r.Body)
	if errors.Is(err, importacao.ErrOffset) {
		escreverJSON(w, http.StatusConflict, ErroAPI{Codigo: "offset", Erro: err.Error(), Offset: &novo})
		return
	}
	if err != nil {
		responderErro(w, err)
		return
	}
	s.avisar("foto")
	escreverJSON(w, http.StatusOK, EstadoFoto{UUID: foto, Offset: novo})
}

func (s *Servidor) getFoto(w http.ResponseWriter, r *http.Request, d importacao.Dispositivo) {
	foto := r.PathValue("foto")
	off, existe, err := s.Importador.EstadoFoto(r.Context(), d, foto)
	if err != nil {
		responderErro(w, err)
		return
	}
	escreverJSON(w, http.StatusOK, EstadoFoto{UUID: foto, Offset: off, Existe: existe})
}
