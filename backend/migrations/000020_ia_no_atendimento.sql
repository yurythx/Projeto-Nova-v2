-- +goose Up
-- Sugestões da IA a quem atende (ADR 065). A IA não muda o chamado: a
-- linha guarda o que ela sugeriu e o que a pessoa decidiu.

CREATE TABLE atendimento_ia_sugestoes (
    id UUID PRIMARY KEY,
    solicitacao_id UUID NOT NULL REFERENCES atendimento_solicitacoes(id) ON DELETE CASCADE,
    tipo TEXT NOT NULL CHECK (tipo IN ('classificacao', 'resumo')),
    conexao TEXT NOT NULL,
    conteudo JSONB NOT NULL,
    pedida_por UUID REFERENCES users(id) ON DELETE SET NULL,
    decisao TEXT CHECK (decisao IN ('aceita', 'recusada')),
    decidida_por UUID REFERENCES users(id) ON DELETE SET NULL,
    decidida_em TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((decisao IS NULL) = (decidida_em IS NULL))
);

CREATE INDEX atendimento_ia_sugestoes_solicitacao_idx ON atendimento_ia_sugestoes (solicitacao_id, created_at DESC);
CREATE INDEX atendimento_ia_sugestoes_created_idx ON atendimento_ia_sugestoes (created_at);

-- A função de IA do atendimento ganha principal e reserva próprios na tela.
ALTER TABLE ia_uso DROP CONSTRAINT ia_uso_funcao_check;
ALTER TABLE ia_uso ADD CONSTRAINT ia_uso_funcao_check
    CHECK (funcao IN ('plataforma.ia', 'atendimento.assistente'));

-- +goose Down
DELETE FROM ia_uso WHERE funcao = 'atendimento.assistente';
ALTER TABLE ia_uso DROP CONSTRAINT ia_uso_funcao_check;
ALTER TABLE ia_uso ADD CONSTRAINT ia_uso_funcao_check CHECK (funcao = 'plataforma.ia');

DROP TABLE IF EXISTS atendimento_ia_sugestoes;
