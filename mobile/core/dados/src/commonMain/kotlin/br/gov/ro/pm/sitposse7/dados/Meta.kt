package br.gov.ro.pm.sitposse7.dados

import androidx.room3.Dao
import androidx.room3.Entity
import androidx.room3.PrimaryKey
import androidx.room3.Query
import androidx.room3.Upsert

/**
 * Metadados da instalação (Fase 0), espelho de desktop/internal/banco/migracoes/0001_meta.sql.
 * O esquema de negócio (docs/architecture/modelo-de-dados.md) entra na Fase 1.
 * Timestamps em epoch ms UTC (ADR-0010).
 */
@Entity(tableName = "meta")
data class Meta(
    @PrimaryKey val chave: String,
    val valor: String,
    val created_at: Long,
    val updated_at: Long,
)

@Dao
interface MetaDao {
    @Upsert
    suspend fun gravar(meta: Meta)

    @Query("SELECT * FROM meta WHERE chave = :chave")
    suspend fun ler(chave: String): Meta?
}
