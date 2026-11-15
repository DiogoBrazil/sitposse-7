package br.gov.ro.pm.sitposse7.dados

import android.content.Context
import androidx.room3.Room

/**
 * Abre o banco no diretório privado do app.
 * Lança [BancoGeracaoAnterior] se o arquivo for da geração do MVP (ADR-0025).
 */
fun abrirBancoAndroid(contexto: Context): BancoCampo {
    val app = contexto.applicationContext
    val arquivo = app.getDatabasePath(BancoCampo.NOME_ARQUIVO)
    conferirGeracao(arquivo.absolutePath, arquivo.exists())
    return Room.databaseBuilder<BancoCampo>(app, arquivo.absolutePath)
        .configurarPadrao()
        .build()
}
