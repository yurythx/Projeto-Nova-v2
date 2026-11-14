-- +goose Up
-- Atos do Avança e uma peça PDF por tipo do Anexo II.
ALTER TABLE fazenda_processos
    ADD COLUMN tqc_assinado_em TIMESTAMPTZ,
    ADD COLUMN selo_criterios INT NOT NULL DEFAULT 0,
    ADD COLUMN isencao_fim INT NOT NULL DEFAULT 0,
    ADD COLUMN cassacao_ciencia TIMESTAMPTZ;

CREATE TABLE fazenda_pecas (
    processo_id UUID NOT NULL REFERENCES fazenda_processos (id) ON DELETE CASCADE,
    tipo TEXT NOT NULL,
    object_key TEXT NOT NULL,
    PRIMARY KEY (processo_id, tipo)
);

-- +goose Down
DROP TABLE IF EXISTS fazenda_pecas;
ALTER TABLE fazenda_processos
    DROP COLUMN IF EXISTS cassacao_ciencia,
    DROP COLUMN IF EXISTS isencao_fim,
    DROP COLUMN IF EXISTS selo_criterios,
    DROP COLUMN IF EXISTS tqc_assinado_em;
