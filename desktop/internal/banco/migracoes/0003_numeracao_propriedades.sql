-- Número fixo da propriedade no estudo (RN-96, ADR-0027).
-- Atribuído quando o cadastro chega à Central, em ordem de recebimento; nunca muda nem é
-- reutilizado (inclusive após exclusão). Migração não destrutiva: só acrescenta uma tabela,
-- preenche os números dos cadastros existentes e recria a visão (visões não guardam dados).

CREATE TABLE numeracao_propriedades (
    propriedade_uuid TEXT PRIMARY KEY REFERENCES propriedades (uuid),
    estudo_uuid      TEXT NOT NULL REFERENCES estudos (uuid),
    numero           INTEGER NOT NULL CHECK (numero >= 1),
    atribuido_em     INTEGER NOT NULL,
    UNIQUE (estudo_uuid, numero)
) STRICT;

CREATE TRIGGER numeracao_sem_update BEFORE UPDATE ON numeracao_propriedades
BEGIN SELECT RAISE(ABORT, 'o número da propriedade no estudo não muda'); END;
CREATE TRIGGER numeracao_sem_delete BEFORE DELETE ON numeracao_propriedades
BEGIN SELECT RAISE(ABORT, 'o número da propriedade no estudo não é apagado'); END;

-- Cadastros já recebidos (inclusive excluídos, para que o número nunca seja reaproveitado).
INSERT INTO numeracao_propriedades (propriedade_uuid, estudo_uuid, numero, atribuido_em)
SELECT p.uuid, j.estudo_uuid,
       ROW_NUMBER() OVER (PARTITION BY j.estudo_uuid ORDER BY p.recebido_em, p.uuid),
       p.recebido_em
FROM propriedades p
JOIN equipes e ON e.uuid = p.equipe_uuid
JOIN jornadas j ON j.uuid = e.jornada_uuid;

DROP VIEW v_propriedades_vigentes;
CREATE VIEW v_propriedades_vigentes AS
    SELECT p.*, e.nome AS equipe_nome, e.codigo AS equipe_codigo,
           j.uuid AS jornada_uuid, j.numero AS jornada_numero, j.data AS jornada_data,
           s.uuid AS estudo_uuid, s.nome AS estudo_nome, s.codigo AS estudo_codigo,
           n.numero AS numero_no_estudo
    FROM propriedades p
    JOIN equipes e ON e.uuid = p.equipe_uuid
    JOIN jornadas j ON j.uuid = e.jornada_uuid
    JOIN estudos s ON s.uuid = j.estudo_uuid
    LEFT JOIN numeracao_propriedades n ON n.propriedade_uuid = p.uuid
    WHERE p.excluido_em IS NULL;
