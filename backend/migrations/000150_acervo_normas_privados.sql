-- +goose Up
-- Acervo, Fase 4, segunda parte (ADR 036): instruções normativas do Arquivo
-- Público com histórico de redações (Lei 10.301/2019, art. 54) e arquivos
-- privados de interesse público e social (arts. 55 a 58).

CREATE TABLE acervo_normas (
    id             UUID PRIMARY KEY,
    numero         TEXT NOT NULL UNIQUE CHECK (numero ~ '^\d{1,4}/\d{4}$'),
    situacao       TEXT NOT NULL DEFAULT 'RASCUNHO' CHECK (situacao IN ('RASCUNHO', 'VIGENTE', 'REVOGADA')),
    revogada_em    DATE,
    revogacao_ato  TEXT NOT NULL DEFAULT '' CHECK (char_length(revogacao_ato) <= 500),
    criada_por     UUID,
    criada_em      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((situacao = 'REVOGADA') = (revogada_em IS NOT NULL AND revogacao_ato <> ''))
);

-- Cada redação é uma versão do texto. Publicada, não muda nem some.
CREATE TABLE acervo_norma_redacoes (
    norma_id       UUID NOT NULL REFERENCES acervo_normas (id) ON DELETE RESTRICT,
    versao         INT NOT NULL CHECK (versao >= 1),
    ementa         TEXT NOT NULL CHECK (char_length(ementa) BETWEEN 1 AND 500),
    texto          TEXT NOT NULL CHECK (char_length(texto) BETWEEN 1 AND 100000),
    motivo         TEXT NOT NULL CHECK (char_length(motivo) BETWEEN 1 AND 500),
    situacao       TEXT NOT NULL DEFAULT 'RASCUNHO' CHECK (situacao IN ('RASCUNHO', 'PUBLICADA')),
    publicada_em   DATE,
    diario         TEXT NOT NULL DEFAULT '' CHECK (char_length(diario) <= 200),
    criada_por     UUID,
    criada_em      TIMESTAMPTZ NOT NULL DEFAULT now(),
    publicada_por  UUID,
    PRIMARY KEY (norma_id, versao),
    CHECK ((situacao = 'PUBLICADA') = (publicada_em IS NOT NULL))
);
-- No máximo um rascunho por norma.
CREATE UNIQUE INDEX uq_acervo_norma_rascunho ON acervo_norma_redacoes (norma_id) WHERE situacao = 'RASCUNHO';

-- +goose StatementBegin
CREATE FUNCTION acervo_redacao_publicada() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.situacao = 'PUBLICADA' THEN
        RAISE EXCEPTION 'acervo: redação publicada de instrução normativa não se altera nem se apaga (art. 54)' USING ERRCODE = 'P0001';
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER trg_acervo_redacao_publicada BEFORE UPDATE OR DELETE ON acervo_norma_redacoes
    FOR EACH ROW EXECUTE FUNCTION acervo_redacao_publicada();
-- +goose StatementEnd

CREATE SEQUENCE acervo_privados_numero_seq;

