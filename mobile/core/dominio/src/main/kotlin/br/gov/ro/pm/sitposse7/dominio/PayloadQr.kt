package br.gov.ro.pm.sitposse7.dominio

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import java.util.UUID

/**
 * Conteúdo do QR exibido pela Central (shared/schemas/qr-pareamento.v1.schema.json, ADR-0007).
 * Na Fase 0 o token é omitido (só diagnóstico de conexão).
 */
@Serializable
data class PayloadQr(
    val v: Int,
    @SerialName("central_id") val centralId: String,
    val ips: List<String>,
    val porta: Int,
    @SerialName("spki_sha256") val spkiSha256: String,
    @SerialName("token_pareamento") val tokenPareamento: String? = null,
    @SerialName("operacao_uuid") val operacaoUuid: String? = null,
    @SerialName("operacao_nome") val operacaoNome: String? = null,
)

sealed interface LeituraQr {
    data class Valido(val payload: PayloadQr) : LeituraQr
    /** [motivo] é técnico (log/diagnóstico); a UI mostra mensagem própria. */
    data class Invalido(val motivo: String) : LeituraQr
}

object LeitorQr {
    // Campos desconhecidos são recusados, como no esquema (additionalProperties=false).
    private val json = Json { ignoreUnknownKeys = false; explicitNulls = false }
    private val pinValido = Regex("^[A-Za-z0-9_-]{43}$")
    private val ipv4 = Regex("^(25[0-5]|2[0-4]\\d|1\\d\\d|[1-9]?\\d)(\\.(25[0-5]|2[0-4]\\d|1\\d\\d|[1-9]?\\d)){3}$")

    fun ler(texto: String): LeituraQr {
        val p = try {
            json.decodeFromString<PayloadQr>(texto)
        } catch (e: SerializationException) {
            return LeituraQr.Invalido("json: ${e.message}")
        } catch (e: IllegalArgumentException) {
            return LeituraQr.Invalido("json: ${e.message}")
        }
        return when {
            p.v != 1 -> LeituraQr.Invalido("versao")
            !uuidValido(p.centralId) -> LeituraQr.Invalido("central_id")
            p.ips.any { !ipv4.matches(it) } -> LeituraQr.Invalido("ips")
            p.porta !in 1..65535 -> LeituraQr.Invalido("porta")
            !pinValido.matches(p.spkiSha256) -> LeituraQr.Invalido("pin")
            p.tokenPareamento != null && p.tokenPareamento.length < 32 -> LeituraQr.Invalido("token")
            p.operacaoUuid != null && !uuidValido(p.operacaoUuid) -> LeituraQr.Invalido("operacao_uuid")
            p.operacaoNome != null && p.operacaoNome.isEmpty() -> LeituraQr.Invalido("operacao_nome")
            else -> LeituraQr.Valido(p)
        }
    }

    private fun uuidValido(s: String) =
        s.length == 36 && runCatching { UUID.fromString(s) }.isSuccess
}
