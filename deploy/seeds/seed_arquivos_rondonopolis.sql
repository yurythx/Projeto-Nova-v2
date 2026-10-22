-- ==============================================================================
-- PROJETO NOVA — SEMENTE DO REPOSITÓRIO DE ARQUIVOS E GED (RONDONÓPOLIS - MT)
-- ==============================================================================
-- Cria a estrutura de pastas corporativas e permissões (ACLs) do GED
-- distribuídas entre as 5 unidades e departamentos de Rondonópolis.
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
    v_u_rodrigo      UUID;

    -- Unidades
    v_unid_centro    UUID := '20000000-0000-0000-0000-000000000001';
    v_unid_aurora    UUID := '20000000-0000-0000-0000-000000000002';
    v_unid_polo      UUID := '20000000-0000-0000-0000-000000000003';
    v_unid_poguba    UUID := '20000000-0000-0000-0000-000000000004';
    v_unid_agro      UUID := '20000000-0000-0000-0000-000000000005';

    -- IDs das Pastas Raiz
    v_f_dir          UUID := '90000000-0000-0000-0000-000000000001';
    v_f_dev          UUID := '90000000-0000-0000-0000-000000000002';
    v_f_ops          UUID := '90000000-0000-0000-0000-000000000003';
    v_f_prot         UUID := '90000000-0000-0000-0000-000000000004';
    v_f_agro         UUID := '90000000-0000-0000-0000-000000000005';

    -- IDs das Subpastas
    v_f_dir_atas     UUID := '90000000-0000-0000-0000-000000000011';
    v_f_dir_plan     UUID := '90000000-0000-0000-0000-000000000012';
    v_f_dev_apis     UUID := '90000000-0000-0000-0000-000000000021';
    v_f_dev_iot      UUID := '90000000-0000-0000-0000-000000000022';
    v_f_ops_laudos   UUID := '90000000-0000-0000-0000-000000000031';
    v_f_ops_pop      UUID := '90000000-0000-0000-0000-000000000032';
    v_f_prot_forms   UUID := '90000000-0000-0000-0000-000000000041';
    v_f_agro_bal     UUID := '90000000-0000-0000-0000-000000000051';

