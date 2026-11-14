package br.gov.ro.pm.sitposse7.dados

import androidx.room3.testing.MigrationTestHelper
import androidx.sqlite.driver.bundled.BundledSQLiteDriver
import androidx.sqlite.execSQL
import kotlinx.coroutines.runBlocking
import java.io.File
import java.nio.file.Files
import kotlin.test.Test
import kotlin.test.assertEquals

/** Migrações não destrutivas testadas a partir de cada versão publicada (mobile/AGENTS.md). */
class MigracaoTest {
    private val schemas = File(checkNotNull(System.getProperty("sitposse7.schemas")) { "propriedade sitposse7.schemas ausente" }).toPath()

    @Test
    fun migra1Para2PreservandoDadosEValidandoOEsquema() = runBlocking<Unit> {
        val dir = Files.createTempDirectory("migracao").toFile()
        val arquivo = dir.resolve(BancoCampo.NOME_ARQUIVO)
        try {
            val helper = MigrationTestHelper(schemas, arquivo.toPath(), BundledSQLiteDriver(), BancoCampo::class, { BancoCampoConstrutor.initialize() }, emptyList())
            helper.createDatabase(1).apply {
                execSQL("INSERT INTO meta (chave, valor, created_at, updated_at) VALUES ('instalacao_uuid', 'a1b2c3d4-0000-4000-8000-000000000001', 1, 1)")
                close()
            }
            // Valida o esquema resultante contra schemas/2.json (falha se a migração divergir do Room).
            helper.runMigrationsAndValidate(2, MIGRACOES.toList()).close()

            // O banco migrado abre pelo caminho normal do app e mantém os dados.
            val db = abrirBancoJvm(arquivo)
            try {
                assertEquals("a1b2c3d4-0000-4000-8000-000000000001", db.metaDao().ler("instalacao_uuid")?.valor)
                assertEquals(null, db.equipeDao().obter())
            } finally {
                db.close()
            }
        } finally {
            dir.deleteRecursively()
        }
    }
}
