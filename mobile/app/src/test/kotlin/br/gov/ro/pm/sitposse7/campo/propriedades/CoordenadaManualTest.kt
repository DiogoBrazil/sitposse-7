package br.gov.ro.pm.sitposse7.campo.propriedades

import br.gov.ro.pm.sitposse7.dominio.Coordenada
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class CoordenadaManualTest {
    private val gnss = Coordenada(-10.123456789012345, -63.123456789012344, "gnss", 8.0, 123456L, 12)

    @Test fun confirmarSemAlterarPreservaPrecisaoEOrigem() {
        assertEquals(gnss, coordenadaManual(gnss.lat.toString(), gnss.lon.toString(), gnss))
        val manual = Coordenada(-10.0, -63.0, "manual")
        assertEquals(manual, coordenadaManual("-10,0", "-63", manual))
    }

    @Test fun editarGnssOuDigitarNovoPontoNaoInventaPrecisao() {
        val esperado = Coordenada(-10.5, -63.5, "manual")
        assertEquals(esperado, coordenadaManual(" -10,5 ", "-63.5", gnss))
        assertEquals(esperado, coordenadaManual("-10.5", "-63,5", null))
        assertEquals(Coordenada(-90.0, 180.0, "manual"), coordenadaManual("-90", "180", null))
    }

    @Test fun recusaPontoIncompletoNaoFinitoOuForaDaFaixa() {
        for ((lat, lon) in listOf("" to "1", "1" to "", "NaN" to "0", "0" to "Infinity", "90.1" to "0", "0" to "-180.1", "abc" to "1")) {
            assertNull(coordenadaManual(lat, lon, gnss))
        }
    }
}
