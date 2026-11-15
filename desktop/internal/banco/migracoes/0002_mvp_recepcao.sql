-- 0002: esquema de negócio e de recepção do MVP (A24).
-- Modelo: docs/architecture/modelo-de-dados.md. Só cria tabelas novas (não destrutiva).
-- Cada entidade recebida guarda rev_origem (última rev do Android aplicada) e
-- registro_json (último payload aplicado, usado como "antes" da auditoria).
-- rev_central/corrigido_desde_rev_origem ficam prontos para as correções da Central
-- (Fase 2/3); no MVP a Central não edita cadastros, então nunca há conflito.

CREATE TABLE operacoes (
    uuid       TEXT PRIMARY KEY,
    nome       TEXT NOT NULL CHECK (length(trim(nome)) > 0),
    descricao  TEXT,
    status     TEXT NOT NULL CHECK (status IN ('ativa', 'encerrada')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
) STRICT;
CREATE UNIQUE INDEX operacoes_uma_ativa ON operacoes (status) WHERE status = 'ativa';

CREATE TABLE equipes (
    uuid               TEXT PRIMARY KEY,
    operacao_uuid      TEXT NOT NULL REFERENCES operacoes (uuid),
    nome               TEXT NOT NULL CHECK (length(trim(nome)) > 0),
    responsavel        TEXT NOT NULL CHECK (length(trim(responsavel)) > 0),
    identificacao_area TEXT,
    created_at         INTEGER NOT NULL,
    updated_at         INTEGER NOT NULL,
    rev_origem         INTEGER NOT NULL CHECK (rev_origem >= 1),
    recebido_em        INTEGER NOT NULL,
    registro_json      TEXT NOT NULL
) STRICT;

CREATE TABLE tokens_pareamento (
    token_hash    TEXT PRIMARY KEY,
    operacao_uuid TEXT NOT NULL REFERENCES operacoes (uuid),
    criado_em     INTEGER NOT NULL,
    expira_em     INTEGER NOT NULL,
    usado_em      INTEGER
) STRICT;

CREATE TABLE dispositivos (
    instalacao_uuid TEXT PRIMARY KEY,
    equipe_uuid     TEXT NOT NULL REFERENCES equipes (uuid),
    credencial_hash TEXT NOT NULL UNIQUE,
    rotulo_aparelho TEXT,
    pareado_em      INTEGER NOT NULL,
    revogado_em     INTEGER
) STRICT;

CREATE TABLE propriedades (
    uuid                        TEXT PRIMARY KEY,
    equipe_uuid                 TEXT NOT NULL REFERENCES equipes (uuid),
    nome_propriedade            TEXT NOT NULL CHECK (length(trim(nome_propriedade)) > 0),
    responsavel_familia         TEXT,
    pessoas_adultos             INTEGER NOT NULL DEFAULT 0 CHECK (pessoas_adultos >= 0),
    pessoas_jovens              INTEGER NOT NULL DEFAULT 0 CHECK (pessoas_jovens >= 0),
    pessoas_criancas            INTEGER NOT NULL DEFAULT 0 CHECK (pessoas_criancas >= 0),
    veic_caminhoes              INTEGER NOT NULL DEFAULT 0 CHECK (veic_caminhoes >= 0),
    veic_automoveis_utilitarios INTEGER NOT NULL DEFAULT 0 CHECK (veic_automoveis_utilitarios >= 0),
    veic_motos_ciclomotores     INTEGER NOT NULL DEFAULT 0 CHECK (veic_motos_ciclomotores >= 0),
    bov_adultos                 INTEGER NOT NULL DEFAULT 0 CHECK (bov_adultos >= 0),
    bov_bezerros                INTEGER NOT NULL DEFAULT 0 CHECK (bov_bezerros >= 0),
    equ_adultos                 INTEGER NOT NULL DEFAULT 0 CHECK (equ_adultos >= 0),
    equ_potros                  INTEGER NOT NULL DEFAULT 0 CHECK (equ_potros >= 0),
    area_valor_milesimos        INTEGER CHECK (area_valor_milesimos >= 0),
    area_unidade                TEXT CHECK (area_unidade IN ('ha', 'alq_paulista')),
    lat                         REAL CHECK (lat BETWEEN -90 AND 90),
    lon                         REAL CHECK (lon BETWEEN -180 AND 180),
    gps_origem                  TEXT CHECK (gps_origem IN ('gnss', 'manual')),
    gps_precisao_m              REAL CHECK (gps_precisao_m >= 0),
    gps_capturado_em            INTEGER,
    gps_satelites               INTEGER CHECK (gps_satelites >= 0),
    created_at                  INTEGER NOT NULL,
    updated_at                  INTEGER NOT NULL,
    excluido_em                 INTEGER,
    rev_origem                  INTEGER NOT NULL CHECK (rev_origem >= 1),
    rev_central                 INTEGER NOT NULL DEFAULT 0,
    corrigido_desde_rev_origem  INTEGER NOT NULL DEFAULT 0 CHECK (corrigido_desde_rev_origem IN (0, 1)),
    recebido_em                 INTEGER NOT NULL,
    origem_instalacao           TEXT NOT NULL,
    registro_json               TEXT NOT NULL,
    CHECK ((area_valor_milesimos IS NULL) = (area_unidade IS NULL)),
    CHECK ((lat IS NULL) = (lon IS NULL)),
    CHECK ((lat IS NULL) = (gps_origem IS NULL)),
    CHECK (gps_origem IS NOT 'manual' OR gps_precisao_m IS NULL)
) STRICT;
CREATE INDEX propriedades_equipe_vigentes ON propriedades (equipe_uuid) WHERE excluido_em IS NULL;

CREATE TABLE catalogo_itens (
    codigo          TEXT PRIMARY KEY,
    categoria       TEXT NOT NULL CHECK (categoria IN ('eletrodomestico', 'movel', 'ave')),
    nome_exibicao   TEXT NOT NULL,
    ordem           INTEGER NOT NULL,
    ativo           INTEGER NOT NULL DEFAULT 1 CHECK (ativo IN (0, 1)),
    versao_catalogo INTEGER NOT NULL
) STRICT;
-- Catálogo inicial (RN-30); igual a shared/test-fixtures/catalogo/itens.v1.json (teste).
INSERT INTO catalogo_itens (codigo, categoria, nome_exibicao, ordem, versao_catalogo) VALUES
    ('eletro.freezer', 'eletrodomestico', 'Freezer', 1, 1),
    ('eletro.geladeira', 'eletrodomestico', 'Geladeira', 2, 1),
    ('eletro.fogao', 'eletrodomestico', 'Fogão', 3, 1),
    ('eletro.maquina_lavar', 'eletrodomestico', 'Máquina de lavar', 4, 1),
    ('eletro.ar_condicionado', 'eletrodomestico', 'Ar-condicionado', 5, 1),
    ('eletro.ventilador', 'eletrodomestico', 'Ventilador', 6, 1),
    ('eletro.outros', 'eletrodomestico', 'Outros', 7, 1),
    ('movel.sofa', 'movel', 'Sofá', 1, 1),
    ('movel.mesa_jantar', 'movel', 'Mesa de jantar', 2, 1),
    ('movel.armario', 'movel', 'Armário', 3, 1),
    ('movel.estante', 'movel', 'Estante', 4, 1),
    ('movel.guarda_roupa', 'movel', 'Guarda-roupa', 5, 1),
    ('movel.outros', 'movel', 'Outros', 6, 1),
    ('ave.galinha', 'ave', 'Galinha', 1, 1),
    ('ave.ganso', 'ave', 'Ganso', 2, 1),
    ('ave.pato', 'ave', 'Pato', 3, 1),
    ('ave.outros', 'ave', 'Outros', 4, 1);

CREATE TABLE itens_propriedade (
    uuid                       TEXT PRIMARY KEY,
    propriedade_uuid           TEXT NOT NULL REFERENCES propriedades (uuid),
    categoria                  TEXT NOT NULL CHECK (categoria IN ('eletrodomestico', 'movel', 'ave')),
    codigo_catalogo            TEXT NOT NULL REFERENCES catalogo_itens (codigo),
    nome_personalizado         TEXT,
    nome_normalizado           TEXT,
    quantidade                 INTEGER NOT NULL CHECK (quantidade >= 0),
    created_at                 INTEGER NOT NULL,
    updated_at                 INTEGER NOT NULL,
    excluido_em                INTEGER,
    rev_origem                 INTEGER NOT NULL CHECK (rev_origem >= 1),
    rev_central                INTEGER NOT NULL DEFAULT 0,
    corrigido_desde_rev_origem INTEGER NOT NULL DEFAULT 0 CHECK (corrigido_desde_rev_origem IN (0, 1)),
    recebido_em                INTEGER NOT NULL,
    origem_instalacao          TEXT NOT NULL,
    registro_json              TEXT NOT NULL,
    CHECK ((codigo_catalogo LIKE '%.outros') = (nome_personalizado IS NOT NULL))
) STRICT;
-- RN-32: item predefinido no máximo uma vez por propriedade (entre vigentes).
CREATE UNIQUE INDEX itens_unico_predefinido ON itens_propriedade (propriedade_uuid, codigo_catalogo)
    WHERE excluido_em IS NULL AND codigo_catalogo NOT LIKE '%.outros';
CREATE INDEX itens_vigentes ON itens_propriedade (propriedade_uuid, categoria, codigo_catalogo) WHERE excluido_em IS NULL;
CREATE INDEX itens_nome_normalizado ON itens_propriedade (nome_normalizado) WHERE excluido_em IS NULL;

CREATE TABLE fotografias (
    uuid                       TEXT PRIMARY KEY,
    propriedade_uuid           TEXT NOT NULL REFERENCES propriedades (uuid),
    caminho_relativo           TEXT,
    mime                       TEXT NOT NULL CHECK (mime = 'image/jpeg'),
    tamanho_bytes              INTEGER NOT NULL CHECK (tamanho_bytes > 0),
    sha256                     TEXT NOT NULL CHECK (length(sha256) = 64),
    largura                    INTEGER NOT NULL,
    altura                     INTEGER NOT NULL,
    capturada_em               INTEGER NOT NULL,
    created_at                 INTEGER NOT NULL,
    updated_at                 INTEGER NOT NULL,
    excluido_em                INTEGER,
    rev_origem                 INTEGER NOT NULL CHECK (rev_origem >= 1),
    rev_central                INTEGER NOT NULL DEFAULT 0,
    corrigido_desde_rev_origem INTEGER NOT NULL DEFAULT 0 CHECK (corrigido_desde_rev_origem IN (0, 1)),
    recebido_em                INTEGER NOT NULL,
    origem_instalacao          TEXT NOT NULL,
    registro_json              TEXT NOT NULL,
    -- Foto vigente sempre aponta para arquivo publicado (invariante 1 de integridade.md).
    CHECK (excluido_em IS NOT NULL OR caminho_relativo IS NOT NULL)
) STRICT;
CREATE INDEX fotografias_vigentes ON fotografias (propriedade_uuid) WHERE excluido_em IS NULL;

CREATE TABLE lotes_recebidos (
    uuid            TEXT PRIMARY KEY,
    instalacao_uuid TEXT NOT NULL REFERENCES dispositivos (instalacao_uuid),
    equipe_uuid     TEXT NOT NULL REFERENCES equipes (uuid),
    canal           TEXT NOT NULL CHECK (canal IN ('wifi', 'usb')),
    status          TEXT NOT NULL CHECK (status IN ('recebendo', 'importado')),
    manifesto_json  TEXT NOT NULL,
    manifesto_sha256 TEXT NOT NULL,
    recibo_json     TEXT,
    recebido_em     INTEGER NOT NULL,
    importado_em    INTEGER,
    ack_em          INTEGER,
    CHECK ((status = 'importado') = (recibo_json IS NOT NULL))
) STRICT;

-- Fotos que um lote ainda em recepção espera receber (para PUT /v1/fotos).
CREATE TABLE fotos_esperadas (
    foto_uuid     TEXT NOT NULL,
    lote_uuid     TEXT NOT NULL REFERENCES lotes_recebidos (uuid),
    tamanho_bytes INTEGER NOT NULL,
    sha256        TEXT NOT NULL,
    PRIMARY KEY (foto_uuid, lote_uuid)
) STRICT;

CREATE TABLE aplicacoes (
    lote_uuid     TEXT NOT NULL REFERENCES lotes_recebidos (uuid),
    entidade      TEXT NOT NULL CHECK (entidade IN ('equipe', 'propriedade', 'item', 'foto')),
    entidade_uuid TEXT NOT NULL,
    rev           INTEGER NOT NULL,
    resultado     TEXT NOT NULL CHECK (resultado IN ('aplicado', 'duplicado', 'conflito', 'rejeitado')),
    motivo        TEXT,
    PRIMARY KEY (lote_uuid, entidade, entidade_uuid)
) STRICT;

-- Somente acréscimo (append-only).
CREATE TABLE auditoria (
    id            INTEGER PRIMARY KEY,
    quando        INTEGER NOT NULL,
    operador      TEXT,
    origem        TEXT NOT NULL,
    entidade      TEXT NOT NULL,
    entidade_uuid TEXT NOT NULL,
    acao          TEXT NOT NULL,
    antes_json    TEXT,
    depois_json   TEXT
) STRICT;
CREATE TRIGGER auditoria_sem_update BEFORE UPDATE ON auditoria
BEGIN SELECT RAISE(ABORT, 'auditoria é somente acréscimo'); END;
CREATE TRIGGER auditoria_sem_delete BEFORE DELETE ON auditoria
BEGIN SELECT RAISE(ABORT, 'auditoria é somente acréscimo'); END;

-- Visões vigentes: única base de estatísticas e consultas (RN-80).
CREATE VIEW v_propriedades_vigentes AS
    SELECT * FROM propriedades WHERE excluido_em IS NULL;
CREATE VIEW v_itens_vigentes AS
    SELECT i.* FROM itens_propriedade i
    JOIN propriedades p ON p.uuid = i.propriedade_uuid
    WHERE i.excluido_em IS NULL AND p.excluido_em IS NULL;
CREATE VIEW v_fotos_vigentes AS
    SELECT f.* FROM fotografias f
    JOIN propriedades p ON p.uuid = f.propriedade_uuid
    WHERE f.excluido_em IS NULL AND p.excluido_em IS NULL;
