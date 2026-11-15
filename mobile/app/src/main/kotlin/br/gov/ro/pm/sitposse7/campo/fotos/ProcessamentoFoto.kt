package br.gov.ro.pm.sitposse7.campo.fotos

import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.Matrix
import android.media.ExifInterface
import java.io.File
import kotlin.math.max
import kotlin.math.roundToInt

/** Foto reduzida e gravada (ADR-0011): lado maior ≤ 2560 px, JPEG 85. */
data class FotoProcessada(val gravada: FotoGravada, val largura: Int, val altura: Int, val capturadaEm: Long)

object ProcessamentoFoto {
    const val LADO_MAXIMO = 2560
    const val QUALIDADE = 85

    /** Decodifica com amostragem (mobile/AGENTS.md), corrige a orientação, reduz e grava no [armazem]. */
    fun processar(captura: File, armazem: ArmazemFotos, agora: Long): FotoProcessada {
        val limites = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        BitmapFactory.decodeFile(captura.absolutePath, limites)
        require(limites.outWidth > 0 && limites.outHeight > 0) { "imagem inválida" }
        var amostra = 1
        while (max(limites.outWidth, limites.outHeight) / (amostra * 2) >= LADO_MAXIMO) amostra *= 2
        val bruto = BitmapFactory.decodeFile(captura.absolutePath, BitmapFactory.Options().apply { inSampleSize = amostra })
            ?: error("não foi possível ler a foto")
        val exif = runCatching { ExifInterface(captura.absolutePath) }.getOrNull()
        val graus = when (exif?.getAttributeInt(ExifInterface.TAG_ORIENTATION, ExifInterface.ORIENTATION_NORMAL)) {
            ExifInterface.ORIENTATION_ROTATE_90 -> 90f
            ExifInterface.ORIENTATION_ROTATE_180 -> 180f
            ExifInterface.ORIENTATION_ROTATE_270 -> 270f
            else -> 0f
        }
        val escala = minOf(1f, LADO_MAXIMO.toFloat() / max(bruto.width, bruto.height))
        val m = Matrix().apply { if (escala < 1f) postScale(escala, escala); if (graus != 0f) postRotate(graus) }
        val final = if (m.isIdentity) bruto else Bitmap.createBitmap(bruto, 0, 0, bruto.width, bruto.height, m, true)
        try {
            val gravada = armazem.gravar { saida -> check(final.compress(Bitmap.CompressFormat.JPEG, QUALIDADE, saida)) { "falha ao comprimir" } }
            return FotoProcessada(gravada, final.width, final.height, agora)
        } finally {
            if (final !== bruto) final.recycle()
            bruto.recycle()
            captura.delete()
        }
    }

    /** Miniatura com amostragem (para listas e grades). */
    fun miniatura(arquivo: File, ladoAlvo: Int = 320): Bitmap? {
        val o = BitmapFactory.Options().apply { inJustDecodeBounds = true }
        BitmapFactory.decodeFile(arquivo.absolutePath, o)
        if (o.outWidth <= 0) return null
        var amostra = 1
        while (max(o.outWidth, o.outHeight) / (amostra * 2) >= ladoAlvo) amostra *= 2
        val b = BitmapFactory.decodeFile(arquivo.absolutePath, BitmapFactory.Options().apply { inSampleSize = amostra }) ?: return null
        val esc = ladoAlvo.toFloat() / max(b.width, b.height)
        return if (esc >= 1f) b else Bitmap.createScaledBitmap(b, (b.width * esc).roundToInt(), (b.height * esc).roundToInt(), true).also { if (it !== b) b.recycle() }
    }
}
