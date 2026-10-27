-- +goose Up
-- Experiência de quem pede (XLA, ADR 069).

-- Facilidade: "foi fácil conseguir o que você precisava?" (1 a 5),
-- opcional, ao lado da nota. As avaliações antigas ficam sem.
ALTER TABLE atendimento_avaliacoes ADD COLUMN facilidade SMALLINT CHECK (facilidade BETWEEN 1 AND 5);

-- As metas de experiência da organização: uma linha só. Meta nula é
-- medida sem meta.
CREATE TABLE atendimento_xla (
    id BOOLEAN PRIMARY KEY DEFAULT true CHECK (id),
    csat_min NUMERIC(2, 1) CHECK (csat_min BETWEEN 1 AND 5),
    facilidade_min NUMERIC(2, 1) CHECK (facilidade_min BETWEEN 1 AND 5),
    reabertura_max NUMERIC(4, 1) CHECK (reabertura_max BETWEEN 0 AND 100),
    idas_max NUMERIC(3, 1) CHECK (idas_max BETWEEN 0 AND 20),
    espera_max_horas INTEGER CHECK (espera_max_horas BETWEEN 1 AND 8760),
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS atendimento_xla;
ALTER TABLE atendimento_avaliacoes DROP COLUMN IF EXISTS facilidade;
