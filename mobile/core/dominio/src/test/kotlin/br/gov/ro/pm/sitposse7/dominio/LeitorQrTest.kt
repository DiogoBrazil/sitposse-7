package br.gov.ro.pm.sitposse7.dominio

import kotlinx.serialization.json.jsonArray
import kotlin.test.Test
import kotlin.test.assertIs

class LeitorQrTest {
    private val fixture = Fixtures.ler("contratos/qr-pareamento.v1.json")

    @Test
    fun validosDaFixture() {
        for (c in fixture.getValue("validos").jsonArray) {
            assertIs<LeituraQr.Valido>(LeitorQr.ler(c.toString()), "caso $c")
        }
    }

    @Test
    fun invalidosDaFixture() {
        for (c in fixture.getValue("invalidos").jsonArray.map { it.obj() }) {
            assertIs<LeituraQr.Invalido>(LeitorQr.ler(c.getValue("dado").toString()), "motivo ${c.texto("motivo")}")
        }
    }

    @Test
    fun textoQueNaoEhJson() {
        assertIs<LeituraQr.Invalido>(LeitorQr.ler("https://exemplo.invalido/qr"))
    }
}
