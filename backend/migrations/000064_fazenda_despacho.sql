-- +goose Up
-- O despacho do Avança abre os marcos de 90 dias e de 24 meses.
-- alerta_limite guarda o marco que já saiu no aviso, para não repetir.
ALTER TABLE fazenda_processos
    ADD COLUMN despacho_em TIMESTAMPTZ,
    ADD COLUMN alerta_limite TIMESTAMPTZ;

-- +goose Down
ALTER TABLE fazenda_processos
    DROP COLUMN IF EXISTS alerta_limite,
    DROP COLUMN IF EXISTS despacho_em;
