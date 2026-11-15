package br.gov.ro.pm.sitposse7.sync

import br.gov.ro.pm.sitposse7.dados.BancoCampo
import br.gov.ro.pm.sitposse7.dados.Contagens
import br.gov.ro.pm.sitposse7.dados.ErroGravacao
import br.gov.ro.pm.sitposse7.dados.GravacaoFoto
import br.gov.ro.pm.sitposse7.dados.GravacaoItem
import br.gov.ro.pm.sitposse7.dados.GravacaoPropriedade
import br.gov.ro.pm.sitposse7.dados.RepositorioCampo
import br.gov.ro.pm.sitposse7.dados.StatusLote
import br.gov.ro.pm.sitposse7.dados.abrirBancoJvm
import br.gov.ro.pm.sitposse7.dominio.Codigos
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
import kotlin.test.assertIs
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds

/**
 * Ponta a ponta: banco Room real (JVM, mesmo driver do aparelho) + Sincronizador + Central Go real.
 * Cenários de testing/estrategia.md §4 (1, 4, 5, 6, 8) e os da A28 (ADR-0024): código válido,
 * código inexistente, cancelamento e correção no celular, correção na Central, destino travado,
 * operador que demora (reconciliação depois) e várias coletas no mesmo aparelho.
 */
class SincronizacaoGoTest {
    private val dir: File = Files.createTempDirectory("sinc").toFile()
    private val fotos = dir.resolve("fotos").apply { mkdirs() }
    private val db: BancoCampo = abrirBancoJvm(dir.resolve(BancoCampo.NOME_ARQUIVO))
    private val agora = { System.currentTimeMillis() }
    private val repo = RepositorioCampo(db, agora) { fotos.resolve(it).isFile }
    private val sinc = Sincronizador(db, agora) { fotos.resolve(it) }
    private val pareador = Pareador(db, agora)
    private var centralAtual: CentralGo? = null
    private val central: CentralGo get() = centralAtual ?: CentralGo.iniciar().also { centralAtual = it }

    @AfterTest
    fun encerrar() {
        centralAtual?.close()
        db.close()
        dir.deleteRecursively()
    }

    private fun qr(token: String = central.token) = PayloadQr(2, central.centralId, listOf("127.0.0.1"), central.porta, central.pin, token)

    private fun novaFoto(tamanho: Int): GravacaoFoto {
        val uuid = UUID.randomUUID().toString()
        val rel = "${uuid.take(2)}/$uuid.jpg"
        val bytes = Random.nextBytes(tamanho)
        fotos.resolve(rel).apply { parentFile.mkdirs() }.writeBytes(bytes)
        return GravacaoFoto(uuid, rel, tamanho.toLong(), sha256Hex(bytes), 2560, 1920, agora())
    }

    private fun prop(nome: String, uuid: String? = null, bov: Long = 10, itens: List<GravacaoItem> = emptyList(), fotos: List<GravacaoFoto> = emptyList(), obs: String? = null) =
        GravacaoPropriedade(uuid, nome, "Responsável", mapOf("bov_adultos" to bov, "pessoas_adultos" to 2), 2420, "ha", -10.5, -62.3, "gnss", 6.0, agora(), 12, itens, fotos, obs)

    /** Código de formato válido (com dígito) que não existe na Central. */
    private fun codigoInexistente(): String {
        val corpo = (1..7).map { Codigos.ALFABETO[Random.nextInt(32)] }.joinToString("")
        return "EQ" + corpo + Codigos.digitoVerificador(corpo)
    }

    private suspend fun parear() {
        val p = pareador.parear(qr(), "aparelho de teste")
        assertEquals(central.centralId, p.centralId)
    }

    private suspend fun enviar(api: ApiCentral, coleta: String, espera: kotlin.time.Duration = 15.seconds) =
        sinc.sincronizarColeta(api, coleta, espera = espera, intervalo = 30.milliseconds)

