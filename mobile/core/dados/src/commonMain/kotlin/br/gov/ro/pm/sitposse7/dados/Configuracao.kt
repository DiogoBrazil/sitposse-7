package br.gov.ro.pm.sitposse7.dados

import androidx.room3.RoomDatabase
import androidx.sqlite.driver.bundled.BundledSQLiteDriver
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.IO

/**
 * Configuração única do banco, usada pelo app e pelos testes (ADR-0002):
 * SQLite empacotado com synchronous=FULL e foreign_keys=ON em toda conexão ([DriverDuravel]), WAL explícito (o modo automático do Room usa TRUNCATE em aparelhos
 * com pouca RAM) e todas as operações fora da main thread.
 * Proibido: fallbackToDestructiveMigration*.
 */
fun RoomDatabase.Builder<BancoCampo>.configurarPadrao(): RoomDatabase.Builder<BancoCampo> =
    setDriver(DriverDuravel(BundledSQLiteDriver()))
        .setJournalMode(RoomDatabase.JournalMode.WRITE_AHEAD_LOGGING)
        .setQueryCoroutineContext(Dispatchers.IO)
        .addMigrations(*MIGRACOES)
