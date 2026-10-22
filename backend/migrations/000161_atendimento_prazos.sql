-- +goose Up
-- Fase 4: prazo em tempo útil. Cada unidade tem uma política (fuso,
-- turnos de expediente, matriz impacto x urgência -> prioridade e as metas
-- de resposta e solução de cada prioridade) e os feriados que valem para
-- ela. Unidade sem política própria usa a da unidade acima e, no topo, a
-- institucional (unidade_id nula). A política é um documento validado no
-- domínio e gravado inteiro: as três partes só fazem sentido juntas.
--
-- A solicitação guarda a prioridade e as metas do momento da
-- classificação: mudar a política depois não muda o compromisso já
-- assumido. O tempo consumido sai das passagens (000160) e não é gravado.

CREATE TABLE atendimento_politicas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unidade_id  UUID REFERENCES unidades (id) ON DELETE CASCADE,
    fuso        TEXT NOT NULL DEFAULT 'America/Sao_Paulo',
    turnos      JSONB NOT NULL DEFAULT '[]',
    matriz      JSONB NOT NULL DEFAULT '[]',
    metas       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_politicas_unidade_uq UNIQUE NULLS NOT DISTINCT (unidade_id)
);

CREATE TRIGGER trg_atendimento_politicas_updated
    BEFORE UPDATE ON atendimento_politicas
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Feriado e ponto facultativo não contam como tempo útil. unidade_id nula
-- = vale para todas; o da unidade vale também para as subunidades.
CREATE TABLE atendimento_feriados (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unidade_id  UUID REFERENCES unidades (id) ON DELETE CASCADE,
    data        DATE NOT NULL,
    nome        TEXT NOT NULL,
    tipo        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_feriados_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 160),
    CONSTRAINT atendimento_feriados_tipo_chk CHECK (tipo IN ('nacional', 'estadual', 'municipal', 'ponto_facultativo')),
    CONSTRAINT atendimento_feriados_dia_uq UNIQUE NULLS NOT DISTINCT (unidade_id, data)
);

CREATE INDEX idx_atendimento_feriados_data ON atendimento_feriados (data);

ALTER TABLE atendimento_solicitacoes
    ADD COLUMN impacto           SMALLINT NOT NULL DEFAULT 2,
    ADD COLUMN urgencia          SMALLINT NOT NULL DEFAULT 2,
    ADD COLUMN prioridade        TEXT,
    ADD COLUMN meta_resposta_min INTEGER,
    ADD COLUMN meta_solucao_min  INTEGER,
    ADD CONSTRAINT atendimento_solicitacoes_impacto_chk CHECK (impacto BETWEEN 1 AND 3),
    ADD CONSTRAINT atendimento_solicitacoes_urgencia_chk CHECK (urgencia BETWEEN 1 AND 3),
    ADD CONSTRAINT atendimento_solicitacoes_prioridade_chk CHECK (prioridade IN ('critica', 'alta', 'media', 'baixa')),
    ADD CONSTRAINT atendimento_solicitacoes_metas_chk CHECK (
        (prioridade IS NULL AND meta_resposta_min IS NULL AND meta_solucao_min IS NULL)
        OR (prioridade IS NOT NULL AND meta_resposta_min > 0 AND meta_solucao_min > 0)
    );

-- O alerta de prazo percorre só o que tem relógio correndo.
CREATE INDEX idx_atendimento_solicitacoes_prazo ON atendimento_solicitacoes (estado)
    WHERE prioridade IS NOT NULL AND estado IN ('aberto', 'em_atendimento');

-- +goose Down
DROP INDEX IF EXISTS idx_atendimento_solicitacoes_prazo;
ALTER TABLE atendimento_solicitacoes
    DROP CONSTRAINT IF EXISTS atendimento_solicitacoes_metas_chk,
    DROP CONSTRAINT IF EXISTS atendimento_solicitacoes_prioridade_chk,
    DROP CONSTRAINT IF EXISTS atendimento_solicitacoes_urgencia_chk,
    DROP CONSTRAINT IF EXISTS atendimento_solicitacoes_impacto_chk,
    DROP COLUMN IF EXISTS meta_solucao_min,
    DROP COLUMN IF EXISTS meta_resposta_min,
    DROP COLUMN IF EXISTS prioridade,
    DROP COLUMN IF EXISTS urgencia,
    DROP COLUMN IF EXISTS impacto;
DROP TABLE IF EXISTS atendimento_feriados;
DROP TABLE IF EXISTS atendimento_politicas;
