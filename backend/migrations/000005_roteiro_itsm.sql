-- +goose Up
-- Leva 2 e 3 do roteiro: campos da oferta, nota de risco, janela de
-- congelamento, relação entre ativos, contratos e a marca da resposta
-- que chegou por e-mail (para não gravar a mesma mensagem duas vezes).

ALTER TABLE atendimento_ofertas
    ADD COLUMN campos jsonb NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE atendimento_solicitacoes
    ADD COLUMN respostas jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE mudancas
    ADD COLUMN risco_score smallint;

ALTER TABLE mudancas
    ADD CONSTRAINT mudancas_risco_score_chk CHECK (risco_score IS NULL OR risco_score BETWEEN 5 AND 15);

CREATE TABLE mudanca_congelamentos (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unidade_id UUID REFERENCES unidades (id) ON DELETE CASCADE,
    inicio     TIMESTAMPTZ NOT NULL,
    fim        TIMESTAMPTZ NOT NULL,
    motivo     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT mudanca_congelamentos_janela_chk CHECK (fim > inicio),
    CONSTRAINT mudanca_congelamentos_motivo_chk CHECK (char_length(motivo) <= 4000)
);

CREATE INDEX mudanca_congelamentos_janela_idx ON mudanca_congelamentos (inicio, fim);

CREATE TABLE ativo_relacoes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    origem_id  UUID NOT NULL REFERENCES ativos (id) ON DELETE CASCADE,
    destino_id UUID NOT NULL REFERENCES ativos (id) ON DELETE CASCADE,
    tipo       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ativo_relacoes_tipo_chk CHECK (tipo IN ('depende', 'composto', 'conectado')),
    CONSTRAINT ativo_relacoes_distintos_chk CHECK (origem_id <> destino_id),
    CONSTRAINT ativo_relacoes_uq UNIQUE (origem_id, destino_id, tipo)
);

CREATE TABLE contratos_fornecedores (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome       TEXT NOT NULL,
    email      TEXT NOT NULL DEFAULT '',
    telefone   TEXT NOT NULL DEFAULT '',
    unidade_id UUID REFERENCES unidades (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT contratos_fornecedores_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 160)
);

CREATE TABLE contratos_contratos (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero         TEXT NOT NULL DEFAULT '',
    titulo         TEXT NOT NULL,
    fornecedor_id  UUID NOT NULL REFERENCES contratos_fornecedores (id),
    inicio         DATE NOT NULL,
    fim            DATE NOT NULL,
    valor_centavos BIGINT NOT NULL DEFAULT 0,
    unidade_id     UUID REFERENCES unidades (id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT contratos_contratos_titulo_chk CHECK (char_length(btrim(titulo)) BETWEEN 1 AND 160),
    CONSTRAINT contratos_contratos_janela_chk CHECK (fim >= inicio),
    CONSTRAINT contratos_contratos_valor_chk CHECK (valor_centavos >= 0)
);

CREATE TABLE contratos_licencas (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome          TEXT NOT NULL,
    fornecedor_id UUID NOT NULL REFERENCES contratos_fornecedores (id),
    fim           DATE,
    quantidade    INT NOT NULL DEFAULT 1,
    unidade_id    UUID REFERENCES unidades (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT contratos_licencas_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 160),
    CONSTRAINT contratos_licencas_qtd_chk CHECK (quantidade >= 1)
);

CREATE INDEX contratos_contratos_fim_idx ON contratos_contratos (fim);
CREATE INDEX contratos_licencas_fim_idx ON contratos_licencas (fim);

CREATE TABLE correio_recebidos (
    message_id     TEXT PRIMARY KEY,
    solicitacao_id UUID NOT NULL REFERENCES atendimento_solicitacoes (id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT correio_recebidos_id_chk CHECK (char_length(message_id) BETWEEN 1 AND 200)
);

-- +goose Down
DROP TABLE IF EXISTS correio_recebidos;
DROP TABLE IF EXISTS contratos_licencas;
DROP TABLE IF EXISTS contratos_contratos;
DROP TABLE IF EXISTS contratos_fornecedores;
DROP TABLE IF EXISTS ativo_relacoes;
DROP TABLE IF EXISTS mudanca_congelamentos;
ALTER TABLE mudancas DROP CONSTRAINT IF EXISTS mudancas_risco_score_chk;
ALTER TABLE mudancas DROP COLUMN IF EXISTS risco_score;
ALTER TABLE atendimento_solicitacoes DROP COLUMN IF EXISTS respostas;
ALTER TABLE atendimento_ofertas DROP COLUMN IF EXISTS campos;
