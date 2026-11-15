package br.gov.ro.pm.sitposse7.sync

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import java.security.MessageDigest
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter

/*
 * Contratos v2 (shared/schemas, arquivos v2; ADR-0017 e ADR-0024), espelho de desktop/internal/contratos.
 * Todas as chaves são escritas, inclusive as nulas (o schema exige as chaves presentes).
 */

const val SCHEMA_VERSAO = 2

/** JSON dos contratos: nulos explícitos na escrita; leitura tolerante a campos novos da Central. */
val JsonContratos = Json {
    explicitNulls = true
    encodeDefaults = true
    ignoreUnknownKeys = true
}

private val formatoInstante = DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'").withZone(ZoneOffset.UTC)

/** Epoch ms UTC → ISO 8601 com ms e Z (ADR-0010). */
fun instante(ms: Long): String = formatoInstante.format(Instant.ofEpochMilli(ms))

fun instanteOuNulo(ms: Long?): String? = ms?.let(::instante)

/** Coleta do celular (RN-05): o lote leva a coleta no lugar da antiga equipe. */
@Serializable
data class ColetaV2(
    val uuid: String,
    @SerialName("codigo_equipe") val codigoEquipe: String,
    @SerialName("codigo_original") val codigoOriginal: String,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    val rev: Long,
)

@Serializable
data class PropriedadeV2(
    val uuid: String,
    @SerialName("coleta_uuid") val coletaUuid: String,
    @SerialName("nome_propriedade") val nomePropriedade: String,
    @SerialName("responsavel_familia") val responsavelFamilia: String,
    @SerialName("observacoes_relevantes") val observacoesRelevantes: String?,
    @SerialName("pessoas_adultos") val pessoasAdultos: Long,
    @SerialName("pessoas_jovens") val pessoasJovens: Long,
    @SerialName("pessoas_criancas") val pessoasCriancas: Long,
    @SerialName("veic_caminhoes") val veicCaminhoes: Long,
    @SerialName("veic_automoveis_utilitarios") val veicAutomoveisUtilitarios: Long,
    @SerialName("veic_motos_ciclomotores") val veicMotosCiclomotores: Long,
    @SerialName("bov_adultos") val bovAdultos: Long,
    @SerialName("bov_bezerros") val bovBezerros: Long,
    @SerialName("equ_adultos") val equAdultos: Long,
    @SerialName("equ_potros") val equPotros: Long,
    @SerialName("area_valor_milesimos") val areaValorMilesimos: Long?,
    @SerialName("area_unidade") val areaUnidade: String?,
    val lat: Double?,
    val lon: Double?,
    @SerialName("gps_origem") val gpsOrigem: String?,
    @SerialName("gps_precisao_m") val gpsPrecisaoM: Double?,
    @SerialName("gps_capturado_em") val gpsCapturadoEm: String?,
    @SerialName("gps_satelites") val gpsSatelites: Long?,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    val rev: Long,
    @SerialName("excluido_em") val excluidoEm: String?,
)

@Serializable
data class ItemV2(
    val uuid: String,
    @SerialName("propriedade_uuid") val propriedadeUuid: String,
    val categoria: String,
    @SerialName("codigo_catalogo") val codigoCatalogo: String,
    @SerialName("nome_personalizado") val nomePersonalizado: String?,
    val quantidade: Long,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    val rev: Long,
    @SerialName("excluido_em") val excluidoEm: String?,
)

@Serializable
data class FotoV2(
    val uuid: String,
    @SerialName("propriedade_uuid") val propriedadeUuid: String,
    val mime: String,
    @SerialName("tamanho_bytes") val tamanhoBytes: Long,
    val sha256: String,
    val largura: Long,
    val altura: Long,
    @SerialName("capturada_em") val capturadaEm: String,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    val rev: Long,
    @SerialName("excluido_em") val excluidoEm: String?,
)

@Serializable
data class LoteV2(
    @SerialName("protocolo_versao") val protocoloVersao: Int = PROTOCOLO_VERSAO,
    @SerialName("schema_versao") val schemaVersao: Int = SCHEMA_VERSAO,
    @SerialName("lote_uuid") val loteUuid: String,
    @SerialName("instalacao_uuid") val instalacaoUuid: String,
    @SerialName("criado_em") val criadoEm: String,
    val coleta: ColetaV2,
    val propriedades: List<PropriedadeV2>,
    val itens: List<ItemV2>,
    val fotos: List<FotoV2>,
)

