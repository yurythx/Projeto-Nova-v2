-- +goose Up
-- A tarefa pode esperar outra do mesmo projeto. Apagar a anterior solta a seguinte.
ALTER TABLE projetos_tarefas
    ADD COLUMN IF NOT EXISTS predecessora_id UUID REFERENCES projetos_tarefas(id) ON DELETE SET NULL;

ALTER TABLE projetos_tarefas
    DROP CONSTRAINT IF EXISTS projetos_tarefas_predecessora_diferente;

ALTER TABLE projetos_tarefas
    ADD CONSTRAINT projetos_tarefas_predecessora_diferente
    CHECK (predecessora_id IS NULL OR predecessora_id <> id);

CREATE INDEX IF NOT EXISTS idx_projetos_tarefas_predecessora
    ON projetos_tarefas (predecessora_id);

-- +goose Down
DROP INDEX IF EXISTS idx_projetos_tarefas_predecessora;
ALTER TABLE projetos_tarefas DROP CONSTRAINT IF EXISTS projetos_tarefas_predecessora_diferente;
ALTER TABLE projetos_tarefas DROP COLUMN IF EXISTS predecessora_id;
