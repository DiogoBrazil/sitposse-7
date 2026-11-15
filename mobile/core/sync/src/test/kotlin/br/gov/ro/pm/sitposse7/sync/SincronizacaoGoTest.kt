package br.gov.ro.pm.sitposse7.sync

import br.gov.ro.pm.sitposse7.dados.BancoCampo
import br.gov.ro.pm.sitposse7.dados.Contagens
import br.gov.ro.pm.sitposse7.dados.GravacaoFoto
import br.gov.ro.pm.sitposse7.dados.GravacaoItem
import br.gov.ro.pm.sitposse7.dados.GravacaoPropriedade
import br.gov.ro.pm.sitposse7.dados.RepositorioCampo
import br.gov.ro.pm.sitposse7.dados.StatusLote
import br.gov.ro.pm.sitposse7.dados.abrirBancoJvm
import br.gov.ro.pm.sitposse7.dominio.PayloadQr
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import java.io.File
import java.nio.file.Files
import java.util.UUID
import kotlin.random.Random
import kotlin.test.AfterTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Ponta a ponta: banco Room real (JVM, mesmo driver do aparelho) + Sincronizador + Central Go real.
 * Cenários de testing/estrategia.md §4: 1 (primeira carga), 4 (reenvio), 5 (ACK perdido),
 * 6 (foto retomada) e 8 (tombstone).
 */
class SincronizacaoGoTest {
    private val dir: File = Files.createTempDirectory("sinc").toFile()
    private val fotos = dir.resolve("fotos").apply { mkdirs() }
    private val db: BancoCampo = abrirBancoJvm(dir.resolve(BancoCampo.NOME_ARQUIVO))
    private val agora = { System.currentTimeMillis() }
    private val repo = RepositorioCampo(db, agora) { fotos.resolve(it).isFile }
    private val sinc = Sincronizador(db, agora) { fotos.resolve(it) }
    private val pareador = Pareador(db, agora)
    private val central = CentralGo.iniciar()

    @AfterTest
    fun encerrar() {
        central.close()
        db.close()
        dir.deleteRecursively()
    }

    private fun qr(token: String = central.token) = PayloadQr(1, central.centralId, listOf("127.0.0.1"), central.porta, central.pin, token, null, null)

    private fun novaFoto(tamanho: Int): GravacaoFoto {
        val uuid = UUID.randomUUID().toString()
        val rel = "${uuid.take(2)}/$uuid.jpg"
        val bytes = Random.nextBytes(tamanho)
        fotos.resolve(rel).apply { parentFile.mkdirs() }.writeBytes(bytes)
        return GravacaoFoto(uuid, rel, tamanho.toLong(), sha256Hex(bytes), 2560, 1920, agora())
    }

    private fun prop(nome: String, uuid: String? = null, bov: Long = 10, itens: List<GravacaoItem> = emptyList(), fotos: List<GravacaoFoto> = emptyList()) =
        GravacaoPropriedade(uuid, nome, "Responsável", mapOf("bov_adultos" to bov, "pessoas_adultos" to 2), 2420, "ha", -10.5, -62.3, "gnss", 6.0, agora(), 12, itens, fotos)

    private suspend fun preparadoEPareado() {
        repo.configurarEquipe("Equipe Linha 25", "Sgt. Teste", "Linha 25")
        val p = pareador.parear(qr(), "aparelho de teste")
        assertEquals(central.centralId, p.centralId)
    }

