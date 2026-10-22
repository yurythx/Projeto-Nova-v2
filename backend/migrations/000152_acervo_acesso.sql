-- +goose Up
-- Acervo, Fase 6 (ADR 038): acesso, busca e difusão (Lei 10.301/2019, arts.
-- 2º, 5º, X e XI, 7º, I, e 13). O ostensivo é a regra; o que vai para a
-- consulta sem login é decidido aqui, no banco, e não só na aplicação:
--   * o representante digital só é liberado à consulta pública se estiver
--     pronto e a unidade for de acesso público e não eliminada;
--   * a unidade que deixa de ser pública (ou é eliminada) tira na hora os
--     seus representantes da consulta pública.

ALTER TABLE acervo_digitais
    ADD COLUMN publicado_em  TIMESTAMPTZ,
    ADD COLUMN publicado_por UUID;
CREATE INDEX idx_acervo_digitais_publicados ON acervo_digitais (publicado_em DESC) WHERE publicado_em IS NOT NULL;

-- +goose StatementBegin
CREATE FUNCTION acervo_digital_publicacao() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.publicado_em IS NOT NULL AND OLD.publicado_em IS NULL THEN
        IF NEW.situacao <> 'PRONTO' THEN
            RAISE EXCEPTION 'acervo: só o representante pronto vai para a consulta pública' USING ERRCODE = 'P0001';
        END IF;
        IF NOT EXISTS (SELECT 1 FROM acervo_unidades u WHERE u.id = NEW.unidade_id
                       AND u.nivel_acesso = 'PUBLICO' AND u.fase <> 'ELIMINADO') THEN
            RAISE EXCEPTION 'acervo: só o documento ostensivo (nível público) vai para a consulta pública (art. 2º)' USING ERRCODE = 'P0001';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_digital_publicacao BEFORE UPDATE OF publicado_em ON acervo_digitais
    FOR EACH ROW EXECUTE FUNCTION acervo_digital_publicacao();
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION acervo_unidade_restringida() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.nivel_acesso <> 'PUBLICO' OR NEW.fase = 'ELIMINADO' THEN
        UPDATE acervo_digitais SET publicado_em = NULL, publicado_por = NULL
         WHERE unidade_id = NEW.id AND publicado_em IS NOT NULL;
    END IF;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_unidade_restringida AFTER UPDATE OF nivel_acesso, fase ON acervo_unidades
    FOR EACH ROW EXECUTE FUNCTION acervo_unidade_restringida();
-- +goose StatementEnd

-- Busca no texto do OCR quando o índice (Typesense) estiver fora do ar. O
-- tsvector tem teto de 1 MB: indexa o começo do texto.
ALTER TABLE acervo_digital_textos
    ADD COLUMN search TSVECTOR GENERATED ALWAYS AS (to_tsvector('portuguese', nexus_unaccent(left(texto, 200000)))) STORED;
CREATE INDEX idx_acervo_digital_textos_search ON acervo_digital_textos USING gin (search);

