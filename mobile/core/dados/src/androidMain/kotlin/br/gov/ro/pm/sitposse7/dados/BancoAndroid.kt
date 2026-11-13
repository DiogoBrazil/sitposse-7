package br.gov.ro.pm.sitposse7.dados

import android.content.Context
import androidx.room3.Room

/** Abre o banco no diretório privado do app. */
fun abrirBancoAndroid(contexto: Context): BancoCampo {
    val app = contexto.applicationContext
    return Room.databaseBuilder<BancoCampo>(app, app.getDatabasePath(BancoCampo.NOME_ARQUIVO).absolutePath)
        .configurarPadrao()
        .build()
}
