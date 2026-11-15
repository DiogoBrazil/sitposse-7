package br.gov.ro.pm.sitposse7.sync

import br.gov.ro.pm.sitposse7.dados.BancoCampo
import br.gov.ro.pm.sitposse7.dados.GravacaoPropriedade
import br.gov.ro.pm.sitposse7.dados.RepositorioCampo
import br.gov.ro.pm.sitposse7.dados.abrirBancoJvm
import br.gov.ro.pm.sitposse7.dominio.PayloadQr
import kotlinx.coroutines.Deferred
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.runBlocking
import java.io.File
import java.nio.file.Files
import kotlin.test.AfterTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertIs
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue
import kotlin.test.fail

/**
 * Pareamento com a Central Go real **sem** confirmação automática: o teste faz o papel do
 * operador (protocolo.md §2). Em toda falha: nada é gravado como pareado, o app recebe uma
 * [FalhaCentral] (mensagem na tela, nunca crash) e os cadastros continuam pendentes no aparelho.
 */
class PareamentoGoTest {
    private val dir: File = Files.createTempDirectory("pareamento").toFile()
    private val db: BancoCampo = abrirBancoJvm(dir.resolve(BancoCampo.NOME_ARQUIVO))
    private val agora = { System.currentTimeMillis() }
    private val repo = RepositorioCampo(db, agora) { true }
    private val pareador = Pareador(db, agora)
    private var central: CentralGo? = null

    @AfterTest
    fun encerrar() {
        central?.close()
        db.close()
        dir.deleteRecursively()
    }

    private fun iniciar(vararg args: String) = CentralGo.iniciar(*args).also { central = it }

    private fun qr(c: CentralGo, token: String) = PayloadQr(2, c.centralId, listOf("127.0.0.1"), c.porta, c.pin, token)

    private suspend fun comCadastro() {
        val coleta = repo.iniciarColeta("EQ7K3M9QX1").coleta.uuid
        repo.salvar(GravacaoPropriedade(null, "Sítio Boa Esperança", "Responsável", mapOf("bov_adultos" to 3L), null, null, null, null, null, null, null, null, emptyList(), emptyList()), coleta)
    }

    /** Pede o pareamento em segundo plano e espera o pedido aparecer para o operador. */
    private fun kotlinx.coroutines.CoroutineScope.pedir(c: CentralGo, token: String): Deferred<Result<*>> {
        val r = async(Dispatchers.IO) { runCatching { pareador.parear(qr(c, token), "aparelho de teste") } }
        val limite = System.currentTimeMillis() + 10_000
        while (c.pendentes() == 0) {
            if (System.currentTimeMillis() > limite) fail("o pedido não chegou à Central")
            Thread.sleep(20)
        }
        return r
    }

    private suspend fun nadaPerdidoENaoPareado() {
        assertNull(db.pareamentoDao().obter(), "não pareado")
        assertEquals(1, db.propriedadeDao().pendentes().size, "cadastro continua pendente no aparelho")
    }

    private fun falhaDe(r: Result<*>): FalhaCentral = assertIs<FalhaCentral>(r.exceptionOrNull(), "esperava FalhaCentral, veio $r")

    @Test
    fun recusadoPeloOperadorEDepoisAceitoComNovoQr() = runBlocking<Unit> {
        val c = iniciar("-confirmar=false")
        comCadastro()
        val recusa = pedir(c, c.token)
        assertEquals(1, c.decidir(aceitar = false))
        falhaDe(recusa.await()).also { assertEquals("recusado", it.codigo); assertTrue(it.fatal) }
        nadaPerdidoENaoPareado()

        // Repetir com o mesmo QR: o token não foi consumido pela recusa, então o pedido chega de novo.
        val aceito = pedir(c, c.token)
        assertEquals(1, c.decidir(aceitar = true))
        assertTrue(aceito.await().isSuccess, "pareado: ${aceito.await()}")
        assertNotNull(db.pareamentoDao().obter())

        // E o QR já usado é recusado com mensagem, sem derrubar nada.
        val repetido = runCatching { pareador.parear(qr(c, c.token), null) }
        assertEquals("token_invalido", falhaDe(repetido).codigo)
    }

    @Test
    fun operadorNaoConfirmaATempo() = runBlocking<Unit> {
        val c = iniciar("-confirmar=false", "-espera=1s")
        comCadastro()
        val r = pedir(c, c.token).await()
        falhaDe(r).also { assertEquals("tempo_esgotado", it.codigo); assertFalse(it.fatal) }
        nadaPerdidoENaoPareado()
    }

    @Test
    fun centralCaiDuranteAEsperaDoOperador() = runBlocking<Unit> {
        val c = iniciar("-confirmar=false")
        comCadastro()
        val r = pedir(c, c.token)
        c.derrubar()
        falhaDe(r.await()).also { assertEquals("rede", it.codigo); assertFalse(it.fatal) }
        nadaPerdidoENaoPareado()
    }

    @Test
    fun centralForaDoArAntesDoPedido() = runBlocking<Unit> {
        val c = iniciar()
        comCadastro()
        val token = c.token
        c.derrubar()
        val r = runCatching { pareador.parear(qr(c, token), null) }
        falhaDe(r).also { assertEquals("rede", it.codigo); assertFalse(it.fatal) }
        nadaPerdidoENaoPareado()
    }
}
