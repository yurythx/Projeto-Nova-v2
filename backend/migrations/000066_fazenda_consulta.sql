-- +goose Up
-- A consulta do titular na área restrita é a data da ciência, se ocorrer até o 10º dia.
ALTER TABLE fazenda_processos
    ADD COLUMN consulta_em TIMESTAMPTZ;

-- +goose Down
ALTER TABLE fazenda_processos
    DROP COLUMN IF EXISTS consulta_em;
