-- +goose Up
-- Subtarefa de um nível: a filha aponta para a tarefa e some com ela.
ALTER TABLE projetos_tarefas
    ADD COLUMN IF NOT EXISTS pai_id UUID REFERENCES projetos_tarefas(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_projetos_tarefas_pai ON projetos_tarefas (pai_id);

-- +goose Down
DROP INDEX IF EXISTS idx_projetos_tarefas_pai;
ALTER TABLE projetos_tarefas DROP COLUMN IF EXISTS pai_id;
