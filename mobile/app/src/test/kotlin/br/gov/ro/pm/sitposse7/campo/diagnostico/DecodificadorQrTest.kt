package br.gov.ro.pm.sitposse7.campo.diagnostico

import com.google.zxing.BarcodeFormat
import com.google.zxing.EncodeHintType
import com.google.zxing.qrcode.QRCodeWriter
import com.google.zxing.qrcode.decoder.ErrorCorrectionLevel
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class DecodificadorQrTest {
    private val json = """{"v":1,"central_id":"3f1c9a52-6a0e-4c1b-9d55-2f7f1f0b8e21","ips":["192.168.137.1"],""" +
        """"porta":47807,"spki_sha256":"LebAdzGYtDPoyDIJZAO3bxa_zvkK8dw6lhVmgEsPdfk","operacao_nome":"Reintegração Gleba Ação"}"""

    /** Gera o plano Y de um "quadro" com o QR, com bytes extras no fim de cada linha (rowStride > largura). */
    private fun quadro(texto: String, lado: Int, sobra: Int): Triple<ByteArray, Int, Int> {
        // Como a biblioteca `qrcode` da Central: bytes UTF-8 em modo byte, SEM segmento ECI.
        // Sem CHARACTER_SET o ZXing codifica em ISO-8859-1 sem ECI; passar os bytes UTF-8
        // "disfarçados" de Latin-1 reproduz exatamente esse QR.
        val bytesUtf8ComoLatin1 = String(texto.toByteArray(Charsets.UTF_8), Charsets.ISO_8859_1)
        val dicas = mapOf<EncodeHintType, Any>(
            EncodeHintType.ERROR_CORRECTION to ErrorCorrectionLevel.M,
            EncodeHintType.MARGIN to 4,
        )
        val m = QRCodeWriter().encode(bytesUtf8ComoLatin1, BarcodeFormat.QR_CODE, lado, lado, dicas)
        val passo = lado + sobra
        val y = ByteArray(passo * lado) { 0x55 }
        for (lin in 0 until lado) for (col in 0 until lado) {
            y[lin * passo + col] = if (m[col, lin]) 0 else 0xFF.toByte()
        }
        return Triple(y, lado, passo)
    }

    @Test
    fun decodificaUtf8SemEciComRowStride() {
        val (y, lado, passo) = quadro(json, 480, 32)
        assertEquals(json, DecodificadorQr().decodificar(y, lado, lado, passo))
    }

    @Test
    fun quadroSemQrRetornaNulo() {
        assertNull(DecodificadorQr().decodificar(ByteArray(320 * 240) { 0x7F }, 320, 240, 320))
    }

    @Test
    fun reutilizavelEntreQuadros() {
        val d = DecodificadorQr()
        val (y, lado, passo) = quadro(json, 400, 0)
        assertNull(d.decodificar(ByteArray(400 * 400), 400, 400, 400))
        assertEquals(json, d.decodificar(y, lado, lado, passo))
    }
}
