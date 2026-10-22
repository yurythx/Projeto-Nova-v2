-- +goose Up
-- Fase 2: o servidor abre a solicitação pela oferta e a equipe da unidade
-- atende a fila. A oferta diz se o pedido é incidente (restaurar o que
-- parou) ou requisição (cumprir o que a oferta promete); a solicitação
-- herda dela o tipo, a unidade que atende e o produto.
--
-- As máquinas de estado são distintas: o par (tipo, estado) é o que o
-- banco aceita; a passagem entre estados fica no domínio. Os dois estados
-- de espera são separados porque pausam o relógio de prazo (fase 4) por
-- motivos diferentes.
--
-- produto_id é cópia da oferta, sem chave estrangeira: plug-ins não
-- compartilham schema (ver 000159).

ALTER TABLE atendimento_ofertas
    ADD COLUMN tipo TEXT NOT NULL DEFAULT 'requisicao',
    ADD CONSTRAINT atendimento_ofertas_tipo_chk CHECK (tipo IN ('incidente', 'requisicao'));

CREATE TABLE atendimento_solicitacoes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero          BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    tipo            TEXT NOT NULL,
    estado          TEXT NOT NULL,
    titulo          TEXT NOT NULL,
    descricao       TEXT NOT NULL DEFAULT '',
    oferta_id       UUID NOT NULL REFERENCES atendimento_ofertas (id) ON DELETE RESTRICT,
    produto_id      UUID,
    unidade_id      UUID REFERENCES unidades (id) ON DELETE SET NULL,
    solicitante_id  UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    responsavel_id  UUID REFERENCES users (id) ON DELETE SET NULL,
    resolvida_em    TIMESTAMPTZ,
    encerrada_em    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_solicitacoes_titulo_chk CHECK (char_length(btrim(titulo)) BETWEEN 1 AND 160),
    CONSTRAINT atendimento_solicitacoes_descricao_chk CHECK (char_length(descricao) <= 4000),
    CONSTRAINT atendimento_solicitacoes_tipo_estado_chk CHECK (
        (tipo = 'incidente' AND estado IN (
            'aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro',
            'resolvido', 'encerrado', 'cancelado'
        ))
        OR
        (tipo = 'requisicao' AND estado IN (
            'aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro',
            'atendido', 'encerrado', 'cancelado'
        ))
    )
);

-- Fila da unidade (estado + ordem de chegada), fila de cada técnico e a
-- lista de quem abriu.
CREATE INDEX idx_atendimento_solicitacoes_fila ON atendimento_solicitacoes (unidade_id, estado, created_at);
CREATE INDEX idx_atendimento_solicitacoes_responsavel ON atendimento_solicitacoes (responsavel_id, estado)
    WHERE responsavel_id IS NOT NULL;
CREATE INDEX idx_atendimento_solicitacoes_solicitante ON atendimento_solicitacoes (solicitante_id, created_at DESC);

CREATE TRIGGER trg_atendimento_solicitacoes_updated
    BEFORE UPDATE ON atendimento_solicitacoes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Cada passagem de estado com o horário: é o fluxo de valor que a fase 4
-- mede em tempo útil. A auditoria guarda o mesmo fato, mas não é base de
-- consulta.
CREATE TABLE atendimento_passagens (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    solicitacao_id  UUID NOT NULL REFERENCES atendimento_solicitacoes (id) ON DELETE CASCADE,
    de              TEXT,
    para            TEXT NOT NULL,
    autor_id        UUID REFERENCES users (id) ON DELETE SET NULL,
    em              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_atendimento_passagens_solicitacao ON atendimento_passagens (solicitacao_id, em);

-- Conversa da solicitação. Nota interna é da equipe e nunca chega a quem
-- abriu; resposta é visível às duas pontas. Comentário não se edita.
CREATE TABLE atendimento_comentarios (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    solicitacao_id  UUID NOT NULL REFERENCES atendimento_solicitacoes (id) ON DELETE CASCADE,
    autor_id        UUID REFERENCES users (id) ON DELETE SET NULL,
    texto           TEXT NOT NULL,
    interno         BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_comentarios_texto_chk CHECK (char_length(btrim(texto)) BETWEEN 1 AND 4000)
);

CREATE INDEX idx_atendimento_comentarios_solicitacao ON atendimento_comentarios (solicitacao_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS atendimento_comentarios;
DROP TABLE IF EXISTS atendimento_passagens;
DROP TABLE IF EXISTS atendimento_solicitacoes;
ALTER TABLE atendimento_ofertas DROP CONSTRAINT IF EXISTS atendimento_ofertas_tipo_chk;
ALTER TABLE atendimento_ofertas DROP COLUMN IF EXISTS tipo;