BEGIN
    SELECT id INTO v_u_carlos   FROM users WHERE username = 'carlos.diretoria';
    SELECT id INTO v_u_mariana  FROM users WHERE username = 'mariana.mkt';
    SELECT id INTO v_u_lucas    FROM users WHERE username = 'lucas.tech';
    SELECT id INTO v_u_marcos   FROM users WHERE username = 'marcos.ops';
    SELECT id INTO v_u_ana      FROM users WHERE username = 'ana.protocolo';
    SELECT id INTO v_u_rodrigo  FROM users WHERE username = 'rodrigo.agro';

    -- 1. Inserção das Pastas Raiz por Unidade
    INSERT INTO files_folders (id, parent_id, name, owner_id, unidade_id, created_at, updated_at)
    VALUES
        (v_f_dir, NULL, 'Diretoria e Governança Corporativa', v_u_carlos, v_unid_centro, now() - interval '20 days', now()),
        (v_f_dev, NULL, 'Engenharia de Software e P&D AgroTech', v_u_lucas, v_unid_aurora, now() - interval '20 days', now()),
        (v_f_ops, NULL, 'Operações Industriais e Logística Regional', v_u_marcos, v_unid_polo, now() - interval '20 days', now()),
        (v_f_prot, NULL, 'Atendimento Integrado e Protocolo CIA Poguba', v_u_ana, v_unid_poguba, now() - interval '20 days', now()),
        (v_f_agro, NULL, 'Terminal Intermodal e Malha Ferroviária', v_u_rodrigo, v_unid_agro, now() - interval '20 days', now())
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        owner_id = EXCLUDED.owner_id,
        unidade_id = EXCLUDED.unidade_id,
        updated_at = now();

    -- 2. Inserção das Subpastas Técnicas e Operacionais
    INSERT INTO files_folders (id, parent_id, name, owner_id, unidade_id, created_at, updated_at)
    VALUES
        (v_f_dir_atas, v_f_dir, 'Atas e Deliberações da Presidência', v_u_carlos, v_unid_centro, now() - interval '18 days', now()),
        (v_f_dir_plan, v_f_dir, 'Planejamento Estratégico e Metas 2026', v_u_carlos, v_unid_centro, now() - interval '18 days', now()),
        (v_f_dev_apis, v_f_dev, 'Especificações de APIs e Arquitetura', v_u_lucas, v_unid_aurora, now() - interval '18 days', now()),
        (v_f_dev_iot, v_f_dev, 'Telemetria de Sensores de Campo IoT', v_u_lucas, v_unid_aurora, now() - interval '18 days', now()),
        (v_f_ops_laudos, v_f_ops, 'Laudos de Inspeção e Manutenção de Bitrens', v_u_marcos, v_unid_polo, now() - interval '18 days', now()),
        (v_f_ops_pop, v_f_ops, 'Procedimentos Operacionais Padrão (SMS e NR-6)', v_u_marcos, v_unid_polo, now() - interval '18 days', now()),
        (v_f_prot_forms, v_f_prot, 'Formulários e Termos de Credenciamento', v_u_ana, v_unid_poguba, now() - interval '18 days', now()),
        (v_f_agro_bal, v_f_agro, 'Boletins de Pesagem e Fluxo de Composições', v_u_rodrigo, v_unid_agro, now() - interval '18 days', now())
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        parent_id = EXCLUDED.parent_id,
        owner_id = EXCLUDED.owner_id,
        unidade_id = EXCLUDED.unidade_id,
        updated_at = now();

    -- 3. Configuração de ACLs (files_folder_acl)
    -- Limpa permissões anteriores dessas pastas para idempotência
    DELETE FROM files_folder_acl WHERE folder_id IN (
        v_f_dir, v_f_dev, v_f_ops, v_f_prot, v_f_agro,
        v_f_dir_atas, v_f_dir_plan, v_f_dev_apis, v_f_dev_iot,
        v_f_ops_laudos, v_f_ops_pop, v_f_prot_forms, v_f_agro_bal
    );

    -- Permissões por Unidade
    -- Sede Centro
    INSERT INTO files_folder_acl (folder_id, subject_type, subject, can_write, granted_by, granted_at)
    VALUES
        (v_f_dir, 'unidade', v_unid_centro::text, true, 'carlos.diretoria', now()),
        (v_f_dir_atas, 'unidade', v_unid_centro::text, true, 'carlos.diretoria', now()),
        (v_f_dir_plan, 'everyone', '*', false, 'carlos.diretoria', now()); -- Todos podem ler o plano estratégico!

    -- Hub Aurora
    INSERT INTO files_folder_acl (folder_id, subject_type, subject, can_write, granted_by, granted_at)
    VALUES
        (v_f_dev, 'unidade', v_unid_aurora::text, true, 'lucas.tech', now()),
        (v_f_dev_apis, 'unidade', v_unid_aurora::text, true, 'lucas.tech', now()),
        (v_f_dev_iot, 'unidade', v_unid_aurora::text, true, 'lucas.tech', now());

    -- Polo Industrial
    INSERT INTO files_folder_acl (folder_id, subject_type, subject, can_write, granted_by, granted_at)
    VALUES
        (v_f_ops, 'unidade', v_unid_polo::text, true, 'marcos.ops', now()),
        (v_f_ops_laudos, 'unidade', v_unid_polo::text, true, 'marcos.ops', now()),
        (v_f_ops_pop, 'everyone', '*', false, 'marcos.ops', now()); -- POP de SMS acessível a todos

    -- CIA Poguba
    INSERT INTO files_folder_acl (folder_id, subject_type, subject, can_write, granted_by, granted_at)
    VALUES
        (v_f_prot, 'unidade', v_unid_poguba::text, true, 'ana.protocolo', now()),
        (v_f_prot_forms, 'everyone', '*', false, 'ana.protocolo', now()); -- Formulários abertos para leitura

    -- Terminal Agro
    INSERT INTO files_folder_acl (folder_id, subject_type, subject, can_write, granted_by, granted_at)
    VALUES
        (v_f_agro, 'unidade', v_unid_agro::text, true, 'rodrigo.agro', now()),
        (v_f_agro_bal, 'unidade', v_unid_agro::text, true, 'rodrigo.agro', now());

END $$;

COMMIT;
