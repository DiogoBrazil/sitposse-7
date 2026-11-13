plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
}

android {
    // Provisório: ver pendência P11 (applicationId definitivo antes da primeira instalação em campo).
    namespace = "br.gov.ro.pm.sitposse7.campo"
    compileSdk { version = release(37) }

    defaultConfig {
        applicationId = "br.gov.ro.pm.sitposse7.campo"
        minSdk { version = release(26) }   // A15
        targetSdk { version = release(37) } // P7
        versionCode = 1
        versionName = "0.0.1-fase0"
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

    testImplementation(kotlin("test-junit5"))
    testImplementation(platform(libs.junit.bom))
    testRuntimeOnly(libs.junit.platform.launcher)
}
