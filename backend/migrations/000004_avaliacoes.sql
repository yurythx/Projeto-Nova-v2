-- +goose Up
-- Pesquisa de satisfação do atendimento. Uma nota por solicitação.
-- O comentário é opcional. Quem abriu é o autor; o nome sai da conta.

CREATE TABLE atendimento_avaliacoes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    solicitacao_id UUID NOT NULL REFERENCES atendimento_solicitacoes (id) ON DELETE CASCADE,
    autor_id       UUID NOT NULL REFERENCES users (id),
    nota           SMALLINT NOT NULL CHECK (nota BETWEEN 1 AND 5),
    comentario     TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_avaliacoes_solicitacao_uq UNIQUE (solicitacao_id)
);

CREATE INDEX atendimento_avaliacoes_created_at_idx ON atendimento_avaliacoes (created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS atendimento_avaliacoes;
