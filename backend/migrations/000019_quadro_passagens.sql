-- +goose Up
-- Passagem a mais, só dentro de um quadro. O fluxo de fábrica continua no
-- código de cada módulo. A coluna nova não copia para os outros quadros.

CREATE TABLE quadro_passagens (
    quadro TEXT NOT NULL,
    de     TEXT NOT NULL,
    para   TEXT NOT NULL,
    PRIMARY KEY (quadro, de, para),
    CONSTRAINT quadro_passagens_distintas_chk CHECK (de <> para),
    CONSTRAINT quadro_passagens_de_fk FOREIGN KEY (quadro, de) REFERENCES quadro_colunas (quadro, chave),
    CONSTRAINT quadro_passagens_para_fk FOREIGN KEY (quadro, para) REFERENCES quadro_colunas (quadro, chave)
);

-- A chave nova cabe no estado. As de fábrica continuam na lista.
ALTER TABLE problemas DROP CONSTRAINT problemas_estado_chk;
ALTER TABLE problemas ADD CONSTRAINT problemas_estado_chk CHECK (
    estado = ANY (ARRAY['registrado', 'em_analise', 'erro_conhecido', 'resolvido', 'encerrado'])
    OR estado ~ '^c[0-9a-f]{8}$'
);

ALTER TABLE mudancas DROP CONSTRAINT mudancas_estado_chk;
ALTER TABLE mudancas ADD CONSTRAINT mudancas_estado_chk CHECK (
    estado = ANY (ARRAY['rascunho', 'em_avaliacao', 'aprovada', 'rejeitada', 'em_execucao', 'concluida', 'falhou', 'cancelada'])
    OR estado ~ '^c[0-9a-f]{8}$'
);

ALTER TABLE ativos DROP CONSTRAINT ativos_estado_chk;
ALTER TABLE ativos ADD CONSTRAINT ativos_estado_chk CHECK (
    estado = ANY (ARRAY['planejado', 'em_uso', 'manutencao', 'desativado'])
    OR estado ~ '^c[0-9a-f]{8}$'
);

ALTER TABLE atendimento_solicitacoes DROP CONSTRAINT atendimento_solicitacoes_tipo_estado_chk;
ALTER TABLE atendimento_solicitacoes ADD CONSTRAINT atendimento_solicitacoes_tipo_estado_chk CHECK (
    (
        tipo = 'incidente' AND estado = ANY (ARRAY['aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro', 'resolvido', 'encerrado', 'cancelado'])
    ) OR (
        tipo = 'requisicao' AND estado = ANY (ARRAY['aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro', 'atendido', 'encerrado', 'cancelado'])
    ) OR (
        tipo = ANY (ARRAY['incidente', 'requisicao']) AND estado ~ '^c[0-9a-f]{8}$'
    )
);

-- +goose Down
ALTER TABLE atendimento_solicitacoes DROP CONSTRAINT atendimento_solicitacoes_tipo_estado_chk;
ALTER TABLE atendimento_solicitacoes ADD CONSTRAINT atendimento_solicitacoes_tipo_estado_chk CHECK (
    (
        (tipo = 'incidente') AND (estado = ANY (ARRAY['aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro', 'resolvido', 'encerrado', 'cancelado']))
    ) OR (
        (tipo = 'requisicao') AND (estado = ANY (ARRAY['aberto', 'em_atendimento', 'aguardando_servidor', 'aguardando_terceiro', 'atendido', 'encerrado', 'cancelado']))
    )
);

ALTER TABLE ativos DROP CONSTRAINT ativos_estado_chk;
ALTER TABLE ativos ADD CONSTRAINT ativos_estado_chk CHECK (
    estado = ANY (ARRAY['planejado', 'em_uso', 'manutencao', 'desativado'])
);

ALTER TABLE mudancas DROP CONSTRAINT mudancas_estado_chk;
ALTER TABLE mudancas ADD CONSTRAINT mudancas_estado_chk CHECK (
    estado = ANY (ARRAY['rascunho', 'em_avaliacao', 'aprovada', 'rejeitada', 'em_execucao', 'concluida', 'falhou', 'cancelada'])
);

ALTER TABLE problemas DROP CONSTRAINT problemas_estado_chk;
ALTER TABLE problemas ADD CONSTRAINT problemas_estado_chk CHECK (
    estado = ANY (ARRAY['registrado', 'em_analise', 'erro_conhecido', 'resolvido', 'encerrado'])
);

DROP TABLE IF EXISTS quadro_passagens;
