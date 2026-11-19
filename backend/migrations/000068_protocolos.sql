-- +goose Up
-- Plug-in protocolos (ADR 085): carta de serviços, formulário e pedido.

CREATE TABLE protocolos_formularios (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unidade_id  UUID NOT NULL REFERENCES unidades (id),
    titulo      TEXT NOT NULL,
    descricao   TEXT NOT NULL DEFAULT '',
    instrucao   TEXT NOT NULL DEFAULT '',
    campos      JSONB NOT NULL DEFAULT '[]',
    ativo       BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE protocolos_servicos (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unidade_id               UUID NOT NULL REFERENCES unidades (id),
    formulario_id            UUID NOT NULL REFERENCES protocolos_formularios (id),
    nome                     TEXT NOT NULL,
    descricao                TEXT NOT NULL DEFAULT '',
    categoria                TEXT NOT NULL DEFAULT '',
    prazo_dias               INT NOT NULL DEFAULT 0,
    taxa_centavos            INT NOT NULL DEFAULT 0,
    destaque                 BOOLEAN NOT NULL DEFAULT false,
    ativo                    BOOLEAN NOT NULL DEFAULT true,
    aprova_sozinho           BOOLEAN NOT NULL DEFAULT false,
    exige_agendamento        BOOLEAN NOT NULL DEFAULT false,
    so_responsavel_tecnico   BOOLEAN NOT NULL DEFAULT false,
    impedir_cpf_aberto       BOOLEAN NOT NULL DEFAULT false,
    campo_anonimo            TEXT NOT NULL DEFAULT '',
    workflow                 JSONB NOT NULL DEFAULT '{"ligado":false,"etapas":[]}',
    agenda                   JSONB NOT NULL DEFAULT '{}',
    requisitos               JSONB NOT NULL DEFAULT '[]',
    documentos               JSONB NOT NULL DEFAULT '[]',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE protocolos_pedidos (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    numero           TEXT NOT NULL,
    servico_id       UUID NOT NULL REFERENCES protocolos_servicos (id),
    formulario_id    UUID NOT NULL,
    unidade_id       UUID NOT NULL REFERENCES unidades (id),
    departamento_id  UUID,
    nome             TEXT NOT NULL,
    cpf              TEXT NOT NULL,
    email            TEXT NOT NULL DEFAULT '',
    telefone         TEXT NOT NULL DEFAULT '',
    respostas        JSONB NOT NULL DEFAULT '{}',
    campos           JSONB NOT NULL DEFAULT '[]',
    status           TEXT NOT NULL,
    etapa_id         TEXT NOT NULL DEFAULT '',
    prioridade       TEXT NOT NULL DEFAULT 'normal',
    prazo_em         TIMESTAMPTZ,
    decisao          TEXT NOT NULL DEFAULT '',
    justificativa    TEXT NOT NULL DEFAULT '',
    complemento      TEXT NOT NULL DEFAULT '',
    usuario_id       UUID,
    responsavel_id   UUID,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT protocolos_pedidos_numero_key UNIQUE (numero)
);

CREATE INDEX idx_protocolos_pedidos_unidade ON protocolos_pedidos (unidade_id, created_at DESC);
CREATE INDEX idx_protocolos_pedidos_cpf ON protocolos_pedidos (servico_id, cpf);

CREATE TABLE protocolos_historico (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pedido_id   UUID NOT NULL REFERENCES protocolos_pedidos (id) ON DELETE CASCADE,
    acao        TEXT NOT NULL,
    descricao   TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT '',
    usuario_id  UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE protocolos_responsaveis (
    id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cpf    TEXT NOT NULL,
    nome   TEXT NOT NULL,
    ativo  BOOLEAN NOT NULL DEFAULT true,
    CONSTRAINT protocolos_responsaveis_cpf_key UNIQUE (cpf)
);

CREATE TABLE protocolos_agendamentos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pedido_id   UUID NOT NULL REFERENCES protocolos_pedidos (id) ON DELETE CASCADE,
    servico_id  UUID NOT NULL,
    data        DATE NOT NULL,
    hora        TEXT NOT NULL,
    status      TEXT NOT NULL,
    CONSTRAINT protocolos_agendamentos_pedido_key UNIQUE (pedido_id)
);

CREATE TABLE protocolos_faltas (
    cpf         TEXT PRIMARY KEY,
    quantidade  INT NOT NULL DEFAULT 0,
    ultima      TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS protocolos_faltas;
DROP TABLE IF EXISTS protocolos_agendamentos;
DROP TABLE IF EXISTS protocolos_responsaveis;
DROP TABLE IF EXISTS protocolos_historico;
DROP TABLE IF EXISTS protocolos_pedidos;
DROP TABLE IF EXISTS protocolos_servicos;
DROP TABLE IF EXISTS protocolos_formularios;
