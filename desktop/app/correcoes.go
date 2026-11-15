package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sitposse7/desktop/internal/correcoes"
)

func (s *ConsultaService) DocumentoEdicao(id string) (string, error) {
	d, e := (&correcoes.Servico{B: s.C.n.B}).Carregar(contexto(), id)
	if e != nil {
		return "", e
	}
	b, e := json.Marshal(d)
	return string(b), e
}
func (s *ConsultaService) SalvarEdicao(id, documento string) error {
	var d correcoes.Documento
	if e := json.Unmarshal([]byte(documento), &d); e != nil {
		return e
	}
	return (&correcoes.Servico{B: s.C.n.B}).Salvar(contexto(), id, &d)
}
func (s *ConsultaService) ExcluirPropriedade(id, token string) error {
	return (&correcoes.Servico{B: s.C.n.B}).Excluir(contexto(), id, token)
}
func (s *ConsultaService) BaixarFoto(id string) (string, error) {
	rel, e := s.C.c.CaminhoFotoVigente(contexto(), id)
	if e != nil {
		return "", e
	}
	if rel == "" || s.C.Arquivos == nil {
		return "", errors.New("foto indisponível")
	}
	dest, e := s.C.Arquivos.EscolherDestino(id+".jpg", "Fotografia JPEG", ".jpg")
	if e != nil || dest == "" {
		return "", e
	}
	b, e := os.ReadFile(filepath.Join(s.C.PastaDados, filepath.FromSlash(rel)))
	if e != nil {
		return "", e
	}
	return dest, GravarAtomico(dest, b)
}
