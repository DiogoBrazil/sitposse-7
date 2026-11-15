package br.gov.ro.pm.sitposse7.sync

import java.security.MessageDigest
import java.security.PublicKey
import java.security.cert.CertificateException
import java.security.cert.X509Certificate
import java.util.Base64

/**
 * Pin SPKI (ADR-0007): SHA-256 do SubjectPublicKeyInfo DER, em base64url sem padding.
 * Mesmo cálculo de `PinDaChavePublica` (desktop/internal/syncserver/identidade.go).
 */
object Pin {
    fun calcular(chave: PublicKey): String {
        // PublicKey.encoded é o SubjectPublicKeyInfo DER (formato "X.509").
        check(chave.format == "X.509") { "formato de chave inesperado: ${chave.format}" }
        val hash = MessageDigest.getInstance("SHA-256").digest(chave.encoded)
        return Base64.getUrlEncoder().withoutPadding().encodeToString(hash)
    }

    fun confere(cert: X509Certificate, pin: String): Boolean =
        MessageDigest.isEqual(calcular(cert.publicKey).toByteArray(), pin.toByteArray())
}

/** O servidor não é a Central pareada. */
class PinDivergenteException : CertificateException("certificado da Central não corresponde ao pareamento (pin SPKI divergente)")
