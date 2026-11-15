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
)

// aplicar grava o lote numa única transação e devolve o recibo persistido.
// Cada registro é aplicado num SAVEPOINT: um registro rejeitado não derruba os demais.
// Regra de aplicação (A24, sem correções na Central): rev > rev_origem → aplicado;
// rev ≤ rev_origem → duplicado. Nada é sobrescrito com rev menor ou igual.
func (im *Importador) aplicar(ctx context.Context, d Dispositivo, l *contratos.Lote) (*contratos.Recibo, error) {
	agora := im.agora().UnixMilli()
	tx, err := im.B.Escrita.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	a := &aplicador{ctx: ctx, tx: tx, d: d, agora: agora, origem: "equipe:" + d.EquipeUUID, lote: l.LoteUUID}

	resultados := map[string]contratos.ResultadoRecibo{}
	chave := func(ent, uuid string) string { return ent + "|" + uuid }
	registrar := func(ent, uuid string, rev int64, res string, motivo string) {
		resultados[chave(ent, uuid)] = contratos.ResultadoRecibo{Entidade: ent, UUID: uuid, Rev: rev, Resultado: res, Motivo: motivo}
	}

	res, motivo := a.comSavepoint(func() (string, error) { return a.equipe(&l.Equipe) })
	registrar("equipe", l.Equipe.UUID, l.Equipe.Rev, res, motivo)
	for i := range l.Propriedades {
		p := &l.Propriedades[i]
		res, motivo := a.comSavepoint(func() (string, error) { return a.propriedade(p, l.Equipe.UUID) })
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
		res, motivo := a.comSavepoint(func() (string, error) { return a.item(it, l.Equipe.UUID) })
		registrar("item", it.UUID, it.Rev, res, motivo)
	}
	for i := range l.Fotos {
		f := &l.Fotos[i]
		res, motivo := a.comSavepoint(func() (string, error) { return a.foto(f, l.Equipe.UUID) })
		registrar("foto", f.UUID, f.Rev, res, motivo)
	}

	// Recibo na ordem do manifesto.
	recibo := &contratos.Recibo{ProtocoloVersao: contratos.ProtocoloVersao, LoteUUID: l.LoteUUID, CentralID: im.CentralID,
		ImportadoEm: contratos.FormatarInstante(agora)}
	add := func(ent, uuid string) {
		r := resultados[chave(ent, uuid)]
		recibo.Resultados = append(recibo.Resultados, r)
	}
	add("equipe", l.Equipe.UUID)
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
	if _, err := tx.ExecContext(ctx, `UPDATE lotes_recebidos SET status = 'importado', recibo_json = ?, importado_em = ? WHERE uuid = ? AND status = 'recebendo'`,
		string(reciboJSON), agora, l.LoteUUID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gravando importação: %w", err)
	}
	return recibo, nil
}

