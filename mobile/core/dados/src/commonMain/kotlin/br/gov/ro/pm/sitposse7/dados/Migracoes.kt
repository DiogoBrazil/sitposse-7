package br.gov.ro.pm.sitposse7.dados

import androidx.room3.migration.Migration
import androidx.sqlite.SQLiteConnection
import androidx.sqlite.execSQL

/**
 * 1 → 2 (MVP, ADR-0018): cria as tabelas de negócio e de sincronização.
 * Não destrutiva: só CREATE TABLE/INDEX; a tabela `meta` e seus dados ficam intactos.
 * O SQL é o mesmo que o Room exporta em `schemas/.../2.json` (o teste de migração
 * confere que o resultado bate com o esquema validado pelo Room).
 */
val MIGRACAO_1_2 = object : Migration(1, 2) {
    override suspend fun migrate(connection: SQLiteConnection) {
        listOf(
            "CREATE TABLE IF NOT EXISTS `equipe` (`uuid` TEXT NOT NULL, `nome` TEXT NOT NULL, `responsavel` TEXT NOT NULL, `identificacao_area` TEXT, `created_at` INTEGER NOT NULL, `updated_at` INTEGER NOT NULL, `rev` INTEGER NOT NULL, `rev_confirmada` INTEGER NOT NULL DEFAULT 0, PRIMARY KEY(`uuid`))",
            "CREATE TABLE IF NOT EXISTS `propriedades` (`uuid` TEXT NOT NULL, `equipe_uuid` TEXT NOT NULL, `nome_propriedade` TEXT NOT NULL, `responsavel_familia` TEXT, `pessoas_adultos` INTEGER NOT NULL DEFAULT 0, `pessoas_jovens` INTEGER NOT NULL DEFAULT 0, `pessoas_criancas` INTEGER NOT NULL DEFAULT 0, `veic_caminhoes` INTEGER NOT NULL DEFAULT 0, `veic_automoveis_utilitarios` INTEGER NOT NULL DEFAULT 0, `veic_motos_ciclomotores` INTEGER NOT NULL DEFAULT 0, `bov_adultos` INTEGER NOT NULL DEFAULT 0, `bov_bezerros` INTEGER NOT NULL DEFAULT 0, `equ_adultos` INTEGER NOT NULL DEFAULT 0, `equ_potros` INTEGER NOT NULL DEFAULT 0, `area_valor_milesimos` INTEGER, `area_unidade` TEXT, `lat` REAL, `lon` REAL, `gps_origem` TEXT, `gps_precisao_m` REAL, `gps_capturado_em` INTEGER, `gps_satelites` INTEGER, `created_at` INTEGER NOT NULL, `updated_at` INTEGER NOT NULL, `rev` INTEGER NOT NULL, `rev_confirmada` INTEGER NOT NULL DEFAULT 0, `excluido_em` INTEGER, PRIMARY KEY(`uuid`), FOREIGN KEY(`equipe_uuid`) REFERENCES `equipe`(`uuid`) ON UPDATE NO ACTION ON DELETE NO ACTION )",
            "CREATE INDEX IF NOT EXISTS `index_propriedades_equipe_uuid` ON `propriedades` (`equipe_uuid`)",
            "CREATE TABLE IF NOT EXISTS `itens_propriedade` (`uuid` TEXT NOT NULL, `propriedade_uuid` TEXT NOT NULL, `categoria` TEXT NOT NULL, `codigo_catalogo` TEXT NOT NULL, `nome_personalizado` TEXT, `nome_normalizado` TEXT, `quantidade` INTEGER NOT NULL, `created_at` INTEGER NOT NULL, `updated_at` INTEGER NOT NULL, `rev` INTEGER NOT NULL, `rev_confirmada` INTEGER NOT NULL DEFAULT 0, `excluido_em` INTEGER, PRIMARY KEY(`uuid`), FOREIGN KEY(`propriedade_uuid`) REFERENCES `propriedades`(`uuid`) ON UPDATE NO ACTION ON DELETE NO ACTION )",
            "CREATE INDEX IF NOT EXISTS `index_itens_propriedade_propriedade_uuid` ON `itens_propriedade` (`propriedade_uuid`)",
            "CREATE TABLE IF NOT EXISTS `fotografias` (`uuid` TEXT NOT NULL, `propriedade_uuid` TEXT NOT NULL, `caminho_relativo` TEXT NOT NULL, `mime` TEXT NOT NULL, `tamanho_bytes` INTEGER NOT NULL, `sha256` TEXT NOT NULL, `largura` INTEGER NOT NULL, `altura` INTEGER NOT NULL, `capturada_em` INTEGER NOT NULL, `created_at` INTEGER NOT NULL, `updated_at` INTEGER NOT NULL, `rev` INTEGER NOT NULL, `rev_confirmada` INTEGER NOT NULL DEFAULT 0, `excluido_em` INTEGER, `arquivo_presente` INTEGER NOT NULL DEFAULT 1, PRIMARY KEY(`uuid`), FOREIGN KEY(`propriedade_uuid`) REFERENCES `propriedades`(`uuid`) ON UPDATE NO ACTION ON DELETE NO ACTION )",
            "CREATE INDEX IF NOT EXISTS `index_fotografias_propriedade_uuid` ON `fotografias` (`propriedade_uuid`)",
            "CREATE TABLE IF NOT EXISTS `rascunhos` (`uuid` TEXT NOT NULL, `propriedade_uuid` TEXT, `json` TEXT NOT NULL, `atualizado_em` INTEGER NOT NULL, PRIMARY KEY(`uuid`))",
            "CREATE TABLE IF NOT EXISTS `lotes_envio` (`uuid` TEXT NOT NULL, `status` TEXT NOT NULL, `canal` TEXT NOT NULL, `payload_json` TEXT NOT NULL, `criado_em` INTEGER NOT NULL, `atualizado_em` INTEGER NOT NULL, `recibo_json` TEXT, `erro` TEXT, PRIMARY KEY(`uuid`))",
            "CREATE TABLE IF NOT EXISTS `lote_itens` (`lote_uuid` TEXT NOT NULL, `entidade` TEXT NOT NULL, `entidade_uuid` TEXT NOT NULL, `rev` INTEGER NOT NULL, `resultado` TEXT, `motivo` TEXT, PRIMARY KEY(`lote_uuid`, `entidade`, `entidade_uuid`), FOREIGN KEY(`lote_uuid`) REFERENCES `lotes_envio`(`uuid`) ON UPDATE NO ACTION ON DELETE NO ACTION )",
            "CREATE TABLE IF NOT EXISTS `pareamento` (`central_id` TEXT NOT NULL, `spki_sha256` TEXT NOT NULL, `ips_json` TEXT NOT NULL, `porta` INTEGER NOT NULL, `credencial` TEXT NOT NULL, `operacao_uuid` TEXT NOT NULL, `operacao_nome` TEXT NOT NULL, `pareado_em` INTEGER NOT NULL, PRIMARY KEY(`central_id`))",
        ).forEach { connection.execSQL(it) }
    }
}

/** Todas as migrações, em ordem. Toda versão nova acrescenta a sua aqui (mobile/AGENTS.md). */
val MIGRACOES: Array<Migration> = arrayOf(MIGRACAO_1_2)
