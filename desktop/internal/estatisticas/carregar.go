package estatisticas

import (
	"context"
	"database/sql"

	"sitposse7/desktop/internal/dominio"
)

// Carregar lê a base vigente do banco: só as visões v_*_vigentes (RN-80), ou seja,
// a última versão aplicada de cada registro, sem tombstones (22-A.5).
func Carregar(ctx context.Context, db *sql.DB) (*Base, error) {
	b := &Base{}

	// Tipos de item: catálogo ativo + grupos de "Outros" recebidos (pelo nome normalizado).
	catItem := map[string]string{} // chave → categoria
	rows, err := db.QueryContext(ctx, `SELECT codigo, categoria, nome_exibicao FROM catalogo_itens WHERE ativo = 1 AND codigo NOT LIKE '%.outros' ORDER BY categoria, ordem`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c Campo
		var codigo string
		if err := rows.Scan(&codigo, &c.Categoria, &c.Rotulo); err != nil {
			rows.Close()
			return nil, err
		}
		c.Chave, c.Grupo = ChaveItem(codigo, ""), categorias[c.Categoria].grupo
		catItem[c.Chave] = c.Categoria
		b.Itens = append(b.Itens, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Quantidade de cada item por propriedade. O "Outros" usa o primeiro texto original
	// recebido como rótulo do grupo (RN-35: o original nunca é alterado).
	itensPorProp := map[string]map[string]int64{}
	rows, err = db.QueryContext(ctx, `SELECT propriedade_uuid, categoria, codigo_catalogo, nome_personalizado, nome_normalizado, quantidade
		FROM v_itens_vigentes ORDER BY created_at, uuid`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var prop, cat, codigo string
		var pers, norm sql.NullString
		var q int64
		if err := rows.Scan(&prop, &cat, &codigo, &pers, &norm, &q); err != nil {
			rows.Close()
			return nil, err
		}
		chave := ChaveItem(codigo, norm.String)
		if _, ok := catItem[chave]; !ok {
			rotulo := pers.String
			if rotulo == "" {
				rotulo = "Outros"
			}
			catItem[chave] = cat
			b.Itens = append(b.Itens, Campo{Chave: chave, Rotulo: rotulo, Grupo: categorias[cat].grupo, Categoria: cat, Outros: true})
		}
		if itensPorProp[prop] == nil {
			itensPorProp[prop] = map[string]int64{}
		}
		itensPorProp[prop][chave] += q
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = db.QueryContext(ctx, `SELECT uuid, nome FROM estudos WHERE excluido_em IS NULL ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var o Opcao
		if err := rows.Scan(&o.Valor, &o.Rotulo); err != nil {
			rows.Close()
			return nil, err
		}
		b.Estudos = append(b.Estudos, o)
	}
	rows.Close()

	rows, err = db.QueryContext(ctx, `SELECT p.uuid, p.nome_propriedade, p.responsavel_familia, p.observacoes_relevantes,
		p.equipe_uuid, p.equipe_nome, p.equipe_codigo, p.jornada_uuid, p.jornada_numero, p.jornada_data, p.estudo_uuid, p.estudo_nome,
		p.pessoas_adultos, p.pessoas_jovens, p.pessoas_criancas, p.veic_caminhoes, p.veic_automoveis_utilitarios, p.veic_motos_ciclomotores,
		p.bov_adultos, p.bov_bezerros, p.equ_adultos, p.equ_potros, p.area_valor_milesimos, p.area_unidade, p.lat IS NOT NULL, p.lat, p.lon, p.gps_origem, p.gps_precisao_m,
		(SELECT count(*) FROM v_fotos_vigentes f WHERE f.propriedade_uuid = p.uuid), p.created_at, p.recebido_em, coalesce(p.numero_no_estudo, 0)
		FROM v_propriedades_vigentes p
		ORDER BY p.nome_propriedade COLLATE NOCASE, p.uuid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p Familia
		var ad, jo, cr, ca, au, mo, ba, bb, ea, ep, fotos int64
		var mil sql.NullInt64
		var un sql.NullString
		if err := rows.Scan(&p.UUID, &p.Nome, &p.Responsavel, &p.Observacoes, &p.EquipeUUID, &p.Equipe, &p.EquipeCodigo,
			&p.JornadaUUID, &p.JornadaNumero, &p.JornadaData, &p.EstudoUUID, &p.Estudo,
			&ad, &jo, &cr, &ca, &au, &mo, &ba, &bb, &ea, &ep, &mil, &un, &p.ComLocalizacao, &p.Lat, &p.Lon, &p.GpsOrigem, &p.GpsPrecisaoM, &fotos, &p.CriadoEm, &p.RecebidoEm, &p.Numero); err != nil {
			return nil, err
		}
		v := map[string]int64{
			"pessoas_adultos": ad, "pessoas_jovens": jo, "pessoas_criancas": cr,
			"veic_caminhoes": ca, "veic_automoveis_utilitarios": au, "veic_motos_ciclomotores": mo,
			"bov_adultos": ba, "bov_bezerros": bb, "equ_adultos": ea, "equ_potros": ep, Fotos: fotos,
		}
		for k, q := range itensPorProp[p.UUID] {
			v[k] = q
		}
		p.Valores = v
		p.Jornada = RotuloJornada(p.JornadaNumero, p.JornadaData)
		if mil.Valid {
			dm2 := dominio.Area{Milesimos: mil.Int64, Unidade: dominio.UnidadeArea(un.String)}.Dm2()
			ha := dominio.FormatarHa(dm2)
			u := un.String
			p.AreaDm2, p.AreaHa, p.AreaUnidade = &dm2, &ha, &u
		}
		b.Familias = append(b.Familias, NovaFamilia(p, catItem))
	}
	return b, rows.Err()
}
