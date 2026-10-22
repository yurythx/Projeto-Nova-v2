-- ==============================================================================
-- PROJETO NEXUS — SEMENTE DO CHAT CORPORATIVO MERCÚRIO (RONDONÓPOLIS - MT)
-- ==============================================================================
-- Cadastra salas departamentais vinculadas aos departamentos de Rondonópolis,
-- associa as personas aos canais e popula mensagens de boas-vindas e alinhamento.
-- Idempotente: pode rodar a qualquer momento.
-- ==============================================================================

BEGIN;

DO $$
DECLARE
    -- Usuários
    v_u_admin        UUID;
    v_u_carlos       UUID;
    v_u_mariana      UUID;
    v_u_lucas        UUID;
    v_u_marcos       UUID;
    v_u_ana          UUID;
    v_u_rodrigo      UUID;

    -- Departamentos
    v_dep_dir        UUID := '30000000-0000-0000-0000-000000000001'; -- DIR-EXEC
    v_dep_mkt        UUID := '30000000-0000-0000-0000-000000000006'; -- MKT
    v_dep_dev        UUID := '30000000-0000-0000-0000-000000000008'; -- DEV
    v_dep_ops        UUID := '30000000-0000-0000-0000-000000000013'; -- OPS
    v_dep_prot       UUID := '30000000-0000-0000-0000-000000000018'; -- PROT
    v_dep_agro       UUID := '30000000-0000-0000-0000-000000000019'; -- INTERMODAL

    -- Salas
    v_r_geral        UUID := 'e7258631-3b1c-42bc-b7d4-29530d1e311e'; -- Global Geral
    v_r_dir          UUID := 'e7258631-3b1c-42bc-b7d4-29530d1e3101';
    v_r_mkt          UUID := 'e7258631-3b1c-42bc-b7d4-29530d1e3102';
    v_r_dev          UUID := 'e7258631-3b1c-42bc-b7d4-29530d1e3103';
    v_r_ops          UUID := 'e7258631-3b1c-42bc-b7d4-29530d1e3104';
    v_r_prot         UUID := 'e7258631-3b1c-42bc-b7d4-29530d1e3105';
    v_r_agro         UUID := 'e7258631-3b1c-42bc-b7d4-29530d1e3106';

