-- +goose Up
-- Credencial da máquina. O segredo não é gravado, só o hash. Revogar
-- não apaga a linha: o uso antigo continua identificável na auditoria.

CREATE TABLE ativos_credenciais (
    id          UUID PRIMARY KEY,
    nome        TEXT NOT NULL,
    unidade_id  UUID NOT NULL REFERENCES unidades (id),
    token_hash  BYTEA NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL,
    revogada_em TIMESTAMPTZ
);

-- +goose Down
DROP TABLE ativos_credenciais;
