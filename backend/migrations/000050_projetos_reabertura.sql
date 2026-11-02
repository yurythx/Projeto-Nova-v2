-- +goose Up
-- Cada reabertura aprovada de um projeto concluído fica registrada com o motivo.
CREATE TABLE IF NOT EXISTS projetos_reaberturas (
    id             UUID PRIMARY KEY,
    projeto_id     UUID NOT NULL REFERENCES projetos(id) ON DELETE CASCADE,
    motivo         TEXT NOT NULL,
    aprovador_id   UUID NOT NULL,
    aprovador_nome TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projetos_reaberturas_projeto
    ON projetos_reaberturas (projeto_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS projetos_reaberturas;
