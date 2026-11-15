package br.gov.ro.pm.sitposse7.dados

import androidx.room3.ColumnInfo
import androidx.room3.Entity
import androidx.room3.ForeignKey
import androidx.room3.Index
import androidx.room3.PrimaryKey

/*
 * Esquema v2 do banco do celular (MVP, A24; docs/architecture/modelo-de-dados.md, ADR-0018).
 * Convenções: UUID v4 em texto; instantes em epoch ms UTC (ADR-0010); exclusão = tombstone
 * (`excluido_em`); toda escrita de entidade sincronizável incrementa `rev` na mesma transação;
 * pendente de envio = `rev > rev_confirmada`.
 * O Room não declara CHECK nem índice parcial: as regras de domínio (RN-xx) são garantidas
 * pelos repositórios, testadas na JVM, e conferidas de novo pela Central na importação.
 */

/** A equipe deste aparelho (uma linha). RN-01..03. */
@Entity(tableName = "equipe")
data class EquipeEntidade(
    @PrimaryKey val uuid: String,
    val nome: String,
    val responsavel: String,
    @ColumnInfo(name = "identificacao_area") val identificacaoArea: String?,
    @ColumnInfo(name = "created_at") val createdAt: Long,
    @ColumnInfo(name = "updated_at") val updatedAt: Long,
    val rev: Long,
    @ColumnInfo(name = "rev_confirmada", defaultValue = "0") val revConfirmada: Long = 0,
)

@Entity(
    tableName = "propriedades",
    foreignKeys = [ForeignKey(entity = EquipeEntidade::class, parentColumns = ["uuid"], childColumns = ["equipe_uuid"])],
    indices = [Index("equipe_uuid")],
)
data class PropriedadeEntidade(
    @PrimaryKey val uuid: String,
    @ColumnInfo(name = "equipe_uuid") val equipeUuid: String,
    @ColumnInfo(name = "nome_propriedade") val nomePropriedade: String,
    @ColumnInfo(name = "responsavel_familia") val responsavelFamilia: String?,
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

/** Rascunho do formulário: nunca sincronizado nem contado (R5, RF-A10). */
@Entity(tableName = "rascunhos")
data class RascunhoEntidade(
    @PrimaryKey val uuid: String,
    @ColumnInfo(name = "propriedade_uuid") val propriedadeUuid: String?,
    val json: String,
    @ColumnInfo(name = "atualizado_em") val atualizadoEm: Long,
)

/** Lote de envio com o JSON congelado (reenviado byte a byte; ADR-0017 §4). */
@Entity(tableName = "lotes_envio")
data class LoteEnvioEntidade(
    @PrimaryKey val uuid: String,
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
    @ColumnInfo(name = "operacao_uuid") val operacaoUuid: String,
    @ColumnInfo(name = "operacao_nome") val operacaoNome: String,
    @ColumnInfo(name = "pareado_em") val pareadoEm: Long,
)

/** Estados do lote (protocolo.md §4). */
object StatusLote {
    const val PREPARADO = "preparado"
    const val ENVIANDO = "enviando"
    const val ERRO_RECUPERAVEL = "erro_recuperavel"
    const val ERRO_FATAL = "erro_fatal"
    const val CONFIRMADO = "confirmado"
    val ABERTOS = listOf(PREPARADO, ENVIANDO, ERRO_RECUPERAVEL, ERRO_FATAL)
}
