-- +goose Up
-- Fase 6: ativos (itens de configuração). O ativo sustenta um produto
-- digital: o servidor, a licença, o link, o equipamento. produto_id é
-- referência sem chave estrangeira — plug-ins não compartilham schema;
-- a aplicação confere a existência e a unidade pela porta do produto.
-- Unidade nula = institucional (ADR 013).

CREATE TABLE ativos (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome           TEXT NOT NULL,
    tipo           TEXT NOT NULL,
    estado         TEXT NOT NULL,
    identificador  TEXT NOT NULL DEFAULT '',
    descricao      TEXT NOT NULL DEFAULT '',
    produto_id     UUID,
    unidade_id     UUID REFERENCES unidades (id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ativos_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 160),
    CONSTRAINT ativos_tipo_chk CHECK (tipo IN ('hardware', 'software', 'servico', 'rede', 'outro')),
    CONSTRAINT ativos_estado_chk CHECK (estado IN ('planejado', 'em_uso', 'manutencao', 'desativado')),
    CONSTRAINT ativos_identificador_chk CHECK (char_length(identificador) <= 80),
    CONSTRAINT ativos_descricao_chk CHECK (char_length(descricao) <= 4000)
);

-- Patrimônio ou número de série: único quando informado.
CREATE UNIQUE INDEX uq_ativos_identificador ON ativos (identificador) WHERE identificador <> '';
CREATE INDEX idx_ativos_unidade ON ativos (unidade_id, nome);
CREATE INDEX idx_ativos_produto ON ativos (produto_id) WHERE produto_id IS NOT NULL;

CREATE TRIGGER trg_ativos_updated
    BEFORE UPDATE ON ativos
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS ativos;
