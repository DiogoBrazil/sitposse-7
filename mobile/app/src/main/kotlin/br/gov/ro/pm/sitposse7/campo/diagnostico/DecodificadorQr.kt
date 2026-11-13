package br.gov.ro.pm.sitposse7.campo.diagnostico

import com.google.zxing.BinaryBitmap
import com.google.zxing.ChecksumException
import com.google.zxing.DecodeHintType
import com.google.zxing.FormatException
import com.google.zxing.NotFoundException
import com.google.zxing.PlanarYUVLuminanceSource
import com.google.zxing.common.HybridBinarizer
import com.google.zxing.qrcode.QRCodeReader

/**
 * Decodifica QR a partir do plano de luminância (Y) de um quadro da câmera (ADR-0015).
 * Totalmente local: ZXing não usa rede nem Google Play Services.
 * Não é thread-safe: uma instância por analisador.
 */
class DecodificadorQr {
    private val leitor = QRCodeReader()

    // A Central gera o QR em modo byte UTF-8 sem ECI (biblioteca `qrcode`); sem esta dica
    // o ZXing pode adivinhar ISO-8859-1 e corromper acentos do nome da operação.
    private val dicas = mapOf(
        DecodeHintType.CHARACTER_SET to Charsets.UTF_8.name(),
        DecodeHintType.POSSIBLE_FORMATS to listOf(com.google.zxing.BarcodeFormat.QR_CODE),
    )

    /**
     * @param y bytes do plano Y; [passoLinha] é o `rowStride` (≥ [largura]).
     * @return o texto do QR, ou `null` se não houver QR legível no quadro.
     */
    fun decodificar(y: ByteArray, largura: Int, altura: Int, passoLinha: Int): String? {
        require(passoLinha >= largura && y.size >= passoLinha * (altura - 1) + largura) { "plano Y inconsistente" }
        val fonte = PlanarYUVLuminanceSource(y, passoLinha, altura, 0, 0, largura, altura, false)
        return try {
            leitor.decode(BinaryBitmap(HybridBinarizer(fonte)), dicas).text
        } catch (_: NotFoundException) {
            null
        } catch (_: ChecksumException) {
            null
        } catch (_: FormatException) {
            null
        } finally {
            leitor.reset()
        }
    }
}
