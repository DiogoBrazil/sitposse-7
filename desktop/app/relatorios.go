package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sitposse7/desktop/internal/estatisticas"
	"sitposse7/desktop/internal/estudos"
	"sitposse7/desktop/internal/relatorios"
)

// Arquivos são as janelas do sistema (salvar, abrir, mostrar na pasta), fornecidas pela
// casca Wails (main.go); a camada app não importa o Wails para continuar testável.
type Arquivos interface {
	// EscolherDestino abre o "Salvar como"; devolve "" se a pessoa cancelar.
	EscolherDestino(nomeSugerido, descricao, extensao string) (string, error)
	Abrir(caminho string) error
	MostrarNaPasta(caminho string) error
}

// RelatorioService exporta relatórios em PDF, DOCX e XLSX (A26, ADR-0021; RF-C07).
type RelatorioService struct{ C *Central }

var formatos = map[string]struct{ ext, descr string }{
	"pdf":  {".pdf", "Documento PDF"},
	"docx": {".docx", "Documento do Word"},
	"xlsx": {".xlsx", "Planilha do Excel"},
}

// ErrArquivoEmUso: o arquivo de destino está aberto em outro programa.
var ErrArquivoEmUso = errors.New("não foi possível gravar: o arquivo está aberto em outro programa (feche-o e tente de novo)")

// nomeEstudo devolve "Nome (SITPOSSE7-XXXX-XXXX)" do estudo em foco ("" = todos).
func (c *Central) nomeEstudo(estudo string) string {
	if estudo == "" {
		return "Todos os estudos"
	}
	e, err := estudos.Detalhe(contexto(), c.n.B.Leitura, estudo)
	if err != nil {
		return ""
	}
	return e.Nome + " (" + e.CodigoExibicao + ")"
}

// gravar escolhe o destino e grava com segurança (arquivo temporário, fsync e rename).
func (s *RelatorioService) gravar(formato, nome string, gerar func() ([]byte, error)) (string, error) {
	f, ok := formatos[formato]
	if !ok {
		return "", fmt.Errorf("formato %q desconhecido", formato)
	}
	if s.C.Arquivos == nil {
		return "", errors.New("janela de salvar indisponível")
	}
	destino, err := s.C.Arquivos.EscolherDestino(nome+f.ext, f.descr, f.ext)
	if err != nil || destino == "" {
		return "", err
	}
	if !strings.EqualFold(filepath.Ext(destino), f.ext) {
		destino += f.ext
	}
	dados, err := gerar()
	if err != nil {
		return "", fmt.Errorf("gerando o arquivo: %w", err)
	}
	if err := GravarAtomico(destino, dados); err != nil {
		return "", err
	}
	return destino, nil
}

// GravarAtomico grava num temporário da mesma pasta, faz fsync e renomeia por cima do destino.
func GravarAtomico(destino string, dados []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(destino), ".sitposse7-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // sem efeito depois do rename
	if _, err := tmp.Write(dados); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), destino); err != nil {
		if errors.Is(err, os.ErrPermission) || strings.Contains(strings.ToLower(err.Error()), "used by another process") {
			return ErrArquivoEmUso
		}
		return err
	}
	return nil
}

// Exportar gera o relatório geral (panorama + famílias + fichas) com os filtros do Painel.
// criterios são os filtros ativos em texto, como aparecem na tela. Devolve o caminho
// gravado ou "" se a pessoa cancelou.
func (s *RelatorioService) Exportar(formato string, filtro estatisticas.Filtro, criterios []string, fichas bool) (string, error) {
	base, err := estatisticas.Carregar(contexto(), s.C.c.DB)
	if err != nil {
		return "", err
	}
	r, err := estatisticas.Calcular(base, filtro)
	if err != nil {
		return "", err
	}
	agora := time.Now()
	d := relatorios.Dados{Estudo: s.C.nomeEstudo(filtro.Estudo), GeradoEm: agora, Criterios: criterios, R: r, Fichas: fichas}
	nomeArq := ""
	if filtro.Estudo != "" && len(base.Estudos) > 0 {
		for _, e := range base.Estudos {
			if e.Valor == filtro.Estudo {
				nomeArq = e.Rotulo
			}
		}
	}
	nome := relatorios.NomeArquivo("SITPOSSE-7", "Panorama", nomeArq, agora.Format("2006-01-02 15h04"))
	return s.gravar(formato, nome, func() ([]byte, error) {
		switch formato {
		case "xlsx":
			return relatorios.XLSX(d)
		case "docx":
			return relatorios.DOCX(relatorios.Montar(d))
		default:
			return relatorios.PDF(relatorios.Montar(d))
		}
	})
}

// ExportarFicha gera a ficha de uma propriedade (PDF ou DOCX).
func (s *RelatorioService) ExportarFicha(formato, uuid string) (string, error) {
	if formato == "xlsx" {
		return "", errors.New("a ficha da propriedade sai em PDF ou Word")
	}
	base, err := estatisticas.Carregar(contexto(), s.C.c.DB)
	if err != nil {
		return "", err
	}
	r, err := estatisticas.Calcular(base, estatisticas.Filtro{})
	if err != nil {
		return "", err
	}
	for _, p := range r.Familias {
		if p.UUID != uuid {
			continue
		}
		doc := relatorios.MontarFicha(p, r.Campos, s.C.nomeEstudo(p.EstudoUUID), time.Now())
		return s.gravar(formato, relatorios.NomeArquivo("SITPOSSE-7", "Ficha", p.Nome), func() ([]byte, error) {
			if formato == "docx" {
				return relatorios.DOCX(doc)
			}
			return relatorios.PDF(doc)
		})
	}
	return "", errors.New("propriedade não encontrada (pode ter sido excluída no celular)")
}

// Abrir abre o arquivo no programa padrão do Windows.
func (s *RelatorioService) Abrir(caminho string) error { return s.C.Arquivos.Abrir(caminho) }

// MostrarNaPasta abre o Explorador com o arquivo selecionado.
func (s *RelatorioService) MostrarNaPasta(caminho string) error {
	return s.C.Arquivos.MostrarNaPasta(caminho)
}
