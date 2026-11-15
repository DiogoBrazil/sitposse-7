package importacao

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"sitposse7/desktop/internal/contratos"
	"sitposse7/desktop/internal/dominio"
	"sitposse7/desktop/internal/estudos"
)

// aplicar grava, numa única transação, o destino da coleta (travado a partir daqui),
// a decisão do operador, os registros do lote e o recibo persistido (ADR-0024).
// Cada registro é aplicado num SAVEPOINT: um registro rejeitado não derruba os demais.
// Regra de aplicação: rev > rev_origem → aplicado; rev ≤ rev_origem → duplicado.
// Nada é sobrescrito com rev menor ou igual.
func (im *Importador) aplicar(ctx context.Context, instalacao string, l *contratos.Lote, dest *estudos.Destino, codigoInformado string, operador any) (*contratos.Recibo, error) {
	agora := im.agora().UnixMilli()
	tx, err := im.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var vigente int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM equipes e JOIN jornadas j ON j.uuid=e.jornada_uuid JOIN estudos s ON s.uuid=j.estudo_uuid WHERE e.uuid=? AND e.excluido_em IS NULL AND j.excluido_em IS NULL AND s.excluido_em IS NULL`, dest.EquipeUUID).Scan(&vigente); err != nil {
		return nil, err
	}
	if vigente == 0 {
		return nil, fmt.Errorf("destino excluído; nada foi importado")
	}
	coleta := l.Coleta.UUID
	a := &aplicador{ctx: ctx, tx: tx, instalacao: instalacao, agora: agora, origem: "coleta:" + coleta, lote: l.LoteUUID, equipe: dest.EquipeUUID}

	// Destino da coleta (o gatilho do banco impede trocar uma equipe já gravada) e decisão.
	if _, err := tx.ExecContext(ctx, `UPDATE coletas SET equipe_uuid = ?, destino_confirmado_em = ? WHERE uuid = ? AND equipe_uuid IS NULL`,
		dest.EquipeUUID, agora, coleta); err != nil {
		return nil, fmt.Errorf("gravando destino da coleta: %w", err)
	}
	acao := "confirmar"
	if dest.EquipeCodigo != codigoInformado {
		acao = "alterar"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO decisoes_destino (lote_uuid, coleta_uuid, acao, codigo_informado, codigo_confirmado, equipe_uuid, jornada_uuid, estudo_uuid, quando, operador)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, l.LoteUUID, coleta, acao, codigoInformado, dest.EquipeCodigo, dest.EquipeUUID, dest.JornadaUUID, dest.EstudoUUID, agora, operador); err != nil {
		return nil, fmt.Errorf("gravando decisão: %w", err)
	}

	resultados := map[string]contratos.ResultadoRecibo{}
	chave := func(ent, uuid string) string { return ent + "|" + uuid }
	registrar := func(ent, uuid string, rev int64, res string, motivo string) {
		resultados[chave(ent, uuid)] = contratos.ResultadoRecibo{Entidade: ent, UUID: uuid, Rev: rev, Resultado: res, Motivo: motivo}
	}

	res, motivo := a.comSavepoint(func() (string, error) { return a.coleta(&l.Coleta) })
	registrar("coleta", coleta, l.Coleta.Rev, res, motivo)
	for i := range l.Propriedades {
		p := &l.Propriedades[i]
		res, motivo := a.comSavepoint(func() (string, error) { return a.propriedade(p, coleta) })
		registrar("propriedade", p.UUID, p.Rev, res, motivo)
	}
	// Exclusões de itens primeiro: liberam o índice único de item predefinido (RN-32)
	// quando, no mesmo lote, um item foi removido e outro do mesmo tipo criado.
	ordemItens := make([]*contratos.Item, 0, len(l.Itens))
	for i := range l.Itens {
		if l.Itens[i].ExcluidoEm != nil {
			ordemItens = append(ordemItens, &l.Itens[i])
		}
	}
	for i := range l.Itens {
		if l.Itens[i].ExcluidoEm == nil {
			ordemItens = append(ordemItens, &l.Itens[i])
		}
	}
	for _, it := range ordemItens {
		res, motivo := a.comSavepoint(func() (string, error) { return a.item(it, coleta) })
		registrar("item", it.UUID, it.Rev, res, motivo)
	}
	for i := range l.Fotos {
		f := &l.Fotos[i]
		res, motivo := a.comSavepoint(func() (string, error) { return a.foto(f, coleta) })
		registrar("foto", f.UUID, f.Rev, res, motivo)
	}

	// Recibo na ordem do manifesto.
	recibo := &contratos.Recibo{ProtocoloVersao: contratos.ProtocoloVersao, LoteUUID: l.LoteUUID, CentralID: im.CentralID,
		ImportadoEm: contratos.FormatarInstante(agora), Destino: contratos.Destino{
			EstudoCodigo: dest.EstudoCodigo, EstudoNome: dest.EstudoNome, JornadaNumero: int64(dest.JornadaNumero), JornadaData: dest.JornadaData,
			EquipeNome: dest.EquipeNome, EquipeCodigo: dest.EquipeCodigo, CodigoInformado: codigoInformado, Corrigido: acao == "alterar"}}
	add := func(ent, uuid string) {
		r := resultados[chave(ent, uuid)]
		recibo.Resultados = append(recibo.Resultados, r)
	}
	add("coleta", coleta)
	for _, p := range l.Propriedades {
		add("propriedade", p.UUID)
	}
	for _, it := range l.Itens {
		add("item", it.UUID)
	}
	for _, f := range l.Fotos {
		add("foto", f.UUID)
	}
	recibo.ReciboSHA256 = recibo.CalcularHash()

	for _, r := range recibo.Resultados {
		var motivo any
		if r.Motivo != "" {
			motivo = r.Motivo
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO aplicacoes (lote_uuid, entidade, entidade_uuid, rev, resultado, motivo) VALUES (?, ?, ?, ?, ?, ?)`,
			l.LoteUUID, r.Entidade, r.UUID, r.Rev, r.Resultado, motivo); err != nil {
			return nil, fmt.Errorf("gravando aplicação: %w", err)
		}
	}
	reciboJSON, err := json.Marshal(recibo)
	if err != nil {
		return nil, err
	}
	res2, err := tx.ExecContext(ctx, `UPDATE lotes_recebidos SET status = 'importado', recibo_json = ?, importado_em = ?, equipe_destino_uuid = ?, decidido_em = ?
		WHERE uuid = ? AND status = 'aguardando_confirmacao'`, string(reciboJSON), agora, dest.EquipeUUID, agora, l.LoteUUID)
	if err != nil {
		return nil, err
	}
	if n, _ := res2.RowsAffected(); n != 1 {
		return nil, ErrJaDecidido
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gravando importação: %w", err)
	}
	return recibo, nil
}

