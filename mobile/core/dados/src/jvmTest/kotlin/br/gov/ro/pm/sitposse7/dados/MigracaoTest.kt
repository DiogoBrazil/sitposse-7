package br.gov.ro.pm.sitposse7.dados

import androidx.room3.testing.MigrationTestHelper
import androidx.sqlite.driver.bundled.BundledSQLiteDriver
import androidx.sqlite.execSQL
import kotlinx.coroutines.runBlocking
import java.io.File
import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith

/**
 * Esquema v3 (linha de base da A28, ADR-0025) e guarda contra bancos do MVP (v1/v2).
 * As migrações a partir da v3 entram aqui com MigrationTestHelper (mobile/AGENTS.md).
 */
class MigracaoTest {
    private val schemas = File(checkNotNull(System.getProperty("sitposse7.schemas")) { "propriedade sitposse7.schemas ausente" }).toPath()

    @Test
    fun bancoNovoCriaOEsquemaV3Validado() = runBlocking<Unit> {
        val dir = Files.createTempDirectory("esquema").toFile()
        val arquivo = dir.resolve(BancoCampo.NOME_ARQUIVO)
        try {
            val helper = MigrationTestHelper(schemas, arquivo.toPath(), BundledSQLiteDriver(), BancoCampo::class, { BancoCampoConstrutor.initialize() }, emptyList())
            // Cria a v3 a partir do esquema exportado e valida que bate com o que o Room espera.
            helper.createDatabase(3).close()
            helper.runMigrationsAndValidate(3, MIGRACOES.toList()).close()
            val db = abrirBancoJvm(arquivo)
            try {
                assertEquals(emptyList(), db.coletaDao().emAndamento())
            } finally {
                db.close()
            }
        } finally {
            dir.deleteRecursively()
        }
    }

    /** Bancos v1 e v2 (MVP) são recusados sem nenhuma alteração no arquivo. */
    @Test
    fun recusaBancoDaGeracaoAnteriorSemAlterar() = runBlocking<Unit> {
        for (versao in listOf(1, 2)) {
            val dir = Files.createTempDirectory("antigo").toFile()
            val arquivo = dir.resolve(BancoCampo.NOME_ARQUIVO)
            try {
                val c = BundledSQLiteDriver().open(arquivo.absolutePath)
                c.execSQL("CREATE TABLE equipe (uuid TEXT PRIMARY KEY, nome TEXT)")
                c.execSQL("INSERT INTO equipe VALUES ('e1', 'Equipe do MVP')")
                c.execSQL("PRAGMA user_version = $versao")
                c.close()
                val antes = arquivo.readBytes()
                val erro = assertFailsWith<BancoGeracaoAnterior> { abrirBancoJvm(arquivo) }
                assertEquals(versao.toLong(), erro.versao)
                assertEquals(antes.toList(), arquivo.readBytes().toList(), "o arquivo não pode mudar")
            } finally {
                dir.deleteRecursively()
            }
        }
    }
}
