-- +goose Up
-- A pasta estreita o segredo dentro da unidade. Não é role do Keycloak.
-- Apagar a pasta devolve o segredo à unidade; a senha não some.

CREATE TABLE cofre_pastas (
    id uuid PRIMARY KEY,
    nome text NOT NULL,
    unidade_id uuid REFERENCES unidades (id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT cofre_pastas_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 160)
);

CREATE TABLE cofre_pasta_membros (
    pasta_id uuid NOT NULL REFERENCES cofre_pastas (id) ON DELETE CASCADE,
    usuario_id uuid NOT NULL REFERENCES users (id),
    pode_revelar boolean NOT NULL DEFAULT false,
    PRIMARY KEY (pasta_id, usuario_id)
);

ALTER TABLE cofre_segredos
    ADD COLUMN pasta_id uuid REFERENCES cofre_pastas (id) ON DELETE SET NULL;

CREATE INDEX cofre_segredos_pasta_idx ON cofre_segredos (pasta_id);

-- +goose Down
DROP INDEX IF EXISTS cofre_segredos_pasta_idx;
ALTER TABLE cofre_segredos DROP COLUMN pasta_id;
DROP TABLE IF EXISTS cofre_pasta_membros;
DROP TABLE IF EXISTS cofre_pastas;