type aplicador struct {
	ctx        context.Context
	tx         *sql.Tx
	instalacao string
	equipe     string // equipe de destino confirmada
	agora      int64
	origem     string
	lote       string
	n          int
}

// rejeicao é um erro esperado que vira "rejeitado" no recibo (não aborta o lote).
type rejeicao struct{ motivo string }

func (r rejeicao) Error() string { return r.motivo }

func rejeitar(format string, a ...any) error { return rejeicao{fmt.Sprintf(format, a...)} }

// comSavepoint executa fn num SAVEPOINT. Rejeição ou erro de restrição do SQLite
// desfazem só este registro. Outros erros também viram rejeição com o motivo, para
// que um registro problemático nunca impeça os demais de serem confirmados.
func (a *aplicador) comSavepoint(fn func() (string, error)) (string, string) {
	a.n++
	sp := fmt.Sprintf("sp%d", a.n)
	if _, err := a.tx.ExecContext(a.ctx, "SAVEPOINT "+sp); err != nil {
		return contratos.ResultadoRejeitado, "erro interno: " + err.Error()
	}
	res, err := fn()
	if err != nil {
		_, _ = a.tx.ExecContext(a.ctx, "ROLLBACK TO "+sp)
		_, _ = a.tx.ExecContext(a.ctx, "RELEASE "+sp)
		var rj rejeicao
		if errors.As(err, &rj) {
			return contratos.ResultadoRejeitado, rj.motivo
		}
		msg := err.Error()
		if strings.Contains(msg, "UNIQUE constraint failed: itens_propriedade") {
			msg = "item predefinido repetido na propriedade (RN-32)"
		}
		return contratos.ResultadoRejeitado, msg
	}
	if _, err := a.tx.ExecContext(a.ctx, "RELEASE "+sp); err != nil {
		return contratos.ResultadoRejeitado, "erro interno: " + err.Error()
	}
	return res, ""
}

