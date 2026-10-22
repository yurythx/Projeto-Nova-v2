-- ==============================================================================
-- PROJETO NEXUS — SEMENTE DE SALAS E EVENTOS DA AGENDA (RONDONÓPOLIS - MT)
-- ==============================================================================
-- Configura salas de reunião corporativas reais distribuídas entre as
-- 5 unidades de Rondonópolis e agenda eventos/reservas com as personas.
-- Idempotente: pode rodar a qualquer momento.
-- ==============================================================================

BEGIN;

-- 1. Criação das Salas de Reunião Corporativas
INSERT INTO calendar_rooms (id, name, location, capacity, resources, active, unidade_id)
VALUES
(
    '50000000-0000-0000-0000-000000000001',
    'Auditório Marechal Rondon',
    'Sede Centro - Bloco A (Térreo)',
    120,
    ARRAY['projetor_4k', 'sistema_som', 'videoconferencia', 'ar_condicionado', 'cabine_traducao'],
    true,
    '20000000-0000-0000-0000-000000000001'
),
(
    '50000000-0000-0000-0000-000000000002',
    'Sala Executiva Pantanal',
    'Sede Centro - 2º Andar (Diretoria)',
    16,
    ARRAY['tv_85_polegadas', 'videoconferencia_cisco', 'ar_condicionado', 'mesa_executiva', 'frigobar'],
    true,
    '20000000-0000-0000-0000-000000000001'
),
(
    '50000000-0000-0000-0000-000000000003',
    'Espaço Inovação & Scrum Hub',
    'Hub Vila Aurora - 1º Andar (Coworking)',
    24,
    ARRAY['lousa_digital', 'videoconferencia', 'ar_condicionado', 'mesas_modulares'],
    true,
    '20000000-0000-0000-0000-000000000002'
),
(
    '50000000-0000-0000-0000-000000000004',
    'Lab de Pesquisa & IoT AgroTech',
    'Hub Vila Aurora - Laboratório 02',
    12,
    ARRAY['bancada_eletronica', 'monitores_ultrawide', 'ar_condicionado', 'impressora_3d'],
    true,
    '20000000-0000-0000-0000-000000000002'
),
(
    '50000000-0000-0000-0000-000000000005',
    'Sala de Operações & Logística Polo',
    'Distrito Industrial - Prédio Administrativo',
    18,
    ARRAY['painel_monitoramento', 'videoconferencia', 'ar_condicionado', 'mesa_reuniao'],
    true,
    '20000000-0000-0000-0000-000000000003'
),
(
    '50000000-0000-0000-0000-000000000006',
    'Sala de Treinamento CIA Poguba',
    'CIA Sagrada Família - Ala de Capacitação',
    30,
    ARRAY['projetor', 'som', 'ar_condicionado', 'carteiras_universitarias'],
    true,
    '20000000-0000-0000-0000-000000000004'
),
(
    '50000000-0000-0000-0000-000000000007',
    'Sala de Briefing do Terminal Agro',
    'Terminal Ferroviário - Prédio da Balança',
    10,
    ARRAY['tv_55_polegadas', 'ar_condicionado', 'radio_vhf', 'computador_balanca'],
    true,
    '20000000-0000-0000-0000-000000000005'
)
ON CONFLICT (name) DO UPDATE SET
    location   = EXCLUDED.location,
    capacity   = EXCLUDED.capacity,
    resources  = EXCLUDED.resources,
    active     = true,
    unidade_id = EXCLUDED.unidade_id;

-- 2. Criação de Eventos e Reservas Dinâmicas
DO $$
DECLARE
    v_u_carlos   UUID;
    v_u_mariana  UUID;
    v_u_lucas    UUID;
    v_u_marcos   UUID;
    v_u_ana      UUID;

    v_ev1_id     UUID := '60000000-0000-0000-0000-000000000001';
    v_ev2_id     UUID := '60000000-0000-0000-0000-000000000002';
    v_ev3_id     UUID := '60000000-0000-0000-0000-000000000003';
    v_ev4_id     UUID := '60000000-0000-0000-0000-000000000004';
    v_ev5_id     UUID := '60000000-0000-0000-0000-000000000005';

    v_unid_aurora UUID := '20000000-0000-0000-0000-000000000002';
