-- +goose Up
-- Plug-in Projetos (internal/modules/projetos)
CREATE TABLE IF NOT EXISTS projetos (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero                BIGSERIAL NOT NULL,
    titulo                TEXT NOT NULL,
    descricao             TEXT NOT NULL DEFAULT '',
    objetivo              TEXT NOT NULL DEFAULT '',
    estado                TEXT NOT NULL DEFAULT 'planejado',
    prioridade            TEXT NOT NULL DEFAULT 'media',
    data_inicio_planejada TIMESTAMPTZ,
    data_fim_planejada    TIMESTAMPTZ,
    data_inicio_real      TIMESTAMPTZ,
    data_fim_real         TIMESTAMPTZ,
    orcamento             NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    unidade_id            UUID,
    produto_id            UUID,
    gerente_id            UUID,
    gerente_nome          TEXT NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_projetos_numero ON projetos (numero);
CREATE INDEX IF NOT EXISTS idx_projetos_estado ON projetos (estado);
CREATE INDEX IF NOT EXISTS idx_projetos_unidade ON projetos (unidade_id);
CREATE INDEX IF NOT EXISTS idx_projetos_created_at ON projetos (created_at DESC);

CREATE TABLE IF NOT EXISTS projetos_tarefas (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    projeto_id       UUID NOT NULL REFERENCES projetos(id) ON DELETE CASCADE,
    titulo           TEXT NOT NULL,
    descricao        TEXT NOT NULL DEFAULT '',
    responsavel_nome TEXT NOT NULL DEFAULT '',
    estado           TEXT NOT NULL DEFAULT 'a_fazer',
    ordem            INT NOT NULL DEFAULT 0,
    data_limite      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projetos_tarefas_projeto ON projetos_tarefas (projeto_id, ordem);

CREATE TABLE IF NOT EXISTS projetos_vinculos (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    projeto_id    UUID NOT NULL REFERENCES projetos(id) ON DELETE CASCADE,
    tipo          TEXT NOT NULL,
    alvo_id       UUID NOT NULL,
    rotulo        TEXT NOT NULL DEFAULT '',
    vinculado_por TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (projeto_id, tipo, alvo_id)
);

CREATE INDEX IF NOT EXISTS idx_projetos_vinculos_proj ON projetos_vinculos (projeto_id);
CREATE INDEX IF NOT EXISTS idx_projetos_vinculos_alvo ON projetos_vinculos (tipo, alvo_id);

-- +goose Down
DROP TABLE IF EXISTS projetos_vinculos;
DROP TABLE IF EXISTS projetos_tarefas;
DROP TABLE IF EXISTS projetos;
