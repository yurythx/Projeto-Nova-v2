-- +goose Up
-- Fase 6: problemas. O problema é a causa por trás de incidentes que se
-- repetem. Com a solução de contorno registrada ele vira erro conhecido —
-- a base que a equipe de atendimento consulta; com a causa raiz, é
-- resolvido. Incidente e ativo são de outros plug-ins: os vínculos guardam
-- só o id (sem chave estrangeira) e a aplicação confere pela porta. Do
-- incidente guardamos número e título, que não mudam depois de abertos.

CREATE TABLE problemas (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero         BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    titulo         TEXT NOT NULL,
    descricao      TEXT NOT NULL DEFAULT '',
    estado         TEXT NOT NULL DEFAULT 'registrado',
    causa_raiz     TEXT NOT NULL DEFAULT '',
    contorno       TEXT NOT NULL DEFAULT '',
    produto_id     UUID,
    unidade_id     UUID REFERENCES unidades (id) ON DELETE SET NULL,
    criado_por     UUID REFERENCES users (id) ON DELETE SET NULL,
    resolvido_em   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT problemas_titulo_chk CHECK (char_length(btrim(titulo)) BETWEEN 1 AND 160),
    CONSTRAINT problemas_textos_chk CHECK (char_length(descricao) <= 4000 AND char_length(causa_raiz) <= 4000 AND char_length(contorno) <= 4000),
    CONSTRAINT problemas_estado_chk CHECK (estado IN ('registrado', 'em_analise', 'erro_conhecido', 'resolvido', 'encerrado')),
    -- Erro conhecido tem contorno; resolvido e encerrado têm causa raiz.
    CONSTRAINT problemas_erro_conhecido_chk CHECK (estado <> 'erro_conhecido' OR btrim(contorno) <> ''),
    CONSTRAINT problemas_resolvido_chk CHECK (estado NOT IN ('resolvido', 'encerrado') OR btrim(causa_raiz) <> '')
);

CREATE INDEX idx_problemas_unidade ON problemas (unidade_id, estado, created_at);
CREATE INDEX idx_problemas_erro_conhecido ON problemas (unidade_id) WHERE estado = 'erro_conhecido';

CREATE TRIGGER trg_problemas_updated
    BEFORE UPDATE ON problemas
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE problemas_incidentes (
    problema_id     UUID NOT NULL REFERENCES problemas (id) ON DELETE CASCADE,
    solicitacao_id  UUID NOT NULL,
    numero          BIGINT NOT NULL,
    titulo          TEXT NOT NULL,
    vinculado_por   UUID REFERENCES users (id) ON DELETE SET NULL,
    vinculado_em    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (problema_id, solicitacao_id)
);

CREATE INDEX idx_problemas_incidentes_solicitacao ON problemas_incidentes (solicitacao_id);

CREATE TABLE problemas_ativos (
    problema_id    UUID NOT NULL REFERENCES problemas (id) ON DELETE CASCADE,
    ativo_id       UUID NOT NULL,
    vinculado_por  UUID REFERENCES users (id) ON DELETE SET NULL,
    vinculado_em   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (problema_id, ativo_id)
);

-- +goose Down
DROP TABLE IF EXISTS problemas_ativos;
DROP TABLE IF EXISTS problemas_incidentes;
DROP TABLE IF EXISTS problemas;
