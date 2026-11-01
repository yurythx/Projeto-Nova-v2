-- +goose Up
-- Migração 47: Matriz de Riscos de TI (ISO/IEC 27001 / GRC) e Agrupamento de Incidentes AIOps

-- 1. Agrupamento de Incidentes AIOps
ALTER TABLE atendimento_solicitacoes
    ADD COLUMN IF NOT EXISTS incidente_pai_id UUID REFERENCES atendimento_solicitacoes(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_atendimento_solicitacoes_pai ON atendimento_solicitacoes(incidente_pai_id);

-- 2. Tabela de Gestão de Riscos de TI (ISO/IEC 27001:2022 / GRC)
CREATE TABLE IF NOT EXISTS riscos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    codigo TEXT NOT NULL UNIQUE,
    titulo TEXT NOT NULL CHECK (length(btrim(titulo)) BETWEEN 3 AND 160),
    descricao TEXT NOT NULL DEFAULT '',
    categoria TEXT NOT NULL CHECK (categoria IN ('seguranca_informacao', 'infraestrutura', 'fornecedores', 'continuidade', 'conformidade_lgpd', 'operacional')),
    ativo_id UUID REFERENCES ativos(id) ON DELETE SET NULL,
    unidade_id UUID REFERENCES unidades(id) ON DELETE CASCADE,
    probabilidade INTEGER NOT NULL CHECK (probabilidade BETWEEN 1 AND 5),
    impacto INTEGER NOT NULL CHECK (impacto BETWEEN 1 AND 5),
    severidade INTEGER GENERATED ALWAYS AS (probabilidade * impacto) STORED,
    estrategia TEXT NOT NULL DEFAULT 'mitigar' CHECK (estrategia IN ('mitigar', 'aceitar', 'transferir', 'evitar')),
    plano_mitigacao TEXT NOT NULL DEFAULT '',
    mudanca_id UUID REFERENCES mudancas(id) ON DELETE SET NULL,
    projeto_id UUID REFERENCES projetos(id) ON DELETE SET NULL,
    responsavel_id UUID REFERENCES users(id) ON DELETE SET NULL,
    responsavel_nome TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'identificado' CHECK (status IN ('identificado', 'em_analise', 'em_mitigacao', 'monitorado', 'resolvido')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_riscos_unidade ON riscos(unidade_id);
CREATE INDEX IF NOT EXISTS idx_riscos_categoria ON riscos(categoria);
CREATE INDEX IF NOT EXISTS idx_riscos_status ON riscos(status);

-- 3. Carga inicial de riscos municipais oficiais (Prefeitura, SMS e SEMECEL)
-- +goose StatementBegin
DO $$
DECLARE
    v_pref_id UUID;
    v_sms_id UUID;
    v_semecel_id UUID;
    v_admin_id UUID;
BEGIN
    SELECT id INTO v_pref_id FROM unidades WHERE sigla = 'PREFEITURA' LIMIT 1;
    IF v_pref_id IS NULL THEN SELECT id INTO v_pref_id FROM unidades LIMIT 1; END IF;
    
    SELECT id INTO v_sms_id FROM unidades WHERE sigla = 'SMS' LIMIT 1;
    IF v_sms_id IS NULL THEN v_sms_id := v_pref_id; END IF;

    SELECT id INTO v_semecel_id FROM unidades WHERE sigla = 'SEMECEL' LIMIT 1;
    IF v_semecel_id IS NULL THEN v_semecel_id := v_pref_id; END IF;

    SELECT id INTO v_admin_id FROM users WHERE username = 'admin' LIMIT 1;
    IF v_admin_id IS NULL THEN SELECT id INTO v_admin_id FROM users LIMIT 1; END IF;

    INSERT INTO riscos (codigo, titulo, descricao, categoria, unidade_id, probabilidade, impacto, estrategia, plano_mitigacao, responsavel_id, responsavel_nome, status)
    VALUES
        ('RSK-001', 'Queda de fornecimento de energia no Datacenter Central', 'Oscilações e falha de gerador ou nobreak principal que podem paralisar os sistemas centrais da Prefeitura.', 'infraestrutura', v_pref_id, 3, 5, 'mitigar', 'Aquisição e manutenção preventiva de nobreaks modulares com bypass e contrato de gerador a diesel.', v_admin_id, 'Gerência de Infraestrutura', 'em_mitigacao'),
        ('RSK-002', 'Ataque de Ransomware / Sequestro de Dados em Servidores Municipais', 'Tentativas de infecção via phishing ou vulnerabilidades em serviços expostos na rede municipal.', 'seguranca_informacao', v_pref_id, 2, 5, 'mitigar', 'Implementação de backup WORM imutável fora do domínio, autenticação MFA e treinamento de conscientização.', v_admin_id, 'Comitê de Segurança da Informação', 'em_analise'),
        ('RSK-003', 'Indisponibilidade do Prontuário Eletrônico nas Unidades Básicas de Saúde (UBS)', 'Falha nos links de comunicação das UBS ou nos servidores centrais de saúde impedindo atendimento médico aos cidadãos.', 'continuidade', v_sms_id, 3, 5, 'mitigar', 'Contratação de links secundários redundantes (SD-WAN / 4G) e replicação de banco de dados em tempo real.', v_admin_id, 'Coordenação de TI da Saúde', 'em_mitigacao'),
        ('RSK-004', 'Vazamento de Dados Pessoais Sensíveis de Pacientes (LGPD)', 'Acesso não autorizado a prontuários e históricos médicos por ausência de duplo fator ou privilégios excessivos.', 'conformidade_lgpd', v_sms_id, 2, 4, 'mitigar', 'Revisão semestral de permissões RBAC, auditoria criptográfica de logs e termo de sigilo para profissionais de saúde.', v_admin_id, 'Encarregado DPO', 'monitorado'),
        ('RSK-005', 'Interrupção do Sistema de Matrículas e Diário de Classe Digital', 'Sobrecarga de acessos no período de matrículas escolares da rede municipal causando travamento do portal.', 'operacional', v_semecel_id, 4, 3, 'mitigar', 'Escalonamento automático em nuvem durante o período de rematrícula e balanceamento de carga.', v_admin_id, 'Equipe de Sistemas da Educação', 'identificado')
    ON CONFLICT (codigo) DO UPDATE
    SET titulo = EXCLUDED.titulo, descricao = EXCLUDED.descricao, plano_mitigacao = EXCLUDED.plano_mitigacao;

END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS riscos;
ALTER TABLE atendimento_solicitacoes DROP COLUMN IF EXISTS incidente_pai_id;
