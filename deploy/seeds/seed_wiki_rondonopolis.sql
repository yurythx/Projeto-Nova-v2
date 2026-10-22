-- ==============================================================================
-- PROJETO NOVA — SEMENTE DA BASE DE CONHECIMENTO WIKI (RONDONÓPOLIS - MT)
-- ==============================================================================
-- Cadastra artigos e manuais operacionais corporativos para as unidades
-- de Rondonópolis, com controle de versão, árvore hierárquica e público-alvo.
-- Idempotente: pode rodar a qualquer momento.
-- ==============================================================================

BEGIN;

DO $$
DECLARE
    -- Usuários
    v_u_carlos       UUID;
    v_u_mariana      UUID;
    v_u_lucas        UUID;
    v_u_marcos       UUID;
    v_u_ana          UUID;

    -- Unidades
    v_unid_centro    UUID := '20000000-0000-0000-0000-000000000001';
    v_unid_aurora    UUID := '20000000-0000-0000-0000-000000000002';
    v_unid_polo      UUID := '20000000-0000-0000-0000-000000000003';
    v_unid_poguba    UUID := '20000000-0000-0000-0000-000000000004';

    -- IDs das Páginas
    v_p_onboarding   UUID := '70000000-0000-0000-0000-000000000001';
    v_p_conduta      UUID := '70000000-0000-0000-0000-000000000002';
    v_p_seguranca    UUID := '70000000-0000-0000-0000-000000000003';
    v_p_dev          UUID := '70000000-0000-0000-0000-000000000004';
    v_p_polo         UUID := '70000000-0000-0000-0000-000000000005';
    v_p_atendimento  UUID := '70000000-0000-0000-0000-000000000006';

