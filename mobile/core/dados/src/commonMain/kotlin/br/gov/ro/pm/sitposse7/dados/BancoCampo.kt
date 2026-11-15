package br.gov.ro.pm.sitposse7.dados

import androidx.room3.ConstructedBy
import androidx.room3.Database
import androidx.room3.RoomDatabase
import androidx.room3.RoomDatabaseConstructor

/**
 * Banco local do aplicativo de campo (ADR-0002).
 * Versão nova exige migração não destrutiva testada (mobile/AGENTS.md).
 */
@Database(
    entities = [
        Meta::class, ColetaEntidade::class, ColetaCodigoEntidade::class, PropriedadeEntidade::class, ItemEntidade::class, FotoEntidade::class,
        RascunhoEntidade::class, LoteEnvioEntidade::class, LoteItemEntidade::class, PareamentoEntidade::class,
    ],
    version = 3,
    exportSchema = true,
)
@ConstructedBy(BancoCampoConstrutor::class)
abstract class BancoCampo : RoomDatabase() {
    abstract fun metaDao(): MetaDao
    abstract fun coletaDao(): ColetaDao
    abstract fun propriedadeDao(): PropriedadeDao
    abstract fun itemDao(): ItemDao
    abstract fun fotoDao(): FotoDao
    abstract fun rascunhoDao(): RascunhoDao
    abstract fun loteDao(): LoteDao
    abstract fun pareamentoDao(): PareamentoDao

    companion object {
        const val NOME_ARQUIVO = "sitposse7-campo.db"

        /** Versões do MVP (v1 e v2) substituídas pela linha de base v3 (ADR-0025). */
        val GERACOES_ANTERIORES = 1L..2L
    }
}

// Implementação gerada pelo Room (KSP) para cada alvo.
@Suppress("KotlinNoActualForExpect")
expect object BancoCampoConstrutor : RoomDatabaseConstructor<BancoCampo> {
    override fun initialize(): BancoCampo
}
