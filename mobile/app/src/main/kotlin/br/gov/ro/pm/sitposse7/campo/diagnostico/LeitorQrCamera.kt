package br.gov.ro.pm.sitposse7.campo.diagnostico

import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageAnalysis
import androidx.camera.core.ImageProxy
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.LocalLifecycleOwner
import java.util.concurrent.Executors

/** Pré-visualização da câmera traseira com leitura de QR (ZXing, offline — ADR-0015). */
@Composable
fun LeitorQrCamera(aoLer: (String) -> Unit, modifier: Modifier = Modifier) {
    val contexto = LocalContext.current
    val dono = LocalLifecycleOwner.current
    val previewView = remember { PreviewView(contexto) }

    DisposableEffect(dono) {
        val executor = Executors.newSingleThreadExecutor()
        val decodificador = DecodificadorQr()
        val futuro = ProcessCameraProvider.getInstance(contexto)
        var provedor: ProcessCameraProvider? = null
        futuro.addListener({
            val p = futuro.get().also { provedor = it }
            val preview = Preview.Builder().build().also { it.surfaceProvider = previewView.surfaceProvider }
            val analise = ImageAnalysis.Builder()
                .setBackpressureStrategy(ImageAnalysis.STRATEGY_KEEP_ONLY_LATEST)
                .build()
                .also { it.setAnalyzer(executor) { img -> img.use { lerQuadro(it, decodificador)?.let(aoLer) } } }
            p.unbindAll()
            p.bindToLifecycle(dono, CameraSelector.DEFAULT_BACK_CAMERA, preview, analise)
        }, ContextCompat.getMainExecutor(contexto))
        onDispose {
            provedor?.unbindAll()
            executor.shutdown()
        }
    }

    AndroidView(factory = { previewView }, modifier = modifier)
}

/** O formato padrão do ImageAnalysis é YUV_420_888; o plano 0 (Y) tem pixelStride 1. */
private fun lerQuadro(img: ImageProxy, decodificador: DecodificadorQr): String? {
    val plano = img.planes[0]
    val buf = plano.buffer.duplicate().apply { rewind() }
    val y = ByteArray(buf.remaining()).also { buf.get(it) }
    return decodificador.decodificar(y, img.width, img.height, plano.rowStride)
}
