package br.gov.ro.pm.sitposse7.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.io.File
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** Mesmo pin que o OpenSSL e o Go (shared/test-fixtures/tls/pin.json). */
class PinTest {
    private val dir = File(System.getProperty("sitposse7.fixtures"), "tls")

    @Test
    fun pinDaFixtureIgualAoDoGoEOpenSsl() {
        val casos = Json.parseToJsonElement(File(dir, "pin.json").readText()).jsonObject.getValue("casos").jsonArray
        assertTrue(casos.isNotEmpty())
        for (c in casos.map { it.jsonObject }) {
            val cert = File(dir, c.getValue("certificado").jsonPrimitive.content).inputStream().use {
                CertificateFactory.getInstance("X.509").generateCertificate(it) as X509Certificate
            }
            val esperado = c.getValue("pin").jsonPrimitive.content
            assertEquals(esperado, Pin.calcular(cert.publicKey))
            assertTrue(Pin.confere(cert, esperado))
        }
    }
}
