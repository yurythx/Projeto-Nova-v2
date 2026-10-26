-- +goose Up
-- A retirada reserva o segredo por um prazo. A senha não entra nesta tabela.
-- Uma linha sem devolvida_em é a reserva em aberto, mesmo depois do prazo:
-- quem for usar de novo a encerra antes de abrir outra.

CREATE TABLE cofre_retiradas (
    id           UUID PRIMARY KEY,
    segredo_id   UUID NOT NULL REFERENCES cofre_segredos (id) ON DELETE CASCADE,
    usuario_id   UUID NOT NULL,
    nome         TEXT NOT NULL DEFAULT '',
    ate          TIMESTAMPTZ NOT NULL,
    devolvida_em TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX cofre_retiradas_aberta_idx
    ON cofre_retiradas (segredo_id)
    WHERE devolvida_em IS NULL;

-- +goose Down
DROP TABLE cofre_retiradas;
