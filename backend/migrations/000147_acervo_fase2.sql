-- +goose Up
-- Acervo, Fase 2 (ADR 033): empréstimo e devolução (Lei 10.301/2019, art. 12,
-- II; TTDD 2.0.03.00) e guia de transferência ao arquivo intermediário, com
-- aceite de quem recebe (art. 29; a TTDD classifica a "Transferência de
-- documentos intermediários" como série própria, 2.0.03.01.00).

CREATE TABLE acervo_emprestimos (
    id                UUID PRIMARY KEY,
    unidade_id        UUID NOT NULL REFERENCES acervo_unidades (id) ON DELETE RESTRICT,
    solicitante_id    UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    finalidade        TEXT NOT NULL CHECK (char_length(finalidade) BETWEEN 1 AND 500),
    devolver_ate      DATE NOT NULL,
    retirado_por      UUID,
    retirado_em       TIMESTAMPTZ NOT NULL DEFAULT now(),
    devolvido_em      TIMESTAMPTZ,
    recebido_por      UUID,
    observacao        TEXT NOT NULL DEFAULT '' CHECK (char_length(observacao) <= 500)
);
-- Uma unidade tem no máximo um empréstimo em aberto.
CREATE UNIQUE INDEX uq_acervo_emprestimo_aberto ON acervo_emprestimos (unidade_id) WHERE devolvido_em IS NULL;
CREATE INDEX idx_acervo_emprestimos_abertos ON acervo_emprestimos (devolver_ate) WHERE devolvido_em IS NULL;

CREATE SEQUENCE acervo_guias_numero_seq;

CREATE TABLE acervo_guias (
    id            UUID PRIMARY KEY,
    numero        TEXT NOT NULL UNIQUE
                      DEFAULT 'GT-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('acervo_guias_numero_seq')::text, 5, '0'),
    origem_id     UUID NOT NULL REFERENCES unidades (id) ON DELETE RESTRICT,
    situacao      TEXT NOT NULL DEFAULT 'ENVIADA' CHECK (situacao IN ('ENVIADA', 'RECEBIDA', 'CANCELADA')),
    observacao    TEXT NOT NULL DEFAULT '' CHECK (char_length(observacao) <= 1000),
    criada_por    UUID,
    criada_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
    recebida_por  UUID,
    recebida_em   TIMESTAMPTZ,
    CHECK ((situacao = 'RECEBIDA') = (recebida_em IS NOT NULL))
);

CREATE TABLE acervo_guia_itens (
    guia_id     UUID NOT NULL REFERENCES acervo_guias (id) ON DELETE RESTRICT,
    unidade_id  UUID NOT NULL REFERENCES acervo_unidades (id) ON DELETE RESTRICT,
    PRIMARY KEY (guia_id, unidade_id)
);
CREATE INDEX idx_acervo_guia_itens_unidade ON acervo_guia_itens (unidade_id);

ALTER TABLE acervo_movimentacoes DROP CONSTRAINT acervo_movimentacoes_tipo_check;
ALTER TABLE acervo_movimentacoes ADD CONSTRAINT acervo_movimentacoes_tipo_check
    CHECK (tipo IN ('CADASTRO', 'ALTERACAO', 'TRANSFERENCIA', 'LOCALIZACAO', 'BLOQUEIO', 'DESBLOQUEIO',
                    'EMPRESTIMO', 'DEVOLUCAO', 'GUIA', 'IMPORTACAO'));
-- Várias movimentações na mesma transação (guia + transferência) precisam de
-- ordem: o relógio de agora, não o do início da transação.
ALTER TABLE acervo_movimentacoes ALTER COLUMN em SET DEFAULT clock_timestamp();

-- +goose Down
ALTER TABLE acervo_movimentacoes ALTER COLUMN em SET DEFAULT now();
ALTER TABLE acervo_movimentacoes DROP CONSTRAINT acervo_movimentacoes_tipo_check;
DELETE FROM acervo_movimentacoes WHERE tipo IN ('EMPRESTIMO', 'DEVOLUCAO', 'GUIA', 'IMPORTACAO');
ALTER TABLE acervo_movimentacoes ADD CONSTRAINT acervo_movimentacoes_tipo_check
    CHECK (tipo IN ('CADASTRO', 'ALTERACAO', 'TRANSFERENCIA', 'LOCALIZACAO', 'BLOQUEIO', 'DESBLOQUEIO'));
DROP TABLE IF EXISTS acervo_guia_itens;
DROP TABLE IF EXISTS acervo_guias;
DROP SEQUENCE IF EXISTS acervo_guias_numero_seq;
DROP TABLE IF EXISTS acervo_emprestimos;
