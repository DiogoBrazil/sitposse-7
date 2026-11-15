package br.gov.ro.pm.sitposse7.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.TlsVersion
import java.io.File
import java.net.ServerSocket
import java.nio.file.Files
import java.util.concurrent.TimeUnit
import kotlin.test.AfterTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertTrue

/**
 * Teste cruzado: cliente Kotlin (JVM) × servidor Go real da Central (desktop/cmd/central-teste).
 * Exige o Go no PATH (ou SITPOSSE7_GO). Falha — não é pulado — se o Go não estiver disponível.
 */
class ClienteCentralGoTest {
    private val pinFixture = "LebAdzGYtDPoyDIJZAO3bxa_zvkK8dw6lhVmgEsPdfk" // outra chave (tls/pin.json)
    private val abertos = mutableListOf<CentralGo>()

    @AfterTest
    fun encerrar() = abertos.forEach { it.close() }

    private fun central() = CentralGo.iniciar().also { abertos += it }

    @Test
    fun statusComPinCorretoUsaTls13() {
        val c = central()
        val r = ClienteCentral(c.pin).status("127.0.0.1", c.porta)
        assertIs<ResultadoStatus.Conectado>(r, "resultado: $r")
        assertEquals("TLSv1.3", r.versaoTls)
        assertEquals(c.centralId, r.status.centralId)
        assertTrue(r.compativel)
        assertTrue(Regex("""^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$""").matches(r.status.horario))
    }

    /** Simula Android 8/9, que não têm TLS 1.3 na plataforma (ADR-0014). */
    @Test
    fun tls12ComSuiteAeadConecta() {
        val c = central()
        val cliente = ClienteCentral(c.pin, null, 5, 15, listOf(TlsVersion.TLS_1_2))
        val r = cliente.status("127.0.0.1", c.porta)
        assertIs<ResultadoStatus.Conectado>(r, "resultado: $r")
        assertEquals("TLSv1.2", r.versaoTls)
    }

    @Test
    fun pinDeOutraCentralRecusadoTambemEmTls12() {
        val c = central()
        val cliente = ClienteCentral(pinFixture, null, 5, 15, listOf(TlsVersion.TLS_1_2))
        assertEquals(ResultadoStatus.PinDivergente, cliente.status("127.0.0.1", c.porta))
    }

    @Test
    fun pinDeOutraCentralRecusado() {
        val c = central()
        assertEquals(ResultadoStatus.PinDivergente, ClienteCentral(pinFixture).status("127.0.0.1", c.porta))
    }

    @Test
    fun duasCentraisTemPinsDiferentes() {
        val a = central()
        val b = central()
        assertTrue(a.pin != b.pin)
        assertEquals(ResultadoStatus.PinDivergente, ClienteCentral(a.pin).status("127.0.0.1", b.porta))
    }

    @Test
    fun portaFechadaEhSemConexao() {
        val porta = ServerSocket(0).use { it.localPort }
        assertIs<ResultadoStatus.SemConexao>(ClienteCentral(pinFixture).status("127.0.0.1", porta))
    }
}
