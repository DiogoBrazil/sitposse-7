// Package nucleo monta a Central (banco, identidade TLS, pareamento, importação e
// servidor HTTPS) do mesmo jeito para o app Wails, para o central-teste e para os testes.
package nucleo

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
	"sitposse7/desktop/internal/estudos"
	"sitposse7/desktop/internal/importacao"
	"sitposse7/desktop/internal/pareamento"
	"sitposse7/desktop/internal/syncserver"
)

// Nucleo agrupa os serviços da Central.
type Nucleo struct {
	Pasta      string
	CentralID  string
	B          *banco.Banco
	Pareamento *pareamento.Servico
	Importador *importacao.Importador
	Estudos    *estudos.Servico
	Servidor   *syncserver.Servidor
	// Reconciliacao é o resultado do reconciliador na abertura.
	Reconciliacao importacao.RelatorioReconciliacao
}

// Abrir abre o banco, garante o ID e a identidade TLS e roda o reconciliador.
// O servidor é criado, mas não iniciado (chame Servidor.Iniciar).
func Abrir(ctx context.Context, pasta string, ips []net.IP) (*Nucleo, error) {
	b, err := banco.Abrir(ctx, filepath.Join(pasta, "sitposse7.db"))
	if err != nil {
		return nil, fmt.Errorf("banco: %w", err)
	}
	n := &Nucleo{Pasta: pasta, B: b}
	if n.CentralID, err = garantirCentralID(ctx, b.Escrita); err != nil {
		b.Fechar()
		return nil, err
	}
	id, err := syncserver.CarregarOuCriarIdentidade(filepath.Join(pasta, "tls"), ips)
	if err != nil {
		b.Fechar()
		return nil, fmt.Errorf("identidade TLS: %w", err)
	}
	n.Pareamento = &pareamento.Servico{B: b, CentralID: n.CentralID}
	n.Importador = &importacao.Importador{B: b, PastaDados: pasta, CentralID: n.CentralID}
	n.Estudos = &estudos.Servico{B: b}
	if n.Reconciliacao, err = n.Importador.Reconciliar(ctx); err != nil {
		b.Fechar()
		return nil, fmt.Errorf("reconciliador: %w", err)
	}
	n.Servidor = &syncserver.Servidor{
		Identidade: id, CentralID: n.CentralID,
		Pareamento: n.Pareamento, Importador: n.Importador,
	}
	return n, nil
}

// Fechar para o servidor e fecha o banco.
func (n *Nucleo) Fechar() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if n.Servidor != nil {
		_ = n.Servidor.Parar(ctx)
	}
	_ = n.B.Fechar()
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
