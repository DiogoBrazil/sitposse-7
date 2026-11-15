package importacao

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"sitposse7/desktop/internal/codigos"
	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/estudos"
)

// LoteAguardando é o que o modal "Confirmar destino da sincronização" mostra (ADR-0024).
type LoteAguardando struct {
	LoteUUID        string  `json:"lote_uuid"`
	InstalacaoUUID  string  `json:"instalacao_uuid"`
	RotuloAparelho  *string `json:"rotulo_aparelho"`
	ColetaUUID      string  `json:"coleta_uuid"`
	CodigoInformado string  `json:"codigo_informado"`
	CodigoExibicao  string  `json:"codigo_exibicao"`
	AguardandoDesde int64   `json:"aguardando_desde"`
	// Contagem do lote: cadastros novos, atualizados e excluídos, e fotos com bytes.
	Novos       int `json:"novos"`
	Atualizados int `json:"atualizados"`
	Excluidos   int `json:"excluidos"`
	Fotos       int `json:"fotos"`
	// Destino resolvido pelo código (nil = código não encontrado).
	Destino *estudos.Destino `json:"destino"`
	// Travado: a coleta já foi importada antes; o destino é DestinoTravado e não muda.
	Travado        bool             `json:"travado"`
	DestinoTravado *estudos.Destino `json:"destino_travado"`
}

// Aguardando lista os lotes que esperam a decisão do operador, do mais antigo ao mais novo.
func (im *Importador) Aguardando(ctx context.Context) ([]LoteAguardando, error) {
	rows, err := im.B.Leitura.QueryContext(ctx, `SELECT l.uuid, l.instalacao_uuid, d.rotulo_aparelho, l.coleta_uuid, l.codigo_informado,
		coalesce(l.aguardando_desde, l.recebido_em), l.manifesto_json, c.equipe_uuid
		FROM lotes_recebidos l JOIN coletas c ON c.uuid = l.coleta_uuid LEFT JOIN dispositivos d ON d.instalacao_uuid = l.instalacao_uuid
		WHERE l.status = 'aguardando_confirmacao' ORDER BY coalesce(l.aguardando_desde, l.recebido_em), l.uuid`)
	if err != nil {
		return nil, err
	}
	type linha struct {
		LoteAguardando
		manifesto string
		travada   sql.NullString
	}
	var ls []linha
	for rows.Next() {
		var x linha
		var rot sql.NullString
		if err := rows.Scan(&x.LoteUUID, &x.InstalacaoUUID, &rot, &x.ColetaUUID, &x.CodigoInformado, &x.AguardandoDesde, &x.manifesto, &x.travada); err != nil {
			rows.Close()
			return nil, err
		}
		if rot.Valid {
			x.RotuloAparelho = &rot.String
		}
		ls = append(ls, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]LoteAguardando, 0, len(ls))
	for _, x := range ls {
		a := x.LoteAguardando
		a.CodigoExibicao = codigos.Exibir(a.CodigoInformado)
		var l contratos.Lote
		if err := json.Unmarshal([]byte(x.manifesto), &l); err != nil {
			return nil, fmt.Errorf("manifesto ilegível: %w", err)
		}
		for _, p := range l.Propriedades {
			var n int
			if err := im.B.Leitura.QueryRowContext(ctx, `SELECT count(*) FROM propriedades WHERE uuid = ?`, p.UUID).Scan(&n); err != nil {
				return nil, err
			}
			switch {
			case p.ExcluidoEm != nil:
				a.Excluidos++
			case n > 0:
				a.Atualizados++
			default:
				a.Novos++
			}
		}
		for _, f := range l.Fotos {
			if f.ExcluidoEm == nil {
				a.Fotos++
			}
		}
		if a.Destino, err = estudos.DestinoPorCodigo(ctx, im.B.Leitura, a.CodigoInformado); err != nil {
			return nil, err
		}
		if x.travada.Valid {
			a.Travado = true
			if a.DestinoTravado, err = estudos.DestinoPorEquipe(ctx, im.B.Leitura, x.travada.String); err != nil {
				return nil, err
			}
		}
		out = append(out, a)
	}
	return out, nil
}

