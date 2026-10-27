-- +goose Up
-- Ocorrências: incidente grave e página de status pública (ADR 066).

CREATE TABLE ocorrencias (
    id UUID PRIMARY KEY,
    titulo TEXT NOT NULL CHECK (length(btrim(titulo)) BETWEEN 1 AND 160),
    impacto TEXT NOT NULL CHECK (impacto IN ('degradado', 'parcial', 'fora')),
    estado TEXT NOT NULL CHECK (estado IN ('investigando', 'identificado', 'monitorando', 'resolvido')),
    servicos TEXT[] NOT NULL CHECK (cardinality(servicos) BETWEEN 1 AND 10),
    chamado_id UUID REFERENCES atendimento_solicitacoes(id) ON DELETE SET NULL,
    coordenador_id UUID REFERENCES users(id) ON DELETE SET NULL,
    iniciada_em TIMESTAMPTZ NOT NULL,
    resolvida_em TIMESTAMPTZ,
    revisao TEXT NOT NULL DEFAULT '',
    revisada_em TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL,
    CHECK ((estado = 'resolvido') = (resolvida_em IS NOT NULL)),
    CHECK (revisada_em IS NULL OR estado = 'resolvido')
);

-- A página lê as abertas e as resolvidas da última semana.
CREATE INDEX ocorrencias_abertas_idx ON ocorrencias (iniciada_em DESC) WHERE estado <> 'resolvido';
CREATE INDEX ocorrencias_resolvidas_idx ON ocorrencias (resolvida_em DESC) WHERE estado = 'resolvido';

CREATE TABLE ocorrencias_atualizacoes (
    id UUID PRIMARY KEY,
    ocorrencia_id UUID NOT NULL REFERENCES ocorrencias(id) ON DELETE CASCADE,
    estado TEXT NOT NULL CHECK (estado IN ('investigando', 'identificado', 'monitorando', 'resolvido')),
    impacto TEXT NOT NULL CHECK (impacto IN ('degradado', 'parcial', 'fora')),
    texto TEXT NOT NULL CHECK (length(btrim(texto)) BETWEEN 1 AND 4000),
    autor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX ocorrencias_atualizacoes_ocorrencia_idx ON ocorrencias_atualizacoes (ocorrencia_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS ocorrencias_atualizacoes;
DROP TABLE IF EXISTS ocorrencias;
