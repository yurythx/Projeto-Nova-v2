-- +goose Up
-- Acervo, Fase 3 (ADR 034): o rito da eliminação da Lei Municipal nº
-- 10.301/2019 — comissões (art. 31), Relação de Eliminação (Anexo I, art. 34),
-- Ata da CSPAD (Anexo IV, art. 33), autorização do Arquivo Público (arts. 19
-- e 65), Edital de Ciência com 30 dias (Anexo III, art. 37), pedidos de
-- desentranhamento, amostragem (art. 39) e Termo de Eliminação (Anexo II,
-- arts. 38 e 40).

CREATE TABLE acervo_comissoes (
    id            UUID PRIMARY KEY,
    tipo          TEXT NOT NULL CHECK (tipo IN ('SETORIAL', 'CENTRAL')),
    -- O órgão (entidade) da CSPAD; a CCPAD é do Arquivo Público (sem órgão).
    entidade_id   UUID REFERENCES entidades (id) ON DELETE RESTRICT,
    nome          TEXT NOT NULL CHECK (char_length(nome) BETWEEN 1 AND 200),
    portaria      TEXT NOT NULL DEFAULT '' CHECK (char_length(portaria) <= 200),
    ativa         BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((tipo = 'CENTRAL') = (entidade_id IS NULL))
);
-- Uma comissão ativa por órgão; uma CCPAD ativa.
CREATE UNIQUE INDEX uq_acervo_comissao_ativa ON acervo_comissoes (COALESCE(entidade_id, '00000000-0000-0000-0000-000000000000'::uuid)) WHERE ativa;