-- Fila do índice do acervo: toda mudança numa unidade, num representante
-- ou no texto do OCR marca a unidade; o worker reindexa e tira a marca.
-- Pega também o que muda fora da aplicação (o recálculo diário, o SQL).
CREATE TABLE acervo_indice_pendentes (
    unidade_id  UUID PRIMARY KEY REFERENCES acervo_unidades (id) ON DELETE CASCADE,
    desde       TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX idx_acervo_indice_pendentes ON acervo_indice_pendentes (desde);

-- +goose StatementBegin
CREATE FUNCTION acervo_marcar_indice() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    alvo UUID;
BEGIN
    IF TG_TABLE_NAME = 'acervo_unidades' THEN
        alvo := NEW.id;
    ELSIF TG_TABLE_NAME = 'acervo_digitais' THEN
        alvo := NEW.unidade_id;
    ELSE
        SELECT d.unidade_id INTO alvo FROM acervo_digitais d WHERE d.id = NEW.digital_id;
    END IF;
    INSERT INTO acervo_indice_pendentes (unidade_id) VALUES (alvo)
        ON CONFLICT (unidade_id) DO UPDATE SET desde = clock_timestamp();
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_indice_unidade_ins AFTER INSERT ON acervo_unidades
    FOR EACH ROW EXECUTE FUNCTION acervo_marcar_indice();
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_indice_unidade_upd AFTER UPDATE ON acervo_unidades
    FOR EACH ROW WHEN (OLD.* IS DISTINCT FROM NEW.*) EXECUTE FUNCTION acervo_marcar_indice();
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_indice_digital AFTER INSERT OR UPDATE ON acervo_digitais
    FOR EACH ROW EXECUTE FUNCTION acervo_marcar_indice();
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_indice_texto AFTER INSERT OR UPDATE ON acervo_digital_textos
    FOR EACH ROW EXECUTE FUNCTION acervo_marcar_indice();
-- +goose StatementEnd

-- O que já existe entra na fila da primeira carga.
INSERT INTO acervo_indice_pendentes (unidade_id) SELECT id FROM acervo_unidades ON CONFLICT DO NOTHING;

-- Exposições virtuais (difusão, art. 5º, X e XI): o Arquivo Público monta
-- e publica; só aparecem os representantes liberados à consulta pública.
CREATE TABLE acervo_exposicoes (
    id             UUID PRIMARY KEY,
    slug           TEXT NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND char_length(slug) <= 80),
    titulo         TEXT NOT NULL CHECK (char_length(titulo) BETWEEN 1 AND 200),
    resumo         TEXT NOT NULL DEFAULT '' CHECK (char_length(resumo) <= 500),
    texto          TEXT NOT NULL DEFAULT '' CHECK (char_length(texto) <= 20000),
    publicada_em   TIMESTAMPTZ,
    criado_por     UUID,
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT now(),
    atualizado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_acervo_exposicoes_publicadas ON acervo_exposicoes (publicada_em DESC) WHERE publicada_em IS NOT NULL;

CREATE TABLE acervo_exposicao_itens (
    exposicao_id  UUID NOT NULL REFERENCES acervo_exposicoes (id) ON DELETE CASCADE,
    digital_id    UUID NOT NULL REFERENCES acervo_digitais (id) ON DELETE RESTRICT,
    ordem         INT NOT NULL CHECK (ordem >= 0),
    legenda       TEXT NOT NULL DEFAULT '' CHECK (char_length(legenda) <= 1000),
    PRIMARY KEY (exposicao_id, digital_id)
);

-- Consultas atendidas por dia (indicador da SEMAD, art. 38, parágrafo
-- único): buscas, fichas e documentos abertos na consulta pública.
CREATE TABLE acervo_consultas (
    dia    DATE NOT NULL,
    tipo   TEXT NOT NULL CHECK (tipo IN ('BUSCA', 'FICHA', 'DOCUMENTO', 'EXPOSICAO')),
    total  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (dia, tipo)
);

-- +goose Down
DROP TABLE IF EXISTS acervo_consultas;
DROP TABLE IF EXISTS acervo_exposicao_itens;
DROP TABLE IF EXISTS acervo_exposicoes;
DROP TRIGGER IF EXISTS trg_acervo_indice_texto ON acervo_digital_textos;
DROP TRIGGER IF EXISTS trg_acervo_indice_digital ON acervo_digitais;
DROP TRIGGER IF EXISTS trg_acervo_indice_unidade_upd ON acervo_unidades;
DROP TRIGGER IF EXISTS trg_acervo_indice_unidade_ins ON acervo_unidades;
DROP FUNCTION IF EXISTS acervo_marcar_indice();
DROP TABLE IF EXISTS acervo_indice_pendentes;
DROP INDEX IF EXISTS idx_acervo_digital_textos_search;
ALTER TABLE acervo_digital_textos DROP COLUMN IF EXISTS search;
DROP TRIGGER IF EXISTS trg_acervo_unidade_restringida ON acervo_unidades;
DROP FUNCTION IF EXISTS acervo_unidade_restringida();
DROP TRIGGER IF EXISTS trg_acervo_digital_publicacao ON acervo_digitais;
DROP FUNCTION IF EXISTS acervo_digital_publicacao();
DROP INDEX IF EXISTS idx_acervo_digitais_publicados;
ALTER TABLE acervo_digitais DROP COLUMN IF EXISTS publicado_por, DROP COLUMN IF EXISTS publicado_em;