    /** Código válido + confirmação (automática no central-teste): cenários 1, 4 e 8 e observações. */
    @Test
    fun primeiraCargaReenvioEdicaoETombstone() = runBlocking<Unit> {
        parear()
        // Token é de uso único: um segundo pareamento com o mesmo QR é recusado.
        assertFailsWith<FalhaCentral> { pareador.parear(qr(), null) }.also { assertEquals("token_invalido", it.codigo) }

        val coleta = repo.iniciarColeta(central.alfa.codigo).coleta.uuid
        val f1 = novaFoto(300_000)
        val f2 = novaFoto(1_500_000) // 2 blocos
        val itens = listOf(GravacaoItem(null, "eletrodomestico", "eletro.geladeira", null, null, 1), GravacaoItem(null, "ave", "ave.outros", "Codorna", "codorna", 40))
        val p1 = repo.salvar(prop("Sítio Boa Esperança", itens = itens, fotos = listOf(f1, f2), obs = "Família pede prazo.\nIdoso acamado."), coleta)
        val p2 = repo.salvar(prop("Chácara Dois"), coleta)

        val api = pareador.conectar()
        val r = assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta)).resumo
        assertEquals(2, r.propriedadesConfirmadas)
        assertEquals(emptyList(), r.rejeitados)
        assertEquals(central.alfa.codigo, r.destino.equipeCodigo)
        assertEquals(false, r.destino.corrigido)
        assertEquals(Contagens(propriedades = 2, pendentes = 0, confirmadas = 2), repo.observarContagens(coleta).first())
        assertEquals(mapOf("propriedades" to 2, "itens" to 2, "fotos" to 2, "lotes" to 1, "coletas" to 1), central.estado().filterKeys { it in setOf("propriedades", "itens", "fotos", "lotes", "coletas") })
        // Destino gravado na coleta com o recibo (destino travado a partir daqui).
        assertNotNull(repo.coleta(coleta)!!.destinoConfirmadoEm)
        assertTrue(repo.coleta(coleta)!!.destinoJson!!.contains("Alfa"))
        assertFailsWith<ErroGravacao> { repo.corrigirCodigo(coleta, central.bravo.codigo) }

        // Nada pendente: não cria lote.
        assertNull(enviar(api, coleta))

        // Edição gera rev nova e só ela vai no próximo lote (destino travado, confirmado sozinho no teste).
        val c = repo.carregar(p1)!!
        repo.salvar(prop("Sítio Boa Esperança", uuid = p1, bov = 99,
            itens = c.itens.map { GravacaoItem(it.uuid, it.categoria, it.codigoCatalogo, it.nomePersonalizado, it.nomeNormalizado, it.quantidade) },
            fotos = c.fotos.map { GravacaoFoto(it.uuid, it.caminhoRelativo, it.tamanhoBytes, it.sha256, it.largura, it.altura, it.capturadaEm) }), coleta)
        val r2 = assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta)).resumo
        assertEquals(1, r2.propriedadesConfirmadas)
        assertEquals(1, r2.aplicados) // a propriedade; a coleta vem como duplicado
        assertEquals(0, repo.observarContagens(coleta).first().pendentes)

        // Tombstone (cenário 8).
        repo.excluir(p2)
        assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta))
        assertEquals(1, central.estado()["propriedades"])
        api.close()
    }

    /** Entrada manual usa o mesmo contrato v2 e continua incremental após edição. */
    @Test
    fun coordenadaManualSincronizaAposEdicaoDeGnss() = runBlocking<Unit> {
        parear()
        val coleta = repo.iniciarColeta(central.alfa.codigo).coleta.uuid
        val uuid = repo.salvar(prop("Ponto editável"), coleta)
        val api = pareador.conectar()
        try {
            assertEquals(emptyList(), assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta)).resumo.rejeitados)
            repo.salvar(prop("Ponto editável", uuid = uuid).copy(
                lat = -10.75, lon = -63.25, gpsOrigem = "manual",
                gpsPrecisaoM = null, gpsCapturadoEm = null, gpsSatelites = null,
            ), coleta)
            val r = assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta)).resumo
            assertEquals(emptyList(), r.rejeitados)
            assertEquals(1, r.propriedadesConfirmadas)
            assertEquals(1, r.aplicados)
            assertEquals(0, repo.observarContagens(coleta).first().pendentes)
            assertEquals(1, central.estado()["propriedades"])
            assertNull(enviar(api, coleta))
        } finally { api.close() }
    }

    /** Cenários 6 (foto retomada) e 5 (ACK perdido: a Central importou e o aparelho não gravou o recibo). */
    @Test
    fun fotoRetomadaEAckPerdido() = runBlocking<Unit> {
        parear()
        val coleta = repo.iniciarColeta(central.alfa.codigo).coleta.uuid
        val grande = novaFoto(2_500_000) // 3 blocos
        repo.salvar(prop("Sítio da Foto Grande", fotos = listOf(grande)), coleta)
        val api = pareador.conectar()

        val lote = assertNotNull(sinc.preparar(coleta))
        val estado = api.enviarLote(lote.uuid, lote.payloadJson)
        assertEquals(grande.uuid, estado.fotosFaltantes.single().uuid)
        val bytes = fotos.resolve(grande.caminhoRelativo).readBytes()
        api.enviarBloco(grande.uuid, 0, bytes.copyOfRange(0, Sincronizador.BLOCO))
        // Retomar: o Sincronizador reaproveita o mesmo lote e continua do offset da Central.
        assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta))
        assertEquals(1, central.estado()["fotos"])

        repo.salvar(prop("Sítio do ACK Perdido"), coleta)
        val lote2 = assertNotNull(sinc.preparar(coleta))
        api.enviarLote(lote2.uuid, lote2.payloadJson)
        api.importar(lote2.uuid)
        // Espera a Central importar (confirmação automática do teste) sem o celular saber.
        val limite = System.currentTimeMillis() + 10_000
        while (api.estadoLote(lote2.uuid)!!.status != StatusCentralLote.IMPORTADO) {
            check(System.currentTimeMillis() < limite)
            Thread.sleep(20)
        }
        assertEquals(1, repo.observarContagens(coleta).first().pendentes, "sem recibo gravado, continua pendente")
        val r = assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta)) // reenvia o mesmo lote e recebe o mesmo recibo
        assertEquals(lote2.uuid, r.resumo.lote)
        assertEquals(0, repo.observarContagens(coleta).first().pendentes)
        assertEquals(StatusLote.CONFIRMADO, db.loteDao().obter(lote2.uuid)!!.status)
        api.close()
    }

    /** Código inexistente → operador cancela → celular corrige o código → reenvio confirmado (A28-2, 4). */
    @Test
    fun codigoInexistenteCanceladoECorrigidoNoCelular() = runBlocking<Unit> {
        parear()
        val errado = codigoInexistente()
        val coleta = repo.iniciarColeta(errado).coleta.uuid
        val foto = novaFoto(50_000)
        val p = repo.salvar(prop("Sítio Digitado Errado", fotos = listOf(foto)), coleta)
        val api = pareador.conectar()

        // A Central não confirma sozinha código inexistente: o celular fica esperando o operador.
        val espera = assertIs<ResultadoColeta.Aguardando>(enviar(api, coleta, espera = 300.milliseconds))
        assertEquals(StatusLote.AGUARDANDO, db.loteDao().obter(espera.lote)!!.status)
        assertFailsWith<ErroGravacao> { repo.corrigirCodigo(coleta, central.alfa.codigo) } // envio em aberto
        val ag = central.esperarAguardando().single()
        assertEquals(false, ag.encontrado)
        assertEquals("cancelado", central.cancelar(ag.lote))

        // Próxima tentativa reconcilia: cancelado, nada importado, nada apagado.
        val cancelado = assertIs<ResultadoColeta.Cancelado>(enviar(api, coleta))
        assertEquals("codigo_inexistente", cancelado.motivo)
        assertEquals(StatusLote.CANCELADO, db.loteDao().obter(cancelado.lote)!!.status)
        assertEquals(0, central.estado()["propriedades"])
        assertEquals(1, repo.observarContagens(coleta).first().pendentes)
        assertTrue(fotos.resolve(foto.caminhoRelativo).isFile)

        // Corrige no celular e reenvia: lote novo, foto reaproveitada do staging, destino Alfa.
        repo.corrigirCodigo(coleta, central.alfa.codigo)
        val ok = assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta))
        assertTrue(ok.resumo.lote != cancelado.lote)
        assertEquals(central.alfa.codigo, ok.resumo.destino.equipeCodigo)
        assertEquals(central.alfa.codigo, ok.resumo.destino.codigoInformado)
        assertEquals(1, central.estado()["propriedades"])
        assertEquals(p, repo.carregar(p)!!.propriedade.uuid)
        assertEquals(listOf(errado to central.alfa.codigo), repo.historicoCodigos(coleta).map { it.codigoAnterior to it.codigoNovo })
        api.close()
    }

    /** Correção pela Central: o operador escolhe outra equipe; o celular recebe o destino corrigido (A28-3, 5, 6). */
    @Test
    fun destinoCorrigidoNaCentralEDepoisTravado() = runBlocking<Unit> {
        centralAtual = CentralGo.iniciar("-destino=manual")
        parear()
        val coleta = repo.iniciarColeta(central.bravo.codigo).coleta.uuid // digitou Bravo, mas era Alfa
        repo.salvar(prop("Sítio Trocado"), coleta)
        val api = pareador.conectar()
        assertIs<ResultadoColeta.Aguardando>(enviar(api, coleta, espera = 200.milliseconds))
        val ag = central.esperarAguardando().single()
        assertEquals(true, ag.encontrado)
        assertEquals("importado", central.confirmar(ag.lote, central.alfa.codigo))
        val r = assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta)).resumo
        assertEquals(true, r.destino.corrigido)
        assertEquals(central.alfa.codigo, r.destino.equipeCodigo)
        assertEquals(central.bravo.codigo, r.destino.codigoInformado)

        // Segundo envio da mesma coleta: destino travado (a Central só aceita a mesma equipe).
        repo.salvar(prop("Sítio Novo da Mesma Coleta"), coleta)
        assertIs<ResultadoColeta.Aguardando>(enviar(api, coleta, espera = 200.milliseconds))
        val ag2 = central.esperarAguardando().single()
        assertEquals(true, ag2.travado)
        assertTrue(central.confirmar(ag2.lote, central.bravo.codigo).contains("destino"), "destino travado não pode mudar")
        assertEquals("importado", central.confirmar(ag2.lote))
        assertEquals(central.alfa.codigo, assertIs<ResultadoColeta.Confirmado>(enviar(api, coleta)).resumo.destino.equipeCodigo)
        assertEquals(2, central.estado()["propriedades"])
        api.close()
    }

    /** Duas coletas (dias diferentes) no mesmo aparelho: lotes e destinos separados (A28-14). */
    @Test
    fun duasColetasNoMesmoAparelho() = runBlocking<Unit> {
        parear()
        val dia1 = repo.iniciarColeta(central.alfa.codigo).coleta.uuid
        repo.salvar(prop("Sítio do Dia 1"), dia1)
        repo.concluirColeta(dia1)
        val dia2 = repo.iniciarColeta(central.bravo.codigo).coleta.uuid
        repo.salvar(prop("Sítio do Dia 2"), dia2)
        repo.salvar(prop("Outro do Dia 2"), dia2)

        val api = pareador.conectar()
        val coletas = sinc.coletasParaEnviar().map { it.uuid }
        assertEquals(listOf(dia1, dia2), coletas)
        val r = coletas.map { assertIs<ResultadoColeta.Confirmado>(enviar(api, it)).resumo }
        assertEquals(listOf(central.alfa.codigo, central.bravo.codigo), r.map { it.destino.equipeCodigo })
        assertEquals(listOf(1, 2), r.map { it.propriedadesConfirmadas })
        assertEquals(emptyList(), sinc.coletasParaEnviar())
        api.close()
    }

    @Test
    fun semRedeOLoteFicaEmErroRecuperavelENadaSePerde() = runBlocking<Unit> {
        parear()
        val coleta = repo.iniciarColeta(central.alfa.codigo).coleta.uuid
        repo.salvar(prop("Sítio Offline"), coleta)
        val api = pareador.conectar()
        central.close() // a Central sai do ar
        centralAtual = null
        val f = assertFailsWith<FalhaCentral> { enviar(api, coleta) }
        assertTrue(!f.fatal)
        val aberto = assertNotNull(db.loteDao().abertoDa(coleta))
        assertEquals(StatusLote.ERRO_RECUPERAVEL, aberto.status)
        assertEquals(1, repo.observarContagens(coleta).first().pendentes)
        api.close()
    }
}
