package br.gov.ro.pm.sitposse7.sync

import java.security.cert.CertificateException
import java.security.cert.X509Certificate
import javax.net.ssl.HostnameVerifier
import javax.net.ssl.SSLSession
import javax.net.ssl.X509TrustManager

/**
 * Aceita **somente** o servidor cuja chave pública tem o pin do QR (ADR-0007).
 * Não desabilita a verificação: troca a âncora de confiança (CA) pelo pin.
 * A validade do certificado não é a âncora (o relógio do aparelho pode estar errado em campo);
 * a confiança vem da chave pareada, como no cliente Go.
 */
class TrustManagerComPin(private val pin: String) : X509TrustManager {
    override fun checkServerTrusted(chain: Array<out X509Certificate>?, authType: String?) {
        val folha = chain?.firstOrNull() ?: throw CertificateException("Central não apresentou certificado")
        if (!Pin.confere(folha, pin)) throw PinDivergenteException()
    }

    override fun checkClientTrusted(chain: Array<out X509Certificate>?, authType: String?): Unit =
        throw CertificateException("autenticação de cliente não suportada")

    override fun getAcceptedIssuers(): Array<X509Certificate> = emptyArray()
}

/**
 * O acesso é por IP e o certificado pode ter sido emitido antes de uma troca de IP;
 * por isso o "hostname" válido é o certificado com o pin pareado (ADR-0007).
 */
class HostnameVerifierComPin(private val pin: String) : HostnameVerifier {
    override fun verify(hostname: String?, session: SSLSession?): Boolean {
        val folha = runCatching { session?.peerCertificates?.firstOrNull() }.getOrNull() as? X509Certificate
        return folha != null && Pin.confere(folha, pin)
    }
}
