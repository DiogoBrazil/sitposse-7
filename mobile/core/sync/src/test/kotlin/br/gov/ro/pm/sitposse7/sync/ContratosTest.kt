package br.gov.ro.pm.sitposse7.sync

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import java.io.File
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/** As mesmas fixtures de contrato que o Go valida (shared/test-fixtures/contratos, ADR-0017). */
class ContratosTest {
    private val raiz = File(checkNotNull(System.getProperty("sitposse7.fixtures")))
    private fun ler(c: String) = Json.parseToJsonElement(File(raiz, c).readText()).jsonObject

    @Test
    fun lotesValidosIdaEVoltaSemPerderCampos() {
        ler("contratos/lote.v1.json").getValue("validos").jsonArray.forEach { v ->
            val lote = JsonContratos.decodeFromString(LoteV1.serializer(), v.toString())
            val volta = Json.parseToJsonElement(JsonContratos.encodeToString(LoteV1.serializer(), lote))
            assertEquals(v, volta, "o JSON gerado pelo app deve ter exatamente os campos do contrato")
        }
    }

    @Test
    fun reciboDaFixtureConfereOHash() {
        val fx = ler("contratos/recibo.v1.json")
        val recibo = JsonContratos.decodeFromString(ReciboV1.serializer(), fx.getValue("validos").jsonArray[0].toString())
        val hash = fx.getValue("hash") as JsonObject
        assertEquals(hash.getValue("texto_canonico").jsonPrimitive.content, recibo.textoCanonico())
        assertEquals(hash.getValue("recibo_sha256").jsonPrimitive.content, recibo.calcularHash())
        assertTrue(recibo.confere(recibo.loteUuid))
        assertTrue(!recibo.confere("outro-lote"))
        assertTrue(!recibo.copy(resultados = recibo.resultados.drop(1)).confere(recibo.loteUuid), "recibo adulterado não confere")
    }

    @Test
    fun estadosEPareamentoDecodificam() {
        ler("contratos/estado-lote.v1.json").getValue("validos").jsonArray.forEach {
            JsonContratos.decodeFromString(EstadoLoteV1.serializer(), it.toString())
        }
        val p = ler("contratos/pareamento.v1.json")
        p.getValue("pedidos_validos").jsonArray.forEach { v ->
            val ped = JsonContratos.decodeFromString(PedidoPareamentoV1.serializer(), v.toString())
            assertEquals(v, Json.parseToJsonElement(JsonContratos.encodeToString(PedidoPareamentoV1.serializer(), ped)))
        }
        p.getValue("respostas_validas").jsonArray.forEach { JsonContratos.decodeFromString(RespostaPareamentoV1.serializer(), it.toString()) }
    }

    @Test
    fun instanteNoFormatoDoContrato() {
        assertEquals("2026-11-14T13:05:10.250Z", instante(1_794_661_510_250L))
        assertEquals("1970-01-01T00:00:00.000Z", instante(0))
    }
}
