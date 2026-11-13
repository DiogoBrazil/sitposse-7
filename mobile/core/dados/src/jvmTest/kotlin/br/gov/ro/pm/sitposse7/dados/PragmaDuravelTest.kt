package br.gov.ro.pm.sitposse7.dados

import kotlin.test.Test
import kotlin.test.assertEquals

class PragmaDuravelTest {
    @Test
    fun elevaSynchronousAbaixoDeFull() {
        for (sql in listOf("PRAGMA synchronous = NORMAL", "pragma synchronous=off", "PRAGMA main.synchronous = 1", "PRAGMA synchronous = 0;", " PRAGMA synchronous='normal' ")) {
            assertEquals("PRAGMA synchronous = FULL", PragmaDuravel.reescrever(sql), sql)
        }
    }

    @Test
    fun mantemDuraveisEOutrosComandos() {
        for (sql in listOf("PRAGMA synchronous = FULL", "PRAGMA synchronous = EXTRA", "PRAGMA synchronous = 2", "PRAGMA synchronous", "PRAGMA journal_mode = WAL", "SELECT 'PRAGMA synchronous = NORMAL'")) {
            assertEquals(sql, PragmaDuravel.reescrever(sql), sql)
        }
    }
}
