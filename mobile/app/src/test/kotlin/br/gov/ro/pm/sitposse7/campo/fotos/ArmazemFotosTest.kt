package br.gov.ro.pm.sitposse7.campo.fotos

import java.io.IOException
import java.nio.file.Files
import java.security.MessageDigest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class ArmazemFotosTest {
    private val raiz = Files.createTempDirectory("fotos").toFile()
    private val armazem = ArmazemFotos(raiz)

    @Test
    fun gravaDuravelComHashDoArquivoGravado() {
        val bytes = ByteArray(300_000) { (it % 251).toByte() }
        val f = armazem.gravar { it.write(bytes) }
        val arq = armazem.arquivo(f.caminhoRelativo)
        assertTrue(arq.isFile)
        assertEquals(bytes.size.toLong(), f.tamanhoBytes)
        assertEquals(MessageDigest.getInstance("SHA-256").digest(arq.readBytes()).joinToString("") { "%02x".format(it) }, f.sha256) // RN-61
        assertEquals("${f.uuid.take(2)}/${f.uuid}.jpg", f.caminhoRelativo)
        assertTrue(raiz.walk().none { it.name.endsWith(".part") })
    }

    @Test
    fun falhaNaGravacaoNaoDeixaArquivo() {
        assertFailsWith<IOException> { armazem.gravar { it.write(1); throw IOException("disco cheio") } }
        assertTrue(raiz.walk().none { it.isFile }, "nem .part nem arquivo final")
    }

    @Test
    fun reconciliadorLimpaRestosEPreservaReferenciados() {
        val registrada = armazem.gravar { it.write(byteArrayOf(1, 2, 3)) }
        val orfa = armazem.gravar { it.write(byteArrayOf(4)) }
        armazem.tmp.mkdirs(); armazem.tmp.resolve("x.capture.jpg").writeText("x")
        armazem.arquivo("ab/resto.jpg.part").apply { parentFile.mkdirs() }.writeText("p")
        val ausentes = armazem.reconciliar(setOf(registrada.caminhoRelativo), mapOf(registrada.uuid to registrada.caminhoRelativo, "u-sumiu" to "zz/sumiu.jpg"))
        assertEquals(listOf("u-sumiu"), ausentes) // registro sem arquivo: alarme, nunca apagado
        assertTrue(armazem.existe(registrada.caminhoRelativo))
        assertFalse(armazem.existe(orfa.caminhoRelativo))
        assertTrue(raiz.resolve("quarentena/${orfa.uuid}.jpg").isFile, "órfã vai para a quarentena, não é apagada")
        assertFalse(armazem.tmp.exists())
        assertTrue(raiz.walk().none { it.name.endsWith(".part") })
    }
}
