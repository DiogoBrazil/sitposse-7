package br.gov.ro.pm.sitposse7.dados

import androidx.room3.ConstructedBy
import androidx.room3.Database
import androidx.room3.RoomDatabase
import androidx.room3.RoomDatabaseConstructor

/**
 * Banco local do aplicativo de campo (ADR-0002).
 * Versão nova exige migração não destrutiva testada (mobile/AGENTS.md).
 */
@Database(entities = [Meta::class], version = 1, exportSchema = true)
@ConstructedBy(BancoCampoConstrutor::class)
abstract class BancoCampo : RoomDatabase() {
    abstract fun metaDao(): MetaDao

    companion object {
        const val NOME_ARQUIVO = "sitposse7-campo.db"
    }
}

// Implementação gerada pelo Room (KSP) para cada alvo.
@Suppress("KotlinNoActualForExpect")
expect object BancoCampoConstrutor : RoomDatabaseConstructor<BancoCampo> {
    override fun initialize(): BancoCampo
}
