-- +goose Up
-- Cancelamento autorizado: o administrador descreve o motivo e o registro fica.
CREATE TABLE IF NOT EXISTS projetos_cancelamentos (
    id             UUID PRIMARY KEY,
    projeto_id     UUID NOT NULL REFERENCES projetos(id) ON DELETE CASCADE,
    motivo         TEXT NOT NULL,
    aprovador_id   UUID NOT NULL,
    aprovador_nome TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projetos_cancelamentos_projeto
    ON projetos_cancelamentos (projeto_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS projetos_cancelamentos;
