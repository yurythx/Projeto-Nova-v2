-- +goose Up
-- Fase 5: atendimento externo. A visita nasce de uma solicitação: check-in
-- ao chegar, evidências (foto e assinatura) e check-out ao sair.
--
-- Localização (LGPD): coletada só no check-in e no check-out, nunca em
-- segundo plano. Base legal: execução de política pública (art. 7º, III,
-- e art. 23) — comprovar a prestação do serviço. As coordenadas são
-- apagadas em localizacao_expira_em (um ano); o horário, as evidências e
-- o fato do check-in ficam. A auditoria e os eventos não levam
-- coordenadas: a trilha é imutável e não poderia ser apagada.
--
-- O id e os horários de captura vêm do aparelho: a visita registrada sem
-- rede sobe depois sem duplicar (mesmo id) e com o horário real.

CREATE TABLE atendimento_visitas (
    id                    UUID PRIMARY KEY,
    solicitacao_id        UUID NOT NULL REFERENCES atendimento_solicitacoes (id) ON DELETE CASCADE,
    tecnico_id            UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    estado                TEXT NOT NULL DEFAULT 'em_andamento',
    local                 TEXT NOT NULL DEFAULT '',
    base_legal            TEXT NOT NULL,
    -- Destino opcional: com ele, o check-in mede a distância.
    destino_lat           DOUBLE PRECISION,
    destino_lng           DOUBLE PRECISION,
    destino_raio_m        INTEGER,
    inicio_em             TIMESTAMPTZ NOT NULL,
    inicio_lat            DOUBLE PRECISION,
    inicio_lng            DOUBLE PRECISION,
    inicio_precisao_m     DOUBLE PRECISION,
    distancia_m           DOUBLE PRECISION,
    fora_da_area          BOOLEAN NOT NULL DEFAULT false,
    fim_em                TIMESTAMPTZ,
    fim_lat               DOUBLE PRECISION,
    fim_lng               DOUBLE PRECISION,
    fim_precisao_m        DOUBLE PRECISION,
    assinante_nome        TEXT NOT NULL DEFAULT '',
    observacao            TEXT NOT NULL DEFAULT '',
    localizacao_expira_em TIMESTAMPTZ NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_visitas_estado_chk CHECK (estado IN ('em_andamento', 'concluida', 'cancelada')),
    CONSTRAINT atendimento_visitas_local_chk CHECK (char_length(local) <= 300),
    CONSTRAINT atendimento_visitas_obs_chk CHECK (char_length(observacao) <= 4000),
    CONSTRAINT atendimento_visitas_assinante_chk CHECK (char_length(assinante_nome) <= 160),
    CONSTRAINT atendimento_visitas_destino_chk CHECK (
        (destino_lat IS NULL AND destino_lng IS NULL AND destino_raio_m IS NULL)
        OR (destino_lat BETWEEN -90 AND 90 AND destino_lng BETWEEN -180 AND 180 AND destino_raio_m BETWEEN 10 AND 50000)
    ),
    CONSTRAINT atendimento_visitas_fim_chk CHECK ((estado = 'em_andamento') = (fim_em IS NULL))
);

-- Um técnico tem no máximo uma visita aberta por solicitação.
CREATE UNIQUE INDEX uq_atendimento_visitas_aberta ON atendimento_visitas (solicitacao_id, tecnico_id)
    WHERE estado = 'em_andamento';
CREATE INDEX idx_atendimento_visitas_solicitacao ON atendimento_visitas (solicitacao_id, inicio_em);
CREATE INDEX idx_atendimento_visitas_andamento ON atendimento_visitas (estado) WHERE estado = 'em_andamento';
CREATE INDEX idx_atendimento_visitas_expira ON atendimento_visitas (localizacao_expira_em)
    WHERE inicio_lat IS NOT NULL OR fim_lat IS NOT NULL OR destino_lat IS NOT NULL;

CREATE TRIGGER trg_atendimento_visitas_updated
    BEFORE UPDATE ON atendimento_visitas
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Foto ou assinatura já enviada ao armazenamento e conferida.
CREATE TABLE atendimento_evidencias (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visita_id     UUID NOT NULL REFERENCES atendimento_visitas (id) ON DELETE CASCADE,
    tipo          TEXT NOT NULL,
    object_key    TEXT NOT NULL UNIQUE,
    content_type  TEXT NOT NULL,
    tamanho       BIGINT NOT NULL,
    autor_id      UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_evidencias_tipo_chk CHECK (tipo IN ('foto', 'assinatura'))
);

CREATE INDEX idx_atendimento_evidencias_visita ON atendimento_evidencias (visita_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS atendimento_evidencias;
DROP TABLE IF EXISTS atendimento_visitas;
