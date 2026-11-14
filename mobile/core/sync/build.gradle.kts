import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.serialization)
}

java {
    sourceCompatibility = JavaVersion.VERSION_17
    targetCompatibility = JavaVersion.VERSION_17
}

kotlin {
    compilerOptions { jvmTarget.set(JvmTarget.JVM_17) }
}

dependencies {
    api(project(":core:dados")) // Sincronizador lê e confirma no banco (variante jvm nos testes, android no app)
    api(project(":core:dominio"))
    implementation(libs.okhttp)
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.kotlinx.coroutines.core)

    testImplementation(kotlin("test"))
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(platform(libs.junit.bom))
    testRuntimeOnly(libs.junit.platform.launcher)
}

val fixtures = rootProject.file("../shared/test-fixtures")
val desktop = rootProject.file("../desktop")

tasks.test {
    useJUnitPlatform()
    systemProperty("sitposse7.fixtures", fixtures.absolutePath)
    systemProperty("sitposse7.desktop", desktop.absolutePath)
    inputs.dir(fixtures).withPathSensitivity(PathSensitivity.RELATIVE)
    // Teste cruzado com o servidor Go real: reexecuta se o código da Central mudar.
    inputs.files(fileTree(desktop) { include("go.mod", "go.sum", "internal/**/*.go", "cmd/**/*.go") })
        .withPathSensitivity(PathSensitivity.RELATIVE)
}
