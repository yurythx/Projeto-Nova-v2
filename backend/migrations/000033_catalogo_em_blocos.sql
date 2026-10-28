-- +goose Up
-- Catálogo em blocos na central do solicitante (ADR 078): as ofertas se
-- agrupam em categorias da instituição, cada uma com ícone, descrição e
-- ordem. Apagar a categoria não apaga a oferta: ela vai para "Outros".
CREATE TABLE atendimento_categorias (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome TEXT NOT NULL CHECK (char_length(btrim(nome)) BETWEEN 1 AND 80),
    descricao TEXT NOT NULL DEFAULT '' CHECK (char_length(descricao) <= 300),
    icone TEXT NOT NULL DEFAULT '' CHECK (icone ~ '^[a-z0-9-]{0,40}$'),
    ordem INTEGER NOT NULL DEFAULT 0 CHECK (ordem BETWEEN 0 AND 999),
    ativa BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX atendimento_categorias_nome ON atendimento_categorias (lower(btrim(nome)));

ALTER TABLE atendimento_ofertas
    ADD COLUMN categoria_id UUID REFERENCES atendimento_categorias (id) ON DELETE SET NULL,
    ADD COLUMN icone TEXT NOT NULL DEFAULT '' CHECK (icone ~ '^[a-z0-9-]{0,40}$'),
    ADD COLUMN ordem INTEGER NOT NULL DEFAULT 0 CHECK (ordem BETWEEN 0 AND 999);
CREATE INDEX atendimento_ofertas_categoria ON atendimento_ofertas (categoria_id);

-- +goose Down
ALTER TABLE atendimento_ofertas DROP COLUMN IF EXISTS ordem, DROP COLUMN IF EXISTS icone, DROP COLUMN IF EXISTS categoria_id;
DROP TABLE IF EXISTS atendimento_categorias;
