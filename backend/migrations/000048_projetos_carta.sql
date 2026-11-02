-- +goose Up
-- Carta 5W2H, 1:1 com o projeto. As etapas do "Como" não ficam aqui:
-- cada uma vira uma tarefa com origem 'how'.

CREATE TABLE IF NOT EXISTS projetos_carta (
    projeto_id   UUID PRIMARY KEY REFERENCES projetos(id) ON DELETE CASCADE,
    onde         TEXT NOT NULL DEFAULT '',
    patrocinador TEXT NOT NULL DEFAULT '',
    metodologia  TEXT NOT NULL DEFAULT '',
    centro_custo TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE projetos_tarefas
    ADD COLUMN IF NOT EXISTS origem TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE projetos_tarefas DROP COLUMN IF EXISTS origem;
DROP TABLE IF EXISTS projetos_carta;
