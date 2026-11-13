package br.gov.ro.pm.sitposse7.sync

import kotlinx.serialization.SerializationException
import kotlinx.serialization.json.Json
import okhttp3.CipherSuite
import okhttp3.ConnectionSpec
import okhttp3.HttpUrl
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.TlsVersion
import java.io.IOException
import java.net.Proxy
import java.util.concurrent.TimeUnit
import javax.net.SocketFactory
import javax.net.ssl.SSLContext

/** Versão de protocolo deste cliente (docs/synchronization/protocolo.md). */
const val PROTOCOLO_VERSAO = 1

sealed interface ResultadoStatus {
    data class Conectado(val status: StatusCentral, val versaoTls: String, val compativel: Boolean) : ResultadoStatus
    /** Alguém respondeu, mas não é a Central pareada. Exige novo pareamento. */
    data object PinDivergente : ResultadoStatus
    data class SemConexao(val detalhe: String) : ResultadoStatus
    data class RespostaInvalida(val detalhe: String) : ResultadoStatus
}

/**
 * Cliente HTTPS da Central com pinning (ADR-0007).
 *
 * @param socketFactory no Android, o `Network.socketFactory` da rede Wi‑Fi obtida por
 *   `requestNetwork` (mobile/docs/arquitetura.md); `null` usa a rede padrão (testes JVM).
 * Chamadas bloqueantes: usar fora da main thread.
 */
class ClienteCentral internal constructor(
    pin: String,
    socketFactory: SocketFactory?,
    tempoConexaoS: Long,
    tempoLeituraS: Long,
    versoesTls: List<TlsVersion>,
) {
    constructor(pin: String, socketFactory: SocketFactory? = null, tempoConexaoS: Long = 5, tempoLeituraS: Long = 15) :
        this(pin, socketFactory, tempoConexaoS, tempoLeituraS, listOf(TlsVersion.TLS_1_3, TlsVersion.TLS_1_2))

    private val json = Json { ignoreUnknownKeys = true }
    private val http: OkHttpClient

    init {
        val tm = TrustManagerComPin(pin)
        val ssl = SSLContext.getInstance("TLS").apply { init(null, arrayOf(tm), null) }
        // ADR-0014: TLS 1.3 preferido; TLS 1.2 (Android 8/9) só com ECDHE + AEAD — mesmas suítes da Central.
        val spec = ConnectionSpec.Builder(ConnectionSpec.RESTRICTED_TLS)
            .tlsVersions(*versoesTls.toTypedArray())
            .cipherSuites(*SUITES_TLS.toTypedArray())
            .build()
        http = OkHttpClient.Builder()
            .apply { if (socketFactory != null) socketFactory(socketFactory) }
            .sslSocketFactory(ssl.socketFactory, tm)
            .hostnameVerifier(HostnameVerifierComPin(pin))
            .connectionSpecs(listOf(spec))
            .proxy(Proxy.NO_PROXY) // rede local: nunca passar por proxy
            .retryOnConnectionFailure(false)
            .connectTimeout(tempoConexaoS, TimeUnit.SECONDS)
            .readTimeout(tempoLeituraS, TimeUnit.SECONDS)
            .build()
    }

    fun status(ip: String, porta: Int): ResultadoStatus {
        val url = HttpUrl.Builder().scheme("https").host(ip).port(porta).addPathSegments("v1/status").build()
        return try {
            http.newCall(Request.Builder().url(url).get().build()).execute().use { resp ->
                val corpo = resp.body.string()
                if (!resp.isSuccessful) return ResultadoStatus.RespostaInvalida("HTTP ${resp.code}")
                val st = json.decodeFromString<StatusCentral>(corpo)
                ResultadoStatus.Conectado(
                    status = st,
                    versaoTls = resp.handshake?.tlsVersion?.javaName ?: "?",
                    compativel = PROTOCOLO_VERSAO in st.protocoloVersaoMin..st.protocoloVersaoMax,
                )
            }
        } catch (e: IOException) {
            if (e.temCausa<PinDivergenteException>()) ResultadoStatus.PinDivergente
            else ResultadoStatus.SemConexao("${e::class.simpleName}: ${e.message}")
        } catch (e: SerializationException) {
            ResultadoStatus.RespostaInvalida("JSON: ${e.message}")
        } catch (e: IllegalArgumentException) {
            ResultadoStatus.RespostaInvalida("JSON: ${e.message}")
        }
    }

    /** Libera conexões e threads do OkHttp. */
    fun fechar() {
        http.dispatcher.executorService.shutdown()
        http.connectionPool.evictAll()
    }
}

/** TLS 1.3 + TLS 1.2 ECDHE-ECDSA com AEAD (espelho de `SuitesTLS12` em desktop/internal/syncserver). */
internal val SUITES_TLS = listOf(
    CipherSuite.TLS_AES_128_GCM_SHA256,
    CipherSuite.TLS_AES_256_GCM_SHA384,
    CipherSuite.TLS_CHACHA20_POLY1305_SHA256,
    CipherSuite.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
    CipherSuite.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
    CipherSuite.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
)

private inline fun <reified T : Throwable> Throwable.temCausa(): Boolean {
    var c: Throwable? = this
    val vistos = HashSet<Throwable>()
    while (c != null && vistos.add(c)) {
        if (c is T) return true
        (c.suppressed).forEach { if (it is T) return true }
        c = c.cause
    }
    return false
}
