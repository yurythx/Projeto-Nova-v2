-- +goose Up
-- Tarefas por equipe na requisição (ADR 068): admissão e desligamento.

-- O modelo da oferta: cada linha vira uma tarefa quando o pedido chega à fila.
CREATE TABLE atendimento_oferta_tarefas (
    oferta_id UUID NOT NULL REFERENCES atendimento_ofertas(id) ON DELETE CASCADE,
    ordem SMALLINT NOT NULL CHECK (ordem BETWEEN 0 AND 19),
    titulo TEXT NOT NULL CHECK (length(btrim(titulo)) BETWEEN 1 AND 120),
    equipe_id UUID NOT NULL REFERENCES atendimento_equipes(id) ON DELETE CASCADE,
    PRIMARY KEY (oferta_id, ordem)
);

-- As tarefas de cada pedido. Mudar o modelo não mexe nas já criadas. A
-- equipe guarda o nome: se ela for apagada, a tarefa fica, sem dona, e
-- quem atende o chamado a dispensa.
CREATE TABLE atendimento_tarefas (
    id UUID PRIMARY KEY,
    solicitacao_id UUID NOT NULL REFERENCES atendimento_solicitacoes(id) ON DELETE CASCADE,
    ordem SMALLINT NOT NULL,
    titulo TEXT NOT NULL,
    equipe_id UUID REFERENCES atendimento_equipes(id) ON DELETE SET NULL,
    equipe_nome TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT 'pendente' CHECK (estado IN ('pendente', 'concluida', 'dispensada')),
    observacao TEXT NOT NULL DEFAULT '' CHECK (length(observacao) <= 1000),
    feita_por UUID REFERENCES users(id) ON DELETE SET NULL,
    feita_em TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (solicitacao_id, ordem),
    CHECK ((estado = 'pendente') = (feita_em IS NULL)),
    CHECK (estado <> 'dispensada' OR length(btrim(observacao)) > 0)
);

-- +goose Down
DROP TABLE IF EXISTS atendimento_tarefas;
DROP TABLE IF EXISTS atendimento_oferta_tarefas;
