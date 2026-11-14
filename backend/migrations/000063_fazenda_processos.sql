-- +goose Up
-- Protocolo da Fazenda (ADR 084). O número é a sequência do ano.
CREATE TABLE fazenda_sequencia (
    ano INT PRIMARY KEY,
    ultimo INT NOT NULL
);

CREATE TABLE fazenda_processos (
    id UUID PRIMARY KEY,
    numero TEXT NOT NULL UNIQUE,
    ano INT NOT NULL,
    seq INT NOT NULL,
    programa TEXT NOT NULL,
    entrada TEXT NOT NULL,
    titular TEXT NOT NULL,
    situacao TEXT NOT NULL,
    recibo_em TIMESTAMPTZ NOT NULL,
    codigo TEXT NOT NULL UNIQUE,
    certidao TEXT NOT NULL DEFAULT '',
    aliquota BIGINT NOT NULL DEFAULT 0,
    execucao TEXT NOT NULL DEFAULT '',
    exigencia_aberta TIMESTAMPTZ,
    exigencia_prazo TIMESTAMPTZ,
    exigencia_prorrogada BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (ano, seq)
);

CREATE INDEX fazenda_processos_titular_idx ON fazenda_processos (titular, recibo_em);

-- +goose Down
DROP TABLE IF EXISTS fazenda_processos;
DROP TABLE IF EXISTS fazenda_sequencia;
