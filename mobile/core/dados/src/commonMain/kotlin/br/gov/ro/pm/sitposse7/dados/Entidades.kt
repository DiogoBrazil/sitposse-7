package br.gov.ro.pm.sitposse7.dados

import androidx.room3.ColumnInfo
import androidx.room3.Entity
import androidx.room3.ForeignKey
import androidx.room3.Index
import androidx.room3.PrimaryKey

/*
 * Esquema v3 do banco do celular (A28, ADR-0024/0025; docs/architecture/modelo-de-dados.md).
 * A v3 é uma nova linha de base: bancos v1/v2 (testes do MVP) são recusados por
 * [conferirGeracao], sem migração nem perda silenciosa.
 * Convenções: UUID v4 em texto; instantes em epoch ms UTC (ADR-0010); exclusão = tombstone
 * (`excluido_em`); toda escrita de entidade sincronizável incrementa `rev` na mesma transação;
 * pendente de envio = `rev > rev_confirmada`.
 * O Room não declara CHECK nem índice parcial: as regras de domínio (RN-xx) são garantidas
 * pelos repositórios, testadas na JVM, e conferidas de novo pela Central na importação.
 */

/**
 * Coleta: cadastros feitos com um código de equipe (RN-05, RN-07). O código é a única
 * identificação da equipe no celular; a Central diz o destino na sincronização.
 * `rev` cobre só o que vai para a Central (código e datas); o estado é local.
 */
@Entity(tableName = "coletas", indices = [Index("codigo_equipe")])
data class ColetaEntidade(
    @PrimaryKey val uuid: String,
    /** Código atual, normalizado (EQ + 8). */
    @ColumnInfo(name = "codigo_equipe") val codigoEquipe: String,
    /** Primeiro código informado (rastreabilidade). */
    @ColumnInfo(name = "codigo_original") val codigoOriginal: String,
    /** [StatusColeta]. */
    val status: String,
    @ColumnInfo(name = "iniciada_em") val iniciadaEm: Long,
    @ColumnInfo(name = "concluida_em") val concluidaEm: Long? = null,
    /** Preenchidos pelo recibo da primeira importação (destino travado, RN-09). */
    @ColumnInfo(name = "destino_confirmado_em") val destinoConfirmadoEm: Long? = null,
    @ColumnInfo(name = "destino_json") val destinoJson: String? = null,
    @ColumnInfo(name = "created_at") val createdAt: Long,
    @ColumnInfo(name = "updated_at") val updatedAt: Long,
    val rev: Long,
    @ColumnInfo(name = "rev_confirmada", defaultValue = "0") val revConfirmada: Long = 0,
)

object StatusColeta {
    const val EM_ANDAMENTO = "em_andamento"
    const val CONCLUIDA = "concluida"
}

/** Histórico das correções do código de uma coleta (somente acréscimo; RN-08). */
@Entity(
    tableName = "coleta_codigos",
    foreignKeys = [ForeignKey(entity = ColetaEntidade::class, parentColumns = ["uuid"], childColumns = ["coleta_uuid"])],
    indices = [Index("coleta_uuid")],
)
data class ColetaCodigoEntidade(
    @PrimaryKey val uuid: String,
    @ColumnInfo(name = "coleta_uuid") val coletaUuid: String,
    @ColumnInfo(name = "codigo_anterior") val codigoAnterior: String,
    @ColumnInfo(name = "codigo_novo") val codigoNovo: String,
    @ColumnInfo(name = "alterado_em") val alteradoEm: Long,
)

