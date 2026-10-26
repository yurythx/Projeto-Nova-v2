-- +goose Up
-- Catálogo das colunas dos quadros. A chave é o estado do registro;
-- nome, cor e ordem são da organização. A coluna não guarda o card.

CREATE TABLE quadro_colunas (
    quadro     TEXT NOT NULL,
    chave      TEXT NOT NULL,
    ordem      INT  NOT NULL,
    nome       TEXT NOT NULL,
    cor        TEXT NOT NULL DEFAULT '',
    tom        TEXT NOT NULL,
    entrada    BOOLEAN NOT NULL DEFAULT false,
    ativa      BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (quadro, chave),
    CONSTRAINT quadro_colunas_quadro_chk CHECK (quadro IN ('problemas', 'mudancas', 'projetos', 'tarefas', 'ativos', 'fila')),
    CONSTRAINT quadro_colunas_nome_chk CHECK (char_length(btrim(nome)) BETWEEN 1 AND 80),
    CONSTRAINT quadro_colunas_cor_chk CHECK (cor = '' OR cor ~ '^#[0-9a-fA-F]{6}$'),
    CONSTRAINT quadro_colunas_ordem_chk CHECK (ordem >= 0),
    CONSTRAINT quadro_colunas_tom_chk CHECK (tom IN ('neutral', 'info', 'warning', 'success', 'danger'))
);

CREATE UNIQUE INDEX quadro_colunas_ordem_ativa ON quadro_colunas (quadro, ordem) WHERE ativa;

INSERT INTO quadro_colunas (quadro, chave, ordem, nome, tom, entrada) VALUES
    ('problemas', 'registrado', 0, 'Registrados', 'info', true),
    ('problemas', 'em_analise', 1, 'Em análise', 'info', false),
    ('problemas', 'erro_conhecido', 2, 'Erros conhecidos', 'warning', false),
    ('problemas', 'resolvido', 3, 'Resolvidos', 'success', false),
    ('problemas', 'encerrado', 4, 'Encerrados', 'neutral', false),
    ('mudancas', 'rascunho', 0, 'Rascunho', 'neutral', true),
    ('mudancas', 'em_avaliacao', 1, 'Em avaliação', 'info', false),
    ('mudancas', 'aprovada', 2, 'Aprovadas', 'success', false),
    ('mudancas', 'em_execucao', 3, 'Em execução', 'info', false),
    ('mudancas', 'concluida', 4, 'Concluídas', 'success', false),
    ('mudancas', 'fechada', 5, 'Rejeitada / Falhou', 'danger', false),
    ('projetos', 'planejado', 0, 'Planejados', 'info', true),
    ('projetos', 'em_andamento', 1, 'Em andamento', 'warning', false),
    ('projetos', 'suspenso', 2, 'Suspensos', 'danger', false),
    ('projetos', 'concluido', 3, 'Concluídos', 'success', false),
    ('projetos', 'cancelado', 4, 'Cancelados', 'neutral', false),
    ('tarefas', 'a_fazer', 0, 'A fazer', 'neutral', true),
    ('tarefas', 'em_andamento', 1, 'Em andamento', 'info', false),
    ('tarefas', 'concluida', 2, 'Concluída', 'success', false),
    ('tarefas', 'bloqueada', 3, 'Bloqueada', 'danger', false),
    ('ativos', 'planejado', 0, 'Planejado', 'info', true),
    ('ativos', 'em_uso', 1, 'Em uso', 'success', false),
    ('ativos', 'manutencao', 2, 'Em manutenção', 'warning', false),
    ('ativos', 'desativado', 3, 'Desativado', 'neutral', false),
    ('fila', 'aberto', 0, 'Abertas', 'info', true),
    ('fila', 'em_atendimento', 1, 'Em atendimento', 'info', false),
    ('fila', 'aguardando_servidor', 2, 'Aguardando o servidor', 'warning', false),
    ('fila', 'aguardando_terceiro', 3, 'Aguardando terceiro', 'warning', false),
    ('fila', 'concluida', 4, 'Concluídas', 'success', false),
    ('fila', 'encerrado', 5, 'Encerradas', 'neutral', false),
    ('fila', 'cancelado', 6, 'Canceladas', 'danger', false);

-- +goose Down
DROP TABLE IF EXISTS quadro_colunas;
