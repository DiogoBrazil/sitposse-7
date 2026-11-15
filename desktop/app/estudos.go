package app

import (
	"database/sql"
	"errors"
	"time"

	"sitposse7/desktop/internal/estudos"
)

// EstudoService: estudos de situação, jornadas e equipes (RF-C01, ADR-0023).
type EstudoService struct{ C *Central }

func (s *EstudoService) srv() *estudos.Servico { return s.C.n.Estudos }

// Listar devolve os estudos, do mais recente para o mais antigo.
func (s *EstudoService) Listar() ([]estudos.Estudo, error) { return s.srv().Listar(contexto()) }

// Detalhe devolve o estudo com jornadas e equipes (códigos incluídos).
func (s *EstudoService) Detalhe(estudo string) (*estudos.EstudoDetalhe, error) {
	return s.srv().Detalhe(contexto(), estudo)
}

// Criar cria o estudo com a primeira jornada (data AAAA-MM-DD) e as equipes informadas,
// e o põe em foco.
func (s *EstudoService) Criar(nome, data string, equipes []string) (*estudos.EstudoDetalhe, error) {
	e, err := s.srv().CriarEstudo(contexto(), nome, data, equipes)
	if err != nil {
		return nil, err
	}
	return e, s.DefinirEmFoco(e.UUID)
}

// Renomear troca o nome do estudo.
func (s *EstudoService) Renomear(estudo, nome string) error {
	return s.srv().RenomearEstudo(contexto(), estudo, nome)
}

// CriarJornada acrescenta uma jornada com as equipes (nomes podem repetir os de outras jornadas;
// cada equipe recebe código novo).
func (s *EstudoService) CriarJornada(estudo, data string, equipes []string) (*estudos.Jornada, error) {
	return s.srv().CriarJornada(contexto(), estudo, data, equipes)
}

// AlterarDataJornada corrige a data da jornada.
func (s *EstudoService) AlterarDataJornada(jornada, data string) error {
	return s.srv().AlterarDataJornada(contexto(), jornada, data)
}

// ExcluirJornada exclui a jornada (só sem cadastros recebidos).
func (s *EstudoService) ExcluirJornada(jornada string) error {
	return s.srv().ExcluirJornada(contexto(), jornada)
}

// AdicionarEquipes cria equipes numa jornada.
func (s *EstudoService) AdicionarEquipes(jornada string, nomes []string) ([]estudos.Equipe, error) {
	return s.srv().AdicionarEquipes(contexto(), jornada, nomes)
}

// RenomearEquipe troca o nome (o código não muda).
func (s *EstudoService) RenomearEquipe(equipe, nome string) error {
	return s.srv().RenomearEquipe(contexto(), equipe, nome)
}

// ExcluirEquipe exclui a equipe (só sem cadastros recebidos).
func (s *EstudoService) ExcluirEquipe(equipe string) error {
	return s.srv().ExcluirEquipe(contexto(), equipe)
}

// EmFoco devolve o estudo em foco guardado ("" se nenhum ou se foi excluído).
// Sem escolha guardada, usa o estudo mais recente.
func (s *EstudoService) EmFoco() (string, error) {
	var v string
	err := s.C.n.B.Leitura.QueryRowContext(contexto(), `SELECT m.valor FROM meta m JOIN estudos e ON e.uuid = m.valor
		WHERE m.chave = 'estudo_em_foco' AND e.excluido_em IS NULL`).Scan(&v)
	if err == nil {
		return v, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	err = s.C.n.B.Leitura.QueryRowContext(contexto(), `SELECT uuid FROM estudos WHERE excluido_em IS NULL ORDER BY created_at DESC LIMIT 1`).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// DefinirEmFoco guarda o estudo em foco (preferência da tela, não dado de negócio).
func (s *EstudoService) DefinirEmFoco(estudo string) error {
	agora := time.Now().UnixMilli()
	_, err := s.C.n.B.Escrita.ExecContext(contexto(), `INSERT INTO meta (chave, valor, created_at, updated_at) VALUES ('estudo_em_foco', ?, ?, ?)
		ON CONFLICT (chave) DO UPDATE SET valor = excluded.valor, updated_at = excluded.updated_at`, estudo, agora, agora)
	return err
}
