-- +goose Up
-- Relato da execução e arquivos da tarefa (fotos e documentos).
ALTER TABLE projetos_tarefas
    ADD COLUMN IF NOT EXISTS relato TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS projetos_tarefa_anexos (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tarefa_id    UUID NOT NULL REFERENCES projetos_tarefas(id) ON DELETE CASCADE,
    nome         TEXT NOT NULL,
    object_key   TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    tamanho      BIGINT NOT NULL,
    autor_id     UUID,
    autor_nome   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projetos_tarefa_anexos_tarefa
    ON projetos_tarefa_anexos (tarefa_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS projetos_tarefa_anexos;
ALTER TABLE projetos_tarefas DROP COLUMN IF EXISTS relato;
