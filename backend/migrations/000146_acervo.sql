-- +goose Up
-- Acervo do SISMARQ (ADR 032; Fase 1 do docs/ROADMAP_SISMARQ.md): unidades de
-- arquivamento, localização física, bloqueio judicial e histórico. As travas
-- da Lei Municipal nº 10.301/2019 valem também aqui, no banco — nenhum
-- caminho fora da aplicação as contorna:
--   * permanente nunca muda de fase (arts. 20 e 42);
--   * eliminado é registro histórico, não volta (art. 38);
--   * nada é eliminado com bloqueio judicial ativo (art. 44) nem sem
--     destinação de eliminação na TTDD (art. 19);
--   * unidade de arquivamento nunca é apagada.

CREATE TABLE acervo_locais (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id   UUID REFERENCES acervo_locais (id) ON DELETE RESTRICT,
    tipo        TEXT NOT NULL CHECK (tipo IN ('PREDIO', 'DEPOSITO', 'SALA', 'CORREDOR', 'ESTANTE', 'PRATELEIRA')),
    nome        TEXT NOT NULL CHECK (char_length(nome) BETWEEN 1 AND 120),
    ativo       BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE UNIQUE INDEX uq_acervo_locais_nome ON acervo_locais (COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(nome));

CREATE SEQUENCE acervo_unidades_numero_seq;

CREATE TABLE acervo_unidades (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero                TEXT NOT NULL UNIQUE
                              DEFAULT 'UA-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('acervo_unidades_numero_seq')::text, 6, '0'),
    tipo                  TEXT NOT NULL CHECK (tipo IN ('CAIXA', 'PASTA', 'LIVRO', 'PROCESSO', 'DOSSIE', 'DOCUMENTO', 'MAPA', 'OUTRO')),
    codigo_classificacao  TEXT NOT NULL CHECK (char_length(codigo_classificacao) BETWEEN 1 AND 32),
    assunto               TEXT NOT NULL DEFAULT '' CHECK (char_length(assunto) <= 500),
    unidade_id            UUID NOT NULL REFERENCES unidades (id) ON DELETE RESTRICT,
    data_inicial          DATE NOT NULL,
    data_final            DATE NOT NULL,
    quantidade            INTEGER NOT NULL DEFAULT 1 CHECK (quantidade BETWEEN 1 AND 100000),
    especificacao         TEXT NOT NULL DEFAULT '' CHECK (char_length(especificacao) <= 120),
    observacoes           TEXT NOT NULL DEFAULT '' CHECK (char_length(observacoes) <= 4000),
    fase                  TEXT NOT NULL DEFAULT 'CORRENTE' CHECK (fase IN ('CORRENTE', 'INTERMEDIARIA', 'PERMANENTE', 'ELIMINADO')),
    destinacao            TEXT NOT NULL CHECK (destinacao IN ('ELIMINACAO', 'GUARDA_PERMANENTE', 'INDEFINIDA')),
    data_evento           DATE,
    contas_aprovadas_em   DATE,
    nivel_acesso          TEXT NOT NULL DEFAULT 'PUBLICO' CHECK (nivel_acesso IN ('PUBLICO', 'RESTRITO', 'SIGILOSO')),
    hipotese_legal        TEXT NOT NULL DEFAULT '' CHECK (char_length(hipotese_legal) <= 500),
    local_id              UUID REFERENCES acervo_locais (id) ON DELETE RESTRICT,
    posicao               TEXT NOT NULL DEFAULT '' CHECK (char_length(posicao) <= 120),
    -- Resultado gravado do cálculo de prazos (recalculado a cada escrita,
    -- diariamente e quando a TTDD muda): filtra e conta sem refazer a conta.
    fim_corrente          DATE,
    fim_intermediaria     DATE,
    situacao              TEXT NOT NULL,
    created_by            UUID,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    search                TSVECTOR GENERATED ALWAYS AS (
        setweight(to_tsvector('portuguese', nexus_unaccent(numero || ' ' || codigo_classificacao)), 'A') ||
        setweight(to_tsvector('portuguese', nexus_unaccent(assunto)), 'B') ||
        setweight(to_tsvector('portuguese', nexus_unaccent(especificacao || ' ' || observacoes)), 'C')
    ) STORED,
    CHECK (data_final >= data_inicial),
    CHECK (nivel_acesso = 'PUBLICO' OR hipotese_legal <> '')
);
CREATE INDEX idx_acervo_unidades_unidade ON acervo_unidades (unidade_id, fase);
CREATE INDEX idx_acervo_unidades_situacao ON acervo_unidades (situacao);
CREATE INDEX idx_acervo_unidades_codigo ON acervo_unidades (codigo_classificacao text_pattern_ops);
CREATE INDEX idx_acervo_unidades_search ON acervo_unidades USING gin (search);

CREATE TABLE acervo_bloqueios (
    id                 UUID PRIMARY KEY,
    unidade_id         UUID NOT NULL REFERENCES acervo_unidades (id) ON DELETE RESTRICT,
    processo           TEXT NOT NULL CHECK (char_length(processo) BETWEEN 1 AND 120),
    motivo             TEXT NOT NULL DEFAULT '' CHECK (char_length(motivo) <= 2000),
    criado_por         UUID,
    criado_em          TIMESTAMPTZ NOT NULL DEFAULT now(),
    liberado_por       UUID,
    liberado_em        TIMESTAMPTZ,
    motivo_liberacao   TEXT NOT NULL DEFAULT '' CHECK (char_length(motivo_liberacao) <= 2000),
    CHECK (liberado_em IS NULL OR motivo_liberacao <> '')
);
CREATE INDEX idx_acervo_bloqueios_ativos ON acervo_bloqueios (unidade_id) WHERE liberado_em IS NULL;

CREATE TABLE acervo_movimentacoes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unidade_id  UUID NOT NULL REFERENCES acervo_unidades (id) ON DELETE RESTRICT,
    tipo        TEXT NOT NULL CHECK (tipo IN ('CADASTRO', 'ALTERACAO', 'TRANSFERENCIA', 'LOCALIZACAO', 'BLOQUEIO', 'DESBLOQUEIO')),
    fase_de     TEXT,
    fase_para   TEXT,
    descricao   TEXT NOT NULL DEFAULT '' CHECK (char_length(descricao) <= 2000),
    por         UUID,
    em          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_acervo_movimentacoes_unidade ON acervo_movimentacoes (unidade_id, em DESC);

-- +goose StatementBegin
CREATE FUNCTION acervo_travas() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'acervo: unidade de arquivamento não se apaga (Lei 10.301/2019)' USING ERRCODE = 'P0001';
    END IF;
    IF OLD.fase = 'PERMANENTE' AND NEW.fase <> 'PERMANENTE' THEN
        RAISE EXCEPTION 'acervo: documento permanente é inalienável (arts. 20 e 42)' USING ERRCODE = 'P0001';
    END IF;
    IF OLD.fase = 'ELIMINADO' AND (NEW.fase <> 'ELIMINADO' OR NEW.codigo_classificacao <> OLD.codigo_classificacao) THEN
        RAISE EXCEPTION 'acervo: unidade eliminada é registro histórico (art. 38)' USING ERRCODE = 'P0001';
    END IF;
    IF NEW.fase = 'ELIMINADO' AND OLD.fase <> 'ELIMINADO' THEN
        IF NEW.destinacao <> 'ELIMINACAO' THEN
            RAISE EXCEPTION 'acervo: a TTDD não destina a série à eliminação (art. 19)' USING ERRCODE = 'P0001';
        END IF;
        IF EXISTS (SELECT 1 FROM acervo_bloqueios b WHERE b.unidade_id = NEW.id AND b.liberado_em IS NULL) THEN
            RAISE EXCEPTION 'acervo: bloqueio judicial ativo (art. 44)' USING ERRCODE = 'P0001';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_acervo_travas BEFORE UPDATE OR DELETE ON acervo_unidades
    FOR EACH ROW EXECUTE FUNCTION acervo_travas();
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_unidades_set_updated_at BEFORE UPDATE ON acervo_unidades
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_locais_set_updated_at BEFORE UPDATE ON acervo_locais
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS acervo_movimentacoes;
DROP TABLE IF EXISTS acervo_bloqueios;
DROP TABLE IF EXISTS acervo_unidades;
DROP FUNCTION IF EXISTS acervo_travas();
DROP SEQUENCE IF EXISTS acervo_unidades_numero_seq;
DROP TABLE IF EXISTS acervo_locais;
