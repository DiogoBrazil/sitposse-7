package br.gov.ro.pm.sitposse7.dados

import androidx.sqlite.SQLiteConnection
import androidx.sqlite.SQLiteDriver
import androidx.sqlite.SQLiteStatement
import androidx.sqlite.execSQL

/**
 * Garante em **toda** conexão `synchronous=FULL` e `foreign_keys=ON` (ADR-0002).
 *
 * Achado do spike (c), Room 3.0.3: em WAL o Room executa `PRAGMA synchronous = NORMAL`
 * em cada conexão do pool *depois* de o driver abri-la, e `RoomDatabase.Callback.onOpen`
 * só roda na primeira conexão. Com NORMAL, uma queda de energia pode desfazer as últimas
 * transações já informadas ao usuário como "Salvo no aparelho" — inaceitável (AGENTS.md §1).
 * Por isso o ajuste fica no driver: aplica os PRAGMAs na abertura e eleva para FULL
 * qualquer tentativa de reduzir `synchronous`. O teste PragmasTest detecta se uma versão
 * futura do Room mudar esse comportamento.
 */
class DriverDuravel(private val base: SQLiteDriver) : SQLiteDriver {
    override val hasConnectionPool: Boolean get() = base.hasConnectionPool

    override fun open(fileName: String): SQLiteConnection {
        val conexao = base.open(fileName)
        try {
            conexao.execSQL("PRAGMA foreign_keys = ON")
            conexao.execSQL("PRAGMA synchronous = FULL")
        } catch (t: Throwable) {
            conexao.close()
            throw t
        }
        return ConexaoDuravel(conexao)
    }
}

private class ConexaoDuravel(private val base: SQLiteConnection) : SQLiteConnection by base {
    override fun prepare(sql: String): SQLiteStatement = base.prepare(PragmaDuravel.reescrever(sql))
}

internal object PragmaDuravel {
    private val pragmaSynchronous =
        Regex("""^\s*PRAGMA\s+(?:main\.)?synchronous\s*=\s*['"]?(\w+)['"]?\s*;?\s*$""", RegexOption.IGNORE_CASE)
    private val duraveis = setOf("FULL", "EXTRA", "2", "3")

    /** Troca `PRAGMA synchronous = <menos que FULL>` por FULL; o resto passa intacto. */
    fun reescrever(sql: String): String {
        val m = pragmaSynchronous.matchEntire(sql) ?: return sql
        return if (m.groupValues[1].uppercase() in duraveis) sql else "PRAGMA synchronous = FULL"
    }
}
