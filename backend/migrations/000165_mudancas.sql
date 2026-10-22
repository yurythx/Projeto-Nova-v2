-- +goose Up
-- Fase 7: mudanças. Nada que está em operação muda sem registro de quem
-- pediu, quem aprovou e como desfazer. A mudança padrão é pré-aprovada; a
-- normal e a emergencial passam por aprovação de quem tem mudancas:aprovar
-- e não foi quem pediu (segregação de funções). Problemas, incidentes e
-- ativos são de outros plug-ins: o vínculo guarda o id e o rótulo do
-- momento, sem chave estrangeira.

CREATE TABLE mudancas (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero            BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE,
    titulo            TEXT NOT NULL,
    descricao         TEXT NOT NULL DEFAULT '',
    justificativa     TEXT NOT NULL DEFAULT '',
    tipo              TEXT NOT NULL,
    risco             TEXT NOT NULL,
    estado            TEXT NOT NULL DEFAULT 'rascunho',
    plano_reversao    TEXT NOT NULL DEFAULT '',
    inicio_planejado  TIMESTAMPTZ,
    fim_planejado     TIMESTAMPTZ,
    resultado         TEXT NOT NULL DEFAULT '',
    executada_inicio  TIMESTAMPTZ,
    executada_fim     TIMESTAMPTZ,
    produto_id        UUID,
    unidade_id        UUID REFERENCES unidades (id) ON DELETE SET NULL,
    solicitante_id    UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT mudancas_titulo_chk CHECK (char_length(btrim(titulo)) BETWEEN 1 AND 160),
    CONSTRAINT mudancas_textos_chk CHECK (
        char_length(descricao) <= 4000 AND char_length(justificativa) <= 4000
        AND char_length(plano_reversao) <= 4000 AND char_length(resultado) <= 4000
    ),
    CONSTRAINT mudancas_tipo_chk CHECK (tipo IN ('padrao', 'normal', 'emergencial')),
    CONSTRAINT mudancas_risco_chk CHECK (risco IN ('baixo', 'medio', 'alto')),
    CONSTRAINT mudancas_estado_chk CHECK (estado IN (
        'rascunho', 'em_avaliacao', 'aprovada', 'rejeitada', 'em_execucao', 'concluida', 'falhou', 'cancelada'
    )),
    CONSTRAINT mudancas_janela_chk CHECK (inicio_planejado IS NULL OR fim_planejado IS NULL OR fim_planejado > inicio_planejado)
);

CREATE INDEX idx_mudancas_unidade ON mudancas (unidade_id, estado, created_at);
CREATE INDEX idx_mudancas_janela ON mudancas (inicio_planejado) WHERE estado IN ('aprovada', 'em_execucao');

CREATE TRIGGER trg_mudancas_updated
    BEFORE UPDATE ON mudancas
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Cada decisão sobre a mudança: quem, quando, qual e por quê.
CREATE TABLE mudancas_aprovacoes (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mudanca_id    UUID NOT NULL REFERENCES mudancas (id) ON DELETE CASCADE,
    aprovador_id  UUID REFERENCES users (id) ON DELETE SET NULL,
    decisao       TEXT NOT NULL,
    parecer       TEXT NOT NULL DEFAULT '',
    em            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT mudancas_aprovacoes_decisao_chk CHECK (decisao IN ('aprovada', 'rejeitada', 'automatica')),
    CONSTRAINT mudancas_aprovacoes_parecer_chk CHECK (char_length(parecer) <= 4000)
);

CREATE INDEX idx_mudancas_aprovacoes_mudanca ON mudancas_aprovacoes (mudanca_id, em);

CREATE TABLE mudancas_vinculos (
    mudanca_id     UUID NOT NULL REFERENCES mudancas (id) ON DELETE CASCADE,
    tipo           TEXT NOT NULL,
    alvo_id        UUID NOT NULL,
    rotulo         TEXT NOT NULL,
    vinculado_por  UUID REFERENCES users (id) ON DELETE SET NULL,
    vinculado_em   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (mudanca_id, tipo, alvo_id),
    CONSTRAINT mudancas_vinculos_tipo_chk CHECK (tipo IN ('problema', 'incidente', 'ativo'))
);

CREATE INDEX idx_mudancas_vinculos_alvo ON mudancas_vinculos (tipo, alvo_id);

-- +goose Down
DROP TABLE IF EXISTS mudancas_vinculos;
DROP TABLE IF EXISTS mudancas_aprovacoes;
DROP TABLE IF EXISTS mudancas;
