package br.gov.ro.pm.sitposse7.dados

import androidx.room3.useReaderConnection
import androidx.room3.useWriterConnection

/** PRAGMAs efetivos de uma conexão (spike c da Fase 0). */
data class Pragmas(
    val journalMode: String,
    /** 0=OFF, 1=NORMAL, 2=FULL, 3=EXTRA. */
    val synchronous: Long,
    val foreignKeys: Long,
    val sqliteVersao: String,
) {
    val conforme: Boolean get() = journalMode.equals("wal", ignoreCase = true) && synchronous >= 2 && foreignKeys == 1L
}

suspend fun BancoCampo.pragmasEscrita(): Pragmas = useWriterConnection { it.lerPragmas() }

suspend fun BancoCampo.pragmasLeitura(): Pragmas = useReaderConnection { it.lerPragmas() }

private suspend fun androidx.room3.Transactor.lerPragmas(): Pragmas {
    suspend fun texto(sql: String) = usePrepared(sql) { st -> st.step(); st.getText(0) }
    suspend fun numero(sql: String) = usePrepared(sql) { st -> st.step(); st.getLong(0) }
    return Pragmas(
        journalMode = texto("PRAGMA journal_mode"),
        synchronous = numero("PRAGMA synchronous"),
        foreignKeys = numero("PRAGMA foreign_keys"),
        sqliteVersao = texto("SELECT sqlite_version()"),
    )
}
