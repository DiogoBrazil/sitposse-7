// Package consultas faz as leituras da Central para as telas (somente leitura).
// Totais e listas usam só as visões vigentes (RN-80); SQL sempre parametrizado.
// O parâmetro estudo ("" = todos) limita ao estudo em foco (A28).
package consultas

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"sitposse7/desktop/internal/codigos"
	"sitposse7/desktop/internal/dominio"
)

// Consultas lê do pool de leitura.
type Consultas struct{ DB *sql.DB }

// Resumo são os números do painel.
type Resumo struct {
	Equipes          int    `json:"equipes"`
	Aparelhos        int    `json:"aparelhos"`
	Propriedades     int    `json:"propriedades"`
	Pessoas          int64  `json:"pessoas"`
	Veiculos         int64  `json:"veiculos"`
	Bovinos          int64  `json:"bovinos"`
	Equinos          int64  `json:"equinos"`
	Aves             int64  `json:"aves"`
	Eletrodomesticos int64  `json:"eletrodomesticos"`
	Moveis           int64  `json:"moveis"`
	Fotos            int    `json:"fotos"`
	AreaHa           string `json:"area_ha"` // soma das áreas informadas, 2 casas
	ComArea          int    `json:"com_area"`
	SemArea          int    `json:"sem_area"` // RN-45: não entra como zero
	ComGPS           int    `json:"com_gps"`
	LotesImportados  int    `json:"lotes_importados"`
	UltimaImportacao *int64 `json:"ultima_importacao"`
}

