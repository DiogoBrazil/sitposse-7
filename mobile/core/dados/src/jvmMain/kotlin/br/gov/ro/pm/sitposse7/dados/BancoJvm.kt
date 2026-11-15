package br.gov.ro.pm.sitposse7.dados

import androidx.room3.Room
import java.io.File

/** Abre o banco num arquivo (testes JVM no host). Mesma guarda de geração do app. */
fun abrirBancoJvm(arquivo: File): BancoCampo {
    conferirGeracao(arquivo.absolutePath, arquivo.exists())
    return Room.databaseBuilder<BancoCampo>(name = arquivo.absolutePath).configurarPadrao().build()
}
