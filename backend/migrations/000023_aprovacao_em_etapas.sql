-- +goose Up
-- Aprovação em etapas na requisição (ADR 068).

-- A requisição de oferta com etapas abre esperando aprovação. etapa_atual
-- só existe nesse estado.
ALTER TABLE atendimento_solicitacoes DROP CONSTRAINT atendimento_solicitacoes_tipo_estado_chk;
ALTER TABLE atendimento_solicitacoes ADD CONSTRAINT atendimento_solicitacoes_tipo_estado_chk CHECK (
    (tipo = 'incidente' AND estado = ANY (ARRAY['aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro', 'resolvido', 'encerrado', 'cancelado']))
    OR (tipo = 'requisicao' AND estado = ANY (ARRAY['aguardando_aprovacao', 'aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro', 'atendido', 'encerrado', 'cancelado']))
);
ALTER TABLE atendimento_solicitacoes ADD COLUMN etapa_atual SMALLINT CHECK (etapa_atual >= 0);
ALTER TABLE atendimento_solicitacoes ADD CONSTRAINT atendimento_solicitacoes_etapa_chk
    CHECK ((estado = 'aguardando_aprovacao') = (etapa_atual IS NOT NULL));

-- As etapas de cada oferta, em ordem; basta um aprovador por etapa.
CREATE TABLE atendimento_oferta_etapas (
    oferta_id UUID NOT NULL REFERENCES atendimento_ofertas(id) ON DELETE CASCADE,
    ordem SMALLINT NOT NULL CHECK (ordem BETWEEN 0 AND 4),
    nome TEXT NOT NULL CHECK (length(btrim(nome)) BETWEEN 1 AND 80),
    aprovadores UUID[] NOT NULL CHECK (cardinality(aprovadores) BETWEEN 1 AND 20),
    PRIMARY KEY (oferta_id, ordem)
);

-- Cada decisão, para quem pediu e para a auditoria.
CREATE TABLE atendimento_aprovacoes (
    id UUID PRIMARY KEY,
    solicitacao_id UUID NOT NULL REFERENCES atendimento_solicitacoes(id) ON DELETE CASCADE,
    etapa SMALLINT NOT NULL,
    etapa_nome TEXT NOT NULL,
    aprovador_id UUID REFERENCES users(id) ON DELETE SET NULL,
    decisao TEXT NOT NULL CHECK (decisao IN ('aprovada', 'recusada')),
    motivo TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    CHECK (decisao = 'aprovada' OR length(btrim(motivo)) > 0)
);

CREATE INDEX atendimento_aprovacoes_solicitacao_idx ON atendimento_aprovacoes (solicitacao_id, created_at);

-- A coluna nova vem antes das abertas no quadro da fila. A ordem ativa é
-- única por quadro: desloca em dois passos para não colidir, e mantém a
-- ordem e os nomes que a organização escolheu (ADR 061).
UPDATE quadro_colunas SET ordem = ordem + 1000 WHERE quadro = 'fila';
UPDATE quadro_colunas SET ordem = ordem - 999 WHERE quadro = 'fila';
INSERT INTO quadro_colunas (quadro, chave, ordem, nome, tom, entrada)
VALUES ('fila', 'aguardando_aprovacao', 0, 'Aguardando aprovação', 'warning', false)
ON CONFLICT (quadro, chave) DO NOTHING;

-- +goose Down
DELETE FROM quadro_colunas WHERE quadro = 'fila' AND chave = 'aguardando_aprovacao';
UPDATE quadro_colunas SET ordem = ordem + 1000 WHERE quadro = 'fila';
UPDATE quadro_colunas SET ordem = ordem - 1001 WHERE quadro = 'fila';

DROP TABLE IF EXISTS atendimento_aprovacoes;
DROP TABLE IF EXISTS atendimento_oferta_etapas;

-- Sem o estado, o pedido que esperava aprovação é cancelado.
UPDATE atendimento_solicitacoes SET estado = 'cancelado', encerrada_em = now(), etapa_atual = NULL
    WHERE estado = 'aguardando_aprovacao';
ALTER TABLE atendimento_solicitacoes DROP CONSTRAINT atendimento_solicitacoes_etapa_chk;
ALTER TABLE atendimento_solicitacoes DROP COLUMN etapa_atual;
ALTER TABLE atendimento_solicitacoes DROP CONSTRAINT atendimento_solicitacoes_tipo_estado_chk;
ALTER TABLE atendimento_solicitacoes ADD CONSTRAINT atendimento_solicitacoes_tipo_estado_chk CHECK (
    (tipo = 'incidente' AND estado = ANY (ARRAY['aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro', 'resolvido', 'encerrado', 'cancelado']))
    OR (tipo = 'requisicao' AND estado = ANY (ARRAY['aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro', 'atendido', 'encerrado', 'cancelado']))
);
