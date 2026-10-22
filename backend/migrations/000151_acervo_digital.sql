-- +goose Up
-- Acervo, Fase 5 (ADR 037): representante digital com valor probatório
-- (Decreto Federal 10.278/2020; Lei 10.301/2019, art. 48). O PDF/A fica no
-- MinIO com object lock; aqui ficam os metadados do Decreto, os hashes, o
-- texto do OCR e o histórico das verificações de integridade.

CREATE SEQUENCE acervo_digitais_numero_seq;

CREATE TABLE acervo_digitais (
    id                  UUID PRIMARY KEY,
    numero              TEXT NOT NULL UNIQUE
                            DEFAULT 'RD-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('acervo_digitais_numero_seq')::text, 6, '0'),
    unidade_id          UUID NOT NULL REFERENCES acervo_unidades (id) ON DELETE RESTRICT,
    descricao           TEXT NOT NULL DEFAULT '' CHECK (char_length(descricao) <= 500),
    tipo_documento      TEXT NOT NULL CHECK (tipo_documento IN ('TEXTO', 'PLANTA')),
    equipamento         TEXT NOT NULL CHECK (char_length(equipamento) BETWEEN 1 AND 200),
    dpi_declarado       INT NOT NULL CHECK (dpi_declarado BETWEEN 72 AND 4800),
    responsavel_id      UUID NOT NULL,
    responsavel_nome    TEXT NOT NULL DEFAULT '' CHECK (char_length(responsavel_nome) <= 200),
    -- Só o CPF mascarado (***.456.789-**): LGPD, minimização.
    responsavel_cpf     TEXT NOT NULL DEFAULT '' CHECK (responsavel_cpf = '' OR responsavel_cpf ~ '^\*\*\*\.\d{3}\.\d{3}-\*\*$'),
    codigo_classificacao TEXT NOT NULL,
    digitalizado_em     TIMESTAMPTZ NOT NULL,
    situacao            TEXT NOT NULL DEFAULT 'ENVIADO'
                            CHECK (situacao IN ('ENVIADO', 'PROCESSANDO', 'PRONTO', 'REPROVADO', 'FALHOU')),
    motivo              TEXT NOT NULL DEFAULT '' CHECK (char_length(motivo) <= 2000),
    tentativas          INT NOT NULL DEFAULT 0,
    processando_desde   TIMESTAMPTZ,
    original_key        TEXT NOT NULL,
    original_sha256     TEXT NOT NULL CHECK (original_sha256 ~ '^[0-9a-f]{64}$'),
    original_bytes      BIGINT NOT NULL CHECK (original_bytes > 0),
    pdfa_key            TEXT NOT NULL DEFAULT '',
    pdfa_sha256         TEXT NOT NULL DEFAULT '' CHECK (pdfa_sha256 = '' OR pdfa_sha256 ~ '^[0-9a-f]{64}$'),
    pdfa_bytes          BIGINT NOT NULL DEFAULT 0,
    paginas             INT NOT NULL DEFAULT 0,
    dpi_medido          INT NOT NULL DEFAULT 0,
    conformidade        TEXT NOT NULL DEFAULT '',
    retencao_ate        DATE,
    processado_em       TIMESTAMPTZ,
    verificado_em       TIMESTAMPTZ,
    integro             BOOLEAN,
    criado_por          UUID,
    criado_em           TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Pronto = PDF/A guardado com hash e retenção.
    CHECK (situacao <> 'PRONTO' OR (pdfa_key <> '' AND pdfa_sha256 <> '' AND retencao_ate IS NOT NULL AND paginas > 0))
);
CREATE INDEX idx_acervo_digitais_unidade ON acervo_digitais (unidade_id, criado_em DESC);
CREATE INDEX idx_acervo_digitais_fila ON acervo_digitais (criado_em) WHERE situacao IN ('ENVIADO', 'PROCESSANDO');
CREATE INDEX idx_acervo_digitais_fixity ON acervo_digitais (verificado_em NULLS FIRST) WHERE situacao = 'PRONTO';

-- O representante pronto é imutável no que prova: hash, objeto e retenção.
-- +goose StatementBegin
CREATE FUNCTION acervo_digital_travado() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'acervo: representante digital não se apaga (Decreto 10.278/2020)' USING ERRCODE = 'P0001';
    END IF;
    IF OLD.situacao = 'PRONTO' AND (NEW.situacao <> 'PRONTO' OR NEW.pdfa_key <> OLD.pdfa_key OR NEW.pdfa_sha256 <> OLD.pdfa_sha256
        OR NEW.original_sha256 <> OLD.original_sha256 OR NEW.retencao_ate < OLD.retencao_ate) THEN
        RAISE EXCEPTION 'acervo: o representante digital pronto não muda: hash, objeto e retenção ficam como gravados' USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_digital_travado BEFORE UPDATE OR DELETE ON acervo_digitais
    FOR EACH ROW EXECUTE FUNCTION acervo_digital_travado();
-- +goose StatementEnd

-- Texto do OCR (para a busca da Fase 6), separado da linha principal.
CREATE TABLE acervo_digital_textos (
    digital_id  UUID PRIMARY KEY REFERENCES acervo_digitais (id) ON DELETE RESTRICT,
    texto       TEXT NOT NULL
);

-- Cada verificação de integridade (fixity), com o hash encontrado.
CREATE TABLE acervo_digital_verificacoes (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    digital_id     UUID NOT NULL REFERENCES acervo_digitais (id) ON DELETE RESTRICT,
    em             TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp(),
    integro        BOOLEAN NOT NULL,
    sha256_lido    TEXT NOT NULL DEFAULT '',
    erro           TEXT NOT NULL DEFAULT '' CHECK (char_length(erro) <= 1000)
);
CREATE INDEX idx_acervo_digital_verificacoes ON acervo_digital_verificacoes (digital_id, em DESC);

ALTER TABLE acervo_movimentacoes DROP CONSTRAINT acervo_movimentacoes_tipo_check;
ALTER TABLE acervo_movimentacoes ADD CONSTRAINT acervo_movimentacoes_tipo_check
    CHECK (tipo IN ('CADASTRO', 'ALTERACAO', 'TRANSFERENCIA', 'LOCALIZACAO', 'BLOQUEIO', 'DESBLOQUEIO',
                    'EMPRESTIMO', 'DEVOLUCAO', 'GUIA', 'IMPORTACAO', 'LISTAGEM', 'AMOSTRA', 'ELIMINACAO', 'RECOLHIMENTO',
                    'DIGITALIZACAO'));

-- +goose Down
ALTER TABLE acervo_movimentacoes DROP CONSTRAINT acervo_movimentacoes_tipo_check;
DELETE FROM acervo_movimentacoes WHERE tipo = 'DIGITALIZACAO';
ALTER TABLE acervo_movimentacoes ADD CONSTRAINT acervo_movimentacoes_tipo_check
    CHECK (tipo IN ('CADASTRO', 'ALTERACAO', 'TRANSFERENCIA', 'LOCALIZACAO', 'BLOQUEIO', 'DESBLOQUEIO',
                    'EMPRESTIMO', 'DEVOLUCAO', 'GUIA', 'IMPORTACAO', 'LISTAGEM', 'AMOSTRA', 'ELIMINACAO', 'RECOLHIMENTO'));
DROP TABLE IF EXISTS acervo_digital_verificacoes;
DROP TABLE IF EXISTS acervo_digital_textos;
DROP TABLE IF EXISTS acervo_digitais;
DROP FUNCTION IF EXISTS acervo_digital_travado();
DROP SEQUENCE IF EXISTS acervo_digitais_numero_seq;