type aplicador struct {
	ctx    context.Context
	tx     *sql.Tx
	d      Dispositivo
	agora  int64
	origem string
	lote   string
	n      int
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

// estadoAtual lê rev_origem e registro_json da entidade (ok=false se não existe).
func (a *aplicador) estadoAtual(tabela, uuid string) (rev int64, registro string, corrigido bool, equipe string, ok bool, err error) {
	var q string
	switch tabela {
	case "equipes":
		q = `SELECT rev_origem, registro_json, 0, uuid FROM equipes WHERE uuid = ?`
	case "propriedades":
		q = `SELECT rev_origem, registro_json, corrigido_desde_rev_origem, equipe_uuid FROM propriedades WHERE uuid = ?`
	case "itens_propriedade":
		q = `SELECT i.rev_origem, i.registro_json, i.corrigido_desde_rev_origem, p.equipe_uuid FROM itens_propriedade i JOIN propriedades p ON p.uuid = i.propriedade_uuid WHERE i.uuid = ?`
	case "fotografias":
		q = `SELECT f.rev_origem, f.registro_json, f.corrigido_desde_rev_origem, p.equipe_uuid FROM fotografias f JOIN propriedades p ON p.uuid = f.propriedade_uuid WHERE f.uuid = ?`
	default:
		return 0, "", false, "", false, fmt.Errorf("tabela desconhecida %s", tabela)
	}
	var c int64
	err = a.tx.QueryRowContext(a.ctx, q, uuid).Scan(&rev, &registro, &c, &equipe)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, "", false, nil
	}
	return rev, registro, c == 1, equipe, err == nil, err
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

func (a *aplicador) equipe(e *contratos.Equipe) (string, error) {
	if err := contratos.ValidarEquipe(e); err != nil {
		return "", rejeitar("%v", err)
	}
	rev, antes, _, _, existe, err := a.estadoAtual("equipes", e.UUID)
	if err != nil {
		return "", err
	}
	if existe && e.Rev <= rev {
		return contratos.ResultadoDuplicado, nil
	}
	js, _ := json.Marshal(e)
	if existe {
		_, err = a.tx.ExecContext(a.ctx, `UPDATE equipes SET nome = ?, responsavel = ?, identificacao_area = ?, updated_at = ?, rev_origem = ?, recebido_em = ?, registro_json = ?
			WHERE uuid = ? AND operacao_uuid = ?`, e.Nome, e.Responsavel, e.IdentificacaoArea, ms(e.UpdatedAt), e.Rev, a.agora, string(js), e.UUID, a.d.OperacaoUUID)
	} else {
		// A equipe é criada no pareamento; este ramo só cobre bancos antigos.
		_, err = a.tx.ExecContext(a.ctx, `INSERT INTO equipes (uuid, operacao_uuid, nome, responsavel, identificacao_area, created_at, updated_at, rev_origem, recebido_em, registro_json)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.UUID, a.d.OperacaoUUID, e.Nome, e.Responsavel, e.IdentificacaoArea, ms(e.CreatedAt), ms(e.UpdatedAt), e.Rev, a.agora, string(js))
	}
	if err != nil {
		return "", err
	}
	return contratos.ResultadoAplicado, a.auditar("equipe", e.UUID, acaoDe(antes, nil), antes, js)
}

func (a *aplicador) propriedade(p *contratos.Propriedade, equipe string) (string, error) {
	if err := contratos.ValidarPropriedade(p); err != nil {
		return "", rejeitar("%v", err)
	}
	if p.EquipeUUID != equipe {
		return "", rejeitar("propriedade de outra equipe")
	}
	rev, antes, corrigido, equipeAtual, existe, err := a.estadoAtual("propriedades", p.UUID)
	if err != nil {
		return "", err
	}
	if existe && equipeAtual != equipe {
		return "", rejeitar("UUID já pertence a cadastro de outra equipe")
	}
	if existe && p.Rev <= rev {
		return contratos.ResultadoDuplicado, nil
	}
	if existe && corrigido {
		// Não ocorre no MVP (a Central não corrige cadastros). Nunca sobrescrever em silêncio.
		return "", rejeitar("cadastro corrigido na Central; resolução de conflitos ainda não disponível")
	}
	js, _ := json.Marshal(p)
	args := []any{p.NomePropriedade, p.ResponsavelFamilia,
		p.PessoasAdultos, p.PessoasJovens, p.PessoasCriancas, p.VeicCaminhoes, p.VeicAutomoveisUtilitarios, p.VeicMotosCiclomotores,
		p.BovAdultos, p.BovBezerros, p.EquAdultos, p.EquPotros, p.AreaValorMilesimos, p.AreaUnidade,
		p.Lat, p.Lon, p.GPSOrigem, p.GPSPrecisaoM, msOpt(p.GPSCapturadoEm), p.GPSSatelites,
		ms(p.UpdatedAt), msOpt(p.ExcluidoEm), p.Rev, a.agora, a.d.InstalacaoUUID, string(js)}
	if existe {
		_, err = a.tx.ExecContext(a.ctx, `UPDATE propriedades SET nome_propriedade = ?, responsavel_familia = ?,
			pessoas_adultos = ?, pessoas_jovens = ?, pessoas_criancas = ?, veic_caminhoes = ?, veic_automoveis_utilitarios = ?, veic_motos_ciclomotores = ?,
			bov_adultos = ?, bov_bezerros = ?, equ_adultos = ?, equ_potros = ?, area_valor_milesimos = ?, area_unidade = ?,
			lat = ?, lon = ?, gps_origem = ?, gps_precisao_m = ?, gps_capturado_em = ?, gps_satelites = ?,
			updated_at = ?, excluido_em = ?, rev_origem = ?, recebido_em = ?, origem_instalacao = ?, registro_json = ?
			WHERE uuid = ?`, append(args, p.UUID)...)
	} else {
		_, err = a.tx.ExecContext(a.ctx, `INSERT INTO propriedades (nome_propriedade, responsavel_familia,
			pessoas_adultos, pessoas_jovens, pessoas_criancas, veic_caminhoes, veic_automoveis_utilitarios, veic_motos_ciclomotores,
			bov_adultos, bov_bezerros, equ_adultos, equ_potros, area_valor_milesimos, area_unidade,
			lat, lon, gps_origem, gps_precisao_m, gps_capturado_em, gps_satelites,
			updated_at, excluido_em, rev_origem, recebido_em, origem_instalacao, registro_json,
			uuid, equipe_uuid, created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			append(args, p.UUID, equipe, ms(p.CreatedAt))...)
	}
	if err != nil {
		return "", err
	}
	return contratos.ResultadoAplicado, a.auditar("propriedade", p.UUID, acaoDe(antes, p.ExcluidoEm), antes, js)
}

// propriedadeDaEquipe confere que a propriedade existe na Central e é da equipe.
func (a *aplicador) propriedadeDaEquipe(uuid, equipe string) error {
	var eq string
	err := a.tx.QueryRowContext(a.ctx, `SELECT equipe_uuid FROM propriedades WHERE uuid = ?`, uuid).Scan(&eq)
	if errors.Is(err, sql.ErrNoRows) {
		return rejeitar("propriedade %s não recebida", uuid)
	}
	if err != nil {
		return err
	}
	if eq != equipe {
		return rejeitar("propriedade de outra equipe")
	}
	return nil
}

func (a *aplicador) item(it *contratos.Item, equipe string) (string, error) {
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
	if err := a.propriedadeDaEquipe(it.PropriedadeUUID, equipe); err != nil {
		return "", err
	}
	rev, antes, corrigido, _, existe, err := a.estadoAtual("itens_propriedade", it.UUID)
	if err != nil {
		return "", err
	}
	if existe && it.Rev <= rev {
		return contratos.ResultadoDuplicado, nil
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
			it.Rev, a.agora, a.d.InstalacaoUUID, string(js), it.UUID)
	} else {
		_, err = a.tx.ExecContext(a.ctx, `INSERT INTO itens_propriedade (uuid, propriedade_uuid, categoria, codigo_catalogo, nome_personalizado, nome_normalizado,
			quantidade, created_at, updated_at, excluido_em, rev_origem, recebido_em, origem_instalacao, registro_json) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			it.UUID, it.PropriedadeUUID, it.Categoria, it.CodigoCatalogo, it.NomePersonalizado, normalizado, it.Quantidade, ms(it.CreatedAt), ms(it.UpdatedAt),
			msOpt(it.ExcluidoEm), it.Rev, a.agora, a.d.InstalacaoUUID, string(js))
	}
	if err != nil {
		return "", err
	}
	return contratos.ResultadoAplicado, a.auditar("item", it.UUID, acaoDe(antes, it.ExcluidoEm), antes, js)
}

func (a *aplicador) foto(f *contratos.Foto, equipe string) (string, error) {
	if err := contratos.ValidarFoto(f); err != nil {
		return "", rejeitar("%v", err)
	}
	if err := a.propriedadeDaEquipe(f.PropriedadeUUID, equipe); err != nil {
		return "", err
	}
	rev, antes, corrigido, _, existe, err := a.estadoAtual("fotografias", f.UUID)
	if err != nil {
		return "", err
	}
	if existe && f.Rev <= rev {
		return contratos.ResultadoDuplicado, nil
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
			f.Rev, a.agora, a.d.InstalacaoUUID, string(js), f.UUID)
	} else {
		_, err = a.tx.ExecContext(a.ctx, `INSERT INTO fotografias (uuid, propriedade_uuid, caminho_relativo, mime, tamanho_bytes, sha256, largura, altura,
			capturada_em, created_at, updated_at, excluido_em, rev_origem, recebido_em, origem_instalacao, registro_json) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			f.UUID, f.PropriedadeUUID, caminho, f.Mime, f.TamanhoBytes, f.SHA256, f.Largura, f.Altura, ms(f.CapturadaEm), ms(f.CreatedAt), ms(f.UpdatedAt),
			msOpt(f.ExcluidoEm), f.Rev, a.agora, a.d.InstalacaoUUID, string(js))
	}
	if err != nil {
		return "", err
	}
	return contratos.ResultadoAplicado, a.auditar("foto", f.UUID, acaoDe(antes, f.ExcluidoEm), antes, js)
}
