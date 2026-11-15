package br.gov.ro.pm.sitposse7.campo.coletas

import androidx.compose.foundation.text.input.TextFieldState
import androidx.compose.foundation.text.input.delete
import androidx.compose.foundation.text.input.insert
import androidx.compose.ui.text.TextRange
import br.gov.ro.pm.sitposse7.dominio.Codigos
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class EntradaCodigoTest {
    @Test fun digitarApagarEEditarNoMeioPreservaCursor() {
        val s = TextFieldState()
        for (c in "g41htbbj") {
            s.edit { insert(selection.end, c.toString()); with(EntradaCodigo) { transformInput() } }
        }
        assertEquals("G41HTBBJ", s.text.toString())
        assertEquals(TextRange(8), s.selection)
        s.edit { selection = TextRange(4); delete(3, 4); with(EntradaCodigo) { transformInput() } }
        assertEquals("G41TBBJ", s.text.toString())
        assertEquals(TextRange(3), s.selection)
        s.edit { insert(selection.end, "h"); with(EntradaCodigo) { transformInput() } }
        assertEquals("G41HTBBJ", s.text.toString())
        assertEquals(TextRange(4), s.selection)
        s.edit { with(MascaraCodigo) { transformOutput() } }
        assertEquals("G41H-TBBJ", s.text.toString())
    }

    @Test fun colarCodigosEManterValidacaoExistente() {
        for (entrada in listOf("EQ-7K3M-9QX1", "eq7k3m9qx1", "7k3m-9qx1", "7k3m 9qx1")) {
            val s = TextFieldState()
            s.edit { insert(0, entrada); with(EntradaCodigo) { transformInput() } }
            assertEquals("7K3M9QX1", s.text.toString())
            assertTrue(Codigos.normalizar(s.text.toString()) is Codigos.Resultado.Valido)
        }
        val s = TextFieldState("G41HTBBJ")
        s.edit { selection = TextRange(0, 8); replace(0, 8, "a"); selection = TextRange(1); with(EntradaCodigo) { transformInput() } }
        assertEquals("A", s.text.toString())
        assertEquals(TextRange(1), s.selection)
        s.edit { delete(0, 1); with(EntradaCodigo) { transformInput() } }
        assertEquals("", s.text.toString())
        val eq = TextFieldState()
        eq.edit { insert(0, "EQ123456"); with(EntradaCodigo) { transformInput() } }
        assertEquals("EQ123456", eq.text.toString()) // EQ pode fazer parte do corpo
        assertTrue(Codigos.normalizar("7K3M9QX2") is Codigos.Resultado.Invalido)
    }
}
