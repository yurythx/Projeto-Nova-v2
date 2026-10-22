-- +goose Up
-- Ingresso SCAN (papel digitalizado, Decreto 10.278) vs NATO_DIGITAL
-- (PDF já nascido eletrônico). A conferência de DPI e de imagens só vale
-- para SCAN. TIFF entra pelo mesmo pipeline (ADR 044).

ALTER TABLE acervo_digitais
    ADD COLUMN origem TEXT NOT NULL DEFAULT 'SCAN'
        CHECK (origem IN ('SCAN', 'NATO_DIGITAL'));

ALTER TABLE acervo_digitais DROP CONSTRAINT acervo_digitais_dpi_declarado_check;

ALTER TABLE acervo_digitais ADD CONSTRAINT acervo_digitais_dpi_origem CHECK (
    (origem = 'SCAN' AND dpi_declarado BETWEEN 72 AND 4800)
    OR (origem = 'NATO_DIGITAL' AND dpi_declarado = 0)
);

-- +goose Down
ALTER TABLE acervo_digitais DROP CONSTRAINT IF EXISTS acervo_digitais_dpi_origem;
ALTER TABLE acervo_digitais DROP COLUMN IF EXISTS origem;
ALTER TABLE acervo_digitais ADD CONSTRAINT acervo_digitais_dpi_declarado_check
    CHECK (dpi_declarado BETWEEN 72 AND 4800);
