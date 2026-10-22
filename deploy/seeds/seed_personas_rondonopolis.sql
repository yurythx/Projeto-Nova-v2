-- ==============================================================================
-- PROJETO NOVA — PERSONAS DE TESTE CORPORATIVAS (RONDONÓPOLIS - MT)
-- ==============================================================================
-- Associa os usuários criados no sistema às unidades e departamentos
-- reais de Rondonópolis, configurando lotações (user_scopes) e perfis do Diretório.
-- Idempotente: pode rodar a qualquer momento.
-- ==============================================================================

BEGIN;

-- 1. Identificadores auxiliares
DO $$
DECLARE
    -- Perfis
    v_perf_admin     UUID := '00ed46c5-20b1-49e2-8c4d-de7dadccf94e';
    v_perf_gestor_c  UUID := 'ebfb9c87-d3cb-4526-b6ff-d213f03d6803';
    v_perf_gestor_i  UUID := '90d1f1ad-84cc-4616-a42b-8274880bcafe';
    v_perf_protocolo UUID := '4e6e18eb-812e-4cd7-938d-7c6e471897f0';
    v_perf_servidor  UUID := '0c1b1e8d-e3be-4f73-a328-c210948e33f1';

    -- Entidades
    v_ent_nova      UUID := '10000000-0000-0000-0000-000000000001';
    v_ent_agrotech   UUID := '10000000-0000-0000-0000-000000000002';

    -- Unidades
    v_unid_centro    UUID := '20000000-0000-0000-0000-000000000001';
    v_unid_aurora    UUID := '20000000-0000-0000-0000-000000000002';
    v_unid_polo      UUID := '20000000-0000-0000-0000-000000000003';
    v_unid_poguba    UUID := '20000000-0000-0000-0000-000000000004';
    v_unid_agro      UUID := '20000000-0000-0000-0000-000000000005';

    -- Departamentos
    v_dep_dir_exec   UUID := '30000000-0000-0000-0000-000000000001';
    v_dep_rh         UUID := '30000000-0000-0000-0000-000000000002';
    v_dep_mkt        UUID := '30000000-0000-0000-0000-000000000006';
    v_dep_dev        UUID := '30000000-0000-0000-0000-000000000008';
    v_dep_ops        UUID := '30000000-0000-0000-0000-000000000013';
    v_dep_prot       UUID := '30000000-0000-0000-0000-000000000018';
    v_dep_intermodal UUID := '30000000-0000-0000-0000-000000000019';

    -- Usuários
    v_u_carlos       UUID;
    v_u_mariana      UUID;
    v_u_lucas        UUID;
    v_u_marcos       UUID;
    v_u_ana          UUID;
    v_u_rodrigo      UUID;
