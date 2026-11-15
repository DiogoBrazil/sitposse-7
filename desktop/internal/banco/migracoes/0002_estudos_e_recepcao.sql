-- 0002: esquema de negócio e de recepção, GERAÇÃO 2 (A28, ADR-0023/0024/0025).
-- Substitui o 0002 do MVP (operação única, equipe vinda do celular). Bancos da
-- geração anterior são recusados por banco.Migrar (ver ADR-0025); nada é apagado.
-- Modelo: docs/architecture/modelo-de-dados.md.
-- Cada entidade recebida guarda rev_origem (última rev do Android aplicada) e
-- registro_json (último payload aplicado, usado como "antes" da auditoria).
-- rev_central/corrigido_desde_rev_origem ficam prontos para as correções da Central.

INSERT INTO meta (chave, valor, created_at, updated_at)
    VALUES ('esquema_geracao', '2', CAST(unixepoch('subsec') * 1000 AS INTEGER), CAST(unixepoch('subsec') * 1000 AS INTEGER));

-- Estudo de situação → jornadas → equipes (RN-01, RN-02).
CREATE TABLE estudos (
    uuid        TEXT PRIMARY KEY,
    codigo      TEXT NOT NULL UNIQUE CHECK (length(codigo) = 17 AND codigo LIKE 'SITPOSSE7%'),
    nome        TEXT NOT NULL CHECK (length(trim(nome)) > 0),
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    excluido_em INTEGER
) STRICT;

CREATE TABLE jornadas (
    uuid        TEXT PRIMARY KEY,
    estudo_uuid TEXT NOT NULL REFERENCES estudos (uuid),
    numero      INTEGER NOT NULL CHECK (numero >= 1),
    data        TEXT NOT NULL CHECK (data GLOB '[0-9][0-9][0-9][0-9]-[0-1][0-9]-[0-3][0-9]'),
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    excluido_em INTEGER,
    UNIQUE (estudo_uuid, numero)
) STRICT;
CREATE INDEX jornadas_estudo_data ON jornadas (estudo_uuid, data);

CREATE TABLE equipes (
    uuid             TEXT PRIMARY KEY,
    jornada_uuid     TEXT NOT NULL REFERENCES jornadas (uuid),
    nome             TEXT NOT NULL CHECK (length(trim(nome)) > 0),
    nome_normalizado TEXT NOT NULL,
    codigo           TEXT NOT NULL UNIQUE CHECK (length(codigo) = 10 AND codigo LIKE 'EQ%'),
    created_at       INTEGER NOT NULL,
    updated_at       INTEGER NOT NULL,
    excluido_em      INTEGER
) STRICT;
CREATE INDEX equipes_jornada ON equipes (jornada_uuid);
CREATE UNIQUE INDEX equipes_nome_na_jornada ON equipes (jornada_uuid, nome_normalizado) WHERE excluido_em IS NULL;

-- Pareamento só do aparelho (ADR-0024).
CREATE TABLE tokens_pareamento (
    token_hash TEXT PRIMARY KEY,
    criado_em  INTEGER NOT NULL,
    expira_em  INTEGER NOT NULL,
    usado_em   INTEGER
) STRICT;

CREATE TABLE dispositivos (
    instalacao_uuid TEXT PRIMARY KEY,
    credencial_hash TEXT NOT NULL UNIQUE,
    rotulo_aparelho TEXT,
    pareado_em      INTEGER NOT NULL,
    revogado_em     INTEGER
) STRICT;

-- Coleta do celular (RN-05). equipe_uuid é preenchido na 1ª confirmação do destino e
-- nunca mais muda (destino travado, RN-09).
CREATE TABLE coletas (
    uuid                  TEXT PRIMARY KEY,
    instalacao_uuid       TEXT NOT NULL REFERENCES dispositivos (instalacao_uuid),
    codigo_informado      TEXT NOT NULL,
    codigo_original       TEXT NOT NULL,
    equipe_uuid           TEXT REFERENCES equipes (uuid),
    destino_confirmado_em INTEGER,
    created_at            INTEGER NOT NULL,
    updated_at            INTEGER NOT NULL,
    rev_origem            INTEGER NOT NULL DEFAULT 0 CHECK (rev_origem >= 0),
    recebido_em           INTEGER NOT NULL,
    registro_json         TEXT,
    CHECK ((equipe_uuid IS NULL) = (destino_confirmado_em IS NULL))
) STRICT;
CREATE INDEX coletas_equipe ON coletas (equipe_uuid);
CREATE TRIGGER coletas_destino_travado BEFORE UPDATE OF equipe_uuid ON coletas
    WHEN OLD.equipe_uuid IS NOT NULL AND NEW.equipe_uuid IS NOT OLD.equipe_uuid
BEGIN SELECT RAISE(ABORT, 'destino da coleta já confirmado'); END;