// Decisao é o que o operador escolheu para um lote aguardando.
type Decisao struct {
	// Cancelar encerra a sincronização sem importar nada.
	Cancelar bool
	// EquipeUUID é o destino confirmado (obrigatório quando não cancela).
	EquipeUUID string
	// Operador é o nome de quem decidiu (opcional; vai para a auditoria).
	Operador string
}

// Decidir aplica a decisão do operador sobre um lote aguardando (ADR-0024):
// cancelar (nada é importado nem apagado) ou importar para a equipe escolhida.
// Na coleta já importada antes, só a equipe travada é aceita. Devolve o novo estado.
func (im *Importador) Decidir(ctx context.Context, lote string, dec Decisao) (*contratos.EstadoLote, error) {
	im.mu.Lock()
	defer im.mu.Unlock()
	var inst, status, coleta, codigo, manifesto string
	var travada sql.NullString
	err := im.B.Escrita.QueryRowContext(ctx, `SELECT l.instalacao_uuid, l.status, l.coleta_uuid, l.codigo_informado, l.manifesto_json, c.equipe_uuid
		FROM lotes_recebidos l JOIN coletas c ON c.uuid = l.coleta_uuid WHERE l.uuid = ?`, lote).
		Scan(&inst, &status, &coleta, &codigo, &manifesto, &travada)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	switch status {
	case contratos.StatusImportado, contratos.StatusCancelado:
		return nil, ErrJaDecidido
	case contratos.StatusRecebendo:
		return nil, ErrNaoAguardando
	}
	porCodigo, err := estudos.DestinoPorCodigo(ctx, im.B.Escrita, codigo)
	if err != nil {
		return nil, err
	}
	agora := im.agora().UnixMilli()
	var operador any
	if dec.Operador != "" {
		operador = dec.Operador
	}

	if dec.Cancelar {
		motivo := contratos.MotivoCanceladoOperador
		if porCodigo == nil && !travada.Valid {
			motivo = contratos.MotivoCodigoInexistente
		}
		tx, err := im.B.Escrita.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, `UPDATE lotes_recebidos SET status = 'cancelado', motivo_cancelamento = ?, decidido_em = ? WHERE uuid = ? AND status = 'aguardando_confirmacao'`,
			motivo, agora, lote); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO decisoes_destino (lote_uuid, coleta_uuid, acao, codigo_informado, quando, operador) VALUES (?, ?, 'cancelar', ?, ?, ?)`,
			lote, coleta, codigo, agora, operador); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return im.estadoLocked(ctx, lote)
	}

	if dec.EquipeUUID == "" {
		return nil, fmt.Errorf("%w: escolha a equipe de destino", ErrInvalido)
	}
	if travada.Valid && dec.EquipeUUID != travada.String {
		return nil, ErrDestinoTravado
	}
	destino, err := estudos.EquipeVigente(ctx, im.B.Escrita, dec.EquipeUUID)
	if err != nil {
		return nil, err
	}
	if destino == nil {
		return nil, fmt.Errorf("%w: equipe de destino", ErrNaoEncontrado)
	}

	prontas, falt, err := im.conferirFotos(ctx, lote)
	if err != nil {
		return nil, err
	}
	if len(falt) > 0 {
		// Staging perdido ou corrompido depois do importar: o lote volta a "recebendo"
		// e o celular reenvia as fotos que faltam.
		if _, err := im.B.Escrita.ExecContext(ctx, `UPDATE lotes_recebidos SET status = 'recebendo', aguardando_desde = NULL WHERE uuid = ? AND status = 'aguardando_confirmacao'`, lote); err != nil {
			return nil, err
		}
		return &contratos.EstadoLote{ProtocoloVersao: contratos.ProtocoloVersao, LoteUUID: lote, Status: contratos.StatusRecebendo, FotosFaltantes: falt}, ErrFotosFaltantes
	}
	if err := im.publicar(prontas); err != nil {
		return nil, err
	}
	var l contratos.Lote
	if err := json.Unmarshal([]byte(manifesto), &l); err != nil {
		return nil, fmt.Errorf("manifesto persistido ilegível: %w", err)
	}
	if _, err := im.aplicar(ctx, inst, &l, destino, codigo, operador); err != nil {
		return nil, err
	}
	return im.estadoLocked(ctx, lote)
}
