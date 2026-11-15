package br.gov.ro.pm.sitposse7.dominio

import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse

/** Mesma fixture da Central (shared/test-fixtures/codigos/codigos.v1.json). */
class CodigosTest {
    private val fx = Fixtures.ler("codigos/codigos.v1.json")

    private fun JsonObject.s(k: String) = getValue(k).jsonPrimitive.content

    @Test
    fun fixtureCompartilhada() {
        assertEquals(Codigos.ALFABETO, fx.s("alfabeto"))
        fx.getValue("digitos").jsonArray.forEach {
            val o = it.jsonObject
            assertEquals(o.s("digito").single(), Codigos.digitoVerificador(o.s("corpo")), o.s("corpo"))
        }
        fx.getValue("normalizacao").jsonArray.forEach {
            val o = it.jsonObject
            val r = Codigos.normalizar(o.s("entrada"), o.s("prefixo"))
            assertEquals(Codigos.Resultado.Valido(o.s("normalizado")), r, o.s("entrada"))
            assertEquals(o.s("exibicao"), Codigos.exibir(o.s("normalizado")))
        }
        val erros = mapOf("vazio" to Codigos.Erro.VAZIO, "tamanho" to Codigos.Erro.TAMANHO, "caractere" to Codigos.Erro.CARACTERE, "digito" to Codigos.Erro.DIGITO)
        fx.getValue("erros").jsonArray.forEach {
            val o = it.jsonObject
            assertEquals(Codigos.Resultado.Invalido(erros.getValue(o.s("erro"))), Codigos.normalizar(o.s("entrada"), o.s("prefixo")), o.s("entrada"))
        }
    }

    @Test
    fun trocaDeUmCaractereSempreDetectada() {
        val corpo7 = "7K3M9QX"
        val corpo = corpo7 + Codigos.digitoVerificador(corpo7)
        for (i in corpo.indices) for (x in Codigos.ALFABETO) {
            if (x == corpo[i]) continue
            val m = corpo.substring(0, i) + x + corpo.substring(i + 1)
            assertFalse(Codigos.normalizar(m) is Codigos.Resultado.Valido, m)
        }
    }

    @Test
    fun formatacaoNaDigitacao() {
        assertEquals("", Codigos.formatarDigitacao(" -"))
        assertEquals("EQ-7K", Codigos.formatarDigitacao("7k"))
        assertEquals("EQ-7K3M", Codigos.formatarDigitacao("eq7k3m"))
        assertEquals("EQ-7K3M-9QX1", Codigos.formatarDigitacao("EQ-7K3M-9QX1"))
        assertEquals("EQ-7K3M-9QX1", Codigos.formatarDigitacao("eq 7k3m 9qx1 99"))
    }
}