@Entity(
    tableName = "propriedades",
    foreignKeys = [ForeignKey(entity = ColetaEntidade::class, parentColumns = ["uuid"], childColumns = ["coleta_uuid"])],
    indices = [Index("coleta_uuid")],
)
data class PropriedadeEntidade(
    @PrimaryKey val uuid: String,
    @ColumnInfo(name = "coleta_uuid") val coletaUuid: String,
    @ColumnInfo(name = "nome_propriedade") val nomePropriedade: String,
    /** Nome do responsável pela propriedade (obrigatório, RN-11). */
    @ColumnInfo(name = "responsavel_familia") val responsavelFamilia: String,
    /** Observações relevantes (RN-15). */
    @ColumnInfo(name = "observacoes_relevantes") val observacoesRelevantes: String? = null,
    @ColumnInfo(name = "pessoas_adultos", defaultValue = "0") val pessoasAdultos: Long = 0,
    @ColumnInfo(name = "pessoas_jovens", defaultValue = "0") val pessoasJovens: Long = 0,
    @ColumnInfo(name = "pessoas_criancas", defaultValue = "0") val pessoasCriancas: Long = 0,
    @ColumnInfo(name = "veic_caminhoes", defaultValue = "0") val veicCaminhoes: Long = 0,
    @ColumnInfo(name = "veic_automoveis_utilitarios", defaultValue = "0") val veicAutomoveisUtilitarios: Long = 0,
    @ColumnInfo(name = "veic_motos_ciclomotores", defaultValue = "0") val veicMotosCiclomotores: Long = 0,
    @ColumnInfo(name = "bov_adultos", defaultValue = "0") val bovAdultos: Long = 0,
    @ColumnInfo(name = "bov_bezerros", defaultValue = "0") val bovBezerros: Long = 0,
    @ColumnInfo(name = "equ_adultos", defaultValue = "0") val equAdultos: Long = 0,
    @ColumnInfo(name = "equ_potros", defaultValue = "0") val equPotros: Long = 0,
    @ColumnInfo(name = "area_valor_milesimos") val areaValorMilesimos: Long? = null,
    @ColumnInfo(name = "area_unidade") val areaUnidade: String? = null,
    val lat: Double? = null,
    val lon: Double? = null,
    @ColumnInfo(name = "gps_origem") val gpsOrigem: String? = null,
    @ColumnInfo(name = "gps_precisao_m") val gpsPrecisaoM: Double? = null,
    @ColumnInfo(name = "gps_capturado_em") val gpsCapturadoEm: Long? = null,
    @ColumnInfo(name = "gps_satelites") val gpsSatelites: Long? = null,
    @ColumnInfo(name = "created_at") val createdAt: Long,
    @ColumnInfo(name = "updated_at") val updatedAt: Long,
    val rev: Long,
    @ColumnInfo(name = "rev_confirmada", defaultValue = "0") val revConfirmada: Long = 0,
    @ColumnInfo(name = "excluido_em") val excluidoEm: Long? = null,
)

@Entity(
    tableName = "itens_propriedade",
    foreignKeys = [ForeignKey(entity = PropriedadeEntidade::class, parentColumns = ["uuid"], childColumns = ["propriedade_uuid"])],
    indices = [Index("propriedade_uuid")],
)
data class ItemEntidade(
    @PrimaryKey val uuid: String,
    @ColumnInfo(name = "propriedade_uuid") val propriedadeUuid: String,
    val categoria: String,
    @ColumnInfo(name = "codigo_catalogo") val codigoCatalogo: String,
    /** Texto original do "Outros" (RN-35: nunca alterado). */
    @ColumnInfo(name = "nome_personalizado") val nomePersonalizado: String?,
    @ColumnInfo(name = "nome_normalizado") val nomeNormalizado: String?,
    val quantidade: Long,
    @ColumnInfo(name = "created_at") val createdAt: Long,
    @ColumnInfo(name = "updated_at") val updatedAt: Long,
    val rev: Long,
    @ColumnInfo(name = "rev_confirmada", defaultValue = "0") val revConfirmada: Long = 0,
    @ColumnInfo(name = "excluido_em") val excluidoEm: Long? = null,
)

