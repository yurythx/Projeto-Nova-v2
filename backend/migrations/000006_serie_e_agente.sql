-- +goose Up
-- Série diária dos indicadores (o dia de hoje é regravado; os anteriores
-- ficam) e a origem do ativo, para o agente não reescrever o cadastro manual.

CREATE TABLE atendimento_indicadores_dia (
    dia             DATE NOT NULL,
    unidade_id      UUID REFERENCES unidades (id) ON DELETE CASCADE,
    mttr_horas      DOUBLE PRECISION,
    mtbf_horas      DOUBLE PRECISION,
    sla_percentual  DOUBLE PRECISION,
    fcr_percentual  DOUBLE PRECISION,
    csat_media      DOUBLE PRECISION,
    soma_mttr_min   INT NOT NULL DEFAULT 0,
    mttr_n          INT NOT NULL DEFAULT 0,
    sla_cumprido    INT NOT NULL DEFAULT 0,
    sla_n           INT NOT NULL DEFAULT 0,
    fcr_n           INT NOT NULL DEFAULT 0,
    concluidas      INT NOT NULL DEFAULT 0,
    incidentes      INT NOT NULL DEFAULT 0,
    csat_soma       INT NOT NULL DEFAULT 0,
    csat_n          INT NOT NULL DEFAULT 0,
    truncado        BOOLEAN NOT NULL DEFAULT false,
    atualizado_em   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT atendimento_indicadores_dia_uq UNIQUE NULLS NOT DISTINCT (dia, unidade_id)
);

CREATE INDEX atendimento_indicadores_dia_dia_idx ON atendimento_indicadores_dia (dia);

ALTER TABLE ativos
    ADD COLUMN origem TEXT NOT NULL DEFAULT 'manual';

ALTER TABLE ativos
    ADD CONSTRAINT ativos_origem_chk CHECK (origem IN ('manual', 'agente'));

-- +goose Down
ALTER TABLE ativos DROP CONSTRAINT IF EXISTS ativos_origem_chk;
ALTER TABLE ativos DROP COLUMN IF EXISTS origem;
DROP TABLE IF EXISTS atendimento_indicadores_dia;