@Serializable
data class FotoFaltanteV2(val uuid: String, val offset: Long)

@Serializable
data class ResultadoReciboV2(
    val entidade: String,
    val uuid: String,
    val rev: Long,
    val resultado: String,
    val motivo: String? = null,
    @SerialName("conflito_uuid") val conflitoUuid: String? = null,
)

/** Equipe confirmada pelo operador para a coleta (ADR-0024). */
@Serializable
data class DestinoV2(
    @SerialName("estudo_codigo") val estudoCodigo: String,
    @SerialName("estudo_nome") val estudoNome: String,
    @SerialName("jornada_numero") val jornadaNumero: Long,
    @SerialName("jornada_data") val jornadaData: String,
    @SerialName("equipe_nome") val equipeNome: String,
    @SerialName("equipe_codigo") val equipeCodigo: String,
    @SerialName("codigo_informado") val codigoInformado: String,
    val corrigido: Boolean,
)

@Serializable
data class ReciboV2(
    @SerialName("protocolo_versao") val protocoloVersao: Int,
    @SerialName("lote_uuid") val loteUuid: String,
    @SerialName("central_id") val centralId: String,
    @SerialName("importado_em") val importadoEm: String,
    val destino: DestinoV2,
    val resultados: List<ResultadoReciboV2>,
    @SerialName("recibo_sha256") val reciboSha256: String,
) {
    /** Entrada do recibo_sha256 (descrição em recibo.v2.schema.json). */
    fun textoCanonico(): String = buildString {
        append(loteUuid).append('\n').append(centralId).append('\n').append(importadoEm).append('\n')
        with(destino) { append("destino|$estudoCodigo|$jornadaNumero|$jornadaData|$equipeCodigo|$codigoInformado|$corrigido\n") }
        resultados.forEach { r -> append("${r.entidade}|${r.uuid}|${r.rev}|${r.resultado}|${r.motivo ?: ""}\n") }
    }

    fun calcularHash(): String = sha256Hex(textoCanonico().toByteArray(Charsets.UTF_8))

    /** O recibo é íntegro e corresponde ao lote esperado. */
    fun confere(lote: String): Boolean = loteUuid == lote && reciboSha256 == calcularHash()
}

/** Estados do lote na Central (estado-lote.v2). */
object StatusCentralLote {
    const val RECEBENDO = "recebendo"
    const val AGUARDANDO = "aguardando_confirmacao"
    const val IMPORTADO = "importado"
    const val CANCELADO = "cancelado"
}

@Serializable
data class EstadoLoteV2(
    @SerialName("protocolo_versao") val protocoloVersao: Int,
    @SerialName("lote_uuid") val loteUuid: String,
    val status: String,
    @SerialName("fotos_faltantes") val fotosFaltantes: List<FotoFaltanteV2>,
    val recibo: ReciboV2?,
    /** Só quando cancelado: "cancelado_operador" | "codigo_inexistente". */
    val motivo: String? = null,
)

@Serializable
data class PedidoPareamentoV2(
    @SerialName("protocolo_versao") val protocoloVersao: Int = PROTOCOLO_VERSAO,
    @SerialName("schema_versao") val schemaVersao: Int = SCHEMA_VERSAO,
    @SerialName("token_pareamento") val token: String,
    @SerialName("instalacao_uuid") val instalacaoUuid: String,
    @SerialName("rotulo_aparelho") val rotuloAparelho: String?,
)

@Serializable
data class RespostaPareamentoV2(
    @SerialName("protocolo_versao") val protocoloVersao: Int,
    @SerialName("central_id") val centralId: String,
    @SerialName("credencial_dispositivo") val credencial: String,
)

/** Corpo de erro da Central: `codigo` estável, `erro` para a pessoa (protocolo.md §9). */
@Serializable
data class ErroApi(val codigo: String = "", val erro: String = "", val offset: Long? = null)

@Serializable
data class EstadoFoto(val uuid: String, val offset: Long, val existe: Boolean = false)

fun sha256Hex(b: ByteArray): String = MessageDigest.getInstance("SHA-256").digest(b).joinToString("") { "%02x".format(it) }
