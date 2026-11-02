-- +goose Up
-- O responsável da tarefa passa a ser uma pessoa (users.id).
-- O nome continua gravado para a lista não depender do diretório.
ALTER TABLE projetos_tarefas
    ADD COLUMN IF NOT EXISTS responsavel_id UUID;

-- +goose Down
ALTER TABLE projetos_tarefas DROP COLUMN IF EXISTS responsavel_id;
