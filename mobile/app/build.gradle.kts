import java.util.Properties

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

// Assinatura do APK de release (A32): chave própria do SITPOSSE-7 FORA do repositório.
// O arquivo de propriedades (arquivo, alias, senha) fica em ~/.sitposse7/assinatura-campo.properties
// ou no caminho da variável SITPOSSE7_ASSINATURA_CAMPO. Sem ele, o release não é gerado (o debug
// continua igual). Atualizações só instalam por cima com a MESMA chave: guarde cópia de segurança.
val arquivoAssinatura = file(
    System.getenv("SITPOSSE7_ASSINATURA_CAMPO")
        ?: "${System.getProperty("user.home")}/.sitposse7/assinatura-campo.properties",
)
val assinatura = Properties().apply { if (arquivoAssinatura.isFile) arquivoAssinatura.inputStream().use { load(it) } }

android {
    // Provisório: ver pendência P11 (applicationId definitivo antes da primeira instalação em campo).
    namespace = "br.gov.ro.pm.sitposse7.campo"
    compileSdk { version = release(37) }

    defaultConfig {
        applicationId = "br.gov.ro.pm.sitposse7.campo"
        minSdk { version = release(26) }   // A15
        targetSdk { version = release(37) } // P7
        versionCode = 5
        versionName = "0.2.1-mvp"
    }

    signingConfigs {
        if (!assinatura.isEmpty) {
            create("release") {
                storeFile = file(assinatura.getProperty("arquivo"))
                storePassword = assinatura.getProperty("senha")
                keyAlias = assinatura.getProperty("alias")
                keyPassword = assinatura.getProperty("senha")
            }
        }
    }

    buildTypes {
        getByName("release") {
            // Sem R8/minificação por enquanto: evita quebrar serialização e reflexão sem testes
            // próprios do release (o ganho de tamanho fica para a Fase 7).
            isMinifyEnabled = false
            signingConfig = signingConfigs.findByName("release")
        }
    }

    buildFeatures { compose = true }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    testOptions {
        unitTests.all { it.useJUnitPlatform() }
    }

    packaging {
        resources.excludes += setOf("META-INF/{AL2.0,LGPL2.1}", "META-INF/versions/9/OSGI-INF/MANIFEST.MF")
    }
}

dependencies {
    implementation(project(":core:dominio"))
    implementation(project(":core:dados"))
    implementation(project(":core:sync"))

    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.runtime.compose)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.compose.ui)
    implementation(libs.androidx.compose.material3)
    implementation(libs.androidx.compose.ui.tooling.preview)
    debugImplementation(libs.androidx.compose.ui.tooling)

    implementation(libs.androidx.camera.camera2)
    implementation(libs.androidx.camera.lifecycle)
    implementation(libs.androidx.camera.view)
    implementation(libs.zxing.core) // QR offline, sem Play Services nem telemetria (ADR-0015)

    implementation(libs.kotlinx.coroutines.android)
    implementation(libs.kotlinx.serialization.json) // rascunho do formulário (JSON)

    testImplementation(kotlin("test-junit5"))
    testImplementation(platform(libs.junit.bom))
    testRuntimeOnly(libs.junit.platform.launcher)
}

// Release sem a chave: falha com explicação em vez de gerar um APK sem assinatura.
run {
    // Só valores locais no doFirst (compatível com o configuration cache).
    val semChave = assinatura.isEmpty
    val caminho = arquivoAssinatura.path
    tasks.matching { it.name == "packageRelease" }.configureEach {
        doFirst {
            check(!semChave) { "Chave de release do SITPOSSE-7 não encontrada em $caminho (ver docs/planning/continuacao.md, A32)." }
        }
    }
}
