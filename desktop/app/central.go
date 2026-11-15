// Package app contém os serviços finos expostos ao frontend via Wails.
// Nenhuma regra de negócio aqui: apenas delega ao núcleo em internal/ (ADR-0003).
package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"sitposse7/desktop/internal/consultas"
	"sitposse7/desktop/internal/estatisticas"
	"sitposse7/desktop/internal/nucleo"
	"sitposse7/desktop/internal/pareamento"
	"sitposse7/desktop/internal/syncserver"
)

// Versao do aplicativo Central.
const Versao = "0.5.0-mvp"

// Central inicializa o núcleo e mantém o servidor de sincronização.
type Central struct {
	PastaDados string
	// Arquivos: janelas de salvar/abrir do sistema (main.go). Pode ser nil nos testes.
	Arquivos Arquivos
	n        *nucleo.Nucleo
	c        *consultas.Consultas
	erroServ string
}

// Info é o diagnóstico exibido na tela de Sincronização.
type Info struct {
	Versao          string   `json:"versao"`
	CentralID       string   `json:"central_id"`
	PastaDados      string   `json:"pasta_dados"`
	IPs             []string `json:"ips"`
	Porta           int      `json:"porta"`
	Pin             string   `json:"pin"`
	ServidorAtivo   bool     `json:"servidor_ativo"`
	ErroServidor    string   `json:"erro_servidor"`
	VersaoEsquema   int      `json:"versao_esquema"`
	FotosSemArquivo []string `json:"fotos_sem_arquivo"` // alarme do reconciliador
}

// Iniciar abre o núcleo e sobe o servidor HTTPS.
func (c *Central) Iniciar(ctx context.Context, porta int) error {
	n, err := nucleo.Abrir(ctx, c.PastaDados, syncserver.IPsLocaisPrivados())
	if err != nil {
		return err
	}
	c.n = n
	c.c = &consultas.Consultas{DB: n.B.Leitura}
	if err := n.Servidor.Iniciar(fmt.Sprintf(":%d", porta)); err != nil {
		// Não impede o uso offline da Central; só a recepção fica indisponível.
		c.erroServ = fmt.Sprintf("Não foi possível escutar na porta %d: %v", porta, err)
	}
	return nil
}

// Encerrar para o servidor e fecha o banco.
func (c *Central) Encerrar() {
	if c.n != nil {
		c.n.Fechar()
	}
}

var reFoto = regexp.MustCompile(`^/fotos/([0-9a-f-]{36})\.jpg$`)

// HandlerFotos serve à tela as fotos vigentes (/fotos/{uuid}.jpg), só leitura.
func (c *Central) HandlerFotos() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := reFoto.FindStringSubmatch(r.URL.Path)
		if m == nil || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		rel, err := c.c.CaminhoFotoVigente(r.Context(), m[1])
		if err != nil || rel == "" || strings.Contains(rel, "..") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "private, max-age=86400")
		http.ServeFile(w, r, filepath.Join(c.PastaDados, filepath.FromSlash(rel)))
	})
}

func (c *Central) info() Info {
	inf := Info{Versao: Versao, PastaDados: c.PastaDados, ErroServidor: c.erroServ, FotosSemArquivo: []string{}}
	if c.n == nil {
		return inf
	}
	inf.CentralID = c.n.CentralID
	inf.FotosSemArquivo = append(inf.FotosSemArquivo, c.n.Reconciliacao.RegistrosSemArquivo...)
	var v int
	_ = c.n.B.Leitura.QueryRow("PRAGMA user_version").Scan(&v)
	inf.VersaoEsquema = v
	s := c.n.Servidor
	inf.Pin = s.Identidade.Pin()
	inf.Porta = s.Porta()
	inf.ServidorAtivo = c.erroServ == "" && inf.Porta > 0
	for _, ip := range s.Identidade.IPs {
		if !ip.IsLoopback() {
			inf.IPs = append(inf.IPs, ip.String())
		}
	}
	return inf
}

// atualizarIPs reavalia os IPs (o computador pode ter ligado o hotspot depois de abrir a Central).
func (c *Central) atualizarIPs() {
	s := c.n.Servidor
	if atuais := syncserver.IPsLocaisPrivados(); !mesmosIPs(atuais, s.Identidade.IPs) {
		if id, err := syncserver.CarregarOuCriarIdentidade(filepath.Join(c.PastaDados, "tls"), atuais); err == nil {
			s.Identidade.IPs = id.IPs
		}
	}
}

