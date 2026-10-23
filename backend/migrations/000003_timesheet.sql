-- +goose Up
-- Tabelas de Apontamento de Horas e Timesheet operacional para Projetos e Atendimento

CREATE TABLE IF NOT EXISTS projetos_apontamentos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    projeto_id UUID NOT NULL REFERENCES projetos(id) ON DELETE CASCADE,
    tarefa_id UUID REFERENCES projetos_tarefas(id) ON DELETE SET NULL,
    usuario_id UUID NOT NULL,
    usuario_nome VARCHAR(200) NOT NULL,
    data_apontamento DATE NOT NULL DEFAULT CURRENT_DATE,
    horas NUMERIC(6, 2) NOT NULL CHECK (horas > 0 AND horas <= 24),
    descricao TEXT NOT NULL,
    custo_hora NUMERIC(12, 2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projetos_apontamentos_projeto ON projetos_apontamentos(projeto_id);
CREATE INDEX IF NOT EXISTS idx_projetos_apontamentos_tarefa ON projetos_apontamentos(tarefa_id);
CREATE INDEX IF NOT EXISTS idx_projetos_apontamentos_data ON projetos_apontamentos(data_apontamento);
CREATE INDEX IF NOT EXISTS idx_projetos_apontamentos_usuario ON projetos_apontamentos(usuario_id);

CREATE TABLE IF NOT EXISTS atendimento_apontamentos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    solicitacao_id UUID NOT NULL REFERENCES atendimento_solicitacoes(id) ON DELETE CASCADE,
    usuario_id UUID NOT NULL,
    usuario_nome VARCHAR(200) NOT NULL,
    data_apontamento DATE NOT NULL DEFAULT CURRENT_DATE,
    horas NUMERIC(6, 2) NOT NULL CHECK (horas > 0 AND horas <= 24),
    descricao TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_atendimento_apontamentos_solic ON atendimento_apontamentos(solicitacao_id);
CREATE INDEX IF NOT EXISTS idx_atendimento_apontamentos_data ON atendimento_apontamentos(data_apontamento);

-- +goose Down
DROP TABLE IF EXISTS atendimento_apontamentos;
DROP TABLE IF EXISTS projetos_apontamentos;