BEGIN
    SELECT id INTO v_u_carlos   FROM users WHERE username = 'carlos.diretoria';
    SELECT id INTO v_u_mariana  FROM users WHERE username = 'mariana.mkt';
    SELECT id INTO v_u_lucas    FROM users WHERE username = 'lucas.tech';
    SELECT id INTO v_u_marcos   FROM users WHERE username = 'marcos.ops';
    SELECT id INTO v_u_ana      FROM users WHERE username = 'ana.protocolo';

    -- =========================================================================
    -- PÁGINA 1: Guia de Onboarding & Integração do Colaborador (Raiz)
    -- =========================================================================
    INSERT INTO wiki_pages (
        id, parent_id, slug, title, body, position, version,
        created_by, updated_by, unidade_id, created_at, updated_at
    ) VALUES (
        v_p_onboarding, NULL, 'onboarding-colaborador',
        'Guia de Boas-Vindas & Integração do Colaborador Nova',
        '# Boas-Vindas à Nova Corporation — Polo Rondonópolis\n\nEste manual reúne as principais informações para apoiar a sua jornada e integração em nossas unidades em Rondonópolis - MT.\n\n## 1. Nossas Unidades Regionais\n- **Sede Corporativa Central:** Av. Marechal Rondon, 1250 (Centro)\n- **Hub de Inovação & P&D:** Av. Lions Internacional, 1500 (Vila Aurora)\n- **Complexo Operacional:** Rodovia BR-364, km 205 (Distrito Industrial Veterinário)\n- **Centro Integrado de Atendimento:** Av. Poguba, 850 (Sagrada Família)\n- **Terminal Intermodal & Agronegócios:** Anel Viário Conrado Sales Brito\n\n## 2. Acesso aos Sistemas Internos\n- **E-mail Corporativo e SSO:** Gerenciado pelo Keycloak integrado ao Active Directory.\n- **Processos & Memorandos:** Acesse o módulo **Trâmite** para autuar ou assinar despachos eletrônicos.\n- **Agendamento de Salas:** Utilize a **Agenda** para reservar salas de reunião em qualquer uma das unidades.\n\nEm caso de dúvidas, procure o departamento de Gente & Gestão (RH) ou o Suporte de TI no ramal `5508`.',
        1, 1, v_u_mariana, v_u_mariana, v_unid_centro, now() - interval '15 days', now() - interval '5 days'
    ) ON CONFLICT (slug) DO UPDATE SET
        title = EXCLUDED.title, body = EXCLUDED.body, updated_at = now();

    -- Revisão 1
    INSERT INTO wiki_revisions (id, page_id, version, title, body, summary, edited_by, edited_at)
    VALUES (
        gen_random_uuid(), v_p_onboarding, 1,
        'Guia de Boas-Vindas & Integração do Colaborador Nova',
        '# Boas-Vindas à Nova Corporation — Polo Rondonópolis...',
        'Publicação inicial do manual de onboarding corporativo.',
        v_u_mariana, now() - interval '15 days'
    ) ON CONFLICT (page_id, version) DO NOTHING;

    -- =========================================================================
    -- PÁGINA 2: Código de Conduta & Integridade (Filha de Onboarding)
    -- =========================================================================
    INSERT INTO wiki_pages (
        id, parent_id, slug, title, body, position, version,
        created_by, updated_by, unidade_id, created_at, updated_at
    ) VALUES (
        v_p_conduta, v_p_onboarding, 'codigo-de-conduta',
        'Código de Ética, Integridade & Conduta Corporativa',
        '# Diretrizes de Ética & Integridade Corporativa\n\nO presente código estabelece os padrões éticos que orientam as relações profissionais, a condução de negócios e o cumprimento da legislação em todas as operações da Nova.\n\n## 1. Princípios Fundamentais\n- **Transparência e Respeito:** Comunicação clara, tratamento igualitário e combate a qualquer forma de discriminação.\n- **Conformidade com a LGPD:** Proteção absoluta dos dados pessoais de colaboradores, clientes e produtores rurais cadastrados.\n- **Conflito de Interesses:** Colaboradores devem comunicar previamente qualquer relação comercial ou familiar que possa influenciar decisões da companhia.\n\n## 2. Canal de Denúncias e Ouvidoria\nDenúncias sobre desvios éticos podem ser encaminhadas de forma anônima à Diretoria de Governança ou protocoladas diretamente no CIA Poguba.',
        1, 1, v_u_carlos, v_u_carlos, v_unid_centro, now() - interval '14 days', now() - interval '4 days'
    ) ON CONFLICT (slug) DO UPDATE SET
        title = EXCLUDED.title, body = EXCLUDED.body, parent_id = v_p_onboarding, updated_at = now();

    INSERT INTO wiki_revisions (id, page_id, version, title, body, summary, edited_by, edited_at)
    VALUES (
        gen_random_uuid(), v_p_conduta, 1,
        'Código de Ética, Integridade & Conduta Corporativa',
        '# Diretrizes de Ética & Integridade Corporativa...',
        'Homologação do código de conduta pela Presidência.',
        v_u_carlos, now() - interval '14 days'
    ) ON CONFLICT (page_id, version) DO NOTHING;

    -- =========================================================================
    -- PÁGINA 3: Política de Segurança da Informação & SOC (Hub Aurora)
    -- =========================================================================
    INSERT INTO wiki_pages (
        id, parent_id, slug, title, body, position, version,
        created_by, updated_by, unidade_id, created_at, updated_at
    ) VALUES (
        v_p_seguranca, NULL, 'politica-seguranca-informacao',
        'Política de Segurança da Informação, Controle de Acessos & SOC',
        '# Política de Segurança da Informação (PSI)\n\nDiretrizes normativas para mitigação de riscos cibernéticos, salvaguarda de ativos computacionais e controle de acessos em todas as redes da Nova.\n\n## 1. Gestão de Senhas e Credenciais\n- Senhas devem conter no mínimo 12 caracteres, combinando letras maiúsculas, minúsculas, números e símbolos.\n- O compartilhamento de credenciais de login ou chaves de API é terminantemente proibido.\n- O lockout progressivo bloqueia contas automaticamente após tentativas inválidas consecutivas.\n\n## 2. Acesso Remoto & Redes Sem Fio\n- Colaboradores em trabalho remoto devem utilizar o cliente corporativo WireGuard.\n- Redes Wi-Fi de visitantes não possuem rota para os servidores de banco de dados ou painel administrativo.\n\n## 3. Notificação de Incidentes\nQualquer suspeita de vazamento de dados ou atividade anômala deve ser comunicada imediatamente ao SOC do Hub Aurora pelo ramal `5510`.',
        2, 1, v_u_lucas, v_u_lucas, v_unid_aurora, now() - interval '12 days', now() - interval '2 days'
    ) ON CONFLICT (slug) DO UPDATE SET
        title = EXCLUDED.title, body = EXCLUDED.body, updated_at = now();

    INSERT INTO wiki_revisions (id, page_id, version, title, body, summary, edited_by, edited_at)
    VALUES (
        gen_random_uuid(), v_p_seguranca, 1,
        'Política de Segurança da Informação, Controle de Acessos & SOC',
        '# Política de Segurança da Informação (PSI)...',
        'Revisão técnica semestral aprovada pela equipe de segurança.',
        v_u_lucas, now() - interval '12 days'
    ) ON CONFLICT (page_id, version) DO NOTHING;

    -- =========================================================================
    -- PÁGINA 4: Manual de Engenharia e P&D AgroTech (Restrito ao Hub Aurora)
    -- =========================================================================
    INSERT INTO wiki_pages (
        id, parent_id, slug, title, body, position, version,
        created_by, updated_by, unidade_id, created_at, updated_at
    ) VALUES (
        v_p_dev, v_p_seguranca, 'guia-desenvolvimento-agrotech',
        'Manual de Boas Práticas de Engenharia e Arquitetura de Software',
        '# Guia de Engenharia & Arquitetura — Nova AgroTech\n\n> **Aviso de Confidencialidade:** Este documento é de circulação restrita à equipe de Pesquisa, Desenvolvimento e Engenharia de Software do Hub de Inovação da Vila Aurora.\n\n## 1. Padrões de Código e Arquitetura\n- **Backend Go:** Clean Architecture com separação rigorosa de domínios, aplicação e adaptadores de infraestrutura.\n- **Frontend Next.js:** React 19, componentes com design tokens Vanilla CSS e zero uso de classes utilitárias dispersas.\n- **Documentação de API:** Todo endpoint deve ser especificado em `docs/openapi.yaml` e validado por testes de contrato.\n\n## 2. Esteira de Integração Contínua (CI)\n- Toda alteração passa pela suíte completa de testes unitários e de integração antes do merge.\n- Cobertura obrigatória mínima mantida em 100% dos pacotes críticos do sistema.\n- Builds de imagem Docker utilizam multi-stage builds reproduzíveis.',
        1, 1, v_u_lucas, v_u_lucas, v_unid_aurora, now() - interval '10 days', now() - interval '1 day'
    ) ON CONFLICT (slug) DO UPDATE SET
        title = EXCLUDED.title, body = EXCLUDED.body, parent_id = v_p_seguranca, updated_at = now();

    INSERT INTO wiki_revisions (id, page_id, version, title, body, summary, edited_by, edited_at)
    VALUES (
        gen_random_uuid(), v_p_dev, 1,
        'Manual de Boas Práticas de Engenharia e Arquitetura de Software',
        '# Guia de Engenharia & Arquitetura — Nova AgroTech...',
        'Atualização dos padrões de CI e multi-stage builds.',
        v_u_lucas, now() - interval '10 days'
    ) ON CONFLICT (page_id, version) DO NOTHING;

    -- Público-alvo da Página 4: Restrito ao Hub Aurora (ADR 014!)
    DELETE FROM wiki_page_publico WHERE page_id = v_p_dev;
    INSERT INTO wiki_page_publico (page_id, entidade_id, unidade_id)
    VALUES (v_p_dev, NULL, v_unid_aurora);

    -- =========================================================================
    -- PÁGINA 5: Manual de Operações Logísticas e Segurança do Trabalho (Polo)
    -- =========================================================================
    INSERT INTO wiki_pages (
        id, parent_id, slug, title, body, position, version,
        created_by, updated_by, unidade_id, created_at, updated_at
    ) VALUES (
        v_p_polo, NULL, 'manual-operacoes-polo-industrial',
        'Manual Operacional do Complexo Logístico e Segurança do Trabalho (SMS)',
        '# Operações Logísticas & Segurança no Distrito Industrial\n\nInstruções normativas para tráfego interno de carretas, manuseio de cargas e segurança do trabalho nas instalações da BR-364.\n\n## 1. Regras de Trânsito no Pátio\n- Limite de velocidade: **20 km/h** em todas as vias internas do complexo.\n- Proibida a circulação de pedestres fora das faixas sinalizadas sem colete reflexivo.\n- Estacionamento de bitrens permitido apenas nas baias demarcadas com calços nas rodas.\n\n## 2. Equipamentos de Proteção Individual (EPIs)\n- Capacete de segurança com jugular e óculos de proteção.\n- Botina de segurança com biqueira de aço.\n- Protetor auricular em áreas com ruído contínuo superior a 80 dB (oficina mecânica e lavador).',
        3, 1, v_u_marcos, v_u_marcos, v_unid_polo, now() - interval '8 days', now() - interval '3 days'
    ) ON CONFLICT (slug) DO UPDATE SET
        title = EXCLUDED.title, body = EXCLUDED.body, updated_at = now();

    INSERT INTO wiki_revisions (id, page_id, version, title, body, summary, edited_by, edited_at)
    VALUES (
        gen_random_uuid(), v_p_polo, 1,
        'Manual Operacional do Complexo Logístico e Segurança do Trabalho (SMS)',
        '# Operações Logísticas & Segurança no Distrito Industrial...',
        'Normatização das regras de tráfego de carretas e segurança SMS.',
        v_u_marcos, now() - interval '8 days'
    ) ON CONFLICT (page_id, version) DO NOTHING;

    -- =========================================================================
    -- PÁGINA 6: Protocolo de Atendimento ao Produtor Rural e Cooperativas (Poguba)
    -- =========================================================================
    INSERT INTO wiki_pages (
        id, parent_id, slug, title, body, position, version,
        created_by, updated_by, unidade_id, created_at, updated_at
    ) VALUES (
        v_p_atendimento, NULL, 'protocolo-atendimento-produtor',
        'Diretrizes de Atendimento Integrado e Triagem Documental',
        '# Procedimentos de Atendimento Presencial — CIA Poguba\n\nManual de atendimento ao cidadão, produtores rurais e cooperativas parceiras nas dependências da Av. Poguba (Sagrada Família).\n\n## 1. Etapas do Atendimento\n1. **Acolhimento e Triagem:** Identificação do interessado e verificação de documentos obrigatórios (CNPJ/CPF, Inscrição Estadual e comprovante de endereço).\n2. **Abertura no Trâmite:** Todo requerimento formal deve gerar um número sequencial único no módulo **Trâmite**.\n3. **Encaminhamento Setorial:**\n   - Dúvidas tributárias/fiscais: Encaminhar para `SEDE-CENTRO / FIN`.\n   - Suporte a sistemas e APIs: Encaminhar para `HUB-AURORA / DEV`.\n   - Logística de cargas e balança: Encaminhar para `POLO-IND` ou `TERM-AGRO`.\n\n## 2. Prazos de Resposta\n- Triagens iniciais: até 24 horas úteis.\n- Emissão de certidões integradas: até 48 horas úteis.',
        4, 1, v_u_ana, v_u_ana, v_unid_poguba, now() - interval '6 days', now() - interval '1 day'
    ) ON CONFLICT (slug) DO UPDATE SET
        title = EXCLUDED.title, body = EXCLUDED.body, updated_at = now();

    INSERT INTO wiki_revisions (id, page_id, version, title, body, summary, edited_by, edited_at)
    VALUES (
        gen_random_uuid(), v_p_atendimento, 1,
        'Diretrizes de Atendimento Integrado e Triagem Documental',
        '# Procedimentos de Atendimento Presencial — CIA Poguba...',
        'Elaboração das diretrizes de atendimento presencial e prazos de resposta.',
        v_u_ana, now() - interval '6 days'
    ) ON CONFLICT (page_id, version) DO NOTHING;

END $$;

COMMIT;
