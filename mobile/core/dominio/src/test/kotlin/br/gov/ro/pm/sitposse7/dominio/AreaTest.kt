package br.gov.ro.pm.sitposse7.dominio

import kotlinx.serialization.json.jsonArray
import kotlin.test.Test
import kotlin.test.assertEquals

class AreaTest {
    private val conversao = Fixtures.ler("area/conversao.json")

    @Test
    fun casosValidosDaFixture() {
        for (c in conversao.getValue("casos").jsonArray.map { it.obj() }) {
            val r = Areas.interpretar(c.texto("entrada")!!, c.texto("unidade"))
            val area = (r as? ResultadoArea.Valida)?.area ?: error("esperado válido para $c, obteve $r")
            assertEquals(c.inteiro("milesimos"), area.milesimos, "milesimos de $c")
            assertEquals(c.inteiro("dm2"), area.dm2, "dm2 de $c")
            assertEquals(c.texto("ha_exibicao"), Areas.formatarHa(area.dm2), "ha de $c")
        }
    }

    @Test
    fun casosInvalidosDaFixture() {
        for (c in conversao.getValue("invalidos").jsonArray.map { it.obj() }) {
            val r = Areas.interpretar(c.texto("entrada")!!, c.texto("unidade"))
            assertEquals(c.texto("erro"), (r as? ResultadoArea.Invalida)?.erro?.codigo, "erro de $c")
        }
    }

    @Test
    fun vazioNaoEhZero() {
        for (c in conversao.getValue("vazios").jsonArray.map { it.obj() }) {
            assertEquals(ResultadoArea.SemArea, Areas.interpretar(c.texto("entrada")!!, c.texto("unidade")))
        }
    }

    @Test
    fun somasDaFixture() {
        for (c in Fixtures.ler("area/somas.json").getValue("casos").jsonArray.map { it.obj() }) {
            val areas = c.getValue("propriedades").jsonArray.map { it.obj() }.map { p ->
                p.inteiro("milesimos")?.let { Area(it, UnidadeArea.doCodigo(p.texto("unidade")!!)!!) }
            }
            val soma = Areas.somar(areas)
            assertEquals(c.inteiro("total_dm2"), soma.totalDm2)
            assertEquals(c.texto("total_ha_exibicao"), Areas.formatarHa(soma.totalDm2))
            assertEquals(c.inteiro("com_area")?.toInt(), soma.comArea)
            assertEquals(c.inteiro("sem_area")?.toInt(), soma.semArea)
        }
    }
}
