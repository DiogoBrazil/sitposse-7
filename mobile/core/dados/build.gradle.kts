import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.android.kmp.library)
    alias(libs.plugins.ksp)
    alias(libs.plugins.room3)
}

// Módulo KMP (android + jvm): o mesmo código Room roda nos testes JVM do host com o
// mesmo BundledSQLiteDriver do aparelho (docs/planning/fase-0.md, spike c).
kotlin {
    android {
        namespace = "br.gov.ro.pm.sitposse7.dados"
        compileSdk = 37
        minSdk = 26
        compilerOptions { jvmTarget.set(JvmTarget.JVM_17) }
    }
    compilerOptions { freeCompilerArgs.add("-Xexpect-actual-classes") } // exigido pelo @ConstructedBy do Room
    jvm {
        compilerOptions { jvmTarget.set(JvmTarget.JVM_17) }
    }

    sourceSets {
        commonMain.dependencies {
            api(libs.room3.runtime) // BancoCampo herda de RoomDatabase
            implementation(libs.sqlite.bundled)
            implementation(libs.kotlinx.coroutines.core)
        }
        jvmTest.dependencies {
            implementation(kotlin("test"))
            implementation(libs.kotlinx.coroutines.test)
        }
    }
}

dependencies {
    add("kspAndroid", libs.room3.compiler)
    add("kspJvm", libs.room3.compiler)
    "jvmTestImplementation"(platform(libs.junit.bom))
    "jvmTestRuntimeOnly"(libs.junit.platform.launcher)
}

room3 {
    schemaDirectory("$projectDir/schemas") // versionado (mobile/AGENTS.md)
}

tasks.named<Test>("jvmTest") {
    useJUnitPlatform()
}