// estadoAtual lê rev_origem e registro_json da entidade e a coleta dona (ok=false se não existe).
func (a *aplicador) estadoAtual(tabela, uuid string) (rev int64, registro string, corrigido bool, coleta string, ok bool, err error) {
	var q string
	switch tabela {
	case "propriedades":
		q = `SELECT rev_origem, registro_json, corrigido_desde_rev_origem, coleta_uuid FROM propriedades WHERE uuid = ?`
	case "itens_propriedade":
		q = `SELECT i.rev_origem, i.registro_json, i.corrigido_desde_rev_origem, p.coleta_uuid FROM itens_propriedade i JOIN propriedades p ON p.uuid = i.propriedade_uuid WHERE i.uuid = ?`
	case "fotografias":
		q = `SELECT f.rev_origem, f.registro_json, f.corrigido_desde_rev_origem, p.coleta_uuid FROM fotografias f JOIN propriedades p ON p.uuid = f.propriedade_uuid WHERE f.uuid = ?`
	default:
		return 0, "", false, "", false, fmt.Errorf("tabela desconhecida %s", tabela)
	}
	var c int64
	err = a.tx.QueryRowContext(a.ctx, q, uuid).Scan(&rev, &registro, &c, &coleta)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, "", false, nil
	}
	return rev, registro, c == 1, coleta, err == nil, err
}

func (a *aplicador) auditar(entidade, uuid, acao, antes string, depois []byte) error {
	var antesV any
	if antes != "" {
		antesV = antes
	}
	_, err := a.tx.ExecContext(a.ctx, `INSERT INTO auditoria (quando, operador, origem, entidade, entidade_uuid, acao, antes_json, depois_json)
		VALUES (?, NULL, ?, ?, ?, ?, ?, ?)`, a.agora, a.origem, entidade, uuid, acao, antesV, string(depois))
	return err
}

func acaoDe(antes string, excluido *string) string {
	switch {
	case excluido != nil:
		return "excluir"
	case antes == "":
		return "criar"
	}
	return "atualizar"
}

func ms(s string) int64 {
	v, _ := contratos.LerInstante(s) // já validado
	return v
}

func msOpt(s *string) any {
	if s == nil {
		return nil
	}
	return ms(*s)
}

func (a *aplicador) coleta(c *contratos.Coleta) (string, error) {
	if err := contratos.ValidarColeta(c); err != nil {
		return "", rejeitar("%v", err)
	}
	var rev int64
	var antes sql.NullString
	if err := a.tx.QueryRowContext(a.ctx, `SELECT rev_origem, registro_json FROM coletas WHERE uuid = ?`, c.UUID).Scan(&rev, &antes); err != nil {
		return "", err
	}
	if c.Rev <= rev {
		return contratos.ResultadoDuplicado, nil
	}
	js, _ := json.Marshal(c)
	if _, err := a.tx.ExecContext(a.ctx, `UPDATE coletas SET codigo_informado = ?, codigo_original = ?, updated_at = ?, rev_origem = ?, recebido_em = ?, registro_json = ? WHERE uuid = ?`,
		c.CodigoEquipe, c.CodigoOriginal, ms(c.UpdatedAt), c.Rev, a.agora, string(js), c.UUID); err != nil {
		return "", err
	}
	return contratos.ResultadoAplicado, a.auditar("coleta", c.UUID, acaoDe(antes.String, nil), antes.String, js)
}