CREATE TABLE acervo_privados (
    id                 UUID PRIMARY KEY,
    numero             TEXT NOT NULL UNIQUE
                           DEFAULT 'AP-' || to_char(now(), 'YYYY') || '-' || lpad(nextval('acervo_privados_numero_seq')::text, 5, '0'),
    titular            TEXT NOT NULL CHECK (char_length(titular) BETWEEN 1 AND 200),
    tipo_titular       TEXT NOT NULL CHECK (tipo_titular IN ('PF', 'PJ')),
    documento          TEXT NOT NULL DEFAULT '' CHECK (char_length(documento) <= 20),
    contato            TEXT NOT NULL DEFAULT '' CHECK (char_length(contato) <= 500),
    descricao          TEXT NOT NULL CHECK (char_length(descricao) BETWEEN 1 AND 8000),
    datas_limite       TEXT NOT NULL DEFAULT '' CHECK (char_length(datas_limite) <= 100),
    mensuracao         TEXT NOT NULL DEFAULT '' CHECK (char_length(mensuracao) <= 200),
    local_guarda       TEXT NOT NULL DEFAULT '' CHECK (char_length(local_guarda) <= 500),
    servico_publico    BOOLEAN NOT NULL DEFAULT false,
    situacao           TEXT NOT NULL CHECK (situacao IN ('CADASTRADO', 'COM_PARECER', 'NAO_DECLARADO', 'DECLARADO')),
    parecer_comissao   TEXT NOT NULL DEFAULT '' CHECK (char_length(parecer_comissao) <= 2000),
    parecer_texto      TEXT NOT NULL DEFAULT '' CHECK (char_length(parecer_texto) <= 8000),
    parecer_favoravel  BOOLEAN,
    parecer_em         DATE,
    decreto_numero     TEXT NOT NULL DEFAULT '' CHECK (char_length(decreto_numero) <= 40),
    decreto_em         DATE,
    decreto_diario     TEXT NOT NULL DEFAULT '' CHECK (char_length(decreto_diario) <= 200),
    acesso_autorizado  BOOLEAN NOT NULL DEFAULT false,
    acesso_condicoes   TEXT NOT NULL DEFAULT '' CHECK (char_length(acesso_condicoes) <= 2000),
    guarda             TEXT NOT NULL DEFAULT 'TITULAR' CHECK (guarda IN ('TITULAR', 'DOADO', 'DEPOSITADO')),
    guarda_em          DATE,
    guarda_termo       TEXT NOT NULL DEFAULT '' CHECK (char_length(guarda_termo) <= 1000),
    convenio           TEXT NOT NULL DEFAULT '' CHECK (char_length(convenio) <= 1000),
    criado_por         UUID,
    criado_em          TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- § 3º: só pessoa jurídica presta serviço público.
    CHECK (NOT servico_publico OR tipo_titular = 'PJ'),
    -- O decreto vem depois do parecer favorável; a entidade de serviço
    -- público é declarada pela própria lei (§ 3º).
    CHECK (situacao <> 'DECLARADO' OR servico_publico OR (parecer_favoravel AND decreto_em IS NOT NULL)),
    CHECK (guarda = 'TITULAR' OR situacao = 'DECLARADO')
);
CREATE INDEX idx_acervo_privados_titular ON acervo_privados (lower(titular));

CREATE TABLE acervo_privado_eventos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    privado_id  UUID NOT NULL REFERENCES acervo_privados (id) ON DELETE RESTRICT,
    tipo        TEXT NOT NULL CHECK (char_length(tipo) BETWEEN 1 AND 40),
    descricao   TEXT NOT NULL CHECK (char_length(descricao) BETWEEN 1 AND 2000),
    por         UUID,
    em          TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX idx_acervo_privado_eventos ON acervo_privado_eventos (privado_id, em DESC);

CREATE TABLE acervo_privado_alienacoes (
    id             UUID PRIMARY KEY,
    privado_id     UUID NOT NULL REFERENCES acervo_privados (id) ON DELETE RESTRICT,
    notificado_em  DATE NOT NULL,
    adquirente     TEXT NOT NULL DEFAULT '' CHECK (char_length(adquirente) <= 200),
    condicoes      TEXT NOT NULL DEFAULT '' CHECK (char_length(condicoes) <= 2000),
    decisao        TEXT NOT NULL DEFAULT 'PENDENTE' CHECK (decisao IN ('PENDENTE', 'INTERESSE', 'SEM_INTERESSE')),
    fundamento     TEXT NOT NULL DEFAULT '' CHECK (char_length(fundamento) <= 2000),
    decidido_por   UUID,
    decidido_em    TIMESTAMPTZ,
    criada_por     UUID,
    CHECK ((decisao = 'PENDENTE') = (decidido_em IS NULL))
);
CREATE INDEX idx_acervo_privado_alienacoes ON acervo_privado_alienacoes (privado_id, notificado_em DESC);

-- +goose Down
DROP TABLE IF EXISTS acervo_privado_alienacoes;
DROP TABLE IF EXISTS acervo_privado_eventos;
DROP TABLE IF EXISTS acervo_privados;
DROP SEQUENCE IF EXISTS acervo_privados_numero_seq;
DROP TRIGGER IF EXISTS trg_acervo_redacao_publicada ON acervo_norma_redacoes;
DROP TABLE IF EXISTS acervo_norma_redacoes;
DROP FUNCTION IF EXISTS acervo_redacao_publicada();
DROP TABLE IF EXISTS acervo_normas;
