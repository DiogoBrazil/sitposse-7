// Package app contém os serviços finos expostos ao frontend via Wails.
// Nenhuma regra de negócio aqui: apenas delega ao núcleo em internal/ (ADR-0003).
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"sitposse7/desktop/internal/banco"
	"sitposse7/desktop/internal/syncserver"
)

// Versao do aplicativo Central.
const Versao = "0.0.1-fase0"

// Central inicializa banco e servidor de sincronização e expõe diagnósticos.
type Central struct {
	PastaDados string
	banco      *banco.Banco
	servidor   *syncserver.Servidor
	centralID  string
	erroServ   string
}

// Info é o diagnóstico exibido na tela de Sincronização.
type Info struct {
	Versao         string   `json:"versao"`
	CentralID      string   `json:"central_id"`
	PastaDados     string   `json:"pasta_dados"`
	IPs            []string `json:"ips"`
	Porta          int      `json:"porta"`
	Pin            string   `json:"pin"`
	ServidorAtivo  bool     `json:"servidor_ativo"`
	ErroServidor   string   `json:"erro_servidor"`
	VersaoEsquema  int      `json:"versao_esquema"`
}

// Iniciar abre o banco, garante a identidade e sobe o servidor HTTPS.
func (c *Central) Iniciar(ctx context.Context, porta int) error {
	b, err := banco.Abrir(ctx, filepath.Join(c.PastaDados, "sitposse7.db"))
	if err != nil {
		return fmt.Errorf("banco: %w", err)
	}
	c.banco = b
	if c.centralID, err = garantirCentralID(ctx, b.Escrita); err != nil {
		return err
	}
	id, err := syncserver.CarregarOuCriarIdentidade(filepath.Join(c.PastaDados, "tls"), syncserver.IPsLocaisPrivados())
	if err != nil {
		return fmt.Errorf("identidade TLS: %w", err)
	}
	c.servidor = &syncserver.Servidor{Identidade: id, CentralID: c.centralID}
	if err := c.servidor.Iniciar(fmt.Sprintf(":%d", porta)); err != nil {
		// Não impede o uso offline da Central; só a recepção fica indisponível.
		c.erroServ = fmt.Sprintf("Não foi possível escutar na porta %d: %v", porta, err)
	}
	return nil
}

// Encerrar para o servidor e fecha o banco.
func (c *Central) Encerrar() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if c.servidor != nil {
		_ = c.servidor.Parar(ctx)
	}
	if c.banco != nil {
		_ = c.banco.Fechar()
	}
}

// SincronizacaoService é o único serviço exposto ao frontend nesta fase.
// Não expõe ciclo de vida (Iniciar/Encerrar) para que a UI não possa fechar o banco.
type SincronizacaoService struct{ C *Central }

// Info devolve o diagnóstico atual.
func (s *SincronizacaoService) Info() Info { return s.C.info() }

// QRPareamento devolve o payload do QR de conexão.
func (s *SincronizacaoService) QRPareamento() (syncserver.PayloadQR, error) { return s.C.qrPareamento() }

func (c *Central) info() Info {
	inf := Info{Versao: Versao, CentralID: c.centralID, PastaDados: c.PastaDados, ErroServidor: c.erroServ, VersaoEsquema: banco.VersaoMaxima()}
	if c.servidor != nil {
		inf.Pin = c.servidor.Identidade.Pin()
		inf.Porta = c.servidor.Porta()
		inf.ServidorAtivo = c.erroServ == "" && inf.Porta > 0
		for _, ip := range c.servidor.Identidade.IPs {
			inf.IPs = append(inf.IPs, ip.String())
		}
	}
	return inf
}

// qrPareamento monta o payload do QR (Fase 0: sem token; o pareamento entra na Fase 2).
func (c *Central) qrPareamento() (syncserver.PayloadQR, error) {
	if c.servidor == nil || c.erroServ != "" {
		return syncserver.PayloadQR{}, errors.New("servidor de sincronização indisponível")
	}
	// Reavalia IPs (o notebook pode ter entrado em outra rede/hotspot).
	if atuais := syncserver.IPsLocaisPrivados(); !mesmosIPs(atuais, c.servidor.Identidade.IPs) {
		if id, err := syncserver.CarregarOuCriarIdentidade(filepath.Join(c.PastaDados, "tls"), atuais); err == nil {
			c.servidor.Identidade.IPs = id.IPs
		}
	}
	return c.servidor.QR("", nil), nil
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

func garantirCentralID(ctx context.Context, db *sql.DB) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `SELECT valor FROM meta WHERE chave = 'central_id'`).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	id = uuid.NewString()
	agora := time.Now().UnixMilli()
	_, err = db.ExecContext(ctx, `INSERT INTO meta (chave, valor, created_at, updated_at) VALUES ('central_id', ?, ?, ?)`, id, agora, agora)
	return id, err
}
