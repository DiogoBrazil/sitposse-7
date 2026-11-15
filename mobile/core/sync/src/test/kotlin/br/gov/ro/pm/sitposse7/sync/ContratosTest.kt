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

/** As mesmas fixtures de contrato v2 que o Go valida (shared/test-fixtures/contratos, ADR-0017/0024). */
class ContratosTest {
    private val raiz = File(checkNotNull(System.getProperty("sitposse7.fixtures")))
    private fun ler(c: String) = Json.parseToJsonElement(File(raiz, c).readText()).jsonObject

    @Test
    fun lotesValidosIdaEVoltaSemPerderCampos() {
        ler("contratos/lote.v2.json").getValue("validos").jsonArray.forEach { v ->
            val lote = JsonContratos.decodeFromString(LoteV2.serializer(), v.toString())
            val volta = Json.parseToJsonElement(JsonContratos.encodeToString(LoteV2.serializer(), lote))
            assertEquals(v, volta, "o JSON gerado pelo app deve ter exatamente os campos do contrato")
        }
    }

    @Test
    fun reciboDaFixtureConfereOHash() {
        val fx = ler("contratos/recibo.v2.json")
        val recibo = JsonContratos.decodeFromString(ReciboV2.serializer(), fx.getValue("validos").jsonArray[0].toString())
        val hash = fx.getValue("hash") as JsonObject
        assertEquals(hash.getValue("texto_canonico").jsonPrimitive.content, recibo.textoCanonico())
        assertEquals(hash.getValue("recibo_sha256").jsonPrimitive.content, recibo.calcularHash())
        assertTrue(recibo.confere(recibo.loteUuid))
        assertTrue(!recibo.confere("outro-lote"))
        assertTrue(!recibo.copy(resultados = recibo.resultados.drop(1)).confere(recibo.loteUuid), "recibo adulterado não confere")
        assertTrue(!recibo.copy(destino = recibo.destino.copy(equipeCodigo = "EQ0000000X")).confere(recibo.loteUuid), "destino adulterado não confere")
    }

    @Test
    fun estadosEPareamentoDecodificam() {
        val estados = ler("contratos/estado-lote.v2.json").getValue("validos").jsonArray.map {
            JsonContratos.decodeFromString(EstadoLoteV2.serializer(), it.toString())
        }
        assertTrue(estados.any { it.status == StatusCentralLote.CANCELADO && it.motivo == "codigo_inexistente" })
        assertTrue(estados.any { it.status == StatusCentralLote.AGUARDANDO })
        val p = ler("contratos/pareamento.v2.json")
        p.getValue("pedidos_validos").jsonArray.forEach { v ->
            val ped = JsonContratos.decodeFromString(PedidoPareamentoV2.serializer(), v.toString())
            assertEquals(v, Json.parseToJsonElement(JsonContratos.encodeToString(PedidoPareamentoV2.serializer(), ped)))
        }
        p.getValue("respostas_validas").jsonArray.forEach { JsonContratos.decodeFromString(RespostaPareamentoV2.serializer(), it.toString()) }
    }

    @Test
    fun instanteNoFormatoDoContrato() {
        assertEquals("2026-11-14T13:05:10.250Z", instante(1_794_661_510_250L))
        assertEquals("1970-01-01T00:00:00.000Z", instante(0))
    }
}