BEGIN
    SELECT id INTO v_u_carlos   FROM users WHERE username = 'carlos.diretoria';
    SELECT id INTO v_u_mariana  FROM users WHERE username = 'mariana.mkt';
    SELECT id INTO v_u_lucas    FROM users WHERE username = 'lucas.tech';
    SELECT id INTO v_u_marcos   FROM users WHERE username = 'marcos.ops';
    SELECT id INTO v_u_ana      FROM users WHERE username = 'ana.protocolo';
    SELECT id INTO v_u_rodrigo  FROM users WHERE username = 'rodrigo.agro';

    -- =========================================================================
    -- PERSONA 1: Carlos Mendes (Diretoria Executiva — Sede Centro)
    -- =========================================================================
    IF v_u_carlos IS NOT NULL THEN
        INSERT INTO directory_profiles (user_id, job_title, phone, extension, unidade_id, departamento_id, bio, visible)
        VALUES (v_u_carlos, 'Diretor Executivo & Relações Institucionais', '(66) 3411-9000', '9001', v_unid_centro, v_dep_dir_exec, 'Liderança executiva da Nova Corporation em Rondonópolis e expansão regional no Centro-Oeste.', true)
        ON CONFLICT (user_id) DO UPDATE SET
            job_title = EXCLUDED.job_title, phone = EXCLUDED.phone, extension = EXCLUDED.extension,
            unidade_id = EXCLUDED.unidade_id, departamento_id = EXCLUDED.departamento_id, bio = EXCLUDED.bio, visible = true;

        DELETE FROM user_scopes WHERE user_id = v_u_carlos;
        INSERT INTO user_scopes (user_id, perfil_id, entidade_id, unidade_id, departamento_id, principal, created_by)
        VALUES (v_u_carlos, v_perf_admin, v_ent_nova, v_unid_centro, NULL, true, 'seed-personas'),
               (v_u_carlos, v_perf_gestor_i, v_ent_nova, NULL, NULL, false, 'seed-personas');
    END IF;

    -- =========================================================================
    -- PERSONA 2: Mariana Silva (Comunicação & Conteúdo — Sede Centro)
    -- =========================================================================
    IF v_u_mariana IS NOT NULL THEN
        INSERT INTO directory_profiles (user_id, job_title, phone, extension, unidade_id, departamento_id, bio, visible)
        VALUES (v_u_mariana, 'Coordenadora de Comunicação & Marketing', '(66) 3411-9000', '9006', v_unid_centro, v_dep_mkt, 'Responsável pela comunicação institucional, branding e publicações editoriais do Nova Blog.', true)
        ON CONFLICT (user_id) DO UPDATE SET
            job_title = EXCLUDED.job_title, phone = EXCLUDED.phone, extension = EXCLUDED.extension,
            unidade_id = EXCLUDED.unidade_id, departamento_id = EXCLUDED.departamento_id, bio = EXCLUDED.bio, visible = true;

        DELETE FROM user_scopes WHERE user_id = v_u_mariana;
        INSERT INTO user_scopes (user_id, perfil_id, entidade_id, unidade_id, departamento_id, principal, created_by)
        VALUES (v_u_mariana, v_perf_gestor_c, v_ent_nova, NULL, NULL, true, 'seed-personas'),
               (v_u_mariana, v_perf_servidor, v_ent_nova, v_unid_centro, v_dep_mkt, false, 'seed-personas');
    END IF;

    -- =========================================================================
    -- PERSONA 3: Lucas Albuquerque (Tech Lead & P&D — Hub Aurora)
    -- =========================================================================
    IF v_u_lucas IS NOT NULL THEN
        INSERT INTO directory_profiles (user_id, job_title, phone, extension, unidade_id, departamento_id, bio, visible)
        VALUES (v_u_lucas, 'Tech Lead & Engenheiro de Software P&D', '(66) 3422-5500', '5508', v_unid_aurora, v_dep_dev, 'Engenharia de software, microsserviços e soluções agro-tecnológicas no Hub de Inovação da Vila Aurora.', true)
        ON CONFLICT (user_id) DO UPDATE SET
            job_title = EXCLUDED.job_title, phone = EXCLUDED.phone, extension = EXCLUDED.extension,
            unidade_id = EXCLUDED.unidade_id, departamento_id = EXCLUDED.departamento_id, bio = EXCLUDED.bio, visible = true;

        DELETE FROM user_scopes WHERE user_id = v_u_lucas;
        INSERT INTO user_scopes (user_id, perfil_id, entidade_id, unidade_id, departamento_id, principal, created_by)
        VALUES (v_u_lucas, v_perf_servidor, v_ent_agrotech, v_unid_aurora, v_dep_dev, true, 'seed-personas'),
               (v_u_lucas, v_perf_gestor_c, v_ent_agrotech, v_unid_aurora, NULL, false, 'seed-personas');
    END IF;

    -- =========================================================================
    -- PERSONA 4: Marcos Roberto Souza (Operações & Logística — Distrito Industrial)
    -- =========================================================================
    IF v_u_marcos IS NOT NULL THEN
        INSERT INTO directory_profiles (user_id, job_title, phone, extension, unidade_id, departamento_id, bio, visible)
        VALUES (v_u_marcos, 'Supervisor de Operações & Logística Regional', '(66) 3439-8800', '8813', v_unid_polo, v_dep_ops, 'Gestão operacional e logística no Distrito Industrial Veterinário de Rondonópolis (BR-364).', true)
        ON CONFLICT (user_id) DO UPDATE SET
            job_title = EXCLUDED.job_title, phone = EXCLUDED.phone, extension = EXCLUDED.extension,
            unidade_id = EXCLUDED.unidade_id, departamento_id = EXCLUDED.departamento_id, bio = EXCLUDED.bio, visible = true;

        DELETE FROM user_scopes WHERE user_id = v_u_marcos;
        INSERT INTO user_scopes (user_id, perfil_id, entidade_id, unidade_id, departamento_id, principal, created_by)
        VALUES (v_u_marcos, v_perf_servidor, v_ent_nova, v_unid_polo, v_dep_ops, true, 'seed-personas'),
               (v_u_marcos, v_perf_protocolo, v_ent_nova, v_unid_polo, NULL, false, 'seed-personas');
    END IF;

    -- =========================================================================
    -- PERSONA 5: Ana Paula Ferreira (Protocolo & Atendimento — CIA Sagrada Família / Poguba)
    -- =========================================================================
    IF v_u_ana IS NOT NULL THEN
        INSERT INTO directory_profiles (user_id, job_title, phone, extension, unidade_id, departamento_id, bio, visible)
        VALUES (v_u_ana, 'Analista de Protocolo Central & Atendimento', '(66) 3425-4100', '4118', v_unid_poguba, v_dep_prot, 'Triagem de processos, abertura de protocolos e atendimento integrado no CIA Poguba / Sagrada Família.', true)
        ON CONFLICT (user_id) DO UPDATE SET
            job_title = EXCLUDED.job_title, phone = EXCLUDED.phone, extension = EXCLUDED.extension,
            unidade_id = EXCLUDED.unidade_id, departamento_id = EXCLUDED.departamento_id, bio = EXCLUDED.bio, visible = true;

        DELETE FROM user_scopes WHERE user_id = v_u_ana;
        INSERT INTO user_scopes (user_id, perfil_id, entidade_id, unidade_id, departamento_id, principal, created_by)
        VALUES (v_u_ana, v_perf_protocolo, v_ent_nova, v_unid_poguba, NULL, true, 'seed-personas'),
               (v_u_ana, v_perf_servidor, v_ent_nova, v_unid_poguba, v_dep_prot, false, 'seed-personas');
    END IF;

    -- =========================================================================
    -- PERSONA 6: Rodrigo Camargo (Operações Intermodais — Terminal Agro / Anel Viário)
    -- =========================================================================
    IF v_u_rodrigo IS NOT NULL THEN
        INSERT INTO directory_profiles (user_id, job_title, phone, extension, unidade_id, departamento_id, bio, visible)
        VALUES (v_u_rodrigo, 'Supervisor Operacional Intermodal', '(66) 3427-1200', '1219', v_unid_agro, v_dep_intermodal, 'Operações ferroviárias e escoamento agroindustrial no Terminal do Anel Viário de Rondonópolis.', true)
        ON CONFLICT (user_id) DO UPDATE SET
            job_title = EXCLUDED.job_title, phone = EXCLUDED.phone, extension = EXCLUDED.extension,
            unidade_id = EXCLUDED.unidade_id, departamento_id = EXCLUDED.departamento_id, bio = EXCLUDED.bio, visible = true;

        DELETE FROM user_scopes WHERE user_id = v_u_rodrigo;
        INSERT INTO user_scopes (user_id, perfil_id, entidade_id, unidade_id, departamento_id, principal, created_by)
        VALUES (v_u_rodrigo, v_perf_servidor, v_ent_agrotech, v_unid_agro, v_dep_intermodal, true, 'seed-personas');
    END IF;

END $$;

COMMIT;
