-- +goose Up
-- Histórico de coluna e de responsável da tarefa.
CREATE TABLE IF NOT EXISTS projetos_tarefa_movimentos (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tarefa_id                UUID NOT NULL REFERENCES projetos_tarefas(id) ON DELETE CASCADE,
    estado_antes             TEXT NOT NULL DEFAULT '',
    estado_depois            TEXT NOT NULL DEFAULT '',
    responsavel_antes_id     UUID,
    responsavel_antes_nome   TEXT NOT NULL DEFAULT '',
    responsavel_depois_id    UUID,
    responsavel_depois_nome  TEXT NOT NULL DEFAULT '',
    autor_id                 UUID,
    autor_nome               TEXT NOT NULL DEFAULT '',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projetos_tarefa_movimentos_tarefa
    ON projetos_tarefa_movimentos (tarefa_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS projetos_tarefa_movimentos;
