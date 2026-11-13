package br.gov.ro.pm.sitposse7.dados

import androidx.room3.Room
import java.io.File

/** Abre o banco num arquivo (testes JVM no host). */
fun abrirBancoJvm(arquivo: File): BancoCampo =
    Room.databaseBuilder<BancoCampo>(name = arquivo.absolutePath).configurarPadrao().build()
