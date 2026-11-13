package br.gov.ro.pm.sitposse7.dados

import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.delay
import kotlinx.coroutines.runBlocking
import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** Spike (c): PRAGMAs efetivos com Room 3 + BundledSQLiteDriver (ADR-0002). */
class PragmasTest {
    @Test
    fun pragmasEfetivosNaEscritaENasLeituras() = runBlocking {
        val dir = Files.createTempDirectory("banco").toFile()
        val db = abrirBancoJvm(dir.resolve(BancoCampo.NOME_ARQUIVO))
        try {
            db.metaDao().gravar(Meta("teste", "1", 1L, 1L)) // força a abertura e a criação
            val escrita = db.pragmasEscrita()
            // Várias leituras simultâneas para obrigar o pool a abrir conexões de leitura distintas.
            val leituras = (1..4).map { async { db.useLeituraLenta() } }.awaitAll()
            println("escrita=$escrita")
            leituras.forEachIndexed { i, p -> println("leitura[$i]=$p") }
            assertEquals("wal", escrita.journalMode.lowercase())
            assertEquals(2L, escrita.synchronous, "synchronous na conexão de escrita")
            assertEquals(1L, escrita.foreignKeys, "foreign_keys na conexão de escrita")
            assertTrue(escrita.conforme)
            leituras.forEach {
                assertEquals(2L, it.synchronous, "synchronous na leitura")
                assertEquals(1L, it.foreignKeys, "foreign_keys na leitura")
                assertEquals("wal", it.journalMode.lowercase())
            }
        } finally {
            db.close()
            dir.deleteRecursively()
        }
    }

    @Test
    fun pragmasPersistemAoReabrir() = runBlocking {
        val dir = Files.createTempDirectory("banco").toFile()
        val arquivo = dir.resolve(BancoCampo.NOME_ARQUIVO)
        try {
            abrirBancoJvm(arquivo).run { metaDao().gravar(Meta("a", "1", 1L, 1L)); close() }
            val db = abrirBancoJvm(arquivo)
            try {
                assertEquals("1", db.metaDao().ler("a")?.valor)
                assertTrue(db.pragmasEscrita().conforme)
                assertTrue(db.pragmasLeitura().conforme)
            } finally {
                db.close()
            }
        } finally {
            dir.deleteRecursively()
        }
    }

    private suspend fun BancoCampo.useLeituraLenta(): Pragmas {
        val p = pragmasLeitura()
        delay(100)
        return p
    }
}
