-- +goose Up
-- Fase 1 da gestão de serviços (ITIL 5): produto digital, com o estágio
-- do ciclo de vida, e a base do atendimento (oferta e equipe).
-- A oferta guarda o id do produto sem chave estrangeira: plug-ins não
-- compartilham schema; a aplicação confere a existência e a unidade.
-- unidade_id nula = recurso institucional (só a gestão global). ON DELETE
-- SET NULL segue o ADR 013.

CREATE TABLE produtos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome        TEXT NOT NULL,
    descricao   TEXT NOT NULL DEFAULT '',
    estagio     TEXT NOT NULL,
    utilidade   TEXT NOT NULL DEFAULT '',
    garantia    TEXT NOT NULL DEFAULT '',
    unidade_id  UUID REFERENCES unidades (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT produtos_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 160),
    CONSTRAINT produtos_estagio_chk CHECK (estagio IN (
        'descoberta', 'desenho', 'aquisicao', 'construcao', 'transicao', 'operacao', 'entrega', 'suporte'
    ))
);

CREATE INDEX idx_produtos_unidade ON produtos (unidade_id);
CREATE INDEX idx_produtos_nome ON produtos (nome);

CREATE TRIGGER trg_produtos_updated
    BEFORE UPDATE ON produtos
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE atendimento_ofertas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome        TEXT NOT NULL,
    descricao   TEXT NOT NULL DEFAULT '',
    produto_id  UUID,
    unidade_id  UUID REFERENCES unidades (id) ON DELETE SET NULL,
    ativa       BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_ofertas_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 160)
);

CREATE INDEX idx_atendimento_ofertas_unidade ON atendimento_ofertas (unidade_id);

CREATE TRIGGER trg_atendimento_ofertas_updated
    BEFORE UPDATE ON atendimento_ofertas
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE atendimento_equipes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome        TEXT NOT NULL,
    unidade_id  UUID REFERENCES unidades (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_equipes_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 160)
);

CREATE INDEX idx_atendimento_equipes_unidade ON atendimento_equipes (unidade_id);

CREATE TRIGGER trg_atendimento_equipes_updated
    BEFORE UPDATE ON atendimento_equipes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE atendimento_equipe_membros (
    equipe_id  UUID NOT NULL REFERENCES atendimento_equipes (id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (equipe_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS atendimento_equipe_membros;
DROP TABLE IF EXISTS atendimento_equipes;
DROP TABLE IF EXISTS atendimento_ofertas;
DROP TABLE IF EXISTS produtos;
