package br.gov.ro.pm.sitposse7.dados

import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import java.io.File
import java.nio.file.Files
import kotlin.test.AfterTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

class RepositorioCampoTest {
    private val dir: File = Files.createTempDirectory("repo").toFile()
    private val db = abrirBancoJvm(dir.resolve(BancoCampo.NOME_ARQUIVO))
    private var relogio = 1_000L
    private val arquivos = mutableSetOf<String>()
    private val repo = RepositorioCampo(db, { relogio++ }, { it in arquivos })

    @AfterTest
    fun fechar() {
        db.close()
        dir.deleteRecursively()
    }

    private fun gravacao(uuid: String? = null, nome: String = "Sítio", itens: List<GravacaoItem> = emptyList(), fotos: List<GravacaoFoto> = emptyList(), bov: Long = 3,
                         responsavel: String = "José", observacoes: String? = null) =
        GravacaoPropriedade(uuid, nome, responsavel, mapOf("bov_adultos" to bov), 12500, "alq_paulista", null, null, null, null, null, null, itens, fotos, observacoes)

    // Códigos válidos (dígito Luhn mod 32 conferido pela fixture compartilhada).
    private val alfa = "EQ7K3M9QX1"
    private val bravo = "EQ4HZ2RTB5"

    /** Coleta em andamento para os cadastros dos testes. */
    private suspend fun coleta(codigo: String = alfa): String = repo.iniciarColeta(codigo).coleta.uuid

    private fun item(codigo: String, qtd: Long = 1, uuid: String? = null, nome: String? = null) =
        GravacaoItem(uuid, codigo.substringBefore('.').let { if (it == "eletro") "eletrodomestico" else it }, codigo, nome, nome?.lowercase(), qtd)

    private fun foto(uuid: String) = GravacaoFoto(uuid, "${uuid.take(2)}/$uuid.jpg", 10, "a".repeat(64), 1, 1, 1)