func (a *aplicador) propriedade(p *contratos.Propriedade, coleta string) (string, error) {
	if err := contratos.ValidarPropriedade(p); err != nil {
		return "", rejeitar("%v", err)
	}
	if p.ColetaUUID != coleta {
		return "", rejeitar("propriedade de outra coleta")
	}
	rev, antes, corrigido, coletaAtual, existe, err := a.estadoAtual("propriedades", p.UUID)
	if err != nil {
		return "", err
	}
	if existe && coletaAtual != coleta {
		return "", rejeitar("UUID já pertence a cadastro de outra coleta")
	}
	if existe && p.Rev <= rev {
		return contratos.ResultadoDuplicado, nil
	}
	if existe && corrigido {
		// Não ocorre no MVP (a Central não corrige cadastros). Nunca sobrescrever em silêncio.
		return "", rejeitar("cadastro corrigido na Central; resolução de conflitos ainda não disponível")
	}
	js, _ := json.Marshal(p)
	args := []any{p.NomePropriedade, p.ResponsavelFamilia, p.ObservacoesRelevantes,
		p.PessoasAdultos, p.PessoasJovens, p.PessoasCriancas, p.VeicCaminhoes, p.VeicAutomoveisUtilitarios, p.VeicMotosCiclomotores,
		p.BovAdultos, p.BovBezerros, p.EquAdultos, p.EquPotros, p.AreaValorMilesimos, p.AreaUnidade,
		p.Lat, p.Lon, p.GPSOrigem, p.GPSPrecisaoM, msOpt(p.GPSCapturadoEm), p.GPSSatelites,
		ms(p.UpdatedAt), msOpt(p.ExcluidoEm), p.Rev, a.agora, a.instalacao, string(js)}
	if existe {
		_, err = a.tx.ExecContext(a.ctx, `UPDATE propriedades SET nome_propriedade = ?, responsavel_familia = ?, observacoes_relevantes = ?,
			pessoas_adultos = ?, pessoas_jovens = ?, pessoas_criancas = ?, veic_caminhoes = ?, veic_automoveis_utilitarios = ?, veic_motos_ciclomotores = ?,
			bov_adultos = ?, bov_bezerros = ?, equ_adultos = ?, equ_potros = ?, area_valor_milesimos = ?, area_unidade = ?,
			lat = ?, lon = ?, gps_origem = ?, gps_precisao_m = ?, gps_capturado_em = ?, gps_satelites = ?,
			updated_at = ?, excluido_em = ?, rev_origem = ?, recebido_em = ?, origem_instalacao = ?, registro_json = ?
			WHERE uuid = ?`, append(args, p.UUID)...)
	} else {
		_, err = a.tx.ExecContext(a.ctx, `INSERT INTO propriedades (nome_propriedade, responsavel_familia, observacoes_relevantes,
			pessoas_adultos, pessoas_jovens, pessoas_criancas, veic_caminhoes, veic_automoveis_utilitarios, veic_motos_ciclomotores,
			bov_adultos, bov_bezerros, equ_adultos, equ_potros, area_valor_milesimos, area_unidade,
			lat, lon, gps_origem, gps_precisao_m, gps_capturado_em, gps_satelites,
			updated_at, excluido_em, rev_origem, recebido_em, origem_instalacao, registro_json,
			uuid, coleta_uuid, equipe_uuid, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			append(args, p.UUID, coleta, a.equipe, ms(p.CreatedAt))...)
		if err == nil {
			err = a.numerar(p.UUID)
		}
	}
	if err != nil {
		return "", err
	}
	return contratos.ResultadoAplicado, a.auditar("propriedade", p.UUID, acaoDe(antes, p.ExcluidoEm), antes, js)
}

// numerar atribui o próximo número da propriedade no estudo de destino (RN-96), na mesma
// transação da primeira inserção. O número nunca muda nem é reaproveitado (triggers da 0003).
func (a *aplicador) numerar(propriedade string) error {
	_, err := a.tx.ExecContext(a.ctx, `INSERT INTO numeracao_propriedades (propriedade_uuid, estudo_uuid, numero, atribuido_em)
		SELECT ?, j.estudo_uuid,
		       COALESCE((SELECT max(n.numero) FROM numeracao_propriedades n WHERE n.estudo_uuid = j.estudo_uuid), 0) + 1, ?
		FROM equipes e JOIN jornadas j ON j.uuid = e.jornada_uuid WHERE e.uuid = ?`, propriedade, a.agora, a.equipe)
	return err
}

// propriedadeDaColeta confere que a propriedade existe na Central e é da coleta.
func (a *aplicador) propriedadeDaColeta(uuid, coleta string) error {
	var c string
	err := a.tx.QueryRowContext(a.ctx, `SELECT coleta_uuid FROM propriedades WHERE uuid = ?`, uuid).Scan(&c)
	if errors.Is(err, sql.ErrNoRows) {
		return rejeitar("propriedade %s não recebida", uuid)
	}
	if err != nil {
		return err
	}
	if c != coleta {
		return rejeitar("propriedade de outra coleta")
	}
	return nil
}

func (a *aplicador) item(it *contratos.Item, coleta string) (string, error) {
	if err := contratos.ValidarItem(it); err != nil {
		return "", rejeitar("%v", err)
	}
	var ativo int
	if err := a.tx.QueryRowContext(a.ctx, `SELECT ativo FROM catalogo_itens WHERE codigo = ?`, it.CodigoCatalogo).Scan(&ativo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", rejeitar("código de catálogo desconhecido: %s", it.CodigoCatalogo)
		}
		return "", err
	}
	if err := a.propriedadeDaColeta(it.PropriedadeUUID, coleta); err != nil {
		return "", err
	}
	rev, antes, corrigido, _, existe, err := a.estadoAtual("itens_propriedade", it.UUID)
	if err != nil {
		return "", err
	}
	if existe && it.Rev <= rev {
		return contratos.ResultadoDuplicado, nil
	}
	var protegido int
	if err := a.tx.QueryRowContext(a.ctx, `SELECT corrigido_desde_rev_origem FROM propriedades WHERE uuid=?`, it.PropriedadeUUID).Scan(&protegido); err != nil {
		return "", err
	}
	if protegido != 0 {
		return "", rejeitar("cadastro corrigido ou excluído na Central; resolução de conflitos ainda não disponível")
	}
	if existe && corrigido {
		return "", rejeitar("item corrigido na Central; resolução de conflitos ainda não disponível")
	}
	var normalizado any
	if it.NomePersonalizado != nil {
		if n, ok := dominio.NormalizarNome(*it.NomePersonalizado); ok {
			normalizado = n
		}
	}
	js, _ := json.Marshal(it)
	if existe {
		_, err = a.tx.ExecContext(a.ctx, `UPDATE itens_propriedade SET propriedade_uuid = ?, categoria = ?, codigo_catalogo = ?, nome_personalizado = ?, nome_normalizado = ?,
			quantidade = ?, updated_at = ?, excluido_em = ?, rev_origem = ?, recebido_em = ?, origem_instalacao = ?, registro_json = ? WHERE uuid = ?`,
			it.PropriedadeUUID, it.Categoria, it.CodigoCatalogo, it.NomePersonalizado, normalizado, it.Quantidade, ms(it.UpdatedAt), msOpt(it.ExcluidoEm),
			it.Rev, a.agora, a.instalacao, string(js), it.UUID)
	} else {
		_, err = a.tx.ExecContext(a.ctx, `INSERT INTO itens_propriedade (uuid, propriedade_uuid, categoria, codigo_catalogo, nome_personalizado, nome_normalizado,
			quantidade, created_at, updated_at, excluido_em, rev_origem, recebido_em, origem_instalacao, registro_json) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			it.UUID, it.PropriedadeUUID, it.Categoria, it.CodigoCatalogo, it.NomePersonalizado, normalizado, it.Quantidade, ms(it.CreatedAt), ms(it.UpdatedAt),
			msOpt(it.ExcluidoEm), it.Rev, a.agora, a.instalacao, string(js))
	}
	if err != nil {
		return "", err
	}
	return contratos.ResultadoAplicado, a.auditar("item", it.UUID, acaoDe(antes, it.ExcluidoEm), antes, js)
}