// Resumo calcula os números gerais do estudo ("" = de todos).
func (c *Consultas) Resumo(ctx context.Context, estudo string) (*Resumo, error) {
	r := &Resumo{}
	const doEstudo = `(?1 = '' OR p.estudo_uuid = ?1)`
	err := c.DB.QueryRowContext(ctx, `SELECT count(*),
		coalesce(sum(pessoas_adultos + pessoas_jovens + pessoas_criancas), 0),
		coalesce(sum(veic_caminhoes + veic_automoveis_utilitarios + veic_motos_ciclomotores), 0),
		coalesce(sum(bov_adultos + bov_bezerros), 0),
		coalesce(sum(equ_adultos + equ_potros), 0),
		coalesce(sum(lat IS NOT NULL), 0)
		FROM v_propriedades_vigentes p WHERE `+doEstudo, estudo).Scan(&r.Propriedades, &r.Pessoas, &r.Veiculos, &r.Bovinos, &r.Equinos, &r.ComGPS)
	if err != nil {
		return nil, err
	}
	err = c.DB.QueryRowContext(ctx, `SELECT
		coalesce(sum(CASE WHEN i.categoria = 'ave' THEN i.quantidade END), 0),
		coalesce(sum(CASE WHEN i.categoria = 'eletrodomestico' THEN i.quantidade END), 0),
		coalesce(sum(CASE WHEN i.categoria = 'movel' THEN i.quantidade END), 0)
		FROM v_itens_vigentes i JOIN v_propriedades_vigentes p ON p.uuid = i.propriedade_uuid WHERE `+doEstudo, estudo).Scan(&r.Aves, &r.Eletrodomesticos, &r.Moveis)
	if err != nil {
		return nil, err
	}
	if err := c.DB.QueryRowContext(ctx, `SELECT count(*) FROM v_fotos_vigentes f JOIN v_propriedades_vigentes p ON p.uuid = f.propriedade_uuid WHERE `+doEstudo, estudo).Scan(&r.Fotos); err != nil {
		return nil, err
	}
	if err := c.DB.QueryRowContext(ctx, `SELECT count(*) FROM equipes e JOIN jornadas j ON j.uuid = e.jornada_uuid
		WHERE e.excluido_em IS NULL AND j.excluido_em IS NULL AND (?1 = '' OR j.estudo_uuid = ?1)`, estudo).Scan(&r.Equipes); err != nil {
		return nil, err
	}
	if err := c.DB.QueryRowContext(ctx, `SELECT count(*) FROM dispositivos WHERE revogado_em IS NULL`).Scan(&r.Aparelhos); err != nil {
		return nil, err
	}
	var ultima sql.NullInt64
	if err := c.DB.QueryRowContext(ctx, `SELECT count(*), max(l.importado_em) FROM lotes_recebidos l
		LEFT JOIN equipes e ON e.uuid = l.equipe_destino_uuid LEFT JOIN jornadas j ON j.uuid = e.jornada_uuid
		WHERE l.status = 'importado' AND (?1 = '' OR j.estudo_uuid = ?1)`, estudo).Scan(&r.LotesImportados, &ultima); err != nil {
		return nil, err
	}
	if ultima.Valid {
		r.UltimaImportacao = &ultima.Int64
	}
	// Área: soma exata em dm² (RN-44/45), formatada só na exibição.
	rows, err := c.DB.QueryContext(ctx, `SELECT area_valor_milesimos, area_unidade FROM v_propriedades_vigentes p WHERE `+doEstudo, estudo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var areas []*dominio.Area
	for rows.Next() {
		var m sql.NullInt64
		var u sql.NullString
		if err := rows.Scan(&m, &u); err != nil {
			return nil, err
		}
		if !m.Valid {
			areas = append(areas, nil)
			continue
		}
		areas = append(areas, &dominio.Area{Milesimos: m.Int64, Unidade: dominio.UnidadeArea(u.String)})
	}
	total, com, sem := dominio.SomaAreas(areas)
	r.AreaHa, r.ComArea, r.SemArea = dominio.FormatarHa(total), com, sem
	return r, rows.Err()
}

// PropriedadeLinha é uma linha da tabela de propriedades.
type PropriedadeLinha struct {
	UUID         string  `json:"uuid"`
	Numero       int     `json:"numero"` // número fixo no estudo (RN-96)
	Nome         string  `json:"nome"`
	Responsavel  string  `json:"responsavel"`
	EquipeUUID   string  `json:"equipe_uuid"`
	Equipe       string  `json:"equipe"`
	Jornada      string  `json:"jornada"` // "Jornada 2 · 2026-10-11"
	JornadaData  string  `json:"jornada_data"`
	Pessoas      int64   `json:"pessoas"`
	Bovinos      int64   `json:"bovinos"`
	AreaHa       *string `json:"area_ha"`
	TemGPS       bool    `json:"tem_gps"`
	Fotos        int     `json:"fotos"`
	AtualizadoEm int64   `json:"atualizado_em"`
	RecebidoEm   int64   `json:"recebido_em"`
}

// Propriedades lista as propriedades vigentes; busca por nome da propriedade ou do
// responsável, sem diferenciar maiúsculas nem acentos (mesma normalização do RN-35).
// O filtro é feito em Go: o volume é pequeno (A13) e o SQLite não remove acentos.
func (c *Consultas) Propriedades(ctx context.Context, busca, estudo, equipe string) ([]PropriedadeLinha, error) {
	q := `SELECT p.uuid, p.nome_propriedade, p.responsavel_familia, p.equipe_uuid, p.equipe_nome, p.jornada_numero, p.jornada_data,
		p.pessoas_adultos + p.pessoas_jovens + p.pessoas_criancas, p.bov_adultos + p.bov_bezerros,
		p.area_valor_milesimos, p.area_unidade, p.lat IS NOT NULL,
		(SELECT count(*) FROM v_fotos_vigentes f WHERE f.propriedade_uuid = p.uuid), p.updated_at, p.recebido_em, coalesce(p.numero_no_estudo, 0)
		FROM v_propriedades_vigentes p WHERE 1 = 1`
	var args []any
	termo, temBusca := dominio.NormalizarNome(busca)
	if estudo != "" {
		q += ` AND p.estudo_uuid = ?`
		args = append(args, estudo)
	}
	if equipe != "" {
		q += ` AND p.equipe_uuid = ?`
		args = append(args, equipe)
	}
	q += ` ORDER BY p.nome_propriedade COLLATE NOCASE, p.uuid`
	rows, err := c.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PropriedadeLinha{}
	for rows.Next() {
		var l PropriedadeLinha
		var m sql.NullInt64
		var u sql.NullString
		var nj int
		if err := rows.Scan(&l.UUID, &l.Nome, &l.Responsavel, &l.EquipeUUID, &l.Equipe, &nj, &l.JornadaData, &l.Pessoas, &l.Bovinos, &m, &u, &l.TemGPS, &l.Fotos, &l.AtualizadoEm, &l.RecebidoEm, &l.Numero); err != nil {
			return nil, err
		}
		l.Jornada = RotuloJornada(nj, l.JornadaData)
		l.AreaHa = areaHa(m, u)
		if temBusca {
			nome, _ := dominio.NormalizarNome(l.Nome)
			resp, _ := dominio.NormalizarNome(l.Responsavel)
			if !strings.Contains(nome, termo) && !strings.Contains(resp, termo) {
				continue
			}
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// RotuloJornada é o texto padrão de uma jornada: "Jornada 2 · 11/10/2026".
func RotuloJornada(numero int, data string) string {
	return "Jornada " + strconv.Itoa(numero) + " · " + DataBR(data)
}

// DataBR converte AAAA-MM-DD em DD/MM/AAAA (texto inválido volta como veio).
func DataBR(data string) string {
	if len(data) != 10 || data[4] != '-' || data[7] != '-' {
		return data
	}
	return data[8:] + "/" + data[5:7] + "/" + data[:4]
}

func areaHa(m sql.NullInt64, u sql.NullString) *string {
	if !m.Valid {
		return nil
	}
	s := dominio.FormatarHa(dominio.Area{Milesimos: m.Int64, Unidade: dominio.UnidadeArea(u.String)}.Dm2())
	return &s
}

// ItemLinha é um item dinâmico na tela de detalhes.
type ItemLinha struct {
	UUID       string `json:"uuid"`
	Categoria  string `json:"categoria"`
	Codigo     string `json:"codigo"`
	Nome       string `json:"nome"` // nome do catálogo ou nome livre original de "Outros"
	Outros     bool   `json:"outros"`
	Quantidade int64  `json:"quantidade"`
}

// FotoLinha são os metadados de uma foto vigente.
type FotoLinha struct {
	UUID         string `json:"uuid"`
	URL          string `json:"url"` // servida pela própria Central (/fotos/{uuid}.jpg)
	Largura      int64  `json:"largura"`
	Altura       int64  `json:"altura"`
	TamanhoBytes int64  `json:"tamanho_bytes"`
	SHA256       string `json:"sha256"`
	CapturadaEm  int64  `json:"capturada_em"`
}

// PropriedadeDetalhe é o cadastro completo.
type PropriedadeDetalhe struct {
	UUID                      string      `json:"uuid"`
	Numero                    int         `json:"numero"` // número fixo no estudo (RN-96)
	Nome                      string      `json:"nome"`
	Responsavel               string      `json:"responsavel"`
	Observacoes               *string     `json:"observacoes"`
	Equipe                    string      `json:"equipe"`
	EquipeCodigo              string      `json:"equipe_codigo"`
	JornadaNumero             int         `json:"jornada_numero"`
	JornadaData               string      `json:"jornada_data"`
	EstudoUUID                string      `json:"estudo_uuid"`
	Estudo                    string      `json:"estudo"`
	EstudoCodigo              string      `json:"estudo_codigo"`
	PessoasAdultos            int64       `json:"pessoas_adultos"`
	PessoasJovens             int64       `json:"pessoas_jovens"`
	PessoasCriancas           int64       `json:"pessoas_criancas"`
	VeicCaminhoes             int64       `json:"veic_caminhoes"`
	VeicAutomoveisUtilitarios int64       `json:"veic_automoveis_utilitarios"`
	VeicMotosCiclomotores     int64       `json:"veic_motos_ciclomotores"`
	BovAdultos                int64       `json:"bov_adultos"`
	BovBezerros               int64       `json:"bov_bezerros"`
	EquAdultos                int64       `json:"equ_adultos"`
	EquPotros                 int64       `json:"equ_potros"`
	AreaMilesimos             *int64      `json:"area_milesimos"`
	AreaUnidade               *string     `json:"area_unidade"`
	AreaHa                    *string     `json:"area_ha"`
	Lat                       *float64    `json:"lat"`
	Lon                       *float64    `json:"lon"`
	GPSOrigem                 *string     `json:"gps_origem"`
	GPSPrecisaoM              *float64    `json:"gps_precisao_m"`
	GPSCapturadoEm            *int64      `json:"gps_capturado_em"`
	GPSSatelites              *int64      `json:"gps_satelites"`
	CriadoEm                  int64       `json:"criado_em"`
	AtualizadoEm              int64       `json:"atualizado_em"`
	RecebidoEm                int64       `json:"recebido_em"`
	Rev                       int64       `json:"rev"`
	Itens                     []ItemLinha `json:"itens"`
	Fotos                     []FotoLinha `json:"fotos"`
}

// Propriedade devolve o detalhe de uma propriedade vigente (nil se não existir).
func (c *Consultas) Propriedade(ctx context.Context, uuid string) (*PropriedadeDetalhe, error) {
	d := &PropriedadeDetalhe{Itens: []ItemLinha{}, Fotos: []FotoLinha{}}
	var m sql.NullInt64
	var u sql.NullString
	err := c.DB.QueryRowContext(ctx, `SELECT p.uuid, p.nome_propriedade, p.responsavel_familia, p.observacoes_relevantes,
		p.equipe_nome, p.equipe_codigo, p.jornada_numero, p.jornada_data, p.estudo_uuid, p.estudo_nome, p.estudo_codigo,
		p.pessoas_adultos, p.pessoas_jovens, p.pessoas_criancas, p.veic_caminhoes, p.veic_automoveis_utilitarios, p.veic_motos_ciclomotores,
		p.bov_adultos, p.bov_bezerros, p.equ_adultos, p.equ_potros, p.area_valor_milesimos, p.area_unidade,
		p.lat, p.lon, p.gps_origem, p.gps_precisao_m, p.gps_capturado_em, p.gps_satelites, p.created_at, p.updated_at, p.recebido_em, p.rev_origem, coalesce(p.numero_no_estudo, 0)
		FROM v_propriedades_vigentes p WHERE p.uuid = ?`, uuid).Scan(
		&d.UUID, &d.Nome, &d.Responsavel, &d.Observacoes, &d.Equipe, &d.EquipeCodigo, &d.JornadaNumero, &d.JornadaData, &d.EstudoUUID, &d.Estudo, &d.EstudoCodigo,
		&d.PessoasAdultos, &d.PessoasJovens, &d.PessoasCriancas, &d.VeicCaminhoes, &d.VeicAutomoveisUtilitarios, &d.VeicMotosCiclomotores,
		&d.BovAdultos, &d.BovBezerros, &d.EquAdultos, &d.EquPotros, &m, &u,
		&d.Lat, &d.Lon, &d.GPSOrigem, &d.GPSPrecisaoM, &d.GPSCapturadoEm, &d.GPSSatelites, &d.CriadoEm, &d.AtualizadoEm, &d.RecebidoEm, &d.Rev, &d.Numero)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.EquipeCodigo, d.EstudoCodigo = codigos.Exibir(d.EquipeCodigo), codigos.Exibir(d.EstudoCodigo)
	if m.Valid {
		d.AreaMilesimos = &m.Int64
		d.AreaUnidade = &u.String
	}
	d.AreaHa = areaHa(m, u)
	rows, err := c.DB.QueryContext(ctx, `SELECT i.uuid, i.categoria, i.codigo_catalogo, coalesce(i.nome_personalizado, c.nome_exibicao), i.nome_personalizado IS NOT NULL, i.quantidade
		FROM v_itens_vigentes i JOIN catalogo_itens c ON c.codigo = i.codigo_catalogo WHERE i.propriedade_uuid = ?
		ORDER BY i.categoria, c.ordem, i.nome_personalizado COLLATE NOCASE`, uuid)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var it ItemLinha
		if err := rows.Scan(&it.UUID, &it.Categoria, &it.Codigo, &it.Nome, &it.Outros, &it.Quantidade); err != nil {
			rows.Close()
			return nil, err
		}
		d.Itens = append(d.Itens, it)
	}
	rows.Close()
	rows, err = c.DB.QueryContext(ctx, `SELECT uuid, largura, altura, tamanho_bytes, sha256, capturada_em FROM v_fotos_vigentes WHERE propriedade_uuid = ? ORDER BY capturada_em, uuid`, uuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f FotoLinha
		if err := rows.Scan(&f.UUID, &f.Largura, &f.Altura, &f.TamanhoBytes, &f.SHA256, &f.CapturadaEm); err != nil {
			return nil, err
		}
		f.URL = "/fotos/" + f.UUID + ".jpg"
		d.Fotos = append(d.Fotos, f)
	}
	return d, rows.Err()
}

// CaminhoFotoVigente devolve o caminho relativo do arquivo de uma foto vigente ("" se não houver).
func (c *Consultas) CaminhoFotoVigente(ctx context.Context, uuid string) (string, error) {
	var cam string
	err := c.DB.QueryRowContext(ctx, `SELECT caminho_relativo FROM v_fotos_vigentes WHERE uuid = ?`, uuid).Scan(&cam)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return cam, err
}

// LoteLinha é uma linha do histórico de recepção.
type LoteLinha struct {
	UUID            string  `json:"uuid"`
	CodigoInformado string  `json:"codigo_informado"` // como veio do celular (exibição com hífens)
	Estudo          *string `json:"estudo"`           // destino (nulo enquanto não decidido ou se cancelado)
	Jornada         *string `json:"jornada"`
	Equipe          *string `json:"equipe"`
	EquipeCodigo    *string `json:"equipe_codigo"`
	Corrigido       bool    `json:"corrigido"` // destino diferente do código recebido
	Motivo          *string `json:"motivo"`    // motivo do cancelamento
	Aparelho        *string `json:"aparelho"`
	Status          string  `json:"status"`
	RecebidoEm      int64   `json:"recebido_em"`
	ImportadoEm     *int64  `json:"importado_em"`
	Aplicados       int     `json:"aplicados"`
	Duplicados      int     `json:"duplicados"`
	Rejeitados      int     `json:"rejeitados"`
	Conflitos       int     `json:"conflitos"`
}

// Lotes lista os últimos lotes recebidos (mais recentes primeiro).
func (c *Consultas) Lotes(ctx context.Context, limite int) ([]LoteLinha, error) {
	if limite <= 0 || limite > 500 {
		limite = 100
	}
	rows, err := c.DB.QueryContext(ctx, `SELECT l.uuid, l.codigo_informado, s.nome, j.numero, j.data, e.nome, e.codigo, l.motivo_cancelamento,
		d.rotulo_aparelho, l.status, l.recebido_em, l.importado_em,
		(SELECT count(*) FROM aplicacoes a WHERE a.lote_uuid = l.uuid AND a.resultado = 'aplicado'),
		(SELECT count(*) FROM aplicacoes a WHERE a.lote_uuid = l.uuid AND a.resultado = 'duplicado'),
		(SELECT count(*) FROM aplicacoes a WHERE a.lote_uuid = l.uuid AND a.resultado = 'rejeitado'),
		(SELECT count(*) FROM aplicacoes a WHERE a.lote_uuid = l.uuid AND a.resultado = 'conflito')
		FROM lotes_recebidos l LEFT JOIN equipes e ON e.uuid = l.equipe_destino_uuid
		LEFT JOIN jornadas j ON j.uuid = e.jornada_uuid LEFT JOIN estudos s ON s.uuid = j.estudo_uuid
		LEFT JOIN dispositivos d ON d.instalacao_uuid = l.instalacao_uuid
		ORDER BY l.recebido_em DESC LIMIT ?`, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LoteLinha{}
	for rows.Next() {
		var l LoteLinha
		var jn sql.NullInt64
		var jd sql.NullString
		if err := rows.Scan(&l.UUID, &l.CodigoInformado, &l.Estudo, &jn, &jd, &l.Equipe, &l.EquipeCodigo, &l.Motivo,
			&l.Aparelho, &l.Status, &l.RecebidoEm, &l.ImportadoEm, &l.Aplicados, &l.Duplicados, &l.Rejeitados, &l.Conflitos); err != nil {
			return nil, err
		}
		l.Corrigido = l.EquipeCodigo != nil && *l.EquipeCodigo != l.CodigoInformado
		l.CodigoInformado = codigos.Exibir(l.CodigoInformado)
		if l.EquipeCodigo != nil {
			e := codigos.Exibir(*l.EquipeCodigo)
			l.EquipeCodigo = &e
		}
		if jn.Valid {
			r := RotuloJornada(int(jn.Int64), jd.String)
			l.Jornada = &r
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// Rejeicao é um registro rejeitado num lote (mostrado ao operador).
type Rejeicao struct {
	Entidade string `json:"entidade"`
	UUID     string `json:"uuid"`
	Motivo   string `json:"motivo"`
}

// Rejeicoes lista os registros rejeitados de um lote.
func (c *Consultas) Rejeicoes(ctx context.Context, lote string) ([]Rejeicao, error) {
	rows, err := c.DB.QueryContext(ctx, `SELECT entidade, entidade_uuid, coalesce(motivo, '') FROM aplicacoes WHERE lote_uuid = ? AND resultado = 'rejeitado' ORDER BY entidade, entidade_uuid`, lote)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Rejeicao{}
	for rows.Next() {
		var r Rejeicao
		if err := rows.Scan(&r.Entidade, &r.UUID, &r.Motivo); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
