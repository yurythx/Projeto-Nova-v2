-- +goose Up
-- Cofre de senhas. A coluna cifrada não é a senha: é o envelope.
-- A chave mestra não fica nesta tabela.

CREATE TABLE cofre_segredos (
    id               UUID PRIMARY KEY,
    nome             TEXT NOT NULL,
    usuario          TEXT NOT NULL DEFAULT '',
    unidade_id       UUID REFERENCES unidades (id) ON DELETE RESTRICT,
    versao           INTEGER NOT NULL DEFAULT 1 CHECK (versao >= 1),
    dek_embrulhada   TEXT NOT NULL,
    segredo_cifrado  TEXT NOT NULL,
    created_by       UUID,
    created_at       TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL
);

CREATE INDEX cofre_segredos_unidade_id_idx ON cofre_segredos (unidade_id);

-- +goose Down
DROP TABLE cofre_segredos;