CREATE TABLE acervo_comissao_membros (
    comissao_id  UUID NOT NULL REFERENCES acervo_comissoes (id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    papel        TEXT NOT NULL CHECK (papel IN ('PRESIDENTE', 'SECRETARIO', 'MEMBRO')),
    area         TEXT NOT NULL CHECK (area IN ('ARQUIVOLOGIA_HISTORIA', 'JURIDICO', 'ADMINISTRACAO_FINANCAS', 'UNIDADE_PRODUTORA', 'ARQUIVO_PUBLICO', 'OUTRA')),
    PRIMARY KEY (comissao_id, user_id)
);
CREATE UNIQUE INDEX uq_acervo_comissao_presidente ON acervo_comissao_membros (comissao_id) WHERE papel = 'PRESIDENTE';

CREATE SEQUENCE acervo_listagens_numero_seq;

CREATE TABLE acervo_listagens (
    id                    UUID PRIMARY KEY,
    numero                TEXT NOT NULL UNIQUE
                              DEFAULT 'LE-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('acervo_listagens_numero_seq')::text, 5, '0'),
    comissao_id           UUID NOT NULL REFERENCES acervo_comissoes (id) ON DELETE RESTRICT,
    unidade_id            UUID NOT NULL REFERENCES unidades (id) ON DELETE RESTRICT,
    situacao              TEXT NOT NULL DEFAULT 'RASCUNHO'
                              CHECK (situacao IN ('RASCUNHO', 'APROVADA', 'AUTORIZADA', 'EM_EDITAL', 'ELIMINADA', 'CANCELADA')),
    responsavel_selecao   UUID REFERENCES users (id) ON DELETE RESTRICT,
    parecer_juridico      TEXT NOT NULL DEFAULT '' CHECK (char_length(parecer_juridico) <= 4000),
    contas_exercicios     TEXT NOT NULL DEFAULT '' CHECK (char_length(contas_exercicios) <= 200),
    contas_aprovadas_em   DATE,
    -- Ata da CSPAD (Anexo IV)
    ata_numero            TEXT NOT NULL DEFAULT '' CHECK (char_length(ata_numero) <= 40),
    ata_em                TIMESTAMPTZ,
    ata_local             TEXT NOT NULL DEFAULT '' CHECK (char_length(ata_local) <= 200),
    ata_manifestacoes     TEXT NOT NULL DEFAULT '' CHECK (char_length(ata_manifestacoes) <= 8000),
    ata_presentes         UUID[] NOT NULL DEFAULT '{}',
    -- Autorização do Arquivo Público (Anexo I, "Autorizo")
    autorizada_por        UUID,
    autorizada_em         TIMESTAMPTZ,
    devolucao_motivo      TEXT NOT NULL DEFAULT '' CHECK (char_length(devolucao_motivo) <= 2000),
    -- Edital de Ciência (Anexo III)
    edital_numero         TEXT NOT NULL DEFAULT '' CHECK (char_length(edital_numero) <= 40),
    edital_publicado_em   DATE,
    edital_diario         TEXT NOT NULL DEFAULT '' CHECK (char_length(edital_diario) <= 200),
    -- Amostragem (art. 39)
    amostra_criterios     TEXT NOT NULL DEFAULT '' CHECK (char_length(amostra_criterios) <= 2000),
    amostra_percentual    NUMERIC(5,2),
    amostra_em            TIMESTAMPTZ,
    -- Termo de Eliminação (Anexo II)
    termo_eliminado_em    DATE,
    termo_acompanhamento  TEXT NOT NULL DEFAULT '' CHECK (char_length(termo_acompanhamento) <= 200),
    termo_destinacao_papel TEXT NOT NULL DEFAULT '' CHECK (char_length(termo_destinacao_papel) <= 500),
    termo_registrado_por  UUID,
    criada_por            UUID,
    criada_em             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Cada etapa exige a anterior.
    CHECK (situacao IN ('RASCUNHO', 'CANCELADA') OR (ata_em IS NOT NULL AND ata_numero <> '')),
    CHECK (situacao NOT IN ('AUTORIZADA', 'EM_EDITAL', 'ELIMINADA') OR autorizada_em IS NOT NULL),
    CHECK (situacao NOT IN ('EM_EDITAL', 'ELIMINADA') OR (edital_publicado_em IS NOT NULL AND edital_numero <> '')),
    CHECK (situacao <> 'ELIMINADA' OR (amostra_em IS NOT NULL AND termo_eliminado_em IS NOT NULL
        AND termo_eliminado_em >= edital_publicado_em + 30))
);
CREATE INDEX idx_acervo_listagens_unidade ON acervo_listagens (unidade_id, situacao);

CREATE TABLE acervo_listagem_itens (
    listagem_id  UUID NOT NULL REFERENCES acervo_listagens (id) ON DELETE CASCADE,
    unidade_id   UUID NOT NULL REFERENCES acervo_unidades (id) ON DELETE RESTRICT,
    amostra      BOOLEAN NOT NULL DEFAULT false,
    retirada     BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (listagem_id, unidade_id)
);
CREATE INDEX idx_acervo_listagem_itens_unidade ON acervo_listagem_itens (unidade_id);

CREATE TABLE acervo_listagem_pedidos (
    id            UUID PRIMARY KEY,
    listagem_id   UUID NOT NULL REFERENCES acervo_listagens (id) ON DELETE CASCADE,
    unidade_id    UUID REFERENCES acervo_unidades (id) ON DELETE RESTRICT,
    requerente    TEXT NOT NULL CHECK (char_length(requerente) BETWEEN 1 AND 200),
    pedido        TEXT NOT NULL CHECK (char_length(pedido) BETWEEN 1 AND 4000),
    recebido_em   DATE NOT NULL,
    situacao      TEXT NOT NULL DEFAULT 'PENDENTE' CHECK (situacao IN ('PENDENTE', 'DEFERIDO', 'INDEFERIDO')),
    decisao       TEXT NOT NULL DEFAULT '' CHECK (char_length(decisao) <= 4000),
    decidido_por  UUID,
    decidido_em   TIMESTAMPTZ,
    CHECK ((situacao = 'PENDENTE') = (decidido_em IS NULL))
);

-- +goose StatementBegin
CREATE TRIGGER trg_acervo_comissoes_set_updated_at BEFORE UPDATE ON acervo_comissoes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_listagens_set_updated_at BEFORE UPDATE ON acervo_listagens
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- Trava do banco: amostra (art. 39) nunca é eliminada.
-- +goose StatementBegin
CREATE FUNCTION acervo_amostra_travada() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.fase = 'ELIMINADO' AND OLD.fase <> 'ELIMINADO' AND EXISTS (
        SELECT 1 FROM acervo_listagem_itens i WHERE i.unidade_id = NEW.id AND i.amostra
    ) THEN
        RAISE EXCEPTION 'acervo: a amostra (art. 39) vai para a guarda permanente' USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_amostra_travada BEFORE UPDATE OF fase ON acervo_unidades
    FOR EACH ROW EXECUTE FUNCTION acervo_amostra_travada();
-- +goose StatementEnd

ALTER TABLE acervo_movimentacoes DROP CONSTRAINT acervo_movimentacoes_tipo_check;
ALTER TABLE acervo_movimentacoes ADD CONSTRAINT acervo_movimentacoes_tipo_check
    CHECK (tipo IN ('CADASTRO', 'ALTERACAO', 'TRANSFERENCIA', 'LOCALIZACAO', 'BLOQUEIO', 'DESBLOQUEIO',
                    'EMPRESTIMO', 'DEVOLUCAO', 'GUIA', 'IMPORTACAO', 'LISTAGEM', 'AMOSTRA', 'ELIMINACAO'));

-- +goose Down
ALTER TABLE acervo_movimentacoes DROP CONSTRAINT acervo_movimentacoes_tipo_check;
DELETE FROM acervo_movimentacoes WHERE tipo IN ('LISTAGEM', 'AMOSTRA', 'ELIMINACAO');
ALTER TABLE acervo_movimentacoes ADD CONSTRAINT acervo_movimentacoes_tipo_check
    CHECK (tipo IN ('CADASTRO', 'ALTERACAO', 'TRANSFERENCIA', 'LOCALIZACAO', 'BLOQUEIO', 'DESBLOQUEIO',
                    'EMPRESTIMO', 'DEVOLUCAO', 'GUIA', 'IMPORTACAO'));
DROP TRIGGER IF EXISTS trg_acervo_amostra_travada ON acervo_unidades;
DROP FUNCTION IF EXISTS acervo_amostra_travada();
DROP TABLE IF EXISTS acervo_listagem_pedidos;
DROP TABLE IF EXISTS acervo_listagem_itens;
DROP TABLE IF EXISTS acervo_listagens;
DROP SEQUENCE IF EXISTS acervo_listagens_numero_seq;
DROP TABLE IF EXISTS acervo_comissao_membros;
DROP TABLE IF EXISTS acervo_comissoes;
