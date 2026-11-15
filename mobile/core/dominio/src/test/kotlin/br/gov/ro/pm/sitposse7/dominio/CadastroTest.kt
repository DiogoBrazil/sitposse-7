package br.gov.ro.pm.sitposse7.dominio

import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonPrimitive
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.test.assertTrue

class CadastroTest {
    @Test
    fun catalogoIgualAFixtureCompartilhada() {
        val fx = Fixtures.ler("catalogo/itens.v1.json")
        assertEquals(Catalogo.VERSAO, fx.getValue("versao_catalogo").jsonPrimitive.int)
        val itens = fx.getValue("itens").jsonArray.map { it.obj() }
        assertEquals(itens.size, Catalogo.itens.size)
        itens.forEach { o ->
            val c = Catalogo.item(o.texto("codigo")!!)
            requireNotNull(c) { "código ausente no app: ${o.texto("codigo")}" }
            assertEquals(o.texto("categoria"), c.categoria.codigo)
            assertEquals(o.texto("nome_exibicao"), c.nome)
            assertEquals(o.inteiro("ordem")!!.toInt(), c.ordem)
            assertTrue(c.codigo.startsWith(c.categoria.prefixo + "."))
        }
    }

    private val valido = CadastroPropriedade(nome = "Sítio Boa Esperança", areaTexto = "12,5", areaUnidade = "alq_paulista",
        itens = listOf(ItemCadastro(codigo = "eletro.geladeira"), ItemCadastro(codigo = "ave.outros", nomePersonalizado = "Codorna", quantidade = 40)))

    @Test
    fun cadastroValidoSoComNome() {
        assertEquals(emptyList(), ValidacaoCadastro.validar(CadastroPropriedade(nome = "Só o nome")))
        assertEquals(emptyList(), ValidacaoCadastro.validar(valido))
    }

    @Test
    fun regrasDoFormulario() {
        fun campos(c: CadastroPropriedade) = ValidacaoCadastro.validar(c).map { it.campo }.toSet()
        assertEquals(setOf("nome"), campos(valido.copy(nome = "   "))) // RN-11
        assertEquals(setOf("area"), campos(valido.copy(areaUnidade = null))) // RN-42
        assertEquals(setOf("bov_adultos"), campos(valido.copy(quantidades = mapOf("bov_adultos" to -1)))) // RN-24
        assertEquals(setOf("item.1"), campos(valido.copy(itens = listOf(ItemCadastro(codigo = "movel.sofa"), ItemCadastro(codigo = "movel.sofa"))))) // RN-32
        assertEquals(setOf("item.0"), campos(valido.copy(itens = listOf(ItemCadastro(codigo = "ave.outros", nomePersonalizado = " "))))) // RN-33
        assertEquals(setOf("item.0"), campos(valido.copy(itens = listOf(ItemCadastro(codigo = "eletro.freezer", quantidade = 0))))) // RN-34
        assertEquals(setOf("coordenada"), campos(valido.copy(coordenada = Coordenada(-10.0, -62.0, "manual", precisaoM = 5.0)))) // RN-52
        assertEquals(setOf("coordenada"), campos(valido.copy(coordenada = Coordenada(-91.0, -62.0, "gnss")))) // RN-51
    }

    @Test
    fun outrosRepetidoAvisaPeloNomeNormalizado() {
        val itens = listOf(ItemCadastro(codigo = "ave.outros", nomePersonalizado = "Codórna  "))
        assertEquals(0, ValidacaoCadastro.outrosRepetido(itens, "ave.outros", "CODORNA"))
        assertNull(ValidacaoCadastro.outrosRepetido(itens, "eletro.outros", "Codorna")) // outra categoria
    }

    @Test
    fun textoDaAreaIdaEVolta() {
        for ((m, t) in listOf(12500L to "12,5", 1000L to "1", 1L to "0,001", 0L to "0", 2420L to "2,42")) {
            assertEquals(t, textoArea(m))
            assertEquals(m, (Areas.interpretar(t, "ha") as ResultadoArea.Valida).area.milesimos)
        }
    }
}
