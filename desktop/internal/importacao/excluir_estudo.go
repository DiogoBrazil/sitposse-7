package importacao

import (
	"context"

	"errors"
	"sitposse7/desktop/internal/correcoes"
	"time"
)

// ExcluirEstudo serializa a cascata com as decisões de importação.
func (im *Importador) ExcluirEstudo(ctx context.Context, id string) error {
	im.mu.Lock()
	defer im.mu.Unlock()
	tx, e := im.B.Escrita.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var nome string
	if e = tx.QueryRowContext(ctx, `SELECT nome FROM estudos WHERE uuid=? AND excluido_em IS NULL`, id).Scan(&nome); e != nil {
		return errors.New("estudo não encontrado ou excluído")
	}
	rows, e := tx.QueryContext(ctx, `SELECT p.uuid FROM propriedades p JOIN equipes e ON e.uuid=p.equipe_uuid JOIN jornadas j ON j.uuid=e.jornada_uuid WHERE j.estudo_uuid=? AND p.excluido_em IS NULL`, id)
	if e != nil {
		return e
	}
	ids := []string{}
	for rows.Next() {
		var u string
		if e = rows.Scan(&u); e != nil {
			rows.Close()
			return e
		}
		ids = append(ids, u)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, u := range ids {
		if e = correcoes.ExcluirCadastro(ctx, tx, u); e != nil {
			return e
		}
	}
	agora := time.Now().UnixMilli()
	rows, e = tx.QueryContext(ctx, `SELECT l.uuid,l.coleta_uuid,l.codigo_informado FROM lotes_recebidos l LEFT JOIN coletas c ON c.uuid=l.coleta_uuid JOIN equipes e ON e.uuid=c.equipe_uuid OR (c.equipe_uuid IS NULL AND e.codigo=l.codigo_informado) JOIN jornadas j ON j.uuid=e.jornada_uuid WHERE j.estudo_uuid=? AND l.status IN ('recebendo','aguardando_confirmacao')`, id)
	if e != nil {
		return e
	}
	type lote struct{ u, c, cod string }
	ls := []lote{}
	for rows.Next() {
		var l lote
		if e = rows.Scan(&l.u, &l.c, &l.cod); e != nil {
			rows.Close()
			return e
		}
		ls = append(ls, l)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, l := range ls {
		if _, e = tx.ExecContext(ctx, `INSERT INTO decisoes_destino(lote_uuid,coleta_uuid,acao,codigo_informado,quando) VALUES(?,?,'cancelar',?,?)`, l.u, l.c, l.cod, agora); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE lotes_recebidos SET status='cancelado',motivo_cancelamento='cancelado_operador',decidido_em=? WHERE uuid=?`, agora, l.u); e != nil {
			return e
		}
	}
	for _, x := range []struct{ t, w string }{{"equipes", "jornada_uuid IN (SELECT uuid FROM jornadas WHERE estudo_uuid=?)"}, {"jornadas", "estudo_uuid=?"}, {"estudos", "uuid=?"}} {
		if e = correcoes.ExcluirEstrutura(ctx, tx, x.t, x.w, id); e != nil {
			return e
		}
	}
	return tx.Commit()
}
