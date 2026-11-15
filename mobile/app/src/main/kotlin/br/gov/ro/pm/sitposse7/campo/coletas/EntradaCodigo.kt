package br.gov.ro.pm.sitposse7.campo.coletas

import androidx.compose.foundation.text.input.InputTransformation
import androidx.compose.foundation.text.input.OutputTransformation
import androidx.compose.foundation.text.input.delete
import androidx.compose.foundation.text.input.insert
import java.util.Locale

/** Apenas o corpo é editável; mudanças locais no buffer preservam seleção e composição. */
internal val EntradaCodigo = InputTransformation {
    val completo = asCharSequence().toString().uppercase(Locale.ROOT)
    val semSeparadores = completo.filterNot { it.isWhitespace() || it in "-._/" }
    if (completo.startsWith("EQ-") || (semSeparadores.length == 10 && semSeparadores.startsWith("EQ"))) {
        delete(0, 2)
    }
    for (i in length - 1 downTo 0) {
        val c = charAt(i)
        when {
            c.isWhitespace() || c in "-._/" -> delete(i, i + 1)
            c in 'a'..'z' -> replace(i, i + 1, c.uppercaseChar().toString())
        }
    }
    if (length > 8) revertAllChanges()
}

/** Separador só visual: o Compose mapeia o cursor e a seleção automaticamente. */
internal val MascaraCodigo = OutputTransformation {
    if (length >= 4) insert(4, "-")
}
