-- +goose Up
-- O nome do programa é o que o agente informa. Vazio não compara.

ALTER TABLE contratos_licencas
    ADD COLUMN programa TEXT NOT NULL DEFAULT '';

ALTER TABLE contratos_licencas
    ADD CONSTRAINT contratos_licencas_programa_chk CHECK (char_length(programa) <= 80);

-- +goose Down
ALTER TABLE contratos_licencas DROP CONSTRAINT contratos_licencas_programa_chk;
ALTER TABLE contratos_licencas DROP COLUMN programa;
