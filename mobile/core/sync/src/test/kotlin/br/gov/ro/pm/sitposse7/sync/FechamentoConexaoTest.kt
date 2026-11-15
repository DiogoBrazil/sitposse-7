package br.gov.ro.pm.sitposse7.sync

import br.gov.ro.pm.sitposse7.dados.BancoCampo
import br.gov.ro.pm.sitposse7.dados.GravacaoPropriedade
import br.gov.ro.pm.sitposse7.dados.RepositorioCampo
import br.gov.ro.pm.sitposse7.dados.abrirBancoJvm
import br.gov.ro.pm.sitposse7.dominio.PayloadQr
import kotlinx.coroutines.runBlocking
import java.io.File
import java.net.InetAddress
import java.net.Socket
import java.nio.file.Files
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import javax.net.SocketFactory
import kotlin.test.AfterTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

/**
 * Regressão do crash no pareamento (mobile/docs/incidentes/2026-11-14-crash-pareamento.md).
 *
 * No aparelho, fechar um socket TLS na thread principal lança `NetworkOnMainThreadException`
 * (o Conscrypt envia o close_notify passando pelo BlockGuard), e o OkHttp repassa essa
 * RuntimeException em `closeQuietly`. A JVM não tem StrictMode: a [FabricaThreadPrincipal]
 * imita a regra, lançando [RedeNaThreadPrincipal] se um socket for fechado na thread marcada
 * como principal (aqui, a do `runBlocking`, que faz o papel do `viewModelScope`).
 */
class FechamentoConexaoTest {
    private val dir: File = Files.createTempDirectory("fechamento").toFile()
    private val db: BancoCampo = abrirBancoJvm(dir.resolve(BancoCampo.NOME_ARQUIVO))
    private val agora = { System.currentTimeMillis() }
    private val repo = RepositorioCampo(db, agora) { true }
    private val central = CentralGo.iniciar()

    @AfterTest
    fun encerrar() {
        central.close()
        db.close()
        dir.deleteRecursively()
    }

    private fun qr() = PayloadQr(1, central.centralId, listOf("127.0.0.1"), central.porta, central.pin, central.token, null, null)

    @Test
    fun parearEEnviarFechandoNaThreadPrincipalNaoDerrubaOApp() = runBlocking<Unit> {
        val fabrica = FabricaThreadPrincipal(Thread.currentThread())
        repo.configurarEquipe("Equipe Linha 25", "Sgt. Teste", "Linha 25")
        repo.salvar(GravacaoPropriedade(null, "Sítio Boa Esperança", "Responsável", mapOf("bov_adultos" to 3L), null, null, null, null, null, null, null, null, emptyList(), emptyList()))

        // Antes da correção: o close() do api.use { } dentro de parear() fechava o socket aqui e lançava.
        val p = Pareador(db, agora).parear(qr(), "aparelho de teste", fabrica)
        assertEquals(central.centralId, p.centralId)
        assertNotNull(db.pareamentoDao().obter(), "pareamento gravado no aparelho")

        // O mesmo vale para o envio (api.use { } no SincronizarViewModel).
        val api = Pareador(db, agora).conectar(fabrica)
        val r = api.use { Sincronizador(db, agora) { error("sem fotos") }.sincronizar(it) }
        assertEquals(1, assertNotNull(r).propriedadesConfirmadas)

        assertTrue(fabrica.fechadosForaDaPrincipal.await(10, TimeUnit.SECONDS), "as conexões são fechadas fora da thread que chamou close()")
        assertEquals(0, fabrica.fechadosNaPrincipal, "nenhum socket fechado na thread principal")
    }

    @Test
    fun clienteDeDiagnosticoTambemFechaForaDaThreadPrincipal() {
        val fabrica = FabricaThreadPrincipal(Thread.currentThread())
        val cliente = ClienteCentral(central.pin, fabrica)
        val r = cliente.status("127.0.0.1", central.porta)
        assertTrue(r is ResultadoStatus.Conectado, "conectou: $r")
        cliente.fechar()
        assertTrue(fabrica.fechadosForaDaPrincipal.await(10, TimeUnit.SECONDS))
        assertEquals(0, fabrica.fechadosNaPrincipal)
    }
}

/** Imita o `NetworkOnMainThreadException` do Android (uma RuntimeException). */
class RedeNaThreadPrincipal : RuntimeException("socket fechado na thread principal")

/** Sockets que lançam [RedeNaThreadPrincipal] se fechados na [principal] (StrictMode do Android). */
class FabricaThreadPrincipal(private val principal: Thread) : SocketFactory() {
    @Volatile var fechadosNaPrincipal = 0
    val fechadosForaDaPrincipal = CountDownLatch(1)

    override fun createSocket(): Socket = object : Socket() {
        override fun close() {
            if (Thread.currentThread() === principal) {
                fechadosNaPrincipal++
                throw RedeNaThreadPrincipal()
            }
            super.close()
            fechadosForaDaPrincipal.countDown()
        }
    }

    override fun createSocket(host: String?, port: Int): Socket = throw UnsupportedOperationException()
    override fun createSocket(host: String?, port: Int, localHost: InetAddress?, localPort: Int): Socket = throw UnsupportedOperationException()
    override fun createSocket(host: InetAddress?, port: Int): Socket = throw UnsupportedOperationException()
    override fun createSocket(address: InetAddress?, port: Int, localAddress: InetAddress?, localPort: Int): Socket = throw UnsupportedOperationException()
}