func (a *aplicador) foto(f *contratos.Foto, coleta string) (string, error) {
	if err := contratos.ValidarFoto(f); err != nil {
		return "", rejeitar("%v", err)
	}
	if err := a.propriedadeDaColeta(f.PropriedadeUUID, coleta); err != nil {
		return "", err
	}
	rev, antes, corrigido, _, existe, err := a.estadoAtual("fotografias", f.UUID)
	if err != nil {
		return "", err
	}
	if existe && f.Rev <= rev {
		return contratos.ResultadoDuplicado, nil
	}
	var protegido int
	if err := a.tx.QueryRowContext(a.ctx, `SELECT corrigido_desde_rev_origem FROM propriedades WHERE uuid=?`, f.PropriedadeUUID).Scan(&protegido); err != nil {
		return "", err
	}
	if protegido != 0 {
		return "", rejeitar("cadastro corrigido ou excluído na Central; resolução de conflitos ainda não disponível")
	}
	if existe && corrigido {
		return "", rejeitar("foto corrigida na Central; resolução de conflitos ainda não disponível")
	}
	// Foto vigente: o arquivo já foi conferido e publicado antes desta transação.
	var caminho any
	if f.ExcluidoEm == nil {
		caminho = CaminhoRelativoFoto(f.UUID)
	} else if existe {
		var c sql.NullString
		_ = a.tx.QueryRowContext(a.ctx, `SELECT caminho_relativo FROM fotografias WHERE uuid = ?`, f.UUID).Scan(&c)
		if c.Valid {
			caminho = c.String
		}
	}
	js, _ := json.Marshal(f)
	if existe {
		_, err = a.tx.ExecContext(a.ctx, `UPDATE fotografias SET propriedade_uuid = ?, caminho_relativo = ?, mime = ?, tamanho_bytes = ?, sha256 = ?, largura = ?, altura = ?,
			capturada_em = ?, updated_at = ?, excluido_em = ?, rev_origem = ?, recebido_em = ?, origem_instalacao = ?, registro_json = ? WHERE uuid = ?`,
			f.PropriedadeUUID, caminho, f.Mime, f.TamanhoBytes, f.SHA256, f.Largura, f.Altura, ms(f.CapturadaEm), ms(f.UpdatedAt), msOpt(f.ExcluidoEm),
			f.Rev, a.agora, a.instalacao, string(js), f.UUID)
	} else {
		_, err = a.tx.ExecContext(a.ctx, `INSERT INTO fotografias (uuid, propriedade_uuid, caminho_relativo, mime, tamanho_bytes, sha256, largura, altura,
			capturada_em, created_at, updated_at, excluido_em, rev_origem, recebido_em, origem_instalacao, registro_json) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			f.UUID, f.PropriedadeUUID, caminho, f.Mime, f.TamanhoBytes, f.SHA256, f.Largura, f.Altura, ms(f.CapturadaEm), ms(f.CreatedAt), ms(f.UpdatedAt),
			msOpt(f.ExcluidoEm), f.Rev, a.agora, a.instalacao, string(js))
	}
	if err != nil {
		return "", err
	}
	return contratos.ResultadoAplicado, a.auditar("foto", f.UUID, acaoDe(antes, f.ExcluidoEm), antes, js)
}