func mesmosIPs(a, b []net.IP) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
}

// ---------- serviços expostos ao frontend ----------
// Não expõem ciclo de vida (Iniciar/Encerrar) para que a UI não possa fechar o banco.

// SincronizacaoService: diagnóstico, QR de pareamento, pedidos pendentes e recepção.
type SincronizacaoService struct{ C *Central }

// Info devolve o diagnóstico atual.
func (s *SincronizacaoService) Info() Info {
	if s.C.n != nil {
		s.C.atualizarIPs()
	}
	return s.C.info()
}

// QRPareamento é o QR com token de uso único e a validade.
type QRPareamento struct {
	Payload  syncserver.PayloadQR `json:"payload"`
	ExpiraEm int64                `json:"expira_em"`
}

// GerarQRPareamento cria um token novo (10 min, uso único) e monta o QR.
func (s *SincronizacaoService) GerarQRPareamento() (*QRPareamento, error) {
	c := s.C
	if c.n == nil || c.erroServ != "" {
		return nil, errors.New("servidor de sincronização indisponível")
	}
	c.atualizarIPs()
	token, op, expira, err := c.n.Pareamento.GerarToken(context.Background())
	if err != nil {
		return nil, err
	}
	p := c.n.Servidor.QR(token, &syncserver.Operacao{UUID: op.UUID, Nome: op.Nome})
	return &QRPareamento{Payload: p, ExpiraEm: expira.UnixMilli()}, nil
}

// Pendentes lista os aparelhos aguardando confirmação.
func (s *SincronizacaoService) Pendentes() []pareamento.PedidoResumo {
	if s.C.n == nil {
		return []pareamento.PedidoResumo{}
	}
	return s.C.n.Pareamento.Pendentes()
}

// Decidir confirma (true) ou recusa (false) o pareamento.
func (s *SincronizacaoService) Decidir(id string, aceitar bool) error {
	return s.C.n.Pareamento.Decidir(id, aceitar)
}

// Lotes lista os últimos lotes recebidos.
func (s *SincronizacaoService) Lotes() ([]consultas.LoteLinha, error) {
	return s.C.c.Lotes(contexto(), 100)
}

// Rejeicoes lista os registros rejeitados de um lote.
func (s *SincronizacaoService) Rejeicoes(lote string) ([]consultas.Rejeicao, error) {
	return s.C.c.Rejeicoes(contexto(), lote)
}

// OperacaoService: operação ativa.
type OperacaoService struct{ C *Central }

// Ativa devolve a operação ativa (nil se não houver).
func (s *OperacaoService) Ativa() (*pareamento.Operacao, error) {
	return s.C.n.Pareamento.OperacaoAtiva(contexto())
}

// Criar cria e ativa a operação.
func (s *OperacaoService) Criar(nome, descricao string) (*pareamento.Operacao, error) {
	return s.C.n.Pareamento.CriarOperacao(contexto(), nome, descricao)
}

// ConsultaService: leituras para o painel, propriedades e equipes (somente leitura).
type ConsultaService struct{ C *Central }

// Resumo devolve os números do painel.
func (s *ConsultaService) Resumo() (*consultas.Resumo, error) { return s.C.c.Resumo(contexto()) }

// Propriedades lista as propriedades vigentes.
func (s *ConsultaService) Propriedades(busca, equipe string) ([]consultas.PropriedadeLinha, error) {
	return s.C.c.Propriedades(contexto(), busca, equipe)
}

// Propriedade devolve o detalhe (nil se não existir).
func (s *ConsultaService) Propriedade(uuid string) (*consultas.PropriedadeDetalhe, error) {
	return s.C.c.Propriedade(contexto(), uuid)
}

// Estatisticas aplica o FiltroEstudo às propriedades vigentes e devolve todos os números
// do painel a partir do mesmo conjunto (RN-81; docs/requirements/estatisticas.md §22-A).
func (s *ConsultaService) Estatisticas(filtro estatisticas.Filtro) (*estatisticas.Resultado, error) {
	base, err := estatisticas.Carregar(contexto(), s.C.c.DB)
	if err != nil {
		return nil, err
	}
	return estatisticas.Calcular(base, filtro)
}

// Equipes lista as equipes.
func (s *ConsultaService) Equipes() ([]consultas.EquipeLinha, error) {
	return s.C.c.Equipes(contexto())
}

func contexto() context.Context { return context.Background() }