@Entity(
    tableName = "fotografias",
    foreignKeys = [ForeignKey(entity = PropriedadeEntidade::class, parentColumns = ["uuid"], childColumns = ["propriedade_uuid"])],
    indices = [Index("propriedade_uuid")],
)
data class FotoEntidade(
    @PrimaryKey val uuid: String,
    @ColumnInfo(name = "propriedade_uuid") val propriedadeUuid: String,
    /** Relativo à pasta privada de fotos do app (ex.: "ab/<uuid>.jpg"). */
    @ColumnInfo(name = "caminho_relativo") val caminhoRelativo: String,
    val mime: String,
    @ColumnInfo(name = "tamanho_bytes") val tamanhoBytes: Long,
    val sha256: String,
    val largura: Long,
    val altura: Long,
    @ColumnInfo(name = "capturada_em") val capturadaEm: Long,
    @ColumnInfo(name = "created_at") val createdAt: Long,
    @ColumnInfo(name = "updated_at") val updatedAt: Long,
    val rev: Long,
    @ColumnInfo(name = "rev_confirmada", defaultValue = "0") val revConfirmada: Long = 0,
    @ColumnInfo(name = "excluido_em") val excluidoEm: Long? = null,
    @ColumnInfo(name = "arquivo_presente", defaultValue = "1") val arquivoPresente: Long = 1,
)

/** Rascunho do formulário: nunca sincronizado nem contado (R5, RF-A10). Pertence a uma coleta (RN-05). */
@Entity(tableName = "rascunhos")
data class RascunhoEntidade(
    @PrimaryKey val uuid: String,
    @ColumnInfo(name = "coleta_uuid") val coletaUuid: String,
    @ColumnInfo(name = "propriedade_uuid") val propriedadeUuid: String?,
    val json: String,
    @ColumnInfo(name = "atualizado_em") val atualizadoEm: Long,
)

/** Lote de envio de uma coleta, com o JSON congelado (reenviado byte a byte; ADR-0017 §4, ADR-0024). */
@Entity(tableName = "lotes_envio", indices = [Index("coleta_uuid")])
data class LoteEnvioEntidade(
    @PrimaryKey val uuid: String,
    @ColumnInfo(name = "coleta_uuid") val coletaUuid: String,
    val status: String,
    val canal: String,
    @ColumnInfo(name = "payload_json") val payloadJson: String,
    @ColumnInfo(name = "criado_em") val criadoEm: Long,
    @ColumnInfo(name = "atualizado_em") val atualizadoEm: Long,
    @ColumnInfo(name = "recibo_json") val reciboJson: String? = null,
    val erro: String? = null,
)

@Entity(
    tableName = "lote_itens",
    primaryKeys = ["lote_uuid", "entidade", "entidade_uuid"],
    foreignKeys = [ForeignKey(entity = LoteEnvioEntidade::class, parentColumns = ["uuid"], childColumns = ["lote_uuid"])],
)
data class LoteItemEntidade(
    @ColumnInfo(name = "lote_uuid") val loteUuid: String,
    val entidade: String,
    @ColumnInfo(name = "entidade_uuid") val entidadeUuid: String,
    val rev: Long,
    /** Preenchido com o resultado do recibo (aplicado, duplicado, conflito, rejeitado). */
    val resultado: String? = null,
    val motivo: String? = null,
)

/** Pareamento com a Central (uma linha; ADR-0007). */
@Entity(tableName = "pareamento")
data class PareamentoEntidade(
    @PrimaryKey @ColumnInfo(name = "central_id") val centralId: String,
    @ColumnInfo(name = "spki_sha256") val spkiSha256: String,
    @ColumnInfo(name = "ips_json") val ipsJson: String,
    val porta: Long,
    val credencial: String,
    @ColumnInfo(name = "pareado_em") val pareadoEm: Long,
)

/** Estados do lote (protocolo.md §4). */
object StatusLote {
    const val PREPARADO = "preparado"
    const val ENVIANDO = "enviando"
    /** A Central recebeu tudo e espera o operador confirmar o destino (ADR-0024). */
    const val AGUARDANDO = "aguardando_confirmacao"
    const val ERRO_RECUPERAVEL = "erro_recuperavel"
    const val ERRO_FATAL = "erro_fatal"
    const val CONFIRMADO = "confirmado"
    /** Cancelado pelo operador na Central (terminal: o mesmo lote nunca é reenviado). */
    const val CANCELADO = "cancelado"
    /** Descartado no celular antes de sair dele (correção de código com lote só preparado). */
    const val DESCARTADO = "descartado"
    val ABERTOS = listOf(PREPARADO, ENVIANDO, AGUARDANDO, ERRO_RECUPERAVEL, ERRO_FATAL)
}