CREATE TABLE propriedades (
    uuid                        TEXT PRIMARY KEY,
    coleta_uuid                 TEXT NOT NULL REFERENCES coletas (uuid),
    equipe_uuid                 TEXT NOT NULL REFERENCES equipes (uuid),
    nome_propriedade            TEXT NOT NULL CHECK (length(trim(nome_propriedade)) > 0),
    responsavel_familia         TEXT NOT NULL CHECK (length(trim(responsavel_familia)) > 0),
    observacoes_relevantes      TEXT CHECK (length(observacoes_relevantes) <= 4000),
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
CREATE INDEX propriedades_coleta ON propriedades (coleta_uuid);

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
    uuid                TEXT PRIMARY KEY,
    instalacao_uuid     TEXT NOT NULL REFERENCES dispositivos (instalacao_uuid),
    coleta_uuid         TEXT NOT NULL REFERENCES coletas (uuid),
    codigo_informado    TEXT NOT NULL,
    canal               TEXT NOT NULL CHECK (canal IN ('wifi', 'usb')),
    status              TEXT NOT NULL CHECK (status IN ('recebendo', 'aguardando_confirmacao', 'importado', 'cancelado')),
    equipe_destino_uuid TEXT REFERENCES equipes (uuid),
    decidido_em         INTEGER,
    motivo_cancelamento TEXT CHECK (motivo_cancelamento IN ('cancelado_operador', 'codigo_inexistente')),
    manifesto_json      TEXT NOT NULL,
    manifesto_sha256    TEXT NOT NULL,
    recibo_json         TEXT,
    recebido_em         INTEGER NOT NULL,
    aguardando_desde    INTEGER,
    importado_em        INTEGER,
    ack_em              INTEGER,
    CHECK ((status = 'importado') = (recibo_json IS NOT NULL)),
    CHECK ((status = 'importado') = (equipe_destino_uuid IS NOT NULL)),
    CHECK ((status = 'cancelado') = (motivo_cancelamento IS NOT NULL)),
    CHECK ((status IN ('importado', 'cancelado')) = (decidido_em IS NOT NULL))
) STRICT;
CREATE INDEX lotes_status ON lotes_recebidos (status);
CREATE INDEX lotes_coleta ON lotes_recebidos (coleta_uuid);

-- Decisões do operador sobre o destino de cada lote (somente acréscimo, ADR-0024).
CREATE TABLE decisoes_destino (
    id                INTEGER PRIMARY KEY,
    lote_uuid         TEXT NOT NULL REFERENCES lotes_recebidos (uuid),
    coleta_uuid       TEXT NOT NULL REFERENCES coletas (uuid),
    acao              TEXT NOT NULL CHECK (acao IN ('confirmar', 'alterar', 'cancelar')),
    codigo_informado  TEXT NOT NULL,
    codigo_confirmado TEXT,
    equipe_uuid       TEXT REFERENCES equipes (uuid),
    jornada_uuid      TEXT REFERENCES jornadas (uuid),
    estudo_uuid       TEXT REFERENCES estudos (uuid),
    quando            INTEGER NOT NULL,
    operador          TEXT,
    CHECK ((acao = 'cancelar') = (equipe_uuid IS NULL))
) STRICT;
CREATE INDEX decisoes_lote ON decisoes_destino (lote_uuid);
CREATE TRIGGER decisoes_sem_update BEFORE UPDATE ON decisoes_destino
BEGIN SELECT RAISE(ABORT, 'decisões de destino são somente acréscimo'); END;
CREATE TRIGGER decisoes_sem_delete BEFORE DELETE ON decisoes_destino
BEGIN SELECT RAISE(ABORT, 'decisões de destino são somente acréscimo'); END;

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
    entidade      TEXT NOT NULL CHECK (entidade IN ('coleta', 'propriedade', 'item', 'foto')),
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
-- v_propriedades_vigentes traz também equipe, jornada e estudo de destino.
CREATE VIEW v_propriedades_vigentes AS
    SELECT p.*, e.nome AS equipe_nome, e.codigo AS equipe_codigo,
           j.uuid AS jornada_uuid, j.numero AS jornada_numero, j.data AS jornada_data,
           s.uuid AS estudo_uuid, s.nome AS estudo_nome, s.codigo AS estudo_codigo
    FROM propriedades p
    JOIN equipes e ON e.uuid = p.equipe_uuid
    JOIN jornadas j ON j.uuid = e.jornada_uuid
    JOIN estudos s ON s.uuid = j.estudo_uuid
    WHERE p.excluido_em IS NULL;
CREATE VIEW v_itens_vigentes AS
    SELECT i.* FROM itens_propriedade i
    JOIN propriedades p ON p.uuid = i.propriedade_uuid
    WHERE i.excluido_em IS NULL AND p.excluido_em IS NULL;
CREATE VIEW v_fotos_vigentes AS
    SELECT f.* FROM fotografias f
    JOIN propriedades p ON p.uuid = f.propriedade_uuid
    WHERE f.excluido_em IS NULL AND p.excluido_em IS NULL;
