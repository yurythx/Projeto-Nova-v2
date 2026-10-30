-- +goose Up
-- Migração 46: SLAs independentes, amarrações nas categorias e importação de Projetos, Mudanças e Problemas do GLPI.

-- 1. Tabela de SLAs independentes (ADR de Gestão de Níveis de Serviço / ITIL v5)
CREATE TABLE IF NOT EXISTS atendimento_slas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nome TEXT NOT NULL CHECK (length(btrim(nome)) BETWEEN 1 AND 120),
    descricao TEXT NOT NULL DEFAULT '',
    tempo_resposta_minutos INTEGER NOT NULL CHECK (tempo_resposta_minutos > 0),
    tempo_solucao_minutos INTEGER NOT NULL CHECK (tempo_solucao_minutos > 0),
    pausar_em_espera BOOLEAN NOT NULL DEFAULT true,
    calendario_tipo TEXT NOT NULL DEFAULT 'expediente' CHECK (calendario_tipo IN ('expediente', 'corrido_24x7')),
    unidade_id UUID REFERENCES unidades(id) ON DELETE CASCADE,
    ativa BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 2. Amarrações de SLA nas Categorias e Ofertas de Atendimento
ALTER TABLE atendimento_categorias ADD COLUMN IF NOT EXISTS sla_id UUID REFERENCES atendimento_slas(id) ON DELETE SET NULL;
ALTER TABLE atendimento_ofertas ADD COLUMN IF NOT EXISTS sla_id UUID REFERENCES atendimento_slas(id) ON DELETE SET NULL;

-- 3. Carga das SLAs oficiais importadas do GLPI
INSERT INTO atendimento_slas (id, nome, descricao, tempo_resposta_minutos, tempo_solucao_minutos, pausar_em_espera, calendario_tipo)
VALUES
    ('a0000001-0000-0000-0000-000000000001'::uuid, 'SLA N1 - Alta Prioridade (4h)', 'Atendimento em até 30 min e Solução em até 4 horas úteis (GLPI N1)', 30, 240, true, 'expediente'),
    ('a0000001-0000-0000-0000-000000000002'::uuid, 'SLA N2 - Média Prioridade (24h)', 'Atendimento em até 2 horas e Solução em até 24 horas úteis (GLPI N2)', 120, 1440, true, 'expediente'),
    ('a0000001-0000-0000-0000-000000000003'::uuid, 'SLA N3 - Baixa Prioridade (36h)', 'Atendimento em até 2 horas e Solução em até 36 horas úteis (GLPI N3)', 120, 2160, true, 'expediente')
ON CONFLICT (id) DO UPDATE
SET nome = EXCLUDED.nome, tempo_resposta_minutos = EXCLUDED.tempo_resposta_minutos, tempo_solucao_minutos = EXCLUDED.tempo_solucao_minutos;

-- 4. Amarração de SLA Padrão nas Categorias de Atendimento existentes
UPDATE atendimento_categorias SET sla_id = 'a0000001-0000-0000-0000-000000000002'::uuid WHERE sla_id IS NULL;