BEGIN
    SELECT id INTO v_u_admin   FROM users WHERE username = 'admin';
    SELECT id INTO v_u_carlos  FROM users WHERE username = 'carlos.diretoria';
    SELECT id INTO v_u_mariana FROM users WHERE username = 'mariana.mkt';
    SELECT id INTO v_u_lucas   FROM users WHERE username = 'lucas.tech';
    SELECT id INTO v_u_marcos  FROM users WHERE username = 'marcos.ops';
    SELECT id INTO v_u_ana     FROM users WHERE username = 'ana.protocolo';
    SELECT id INTO v_u_rodrigo FROM users WHERE username = 'rodrigo.agro';

    -- 1. Sala Geral (Global - atualiza descrição)
    UPDATE mercurio_rooms
    SET description = 'Canal aberto a todos os colaboradores de todas as unidades de Rondonópolis.'
    WHERE id = v_r_geral;

    -- 2. Salas Departamentais
    INSERT INTO mercurio_rooms (id, kind, name, description, ad_group, departamento_id, created_by, created_at, updated_at)
    VALUES
        (v_r_dir, 'department', 'Diretoria Executiva & Estratégia', 'Alinhamento da liderança e diretoria executiva na Sede Centro.', 'NEXUS-DIR', v_dep_dir, v_u_carlos, now() - interval '10 days', now()),
        (v_r_mkt, 'department', 'Comunicação & Marketing', 'Coordenação de eventos, comunicados corporativos e publicações do Blog.', 'NEXUS-MKT', v_dep_mkt, v_u_mariana, now() - interval '10 days', now()),
        (v_r_dev, 'department', 'Engenharia de Software & P&D', 'Discussões de arquitetura, deploys e projetos do Hub Aurora.', 'TECH-DEV', v_dep_dev, v_u_lucas, now() - interval '10 days', now()),
        (v_r_ops, 'department', 'Operações & Logística Regional', 'Acompanhamento do tráfego de frotas, armazém e logística no Polo Industrial.', 'LOG-OPS', v_dep_ops, v_u_marcos, now() - interval '10 days', now()),
        (v_r_prot, 'department', 'Protocolo & Atendimento Integrado', 'Recepção de processos físicos, triagem e atendimento presencial no CIA Poguba.', 'ATD-PROT', v_dep_prot, v_u_ana, now() - interval '10 days', now()),
        (v_r_agro, 'department', 'Terminal Agro & Balança Ferroviária', 'Monitoramento da balança ferroviária, descargas e operações intermodais.', 'AGRO-BAL', v_dep_agro, v_u_rodrigo, now() - interval '10 days', now())
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        description = EXCLUDED.description,
        departamento_id = EXCLUDED.departamento_id,
        ad_group = EXCLUDED.ad_group,
        updated_at = now();

    -- 3. Membros nas salas
    -- Sala Geral: todos são membros
    INSERT INTO mercurio_room_members (room_id, user_id, joined_at)
    VALUES
        (v_r_geral, v_u_carlos, now() - interval '10 days'),
        (v_r_geral, v_u_mariana, now() - interval '10 days'),
        (v_r_geral, v_u_lucas, now() - interval '10 days'),
        (v_r_geral, v_u_marcos, now() - interval '10 days'),
        (v_r_geral, v_u_ana, now() - interval '10 days'),
        (v_r_geral, v_u_rodrigo, now() - interval '10 days')
    ON CONFLICT DO NOTHING;

    -- Salas departamentais
    INSERT INTO mercurio_room_members (room_id, user_id, joined_at)
    VALUES
        (v_r_dir, v_u_carlos, now() - interval '10 days'),
        (v_r_mkt, v_u_mariana, now() - interval '10 days'),
        (v_r_dev, v_u_lucas, now() - interval '10 days'),
        (v_r_ops, v_u_marcos, now() - interval '10 days'),
        (v_r_prot, v_u_ana, now() - interval '10 days'),
        (v_r_agro, v_u_rodrigo, now() - interval '10 days')
    ON CONFLICT DO NOTHING;

    -- 4. Mensagens Contextuais
    -- Canal Geral
    INSERT INTO mercurio_messages (id, room_id, author_id, body, created_at)
    VALUES
        (gen_random_uuid(), v_r_geral, v_u_carlos, 'Bom dia a todas as equipes de Rondonópolis! Sejam bem-vindos ao novo ambiente unificado da Plataforma Nexus.', now() - interval '4 days'),
        (gen_random_uuid(), v_r_geral, v_u_mariana, 'Lembrando que o cronograma da Exposul 2026 já está publicado no Blog Corporativo e aberto a comentários!', now() - interval '3 days'),
        (gen_random_uuid(), v_r_geral, v_u_lucas, 'Equipe Hub Aurora à disposição para integrações e suporte técnico no ramal 5508.', now() - interval '2 days'),
        (gen_random_uuid(), v_r_geral, v_u_marcos, 'Operações no Polo Industrial normalizadas após a manutenção preventiva da frota bitrem.', now() - interval '1 day'),
        (gen_random_uuid(), v_r_geral, v_u_ana, 'Atendimento presencial no CIA Poguba funcionando normalmente com triagem digital pelo módulo Trâmite.', now() - interval '12 hours'),
        (gen_random_uuid(), v_r_geral, v_u_rodrigo, 'Pátio ferroviário com fluxo ágil de transbordo e pesagem contínua no Terminal Agro.', now() - interval '2 hours')
    ON CONFLICT DO NOTHING;

    -- Canal Engenharia & Dev
    INSERT INTO mercurio_messages (id, room_id, author_id, body, created_at)
    VALUES
        (gen_random_uuid(), v_r_dev, v_u_lucas, 'Iniciamos os testes de integração da esteira de CI com multi-stage builds. Guia disponível na Wiki corporativa.', now() - interval '3 days'),
        (gen_random_uuid(), v_r_dev, v_u_lucas, 'Deploy da nova versão com isolamento de público-alvo (ADR 030) concluído com 100% de testes verdes.', now() - interval '6 hours')
    ON CONFLICT DO NOTHING;

    -- Canal Operações Polo
    INSERT INTO mercurio_messages (id, room_id, author_id, body, created_at)
    VALUES
        (gen_random_uuid(), v_r_ops, v_u_marcos, 'Atenção motoristas: limite de 20 km/h estritamente fiscalizado em todas as vias internas do Distrito Industrial.', now() - interval '2 days'),
        (gen_random_uuid(), v_r_ops, v_u_marcos, 'Conferência de estoque de EPIs e sinalização viária concluída com o setor de SMS.', now() - interval '5 hours')
    ON CONFLICT DO NOTHING;

    -- Canal Protocolo Poguba
    INSERT INTO mercurio_messages (id, room_id, author_id, body, created_at)
    VALUES
        (gen_random_uuid(), v_r_prot, v_u_ana, 'Encaminhado ao Hub Aurora o processo de adesão de cooperativas parceiras pelo Trâmite eletrônico.', now() - interval '1 day')
    ON CONFLICT DO NOTHING;

END $$;

COMMIT;