    @Test
    fun primeiraCargaReenvioEdicaoETombstone() = runBlocking<Unit> {
        preparadoEPareado()
        // Token é de uso único: um segundo pareamento com o mesmo QR é recusado.
        assertFailsWith<FalhaCentral> { pareador.parear(qr(), null) }.also { assertEquals("token_invalido", it.codigo) }

        val f1 = novaFoto(300_000)
        val f2 = novaFoto(1_500_000) // 2 blocos
        val itens = listOf(GravacaoItem(null, "eletrodomestico", "eletro.geladeira", null, null, 1), GravacaoItem(null, "ave", "ave.outros", "Codorna", "codorna", 40))
        val p1 = repo.salvar(prop("Sítio Boa Esperança", itens = itens, fotos = listOf(f1, f2)))
        val p2 = repo.salvar(prop("Chácara Dois"))

        val api = pareador.conectar()
        val r = assertNotNull(sinc.sincronizar(api))
        assertEquals(2, r.propriedadesConfirmadas)
        assertEquals(emptyList(), r.rejeitados)
        assertEquals(Contagens(propriedades = 2, pendentes = 0, confirmadas = 2), repo.observarContagens().first())
        assertEquals(mapOf("propriedades" to 2, "itens" to 2, "fotos" to 2, "lotes" to 1), central.estado().filterKeys { it != "aplicacoes" })

        // Nada pendente: não cria lote.
        assertNull(sinc.sincronizar(api))

        // Edição gera rev nova e só ela vai no próximo lote.
        val c = repo.carregar(p1)!!
        repo.salvar(prop("Sítio Boa Esperança", uuid = p1, bov = 99,
            itens = c.itens.map { GravacaoItem(it.uuid, it.categoria, it.codigoCatalogo, it.nomePersonalizado, it.nomeNormalizado, it.quantidade) },
            fotos = c.fotos.map { GravacaoFoto(it.uuid, it.caminhoRelativo, it.tamanhoBytes, it.sha256, it.largura, it.altura, it.capturadaEm) }))
        val r2 = assertNotNull(sinc.sincronizar(api))
        assertEquals(1, r2.propriedadesConfirmadas)
        assertEquals(1, r2.aplicados) // a propriedade; a equipe vem como duplicado
        assertEquals(0, repo.observarContagens().first().pendentes)

        // Tombstone (cenário 8).
        repo.excluir(p2)
        assertNotNull(sinc.sincronizar(api))
        assertEquals(1, central.estado()["propriedades"])
        api.close()
    }

    @Test
    fun fotoRetomadaEAckPerdido() = runBlocking<Unit> {
        preparadoEPareado()
        val grande = novaFoto(2_500_000) // 3 blocos
        repo.salvar(prop("Sítio da Foto Grande", fotos = listOf(grande)))
        val api = pareador.conectar()

        // Cenário 6: o primeiro bloco chega e a conexão "cai" antes do resto.
        val lote = assertNotNull(sinc.preparar())
        val estado = api.enviarLote(lote.uuid, lote.payloadJson)
        assertEquals(grande.uuid, estado.fotosFaltantes.single().uuid)
        val bytes = fotos.resolve(grande.caminhoRelativo).readBytes()
        api.enviarBloco(grande.uuid, 0, bytes.copyOfRange(0, Sincronizador.BLOCO))
        // Retomar: o Sincronizador reaproveita o mesmo lote e continua do offset da Central.
        assertNotNull(sinc.sincronizar(api))
        assertEquals(1, central.estado()["fotos"])

        // Cenário 5: a Central importou, mas o aparelho não gravou o recibo (resposta perdida).
        repo.salvar(prop("Sítio do ACK Perdido"))
        val lote2 = assertNotNull(sinc.preparar())
        api.enviarLote(lote2.uuid, lote2.payloadJson)
        assertTrue(api.importar(lote2.uuid) is ApiCentral.ResultadoImportacao.Importado)
        assertEquals(1, repo.observarContagens().first().pendentes, "sem recibo gravado, continua pendente")
        val r = assertNotNull(sinc.sincronizar(api)) // reenvia o mesmo lote e recebe o mesmo recibo
        assertEquals(lote2.uuid, r.lote)
        assertEquals(0, repo.observarContagens().first().pendentes)
        assertEquals(2, central.estado()["lotes"])
        assertEquals(StatusLote.CONFIRMADO, db.loteDao().obter(lote2.uuid)!!.status)
        api.close()
    }

    @Test
    fun semRedeOLoteFicaEmErroRecuperavelENadaSePerde() = runBlocking<Unit> {
        preparadoEPareado()
        repo.salvar(prop("Sítio Offline"))
        val api = pareador.conectar()
        central.close() // a Central sai do ar
        val f = assertFailsWith<FalhaCentral> { sinc.sincronizar(api) }
        assertTrue(!f.fatal)
        val aberto = assertNotNull(db.loteDao().aberto())
        assertEquals(StatusLote.ERRO_RECUPERAVEL, aberto.status)
        assertEquals(1, repo.observarContagens().first().pendentes)
        api.close()
    }
}
