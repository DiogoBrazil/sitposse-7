package br.gov.ro.pm.sitposse7.dados

import androidx.room3.migration.Migration
import androidx.sqlite.SQLiteDriver
import androidx.sqlite.driver.bundled.BundledSQLiteDriver

/*
 * Migrações do banco do celular (mobile/AGENTS.md: não destrutivas e testadas).
 *
 * A versão 3 é uma nova linha de base (A28, ADR-0025): os bancos v1/v2 eram só de teste do
 * MVP e foram substituídos por decisão do usuário. Não há migração 2 → 3: o app recusa abrir
 * esses bancos ([conferirGeracao]) e pede para reinstalar, sem apagar nada sozinho.
 * Daqui em diante, cada versão nova acrescenta a sua migração (MIGRACAO_3_4, ...) e o teste.
 */

/** Todas as migrações, em ordem. Toda versão nova acrescenta a sua aqui (mobile/AGENTS.md). */
val MIGRACOES: Array<Migration> = arrayOf()

/** O arquivo é de uma geração anterior do esquema (MVP); nada foi alterado nele. */
class BancoGeracaoAnterior(val versao: Long) :
    Exception("Os dados deste aparelho são de uma versão de teste anterior do app (banco v$versao) e não podem ser abertos por esta versão. Desinstale o app e instale-o de novo.")

/**
 * Confere, antes de abrir o Room, se o arquivo existente é de uma geração anterior
 * (ADR-0025). Só lê `PRAGMA user_version`; não escreve nada.
 */
fun conferirGeracao(caminho: String, existe: Boolean, driver: SQLiteDriver = BundledSQLiteDriver()) {
    if (!existe) return
    val c = driver.open(caminho)
    try {
        val st = c.prepare("PRAGMA user_version")
        val v = try {
            st.step()
            st.getLong(0)
        } finally {
            st.close()
        }
        if (v in BancoCampo.GERACOES_ANTERIORES) throw BancoGeracaoAnterior(v)
    } finally {
        c.close()
    }
}
