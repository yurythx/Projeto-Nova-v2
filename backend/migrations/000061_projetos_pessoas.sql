-- +goose Up
-- O projeto tem um gerente e vários participantes.
-- A tarefa e a subtarefa têm a própria lista, só com gente do projeto.
CREATE TABLE IF NOT EXISTS projetos_participantes (
    projeto_id UUID NOT NULL REFERENCES projetos(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL,
    nome       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (projeto_id, user_id)
);

CREATE TABLE IF NOT EXISTS projetos_tarefa_pessoas (
    tarefa_id  UUID NOT NULL REFERENCES projetos_tarefas(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL,
    nome       TEXT NOT NULL DEFAULT '',
    ordem      INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tarefa_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_projetos_tarefa_pessoas_user
    ON projetos_tarefa_pessoas (user_id);

INSERT INTO projetos_participantes (projeto_id, user_id, nome)
SELECT id, gerente_id, gerente_nome
FROM projetos
WHERE gerente_id IS NOT NULL
ON CONFLICT (projeto_id, user_id) DO NOTHING;

INSERT INTO projetos_participantes (projeto_id, user_id, nome)
SELECT DISTINCT ON (projeto_id, responsavel_id) projeto_id, responsavel_id, responsavel_nome
FROM projetos_tarefas
WHERE responsavel_id IS NOT NULL
ORDER BY projeto_id, responsavel_id, updated_at DESC
ON CONFLICT (projeto_id, user_id) DO NOTHING;

INSERT INTO projetos_tarefa_pessoas (tarefa_id, user_id, nome, ordem)
SELECT id, responsavel_id, responsavel_nome, 0
FROM projetos_tarefas
WHERE responsavel_id IS NOT NULL
ON CONFLICT (tarefa_id, user_id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS projetos_tarefa_pessoas;
DROP TABLE IF EXISTS projetos_participantes;
