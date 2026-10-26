-- +goose Up
-- Triagem curta, resposta pronta, chamado mensal e a vista do artigo
-- que não virou pedido. Sem motor de regras e sem amostra inventada.

CREATE TABLE atendimento_regras (
    id uuid PRIMARY KEY,
    unidade_id uuid REFERENCES unidades (id) ON DELETE RESTRICT,
    ordem integer NOT NULL DEFAULT 0,
    nome text NOT NULL,
    oferta_id uuid REFERENCES atendimento_ofertas (id) ON DELETE CASCADE,
    palavra text NOT NULL DEFAULT '',
    unidade_solicitante_id uuid REFERENCES unidades (id) ON DELETE RESTRICT,
    prioridade text NOT NULL DEFAULT '',
    equipe_id uuid REFERENCES atendimento_equipes (id) ON DELETE SET NULL,
    ativa boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT atendimento_regras_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 160),
    CONSTRAINT atendimento_regras_ordem_chk CHECK (ordem BETWEEN 0 AND 999),
    CONSTRAINT atendimento_regras_palavra_chk CHECK (char_length(palavra) <= 80),
    CONSTRAINT atendimento_regras_prioridade_chk CHECK (prioridade IN ('', 'critica', 'alta', 'media', 'baixa')),
    CONSTRAINT atendimento_regras_acao_chk CHECK (prioridade <> '' OR equipe_id IS NOT NULL)
);

CREATE INDEX atendimento_regras_unidade_idx ON atendimento_regras (unidade_id, ordem, id);

CREATE TABLE atendimento_respostas (
    oferta_id uuid PRIMARY KEY REFERENCES atendimento_ofertas (id) ON DELETE CASCADE,
    publico text NOT NULL DEFAULT '',
    interna text NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL,
    CONSTRAINT atendimento_respostas_texto_chk CHECK (char_length(publico) <= 4000 AND char_length(interna) <= 4000)
);

CREATE TABLE atendimento_recorrencias (
    id uuid PRIMARY KEY,
    unidade_id uuid REFERENCES unidades (id) ON DELETE RESTRICT,
    oferta_id uuid NOT NULL REFERENCES atendimento_ofertas (id) ON DELETE CASCADE,
    titulo text NOT NULL,
    descricao text NOT NULL DEFAULT '',
    dia smallint NOT NULL,
    criado_por uuid NOT NULL REFERENCES users (id),
    ativa boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    CONSTRAINT atendimento_recorrencias_titulo_chk CHECK (char_length(btrim(titulo)) BETWEEN 1 AND 160),
    CONSTRAINT atendimento_recorrencias_descricao_chk CHECK (char_length(descricao) <= 4000),
    CONSTRAINT atendimento_recorrencias_dia_chk CHECK (dia BETWEEN 1 AND 28)
);

CREATE TABLE atendimento_recorrencia_aberturas (
    recorrencia_id uuid NOT NULL REFERENCES atendimento_recorrencias (id) ON DELETE CASCADE,
    ano integer NOT NULL,
    mes integer NOT NULL,
    solicitacao_id uuid,
    tentado_em timestamptz NOT NULL,
    PRIMARY KEY (recorrencia_id, ano, mes)
);

CREATE TABLE atendimento_deflexoes (
    usuario_id uuid NOT NULL REFERENCES users (id),
    artigo_id text NOT NULL,
    dia date NOT NULL,
    unidade_id uuid REFERENCES unidades (id) ON DELETE SET NULL,
    resolveu boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (usuario_id, artigo_id, dia),
    CONSTRAINT atendimento_deflexoes_artigo_chk CHECK (char_length(artigo_id) BETWEEN 1 AND 80)
);

CREATE INDEX atendimento_deflexoes_dia_idx ON atendimento_deflexoes (dia);

-- +goose Down
DROP TABLE IF EXISTS atendimento_deflexoes;
DROP TABLE IF EXISTS atendimento_recorrencia_aberturas;
DROP TABLE IF EXISTS atendimento_recorrencias;
DROP TABLE IF EXISTS atendimento_respostas;
DROP TABLE IF EXISTS atendimento_regras;
