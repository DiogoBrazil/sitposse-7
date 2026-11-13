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
    implementation(libs.kotlinx.serialization.json)

    testImplementation(kotlin("test"))
    testImplementation(platform(libs.junit.bom))
    testRuntimeOnly(libs.junit.platform.launcher)
}

tasks.test {
    useJUnitPlatform()
    // Fixtures compartilhadas com a Central (Go): a mesma entrada, o mesmo resultado.
    systemProperty("sitposse7.fixtures", rootProject.file("../shared/test-fixtures").absolutePath)
    inputs.dir(rootProject.file("../shared/test-fixtures")).withPathSensitivity(PathSensitivity.RELATIVE)
}
