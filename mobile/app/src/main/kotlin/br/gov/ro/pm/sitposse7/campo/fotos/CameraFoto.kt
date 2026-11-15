package br.gov.ro.pm.sitposse7.campo.fotos

import androidx.camera.core.CameraSelector
import androidx.camera.core.ImageCapture
import androidx.camera.core.ImageCaptureException
import androidx.camera.core.Preview
import androidx.camera.lifecycle.ProcessCameraProvider
import androidx.camera.view.PreviewView
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilledIconButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButtonDefaults
import androidx.compose.material3.TextButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.content.ContextCompat
import androidx.lifecycle.compose.LocalLifecycleOwner
import br.gov.ro.pm.sitposse7.campo.R
import java.io.File
import java.util.UUID

/** Câmera em tela cheia: captura em `tmp/` e devolve o arquivo bruto para processar. */
@Composable
fun CameraFoto(pastaTmp: File, aoCapturar: (File) -> Unit, aoCancelar: () -> Unit) {
    val contexto = LocalContext.current
    val dono = LocalLifecycleOwner.current
    val previewView = remember { PreviewView(contexto) }
    val captura = remember { ImageCapture.Builder().setCaptureMode(ImageCapture.CAPTURE_MODE_MAXIMIZE_QUALITY).build() }
    var ocupado by remember { mutableStateOf(false) }
    var erro by remember { mutableStateOf<String?>(null) }

    DisposableEffect(dono) {
        val futuro = ProcessCameraProvider.getInstance(contexto)
        var provedor: ProcessCameraProvider? = null
        futuro.addListener({
            val p = futuro.get().also { provedor = it }
            val preview = Preview.Builder().build().also { it.surfaceProvider = previewView.surfaceProvider }
            p.unbindAll()
            runCatching { p.bindToLifecycle(dono, CameraSelector.DEFAULT_BACK_CAMERA, preview, captura) }.onFailure { erro = it.message }
        }, ContextCompat.getMainExecutor(contexto))
        onDispose { provedor?.unbindAll() }
    }

    Box(Modifier.fillMaxSize().background(Color.Black)) {
        AndroidView(factory = { previewView }, modifier = Modifier.fillMaxSize())
        erro?.let { Text(it, color = Color.White, modifier = Modifier.align(Alignment.Center).padding(24.dp)) }
        Row(
            Modifier.align(Alignment.BottomCenter).fillMaxWidth().navigationBarsPadding().padding(24.dp),
            horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically,
        ) {
            TextButton(onClick = aoCancelar) { Text(stringResource(R.string.cancelar), color = Color.White) }
            if (ocupado) CircularProgressIndicator(color = Color.White, modifier = Modifier.size(72.dp))
            else FilledIconButton(
                onClick = {
                    ocupado = true
                    pastaTmp.mkdirs()
                    val arquivo = File(pastaTmp, "${UUID.randomUUID()}.capture.jpg")
                    captura.takePicture(
                        ImageCapture.OutputFileOptions.Builder(arquivo).build(), ContextCompat.getMainExecutor(contexto),
                        object : ImageCapture.OnImageSavedCallback {
                            override fun onImageSaved(r: ImageCapture.OutputFileResults) { ocupado = false; aoCapturar(arquivo) }
                            override fun onError(e: ImageCaptureException) { ocupado = false; erro = e.message; arquivo.delete() }
                        },
                    )
                },
                modifier = Modifier.size(72.dp), shape = CircleShape,
                colors = IconButtonDefaults.filledIconButtonColors(containerColor = Color.White, contentColor = Color.Black),
            ) { Icon(painterResource(R.drawable.ic_camera), stringResource(R.string.foto_tirar), Modifier.size(36.dp)) }
            Box(Modifier.size(64.dp))
        }
    }
}
