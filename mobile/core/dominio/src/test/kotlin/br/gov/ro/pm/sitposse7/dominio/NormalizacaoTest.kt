package br.gov.ro.pm.sitposse7.dominio

import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive
import kotlin.test.Test
import kotlin.test.assertEquals

class NormalizacaoTest {
    @Test
    fun casosDaFixture() {
        for (c in Fixtures.ler("normalizacao/nomes.json").getValue("casos").jsonArray.map { it.obj() }) {
            val n = Normalizacao.normalizarNome(c.texto("original")!!)
            val valido = c.getValue("valido").jsonPrimitive.boolean
            assertEquals(if (valido) c.texto("normalizado") else null, n, "caso $c")
        }
    }
}
