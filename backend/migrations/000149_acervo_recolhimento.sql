-- +goose Up
-- Acervo, Fase 4 (ADR 035): recolhimento ao Arquivo Público (Lei
-- 10.301/2019, arts. 5º, VII, 16, 17, 41, 42, 47, 50, 51 e 53).

-- Guarda do Arquivo Público: a entrada fica registrada na unidade. A unidade
-- produtora (unidade_id) segue como proveniência e não muda mais.
ALTER TABLE acervo_unidades
    ADD COLUMN recolhida_em    TIMESTAMPTZ,
    ADD COLUMN recolhimento_id UUID,
    ADD CONSTRAINT ck_acervo_recolhida_permanente CHECK (recolhida_em IS NULL OR fase = 'PERMANENTE'),
    ADD CONSTRAINT ck_acervo_recolhida_par CHECK ((recolhida_em IS NULL) = (recolhimento_id IS NULL));

CREATE SEQUENCE acervo_recolhimentos_numero_seq;
CREATE SEQUENCE acervo_registro_entrada_seq;

CREATE TABLE acervo_recolhimentos (
    id                UUID PRIMARY KEY,
    numero            TEXT NOT NULL UNIQUE
                          DEFAULT 'RC-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('acervo_recolhimentos_numero_seq')::text, 5, '0'),
    origem_id         UUID NOT NULL REFERENCES unidades (id) ON DELETE RESTRICT,
    motivo            TEXT NOT NULL CHECK (motivo IN ('REGULAR', 'EXTINCAO', 'DESESTATIZACAO')),
    fundamento        TEXT NOT NULL DEFAULT '' CHECK (char_length(fundamento) <= 500),
    -- Art. 53: o preparo declarado pela origem.
    classificados     BOOLEAN NOT NULL,
    avaliados         BOOLEAN NOT NULL,
    organizados       BOOLEAN NOT NULL,
    higienizados      BOOLEAN NOT NULL,
    acondicionados    BOOLEAN NOT NULL,
    historico         TEXT NOT NULL DEFAULT '' CHECK (char_length(historico) <= 8000),
    observacao        TEXT NOT NULL DEFAULT '' CHECK (char_length(observacao) <= 1000),
    situacao          TEXT NOT NULL DEFAULT 'ENVIADO' CHECK (situacao IN ('ENVIADO', 'RECEBIDO', 'DEVOLVIDO', 'CANCELADO')),
    criado_por        UUID,
    criado_em         TIMESTAMPTZ NOT NULL DEFAULT now(),
    recebido_por      UUID,
    recebido_em       TIMESTAMPTZ,
    registro_entrada  TEXT UNIQUE,
    local_id          UUID REFERENCES acervo_locais (id) ON DELETE RESTRICT,
    devolucao_motivo  TEXT NOT NULL DEFAULT '' CHECK (char_length(devolucao_motivo) <= 2000),
    CHECK (classificados AND avaliados AND organizados AND higienizados AND acondicionados),
    CHECK (motivo = 'REGULAR' OR fundamento <> ''),
    CHECK ((situacao = 'RECEBIDO') = (recebido_em IS NOT NULL AND registro_entrada IS NOT NULL)),
    CHECK ((situacao = 'DEVOLVIDO') = (devolucao_motivo <> ''))
);
CREATE INDEX idx_acervo_recolhimentos_origem ON acervo_recolhimentos (origem_id, criado_em DESC);

CREATE TABLE acervo_recolhimento_itens (
    recolhimento_id  UUID NOT NULL REFERENCES acervo_recolhimentos (id) ON DELETE RESTRICT,
    unidade_id       UUID NOT NULL REFERENCES acervo_unidades (id) ON DELETE RESTRICT,
    PRIMARY KEY (recolhimento_id, unidade_id)
);
CREATE INDEX idx_acervo_recolhimento_itens_unidade ON acervo_recolhimento_itens (unidade_id);

ALTER TABLE acervo_unidades ADD CONSTRAINT fk_acervo_unidades_recolhimento
    FOREIGN KEY (recolhimento_id) REFERENCES acervo_recolhimentos (id) ON DELETE RESTRICT;

-- Recolhida não volta ao órgão nem muda de proveniência (arts. 5º, VII, e 42).
-- +goose StatementBegin
CREATE FUNCTION acervo_recolhida_travada() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.recolhida_em IS NOT NULL AND (NEW.recolhida_em IS DISTINCT FROM OLD.recolhida_em
        OR NEW.recolhimento_id IS DISTINCT FROM OLD.recolhimento_id OR NEW.unidade_id <> OLD.unidade_id) THEN
        RAISE EXCEPTION 'acervo: unidade recolhida fica sob a guarda do Arquivo Público, com a mesma proveniência (art. 5º, VII)'
            USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_recolhida_travada BEFORE UPDATE ON acervo_unidades
    FOR EACH ROW EXECUTE FUNCTION acervo_recolhida_travada();
-- +goose StatementEnd

ALTER TABLE acervo_movimentacoes DROP CONSTRAINT acervo_movimentacoes_tipo_check;
ALTER TABLE acervo_movimentacoes ADD CONSTRAINT acervo_movimentacoes_tipo_check
    CHECK (tipo IN ('CADASTRO', 'ALTERACAO', 'TRANSFERENCIA', 'LOCALIZACAO', 'BLOQUEIO', 'DESBLOQUEIO',
                    'EMPRESTIMO', 'DEVOLUCAO', 'GUIA', 'IMPORTACAO', 'LISTAGEM', 'AMOSTRA', 'ELIMINACAO', 'RECOLHIMENTO'));

-- +goose Down
ALTER TABLE acervo_movimentacoes DROP CONSTRAINT acervo_movimentacoes_tipo_check;
DELETE FROM acervo_movimentacoes WHERE tipo = 'RECOLHIMENTO';
ALTER TABLE acervo_movimentacoes ADD CONSTRAINT acervo_movimentacoes_tipo_check
    CHECK (tipo IN ('CADASTRO', 'ALTERACAO', 'TRANSFERENCIA', 'LOCALIZACAO', 'BLOQUEIO', 'DESBLOQUEIO',
                    'EMPRESTIMO', 'DEVOLUCAO', 'GUIA', 'IMPORTACAO', 'LISTAGEM', 'AMOSTRA', 'ELIMINACAO'));
DROP TRIGGER IF EXISTS trg_acervo_recolhida_travada ON acervo_unidades;
DROP FUNCTION IF EXISTS acervo_recolhida_travada();
ALTER TABLE acervo_unidades DROP CONSTRAINT IF EXISTS fk_acervo_unidades_recolhimento;
DROP TABLE IF EXISTS acervo_recolhimento_itens;
DROP TABLE IF EXISTS acervo_recolhimentos;
DROP SEQUENCE IF EXISTS acervo_registro_entrada_seq;
DROP SEQUENCE IF EXISTS acervo_recolhimentos_numero_seq;
ALTER TABLE acervo_unidades
    DROP CONSTRAINT IF EXISTS ck_acervo_recolhida_par,
    DROP CONSTRAINT IF EXISTS ck_acervo_recolhida_permanente,
    DROP COLUMN IF EXISTS recolhimento_id,
    DROP COLUMN IF EXISTS recolhida_em;
