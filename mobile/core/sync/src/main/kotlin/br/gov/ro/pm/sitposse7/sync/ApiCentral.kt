package br.gov.ro.pm.sitposse7.sync

import kotlinx.serialization.KSerializer
import kotlinx.serialization.SerializationException
import okhttp3.ConnectionSpec
import okhttp3.HttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.TlsVersion
import java.io.IOException
import java.net.Proxy
import java.util.concurrent.TimeUnit
import javax.net.SocketFactory
import javax.net.ssl.SSLContext

/** Falha numa chamada à Central. [fatal] = exige ação da pessoa (não adianta repetir). */
class FalhaCentral(val codigo: String, mensagem: String, val fatal: Boolean, causa: Throwable? = null) : Exception(mensagem, causa)

/**
 * Cliente da API v1 da Central (protocolo.md §3, ADR-0017), com o mesmo TLS e pinning
 * do [ClienteCentral] (ADR-0007/0014). Chamadas bloqueantes: usar fora da main thread.
 * Sem repetição automática do OkHttp: quem repete é o [Sincronizador], com idempotência.
 */
class ApiCentral(
    pin: String,
    private val ip: String,
    private val porta: Int,
    socketFactory: SocketFactory? = null,
    versoesTls: List<TlsVersion> = listOf(TlsVersion.TLS_1_3, TlsVersion.TLS_1_2),
) : AutoCloseable {
    private val http: OkHttpClient
    /** O pareamento espera o operador confirmar na Central (até 90 s). */
    private val httpPareamento: OkHttpClient
    var credencial: String? = null

    init {
        val tm = TrustManagerComPin(pin)
        val ssl = SSLContext.getInstance("TLS").apply { init(null, arrayOf(tm), null) }
        val spec = ConnectionSpec.Builder(ConnectionSpec.RESTRICTED_TLS)
            .tlsVersions(*versoesTls.toTypedArray())
            .cipherSuites(*SUITES_TLS.toTypedArray())
            .build()
        http = OkHttpClient.Builder()
            .apply { if (socketFactory != null) socketFactory(socketFactory) }
            .sslSocketFactory(ssl.socketFactory, tm)
            .hostnameVerifier(HostnameVerifierComPin(pin))
            .connectionSpecs(listOf(spec))
            .proxy(Proxy.NO_PROXY)
            .retryOnConnectionFailure(false)
            .connectTimeout(5, TimeUnit.SECONDS)
            .readTimeout(60, TimeUnit.SECONDS)
            .writeTimeout(60, TimeUnit.SECONDS)
            .build()
        httpPareamento = http.newBuilder().readTimeout(120, TimeUnit.SECONDS).build()
    }

    private fun url(caminho: String, consulta: Map<String, String> = emptyMap()): HttpUrl =
        HttpUrl.Builder().scheme("https").host(ip).port(porta).addPathSegments(caminho)
            .apply { consulta.forEach { (k, v) -> addQueryParameter(k, v) } }.build()

    private fun req(caminho: String, consulta: Map<String, String> = emptyMap()) =
        Request.Builder().url(url(caminho, consulta)).apply { credencial?.let { header("Authorization", "Bearer $it") } }

    private val tipoJson = "application/json; charset=utf-8".toMediaType()
    private val tipoBinario = "application/octet-stream".toMediaType()

    /** Executa e devolve (código HTTP, corpo). Erros de rede viram [FalhaCentral] recuperável. */
    private fun executar(cliente: OkHttpClient, r: Request): Pair<Int, String> = try {
        cliente.newCall(r).execute().use { it.code to it.body.string() }
    } catch (e: IOException) {
        if (e.temCausaPin()) throw FalhaCentral("pin", "Esta não é a Central pareada. Leia o QR da Central de novo.", fatal = true, e)
        throw FalhaCentral("rede", "Sem conexão com a Central: ${e.message}", fatal = false, e)
    }

    private fun <T> decodificar(s: KSerializer<T>, corpo: String): T = try {
        JsonContratos.decodeFromString(s, corpo)
    } catch (e: SerializationException) {
        throw FalhaCentral("resposta", "Resposta inesperada da Central.", fatal = false, e)
    } catch (e: IllegalArgumentException) {
        throw FalhaCentral("resposta", "Resposta inesperada da Central.", fatal = false, e)
    }

    private fun falha(codigoHttp: Int, corpo: String): FalhaCentral {
        val e = runCatching { JsonContratos.decodeFromString(ErroApiV1.serializer(), corpo) }.getOrNull()
        val codigo = e?.codigo?.ifEmpty { null } ?: "http_$codigoHttp"
        val fatal = codigoHttp in setOf(401, 403, 410, 422) || codigo in setOf("outra_operacao", "lote_divergente", "sem_operacao", "recusado")
        return FalhaCentral(codigo, e?.erro?.ifEmpty { null } ?: "A Central respondeu HTTP $codigoHttp.", fatal)
    }

    fun parear(pedido: PedidoPareamentoV1): RespostaPareamentoV1 {
        val corpo = JsonContratos.encodeToString(PedidoPareamentoV1.serializer(), pedido)
        val (st, resp) = executar(httpPareamento, req("v1/pareamento").post(corpo.toRequestBody(tipoJson)).build())
        if (st != 200) throw falha(st, resp)
        return decodificar(RespostaPareamentoV1.serializer(), resp)
    }

    fun status(): StatusCentral {
        val (st, resp) = executar(http, req("v1/status").get().build())
        if (st != 200) throw falha(st, resp)
        return decodificar(StatusCentral.serializer(), resp)
    }

    /** PUT /v1/lotes/{lote} com os bytes congelados do lote (idempotente). */
    fun enviarLote(lote: String, json: String): EstadoLoteV1 {
        val (st, resp) = executar(http, req("v1/lotes/$lote").put(json.toByteArray(Charsets.UTF_8).toRequestBody(tipoJson)).build())
        if (st != 200) throw falha(st, resp)
        return decodificar(EstadoLoteV1.serializer(), resp)
    }

    /** GET /v1/lotes/{lote}; null se a Central não conhece o lote. */
    fun estadoLote(lote: String): EstadoLoteV1? {
        val (st, resp) = executar(http, req("v1/lotes/$lote").get().build())
        if (st == 404) return null
        if (st != 200) throw falha(st, resp)
        return decodificar(EstadoLoteV1.serializer(), resp)
    }

    sealed interface ResultadoBloco {
        data class Aceito(val offset: Long) : ResultadoBloco
        /** A Central tem outro offset: continuar dele. */
        data class OffsetDiferente(val offset: Long) : ResultadoBloco
    }

    fun enviarBloco(foto: String, offset: Long, bytes: ByteArray, tamanho: Int = bytes.size): ResultadoBloco {
        val corpo = bytes.toRequestBody(tipoBinario, 0, tamanho)
        val (st, resp) = executar(http, req("v1/fotos/$foto", mapOf("offset" to offset.toString())).put(corpo).build())
        return when (st) {
            200 -> ResultadoBloco.Aceito(decodificar(EstadoFotoV1.serializer(), resp).offset)
            409 -> {
                val e = runCatching { JsonContratos.decodeFromString(ErroApiV1.serializer(), resp) }.getOrNull()
                e?.offset?.let { ResultadoBloco.OffsetDiferente(it) } ?: throw falha(st, resp)
            }
            else -> throw falha(st, resp)
        }
    }

    sealed interface ResultadoImportacao {
        data class Importado(val estado: EstadoLoteV1) : ResultadoImportacao
        data class FaltamFotos(val estado: EstadoLoteV1) : ResultadoImportacao
    }

    fun importar(lote: String): ResultadoImportacao {
        val (st, resp) = executar(http, req("v1/lotes/$lote/importar").post(ByteArray(0).toRequestBody(null)).build())
        return when (st) {
            200 -> ResultadoImportacao.Importado(decodificar(EstadoLoteV1.serializer(), resp))
            409 -> runCatching { JsonContratos.decodeFromString(EstadoLoteV1.serializer(), resp) }.getOrNull()
                ?.let { ResultadoImportacao.FaltamFotos(it) } ?: throw falha(st, resp)
            else -> throw falha(st, resp)
        }
    }

    /** Informativo: falhar aqui não muda nada no aparelho. */
    fun ack(lote: String) {
        runCatching { executar(http, req("v1/lotes/$lote/ack").post(ByteArray(0).toRequestBody(null)).build()) }
    }

    /** Pode ser chamado em qualquer thread, inclusive a principal (ver [liberarForaDaThreadAtual]). */
    override fun close() = http.liberarForaDaThreadAtual()
}

internal fun Throwable.temCausaPin(): Boolean {
    var c: Throwable? = this
    val vistos = HashSet<Throwable>()
    while (c != null && vistos.add(c)) {
        if (c is PinDivergenteException || c.suppressed.any { it is PinDivergenteException }) return true
        c = c.cause
    }
    return false
}