-- Variáveis auxiliares para IDs de unidades e usuários
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

    -- 5. Importação dos Projetos do GLPI
    INSERT INTO projetos (titulo, descricao, objetivo, estado, prioridade, data_inicio_planejada, data_fim_planejada, data_inicio_real, data_fim_real, orcamento, unidade_id, gerente_id, gerente_nome, created_at, updated_at)
    VALUES ('Projeto - Sistema AMTC', 'Sistema de Acompanhamento de Transporte Público – AMTC O projeto consiste no desenvolvimento de um aplicativo mobile voltado aos usuários do transporte público da AMTC, com o objetivo de oferecer praticidade, agilidade e transparência no acompanhamento da frota de ônibus em tempo real. Principais funcionalidades: Compra de créditos diretamente pelo aplicativo, com integração a diferentes formas de pagamento; Monitoramento em tempo real dos ônibus, permitindo que o usuário saiba exatamente onde o veículo está; Consulta de itinerários, com visualização detalhada das linhas e seus respectivos horários; Favoritos, para que o usuário possa salvar suas rotas preferidas e acessá-las rapidamente; Busca de rotas por destino, facilitando o planejamento do trajeto até o local desejado.', 'Objetivo do projeto cadastrado no GLPI da Prefeitura', 'em_andamento', 'media', '2025-03-10 12:00:00', NULL, '2025-03-10 12:00:00', NULL, 0.00, v_pref_id, v_admin_id, 'Gerência de TI', '2025-04-10 15:24:06', '2025-05-12 16:51:38')
    ON CONFLICT DO NOTHING;

    INSERT INTO projetos (titulo, descricao, objetivo, estado, prioridade, data_inicio_planejada, data_fim_planejada, data_inicio_real, data_fim_real, orcamento, unidade_id, gerente_id, gerente_nome, created_at, updated_at)
    VALUES ('Projeto - Portais das Secretarias', 'O objetivo é desenvolver um portal dedicado para cada Secretaria, utilizando uma base comum que garanta consistência, eficiência e facilidade de manutenção. Este projeto abrange as seguintes etapas e elementos: Análise de Requisitos: Levantamento das necessidades específicas de cada Secretaria para definir funcionalidades, design e integrações. Arquitetura Modular: Desenvolvimento de uma base unificada que permita a customização individual dos portais, assegurando a padronização e a flexibilidade. Design Responsivo e Acessibilidade: Criação de interfaces intuitivas e compatíveis com diferentes dispositivos, garantindo uma experiência de uso inclusiva. Segurança e Desempenho: Implementação de práticas de segurança da informação e otimização da performance para oferecer um ambiente confiável e ágil. Testes e Implantação: Realização de testes abrangentes e implantação gradual para assegurar a qualidade e a estabilidade dos portais. Manutenção e Suporte Contínuo: Planejamento de atualizações e suporte para adaptar o sistema às mudanças e demandas futuras.', 'Objetivo do projeto cadastrado no GLPI da Prefeitura', 'planejado', 'media', '2025-03-19 12:00:00', NULL, '2025-03-19 12:00:00', NULL, 0.00, v_pref_id, v_admin_id, 'Gerência de TI', '2025-04-10 15:25:27', '2025-04-28 12:30:32')
    ON CONFLICT DO NOTHING;

    INSERT INTO projetos (titulo, descricao, objetivo, estado, prioridade, data_inicio_planejada, data_fim_planejada, data_inicio_real, data_fim_real, orcamento, unidade_id, gerente_id, gerente_nome, created_at, updated_at)
    VALUES ('Projeto - Painel de Controle Administrativo', 'Este projeto tem o objetivo de desenvolver um novo painel administrativo, que servirá como base para os painéis individuais de cada projeto, como os Portais das Secretarias, Portais Municipais, entre outros.', 'Objetivo do projeto cadastrado no GLPI da Prefeitura', 'planejado', 'media', '2025-04-14 12:00:00', NULL, '2025-04-14 12:00:00', NULL, 0.00, v_pref_id, v_admin_id, 'Gerência de TI', '2025-04-10 15:26:25', '2025-04-17 09:42:37')
    ON CONFLICT DO NOTHING;

    INSERT INTO projetos (titulo, descricao, objetivo, estado, prioridade, data_inicio_planejada, data_fim_planejada, data_inicio_real, data_fim_real, orcamento, unidade_id, gerente_id, gerente_nome, created_at, updated_at)
    VALUES ('Projeto - Portal Municipal', 'Projeto Referente ao desenvolvimento, manutenção e melhorias do Portal Municipal', 'Objetivo do projeto cadastrado no GLPI da Prefeitura', 'em_andamento', 'media', '2025-01-01 12:00:00', NULL, '2025-01-01 12:00:00', NULL, 0.00, v_pref_id, v_admin_id, 'Gerência de TI', '2025-04-10 15:28:12', '2025-07-02 08:16:01')
    ON CONFLICT DO NOTHING;

    INSERT INTO projetos (titulo, descricao, objetivo, estado, prioridade, data_inicio_planejada, data_fim_planejada, data_inicio_real, data_fim_real, orcamento, unidade_id, gerente_id, gerente_nome, created_at, updated_at)
    VALUES ('Projeto - Intranet', 'Este projeto tem como objetivo desenvolver um novo intranet para a prefeitura', 'Objetivo do projeto cadastrado no GLPI da Prefeitura', 'planejado', 'media', NULL, NULL, NULL, NULL, 0.00, v_pref_id, v_admin_id, 'Gerência de TI', '2025-04-11 08:23:00', '2025-04-17 09:44:10')
    ON CONFLICT DO NOTHING;

    INSERT INTO projetos (titulo, descricao, objetivo, estado, prioridade, data_inicio_planejada, data_fim_planejada, data_inicio_real, data_fim_real, orcamento, unidade_id, gerente_id, gerente_nome, created_at, updated_at)
    VALUES ('Sistema - Centro Pop', 'Este projeto tem como objetivo desenvolver um sistema que modernize o Centro POP, migrando formulários e relatórios para plataformas digitais. Inicialmente, serão criados dois formulários: - Formulário PIA: relatório completo dos usuários cadastrados no Centro POP; - Formulário da Recepção: registro de todas as pessoas que visitam o Centro POP. - Além disso, o projeto prevê o desenvolvimento de dois tipos de relatórios: - Relatório Final: resumo consolidado de todas as informações geradas pelos formulários; - Relatório Individual: possibilita a geração de relatórios específicos para diferentes grupos de dados coletados;', 'Objetivo do projeto cadastrado no GLPI da Prefeitura', 'em_andamento', 'media', '2025-03-19 12:00:00', '2025-05-30 12:00:00', '2025-03-19 12:00:00', NULL, 0.00, v_pref_id, v_admin_id, 'Gerência de TI', '2025-04-11 08:24:29', '2025-05-22 14:26:40')
    ON CONFLICT DO NOTHING;

    -- 6. Importação das Mudanças do GLPI
    INSERT INTO mudancas (titulo, descricao, justificativa, tipo, risco, estado, plano_reversao, inicio_planejado, fim_planejado, resultado, solicitante_id, unidade_id, created_at, updated_at)
    VALUES ('mudança nos pontos de rede da casa abrigo', 'foi passado os cabos e as canaletas com as caixinhas, falta so climpar os conectores femeas', 'Mudança de infraestrutura e conectividade executada pela equipe técnica.', 'normal', 'medio', 'em_execucao', 'Retorno ao cabeamento e conexão anterior em caso de anomalia.', '2026-04-24 09:09:18', NULL, '', v_admin_id, v_pref_id, '2026-04-24 09:09:18', '2026-04-24 09:10:29')
    ON CONFLICT DO NOTHING;

    INSERT INTO mudancas (titulo, descricao, justificativa, tipo, risco, estado, plano_reversao, inicio_planejado, fim_planejado, resultado, solicitante_id, unidade_id, created_at, updated_at)
    VALUES ('mudança nos pontos de rede da casa abrigo', 'mudança nos pontos de rede da casa abrigo, falta climpar os femeas', 'Mudança de infraestrutura e conectividade executada pela equipe técnica.', 'normal', 'medio', 'concluida', 'Retorno ao cabeamento e conexão anterior em caso de anomalia.', '2026-04-24 14:03:50', '2026-06-15 10:37:41', 'Mudança executada e validada com sucesso.', v_admin_id, v_pref_id, '2026-04-24 14:03:50', '2026-06-15 10:37:45')
    ON CONFLICT DO NOTHING;

    INSERT INTO mudancas (titulo, descricao, justificativa, tipo, risco, estado, plano_reversao, inicio_planejado, fim_planejado, resultado, solicitante_id, unidade_id, created_at, updated_at)
    VALUES ('mudança na procuradoria', 'implementação de novo sweet e 12 novos pontos de rede em nova sala, canaletas e rj femeas serão colocadas', 'Mudança de infraestrutura e conectividade executada pela equipe técnica.', 'normal', 'medio', 'concluida', 'Retorno ao cabeamento e conexão anterior em caso de anomalia.', '2026-04-24 14:35:54', '2026-05-29 13:54:13', 'Mudança executada e validada com sucesso.', v_admin_id, v_pref_id, '2026-04-24 14:35:54', '2026-05-29 13:54:13')
    ON CONFLICT DO NOTHING;

    INSERT INTO mudancas (titulo, descricao, justificativa, tipo, risco, estado, plano_reversao, inicio_planejado, fim_planejado, resultado, solicitante_id, unidade_id, created_at, updated_at)
    VALUES ('Instalação de link de internet na praça do Alfredo de Castro, será ins', 'Instalação de link de internet na praça do Alfredo de Castro, será instalado no dia 27/04/2026', 'Mudança de infraestrutura e conectividade executada pela equipe técnica.', 'normal', 'medio', 'concluida', 'Retorno ao cabeamento e conexão anterior em caso de anomalia.', '2026-04-24 14:39:39', '2026-04-29 10:17:37', 'Mudança executada e validada com sucesso.', v_admin_id, v_pref_id, '2026-04-24 14:40:29', '2026-04-29 10:17:37')
    ON CONFLICT DO NOTHING;

    INSERT INTO mudancas (titulo, descricao, justificativa, tipo, risco, estado, plano_reversao, inicio_planejado, fim_planejado, resultado, solicitante_id, unidade_id, created_at, updated_at)
    VALUES ('mudança centro recreativo alfredo de castro', 'feita a vistoria pra instalação de rede complet para 12 pontos', 'Mudança de infraestrutura e conectividade executada pela equipe técnica.', 'normal', 'medio', 'concluida', 'Retorno ao cabeamento e conexão anterior em caso de anomalia.', '2026-04-24 14:51:07', '2026-06-15 10:36:11', 'Mudança executada e validada com sucesso.', v_admin_id, v_pref_id, '2026-04-24 14:51:07', '2026-06-15 10:36:16')
    ON CONFLICT DO NOTHING;

    INSERT INTO mudancas (titulo, descricao, justificativa, tipo, risco, estado, plano_reversao, inicio_planejado, fim_planejado, resultado, solicitante_id, unidade_id, created_at, updated_at)
    VALUES ('Mudança ginasio marechal rondon', 'instalação link de internet no ginasio marechal rondon para distribuição de wifi pro administrativo apenas', 'Mudança de infraestrutura e conectividade executada pela equipe técnica.', 'normal', 'medio', 'concluida', 'Retorno ao cabeamento e conexão anterior em caso de anomalia.', '2026-04-24 15:01:17', '2026-05-11 17:00:39', 'Mudança executada e validada com sucesso.', v_admin_id, v_pref_id, '2026-04-24 15:01:17', '2026-05-11 17:00:43')
    ON CONFLICT DO NOTHING;

    -- 7. Importação dos Problemas do GLPI
    INSERT INTO problemas (titulo, descricao, estado, causa_raiz, contorno, unidade_id, criado_por, resolvido_em, created_at, updated_at)
    VALUES ('mudança nos pontos de rede da casa abrigo', 'passado o cabeamento e as caixinhas. falta climpar as femeas', 'encerrado', 'passado o cabeamento e as caixinhas. falta climpar as femeas', 'Substituição provisória do equipamento e remanejamento de pontos de rede.', v_pref_id, v_admin_id, '2026-04-24 09:06:36', '2026-04-24 09:05:54', '2026-04-24 09:06:36')
    ON CONFLICT DO NOTHING;

    INSERT INTO problemas (titulo, descricao, estado, causa_raiz, contorno, unidade_id, criado_por, resolvido_em, created_at, updated_at)
    VALUES ('maquina do atendimento deu defeito e vai ser acionado a garantia para', 'maquina do atendimento deu defeito e vai ser acionado a garantia para manutenção', 'encerrado', 'maquina do atendimento deu defeito e vai ser acionado a garantia para manutenção', 'Substituição provisória do equipamento e remanejamento de pontos de rede.', v_pref_id, v_admin_id, '2026-06-17 13:45:43', '2026-04-24 15:41:18', '2026-06-17 13:45:50')
    ON CONFLICT DO NOTHING;

    INSERT INTO problemas (titulo, descricao, estado, causa_raiz, contorno, unidade_id, criado_por, resolvido_em, created_at, updated_at)
    VALUES ('Conclusão da instalação de pontos fêmea e configuração definitiva de energia no rack casa da mulher.', 'De um total de 15 pontos fêmea previstos, restam 04 pontos para serem instalados. Os pontos de energia externos já foram concluídos. No entanto, o ponto de energia do rack ainda precisa de finalização (atualmente operando de forma provisória via extensão). Foi feita uma tentativa de alinhamento com a Naiane para o envio de um carro e finalização nesta semana, porém o horário não foi confirmado a tempo.', 'resolvido', 'De um total de 15 pontos fêmea previstos, restam 04 pontos para serem instalados. Os pontos de energia externos já foram concluídos. No entanto, o ponto de energia do rack ainda precisa de finalização (atualmente operando de forma provisória via extensão). Foi feita uma tentativa de alinhamento com a Naiane para o envio de um carro e finalização nesta semana, porém o horário não foi confirmado a tempo.', 'Substituição provisória do equipamento e remanejamento de pontos de rede.', v_pref_id, v_admin_id, '2026-05-04 07:48:35', '2026-04-28 12:48:57', '2026-05-04 07:48:35')
    ON CONFLICT DO NOTHING;

END $$;

-- +goose Down
-- Reversão da migration 46
DELETE FROM atendimento_slas WHERE id IN ('a0000001-0000-0000-0000-000000000001'::uuid, 'a0000001-0000-0000-0000-000000000002'::uuid, 'a0000001-0000-0000-0000-000000000003'::uuid);
ALTER TABLE atendimento_ofertas DROP COLUMN IF EXISTS sla_id;
ALTER TABLE atendimento_categorias DROP COLUMN IF EXISTS sla_id;
DROP TABLE IF EXISTS atendimento_slas;
