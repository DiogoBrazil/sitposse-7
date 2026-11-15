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

    private fun gravacao(uuid: String? = null, nome: String = "Sítio", itens: List<GravacaoItem> = emptyList(), fotos: List<GravacaoFoto> = emptyList(), bov: Long = 3) =
        GravacaoPropriedade(uuid, nome, null, mapOf("bov_adultos" to bov), 12500, "alq_paulista", null, null, null, null, null, null, itens, fotos)

    private fun item(codigo: String, qtd: Long = 1, uuid: String? = null, nome: String? = null) =
        GravacaoItem(uuid, codigo.substringBefore('.').let { if (it == "eletro") "eletrodomestico" else it }, codigo, nome, nome?.lowercase(), qtd)

    private fun foto(uuid: String) = GravacaoFoto(uuid, "${uuid.take(2)}/$uuid.jpg", 10, "a".repeat(64), 1, 1, 1)

    @Test
    fun equipeEInstalacao() = runBlocking<Unit> {
        val e = repo.configurarEquipe("Equipe 1", "Sgt. A", " ")
        assertNull(e.identificacaoArea)
        assertEquals(36, repo.instalacaoUuid().length)
        assertFailsWith<ErroGravacao> { repo.configurarEquipe("Outra", "B", null) }
        val editada = repo.editarEquipe("Equipe 1", "Sgt. B", "Linha 25")
        assertEquals(e.uuid, editada.uuid) // RN-02
        assertEquals(2, editada.rev)
        assertEquals(2, repo.editarEquipe("Equipe 1", "Sgt. B", "Linha 25").rev) // sem mudança, sem rev nova
    }

    @Test
    fun salvarEditarEExcluirControlaRevETombstones() = runBlocking<Unit> {
        repo.configurarEquipe("Equipe 1", "Sgt. A", null)
        repo.gravarRascunho("r1", null, "{}")
        val uuid = repo.salvar(gravacao(itens = listOf(item("eletro.geladeira"), item("ave.outros", 40, nome = "Codorna"))), rascunhoUuid = "r1")
        assertNull(repo.rascunho("r1"), "rascunho apagado na mesma transação")
        var c = assertNotNull(repo.carregar(uuid))
        assertEquals(1, c.propriedade.rev)
        assertEquals(2, c.itens.size)

        // Salvar sem mudança não gera rev nova.
        val semMudanca = gravacao(uuid, itens = c.itens.map { item(it.codigoCatalogo, it.quantidade, it.uuid, it.nomePersonalizado) })
        repo.salvar(semMudanca)
        assertEquals(1, repo.carregar(uuid)!!.propriedade.rev)

        // Muda quantidade da geladeira, remove "Outros", adiciona sofá.
        val gel = c.itens.first { it.codigoCatalogo == "eletro.geladeira" }
        repo.salvar(gravacao(uuid, bov = 5, itens = listOf(item("eletro.geladeira", 2, gel.uuid), item("movel.sofa"))))
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
        repo.configurarEquipe("Equipe 1", "Sgt. A", null)
        val f = foto("ab345678-0000-4000-8000-000000000001")
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(fotos = listOf(f))) }
        assertEquals(0, db.propriedadeDao().pendentes().size, "nada gravado quando a foto falha")
        arquivos += f.caminhoRelativo
        val uuid = repo.salvar(gravacao(fotos = listOf(f)))
        assertEquals(1, repo.carregar(uuid)!!.fotos.size)
        repo.salvar(gravacao(uuid, fotos = emptyList())) // remove a foto
        assertEquals(0, repo.carregar(uuid)!!.fotos.size)
        assertNotNull(db.fotoDao().obter(f.uuid)!!.excluidoEm)
    }

    @Test
    fun regrasDeIntegridade() = runBlocking<Unit> {
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao()) } // sem equipe
        repo.configurarEquipe("Equipe 1", "Sgt. A", null)
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(nome = "  ")) }
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(itens = listOf(item("movel.sofa"), item("movel.sofa")))) } // RN-32
        assertFailsWith<ErroGravacao> { repo.salvar(gravacao(itens = listOf(item("ave.outros", nome = null)))) } // RN-33
        assertEquals(0, db.propriedadeDao().pendentes().size)
        val contagens = repo.observarContagens().first()
        assertEquals(Contagens(0, 0, 0), contagens)
    }
}