BEGIN
    SELECT id INTO v_u_carlos  FROM users WHERE username = 'carlos.diretoria';
    SELECT id INTO v_u_mariana FROM users WHERE username = 'mariana.mkt';
    SELECT id INTO v_u_lucas   FROM users WHERE username = 'lucas.tech';
    SELECT id INTO v_u_marcos  FROM users WHERE username = 'marcos.ops';
    SELECT id INTO v_u_ana     FROM users WHERE username = 'ana.protocolo';

    -- EVENTO 1: Reunião do Comitê Executivo Nexus (Sede Centro)
    INSERT INTO calendar_events (
        id, title, description, location, room_id,
        starts_at, ends_at, all_day, visibility, status, organizer_id
    ) VALUES (
        v_ev1_id,
        'Reunião do Comitê Executivo Nexus Corporation',
        'Alinhamento trimestral de metas, governança corporativa e expansão dos novos polos no Centro-Oeste.',
        'Sede Centro - Sala Executiva Pantanal',
        '50000000-0000-0000-0000-000000000002',
        (CURRENT_DATE + interval '1 day' + time '09:00:00') AT TIME ZONE 'UTC',
        (CURRENT_DATE + interval '1 day' + time '11:30:00') AT TIME ZONE 'UTC',
        false, 'internal', 'confirmed', v_u_carlos
    ) ON CONFLICT (id) DO UPDATE SET
        title = EXCLUDED.title, description = EXCLUDED.description,
        starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at,
        status = 'confirmed';

    -- EVENTO 2: Sprint Review & Demo AgroTech (Hub Aurora — Restrito à Unidade)
    INSERT INTO calendar_events (
        id, title, description, location, room_id,
        starts_at, ends_at, all_day, visibility, status, organizer_id
    ) VALUES (
        v_ev2_id,
        'Sprint Review: Plataforma de Telemetria e IA AgroTech',
        'Apresentação dos novos microsserviços de visão computacional e telemetria de colheita para a equipe de P&D.',
        'Hub Vila Aurora - Espaço Inovação & Scrum Hub',
        '50000000-0000-0000-0000-000000000003',
        (CURRENT_DATE + interval '2 days' + time '14:00:00') AT TIME ZONE 'UTC',
        (CURRENT_DATE + interval '2 days' + time '16:00:00') AT TIME ZONE 'UTC',
        false, 'internal', 'confirmed', v_u_lucas
    ) ON CONFLICT (id) DO UPDATE SET
        title = EXCLUDED.title, description = EXCLUDED.description,
        starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at,
        status = 'confirmed';

    -- Público-alvo do Evento 2: Restrito ao Hub Aurora (ADR 014)
    DELETE FROM calendar_event_publico WHERE event_id = v_ev2_id;
    INSERT INTO calendar_event_publico (event_id, entidade_id, unidade_id)
    VALUES (v_ev2_id, NULL, v_unid_aurora);

    -- EVENTO 3: Alinhamento de Frotas e Escala da Safra (Polo Industrial)
    INSERT INTO calendar_events (
        id, title, description, location, room_id,
        starts_at, ends_at, all_day, visibility, status, organizer_id
    ) VALUES (
        v_ev3_id,
        'Alinhamento Operacional: Logística e Manutenção de Frotas',
        'Coordenação das janelas de revisão de bitrens e agendamento de chegada de caminhões no terminal ferroviário.',
        'Distrito Industrial - Sala de Operações & Logística Polo',
        '50000000-0000-0000-0000-000000000005',
        (CURRENT_DATE + interval '3 days' + time '08:30:00') AT TIME ZONE 'UTC',
        (CURRENT_DATE + interval '3 days' + time '10:30:00') AT TIME ZONE 'UTC',
        false, 'internal', 'confirmed', v_u_marcos
    ) ON CONFLICT (id) DO UPDATE SET
        title = EXCLUDED.title, description = EXCLUDED.description,
        starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at,
        status = 'confirmed';

    -- EVENTO 4: Treinamento de Atendimento e LGPD (CIA Poguba)
    INSERT INTO calendar_events (
        id, title, description, location, room_id,
        starts_at, ends_at, all_day, visibility, status, organizer_id
    ) VALUES (
        v_ev4_id,
        'Workshop de Boas Práticas: Atendimento Integrado e LGPD',
        'Capacitação da equipe de atendimento presencial e protocolo nos novos padrões de privacidade e segurança documental.',
        'CIA Poguba - Sala de Treinamento',
        '50000000-0000-0000-0000-000000000006',
        (CURRENT_DATE + interval '4 days' + time '13:30:00') AT TIME ZONE 'UTC',
        (CURRENT_DATE + interval '4 days' + time '17:00:00') AT TIME ZONE 'UTC',
        false, 'internal', 'confirmed', v_u_ana
    ) ON CONFLICT (id) DO UPDATE SET
        title = EXCLUDED.title, description = EXCLUDED.description,
        starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at,
        status = 'confirmed';

    -- EVENTO 5: Workshop Aberto: Inovação no Agronegócio (Auditório Sede Centro — Público)
    INSERT INTO calendar_events (
        id, title, description, location, room_id,
        starts_at, ends_at, all_day, visibility, status, organizer_id
    ) VALUES (
        v_ev5_id,
        'Workshop Nexus: O Futuro da Tecnologia no Campo',
        'Painel aberto com cooperativas e produtores rurais de Rondonópolis sobre inteligência artificial e conectividade agrícola.',
        'Sede Centro - Auditório Marechal Rondon',
        '50000000-0000-0000-0000-000000000001',
        (CURRENT_DATE + interval '7 days' + time '19:00:00') AT TIME ZONE 'UTC',
        (CURRENT_DATE + interval '7 days' + time '21:30:00') AT TIME ZONE 'UTC',
        false, 'public', 'confirmed', v_u_mariana
    ) ON CONFLICT (id) DO UPDATE SET
        title = EXCLUDED.title, description = EXCLUDED.description,
        starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at,
        status = 'confirmed';

END $$;

COMMIT;
