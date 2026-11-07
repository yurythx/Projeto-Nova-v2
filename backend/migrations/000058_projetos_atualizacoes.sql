-- +goose Up
-- Fio de atualizações da tarefa, separado do relato de como ela foi feita.
-- O índice do prazo sustenta o calendário do mês.
CREATE TABLE IF NOT EXISTS projetos_tarefa_atualizacoes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tarefa_id  UUID NOT NULL REFERENCES projetos_tarefas(id) ON DELETE CASCADE,
    texto      TEXT NOT NULL,
    autor_id   UUID,
    autor_nome TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projetos_tarefa_atualizacoes_tarefa
    ON projetos_tarefa_atualizacoes (tarefa_id, created_at);

CREATE INDEX IF NOT EXISTS idx_projetos_tarefas_prazo
    ON projetos_tarefas (data_limite)
    WHERE data_limite IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_projetos_tarefas_prazo;
DROP TABLE IF EXISTS projetos_tarefa_atualizacoes;