    @Test
    fun cicloDaColeta() = runBlocking<Unit> { // RN-05, RN-07
        val nova = repo.iniciarColeta(alfa)
        assertTrue(nova is InicioColeta.Nova)
        assertEquals(36, repo.instalacaoUuid().length)
        assertEquals(StatusColeta.EM_ANDAMENTO, nova.coleta.status)
        // O mesmo código não duplica: devolve a coleta existente.
        val de_novo = repo.iniciarColeta(alfa)
        assertTrue(de_novo is InicioColeta.Existente)
        assertEquals(nova.coleta.uuid, de_novo.coleta.uuid)
        assertFailsWith<ErroGravacao> { repo.iniciarColeta("EQ-7K3M-9QX1") } // o repositório só aceita a forma normalizada

        val b = repo.iniciarColeta(bravo).coleta.uuid
        assertEquals(2, repo.coletasEmAndamento().size)
        val p = repo.salvar(gravacao(), b)
        val concluida = repo.concluirColeta(b)
        assertEquals(StatusColeta.CONCLUIDA, concluida.status)
        assertNotNull(concluida.concluidaEm)
        assertEquals(1, concluida.rev) // estado local: não gera envio
        // Concluída: não aceita cadastro novo, mas o existente pode ser editado e excluído.
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(nome = "Novo"), b) }
        repo.salvar(gravacao(p, bov = 9), b)
        assertEquals(2, repo.carregar(p)!!.propriedade.rev)
        repo.reabrirColeta(b)
        repo.salvar(gravacao(nome = "Novo"), b)
        assertEquals(2, repo.observarPropriedades(b).first().size)
        assertEquals(0, repo.observarPropriedades(nova.coleta.uuid).first().size)

        repo.definirColetaAtual(b)
        assertEquals(b, repo.coletaAtual())
        repo.definirColetaAtual(null)
        assertNull(repo.coletaAtual())
        val resumos = repo.observarColetas().first()
        assertEquals(2, resumos.size)
        assertEquals(2, resumos.first { it.uuid == b }.cadastros)
        assertTrue(resumos.first { it.uuid == b }.temPendencia)
    }

    @Test
    fun correcaoDoCodigo() = runBlocking<Unit> { // RN-08
        val c = coleta()
        val p = repo.salvar(gravacao(), c)
        val antes = repo.carregar(p)!!
        val corrigida = repo.corrigirCodigo(c, bravo)
        assertEquals(bravo, corrigida.codigoEquipe)
        assertEquals(alfa, corrigida.codigoOriginal)
        assertEquals(2, corrigida.rev)
        assertEquals(listOf(alfa to bravo), repo.historicoCodigos(c).map { it.codigoAnterior to it.codigoNovo })
        assertEquals(antes, repo.carregar(p)) // nada dos cadastros muda

        // Lote só preparado: descartado na correção.
        val t = 5L
        db.loteDao().inserir(LoteEnvioEntidade("l1", c, StatusLote.PREPARADO, "wifi", "{}", t, t))
        repo.corrigirCodigo(c, alfa)
        assertEquals(StatusLote.DESCARTADO, db.loteDao().obter("l1")!!.status)
        // Lote que já saiu do aparelho: bloqueia até reconciliar.
        db.loteDao().inserir(LoteEnvioEntidade("l2", c, StatusLote.AGUARDANDO, "wifi", "{}", t, t))
        assertFailsWith<ErroGravacao> { repo.corrigirCodigo(c, bravo) }
        db.loteDao().mudarStatus("l2", StatusLote.CANCELADO, null, t)
        repo.corrigirCodigo(c, bravo)
        // Destino confirmado: não corrige mais.
        db.coletaDao().gravarDestino(c, "{}", t)
        assertFailsWith<ErroGravacao> { repo.corrigirCodigo(c, alfa) }
    }

    @Test
    fun rascunhoPresoAColeta() = runBlocking<Unit> { // RN-05
        val a = coleta(alfa)
        val b = coleta(bravo)
        repo.gravarRascunho("r1", a, null, "{}")
        repo.gravarRascunho("r2", b, null, "{}")
        assertEquals(listOf("r1"), repo.rascunhos(a).map { it.uuid })
        assertEquals(2, repo.rascunhos().size)
    }

    @Test
    fun responsavelEObservacoes() = runBlocking<Unit> { // RN-11, RN-15
        val c = coleta()
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(responsavel = " "), c) }
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(observacoes = "a".repeat(4001)), c) }
        val obs = "Família pede prazo.\nIdoso acamado."
        val p = repo.salvar(gravacao(observacoes = obs), c)
        assertEquals(obs, repo.carregar(p)!!.propriedade.observacoesRelevantes)
        assertEquals("José", repo.carregar(p)!!.propriedade.responsavelFamilia)
    }

    @Test
    fun salvarEditarEExcluirControlaRevETombstones() = runBlocking<Unit> {
        val col = coleta()
        repo.gravarRascunho("r1", col, null, "{}")
        val uuid = repo.salvar(gravacao(itens = listOf(item("eletro.geladeira"), item("ave.outros", 40, nome = "Codorna"))), col, rascunhoUuid = "r1")
        assertNull(repo.rascunho("r1"), "rascunho apagado na mesma transação")
        var c = assertNotNull(repo.carregar(uuid))
        assertEquals(1, c.propriedade.rev)
        assertEquals(2, c.itens.size)

        // Salvar sem mudança não gera rev nova.
        val semMudanca = gravacao(uuid, itens = c.itens.map { item(it.codigoCatalogo, it.quantidade, it.uuid, it.nomePersonalizado) })
        repo.salvar(semMudanca, col)
        assertEquals(1, repo.carregar(uuid)!!.propriedade.rev)

        // Muda quantidade da geladeira, remove "Outros", adiciona sofá.
        val gel = c.itens.first { it.codigoCatalogo == "eletro.geladeira" }
        repo.salvar(gravacao(uuid, bov = 5, itens = listOf(item("eletro.geladeira", 2, gel.uuid), item("movel.sofa"))), col)
        c = repo.carregar(uuid)!!
        assertEquals(2, c.propriedade.rev)
        assertEquals(setOf("eletro.geladeira", "movel.sofa"), c.itens.map { it.codigoCatalogo }.toSet())
        assertEquals(2, c.itens.first { it.codigoCatalogo == "eletro.geladeira" }.rev)
        val pendentes = db.itemDao().pendentes()
        assertEquals(3, pendentes.size) // geladeira (rev 2), sofá (novo) e codorna (tombstone)
        assertTrue(pendentes.any { it.nomePersonalizado == "Codorna" && it.excluidoEm != null && it.rev == 2L })

        repo.excluir(uuid)
        assertNull(repo.carregar(uuid))
        assertEquals(0, repo.observarPropriedades().first().size)
        val p = db.propriedadeDao().obter(uuid)!!
        assertEquals(3, p.rev)
        assertNotNull(p.excluidoEm) // tombstone, registro continua
    }

    @Test
    fun fotoSoComArquivoGravado() = runBlocking<Unit> {
        val c = coleta()
        val f = foto("ab345678-0000-4000-8000-000000000001")
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(fotos = listOf(f)), c) }
        assertEquals(0, db.propriedadeDao().pendentes().size, "nada gravado quando a foto falha")
        arquivos += f.caminhoRelativo
        val uuid = repo.salvar(gravacao(fotos = listOf(f)), c)
        assertEquals(1, repo.carregar(uuid)!!.fotos.size)
        repo.salvar(gravacao(uuid, fotos = emptyList()), c) // remove a foto
        assertEquals(0, repo.carregar(uuid)!!.fotos.size)
        assertNotNull(db.fotoDao().obter(f.uuid)!!.excluidoEm)
    }

    @Test
    fun regrasDeIntegridade() = runBlocking<Unit> {
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(), "coleta-inexistente") } // sem coleta
        val c = coleta()
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(nome = "  "), c) }
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(itens = listOf(item("movel.sofa"), item("movel.sofa"))), c) } // RN-32
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(itens = listOf(item("ave.outros", nome = null))), c) } // RN-33
        assertEquals(0, db.propriedadeDao().pendentes().size)
        val contagens = repo.observarContagens().first()
        assertEquals(Contagens(0, 0, 0), contagens)
    }
}
