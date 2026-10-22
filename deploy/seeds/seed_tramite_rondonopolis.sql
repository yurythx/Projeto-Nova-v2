-- ==============================================================================
-- PROJETO NOVA — SEMENTE DE PROCESSOS ELETRÔNICOS E TRÂMITE (RONDONÓPOLIS)
-- ==============================================================================
-- Cria processos realistas tramitando entre as 5 unidades corporativas de
-- Rondonópolis, com documentos redigidos e histórico de movimentações das personas.
-- Idempotente: pode rodar a qualquer momento sem duplicar registros.
-- ==============================================================================

BEGIN;

-- Ajusta a numeração sequencial anual
INSERT INTO tramite_numeracao (ano, ultimo)
VALUES (2026, 4)
ON CONFLICT (ano) DO UPDATE SET ultimo = GREATEST(tramite_numeracao.ultimo, 4);

DO $$
DECLARE
    -- Tipos
    v_tipo_contratacao UUID := '19e22959-128e-4001-9cb0-a481e3010fd7';
    v_tipo_requerimento UUID := '951b5bad-bcce-45db-8054-e4771c23a77e';
    v_tipo_memorando    UUID := '19404951-ca95-4848-ae27-53029c0f3d70';

    -- Unidades
    v_unid_centro    UUID := '20000000-0000-0000-0000-000000000001';
    v_unid_aurora    UUID := '20000000-0000-0000-0000-000000000002';
    v_unid_polo      UUID := '20000000-0000-0000-0000-000000000003';
    v_unid_poguba    UUID := '20000000-0000-0000-0000-000000000004';
    v_unid_agro      UUID := '20000000-0000-0000-0000-000000000005';

    -- Usuários
    v_u_carlos       UUID;
    v_u_mariana      UUID;
    v_u_lucas        UUID;
    v_u_marcos       UUID;
    v_u_ana          UUID;
    v_u_rodrigo      UUID;

    -- Processos
    v_proc1_id       UUID := '40000000-0000-0000-0000-000000000001';
    v_proc2_id       UUID := '40000000-0000-0000-0000-000000000002';
    v_proc3_id       UUID := '40000000-0000-0000-0000-000000000003';
    v_proc4_id       UUID := '40000000-0000-0000-0000-000000000004';

BEGIN
    SELECT id INTO v_u_carlos   FROM users WHERE username = 'carlos.diretoria';
    SELECT id INTO v_u_mariana  FROM users WHERE username = 'mariana.mkt';
    SELECT id INTO v_u_lucas    FROM users WHERE username = 'lucas.tech';
    SELECT id INTO v_u_marcos   FROM users WHERE username = 'marcos.ops';
    SELECT id INTO v_u_ana      FROM users WHERE username = 'ana.protocolo';
    SELECT id INTO v_u_rodrigo  FROM users WHERE username = 'rodrigo.agro';

    -- =========================================================================
    -- PROCESSO 1: Hub Aurora -> Sede Centro (Servidores de IA e AgroTech)
    -- =========================================================================
    IF NOT EXISTS (SELECT 1 FROM tramite_processos WHERE id = v_proc1_id) THEN
        INSERT INTO tramite_processos (
            id, numero, tipo_id, assunto, interessado, descricao, sigilo, status,
            unidade_origem_id, unidade_atual_id, created_by, created_at, updated_at
        ) VALUES (
            v_proc1_id, '000001/2026', v_tipo_contratacao,
            'Expansão de Infraestrutura de Servidores para o Laboratório AgroTech',
            'Nova AgroTech & Soluções Digitais',
            'Aquisição de servidores de alta densidade para processamento em tempo real de telemetria de frotas e IA preditiva no Hub da Vila Aurora.',
            'publico', 'em_tramitacao',
            v_unid_aurora, v_unid_centro, v_u_lucas, now() - interval '3 days', now() - interval '1 day'
        );

        -- Documentos
        INSERT INTO tramite_documentos (id, processo_id, tipo, titulo, origem, conteudo, status, created_by, created_at)
        VALUES (
            gen_random_uuid(), v_proc1_id, 'memorando',
            'Memorando Técnico nº 12/2026 - Especificação de Servidores',
            'redigido',
            '# Especificação Técnica de Infraestrutura\n\n- **Ambiente:** Hub de Inovação & P&D - Vila Aurora\n- **Carga:** Processamento de modelos de visão computacional para detecção de pragas e telemetria de colheita.\n- **Configuração:** 2x Nós Rack 2U com aceleradores GPU e 256GB de RAM.\n\nSolicitamos deliberação orçamentária da Diretoria Executiva.',
            'assinado', v_u_lucas, now() - interval '3 days'
        ),
        (
            gen_random_uuid(), v_proc1_id, 'despacho',
            'Despacho Decisório nº 44/2026 - Parecer Favorável da Diretoria',
            'redigido',
            '# Despacho da Presidência & Diretoria Executiva\n\nAprovo a dotação orçamentária no valor de R$ 180.000,00 para a expansão do Hub de Inovação. Encaminho ao departamento de Compras e Suprimentos da Sede Centro para abertura de cotação.',
            'assinado', v_u_carlos, now() - interval '1 day'
        );

        -- Movimentações históricas
        INSERT INTO tramite_movimentos (id, processo_id, acao, de_unidade_id, para_unidade_id, despacho, actor_id, created_at)
        VALUES
        (
            gen_random_uuid(), v_proc1_id, 'abertura', NULL, v_unid_aurora,
            'Processo autuado pelo Tech Lead Lucas Albuquerque no Hub de Inovação da Vila Aurora.',
            v_u_lucas, now() - interval '3 days'
        ),
        (
            gen_random_uuid(), v_proc1_id, 'documento', v_unid_aurora, v_unid_aurora,
            'Documento juntado: Memorando Técnico nº 12/2026.',
            v_u_lucas, now() - interval '3 days'
        ),
        (
            gen_random_uuid(), v_proc1_id, 'tramitacao', v_unid_aurora, v_unid_centro,
            'Encaminhado à Diretoria Executiva na Sede Centro para aprovação orçamentária.',
            v_u_lucas, now() - interval '2 days'
        ),
        (
            gen_random_uuid(), v_proc1_id, 'documento', v_unid_centro, v_unid_centro,
            'Documento juntado: Despacho Decisório nº 44/2026.',
            v_u_carlos, now() - interval '1 day'
        );
    END IF;

    -- =========================================================================
    -- PROCESSO 2: Polo Industrial -> Terminal Agro (Logística e Frotas)
    -- =========================================================================
    IF NOT EXISTS (SELECT 1 FROM tramite_processos WHERE id = v_proc2_id) THEN
        INSERT INTO tramite_processos (
            id, numero, tipo_id, assunto, interessado, descricao, sigilo, status,
            unidade_origem_id, unidade_atual_id, created_by, created_at, updated_at
        ) VALUES (
            v_proc2_id, '000002/2026', v_tipo_requerimento,
            'Revisão e Manutenção Preventiva da Frota Bitrem (Safra 2026/2027)',
            'Departamento de Gestão de Frotas & Transporte Rodoviário',
            'Cronograma de manutenção mecânica preventiva e alinhamento de frotas rodoviárias que operam no corredor logístico da BR-364.',
            'publico', 'em_tramitacao',
            v_unid_polo, v_unid_agro, v_u_marcos, now() - interval '5 days', now() - interval '2 days'
        );

        -- Documentos
        INSERT INTO tramite_documentos (id, processo_id, tipo, titulo, origem, conteudo, status, created_by, created_at)
        VALUES (
            gen_random_uuid(), v_proc2_id, 'despacho',
            'Laudo de Vistoria de Frotas nº 08/2026',
            'redigido',
            '# Laudo Operacional de Vistoria Mecânica\n\nForam inspecionadas 18 carretas bitrem no pátio do Distrito Industrial Veterinário. Necessidade de substituição de freios e pneus em 4 unidades antes do pico do escoamento de grãos.',
            'assinado', v_u_marcos, now() - interval '5 days'
        ),
        (
            gen_random_uuid(), v_proc2_id, 'despacho',
            'Parecer Técnico do Terminal Intermodal nº 19/2026',
            'redigido',
            '# Alinhamento da Janela Ferroviária\n\nConfirmamos o recebimento. Solicitamos que as manutenções sejam realizadas entre terça e quinta-feira, período com menor fila na balança de pesagem do Anel Viário.',
            'assinado', v_u_rodrigo, now() - interval '2 days'
        );

        -- Movimentações
        INSERT INTO tramite_movimentos (id, processo_id, acao, de_unidade_id, para_unidade_id, despacho, actor_id, created_at)
        VALUES
        (
            gen_random_uuid(), v_proc2_id, 'abertura', NULL, v_unid_polo,
            'Processo autuado no Complexo Operacional do Distrito Industrial por Marcos Roberto Souza.',
            v_u_marcos, now() - interval '5 days'
        ),
        (
            gen_random_uuid(), v_proc2_id, 'documento', v_unid_polo, v_unid_polo,
            'Documento juntado: Laudo de Vistoria de Frotas nº 08/2026.',
            v_u_marcos, now() - interval '5 days'
        ),
        (
            gen_random_uuid(), v_proc2_id, 'tramitacao', v_unid_polo, v_unid_agro,
            'Encaminhado ao Terminal Intermodal para alinhamento com a balança ferroviária.',
            v_u_marcos, now() - interval '4 days'
        ),
        (
            gen_random_uuid(), v_proc2_id, 'documento', v_unid_agro, v_unid_agro,
            'Documento juntado: Parecer Técnico do Terminal Intermodal nº 19/2026.',
            v_u_rodrigo, now() - interval '2 days'
        );
    END IF;

    -- =========================================================================
    -- PROCESSO 3: CIA Poguba -> Hub Aurora (Integração de Cooperativa)
    -- =========================================================================
    IF NOT EXISTS (SELECT 1 FROM tramite_processos WHERE id = v_proc3_id) THEN
        INSERT INTO tramite_processos (
            id, numero, tipo_id, assunto, interessado, descricao, sigilo, status,
            unidade_origem_id, unidade_atual_id, created_by, created_at, updated_at
        ) VALUES (
            v_proc3_id, '000003/2026', v_tipo_requerimento,
            'Cadastro e Integração de Cooperativa Agropecuária no Portal Nova',
            'Cooperativa Agrícola do Sul de Mato Grosso (COOPERSUL)',
            'Solicitação de convênio e credenciamento digital para integração de dados de rastreabilidade de grãos e agendamento de descargas no terminal.',
            'publico', 'em_tramitacao',
            v_unid_poguba, v_unid_aurora, v_u_ana, now() - interval '2 days', now() - interval '4 hours'
        );

        -- Documentos
        INSERT INTO tramite_documentos (id, processo_id, tipo, titulo, origem, conteudo, status, created_by, created_at)
        VALUES (
            gen_random_uuid(), v_proc3_id, 'requerimento',
            'Formulário de Triagem e Solicitação de Convênio Digital',
            'redigido',
            '# Termo de Atendimento e Triagem\n\n- **Solicitante:** Cooperativa COOPERSUL (CNPJ: 04.992.118/0001-50)\n- **Representante:** João Batista Silva\n- **Demanda:** Integração via Webhook/API com o Portal Nova para envio automatizado de romaneios de entrada.',
            'assinado', v_u_ana, now() - interval '2 days'
        );

        -- Movimentações
        INSERT INTO tramite_movimentos (id, processo_id, acao, de_unidade_id, para_unidade_id, despacho, actor_id, created_at)
        VALUES
        (
            gen_random_uuid(), v_proc3_id, 'abertura', NULL, v_unid_poguba,
            'Atendimento presencial realizado no Centro Integrado de Atendimento Poguba por Ana Paula Ferreira.',
            v_u_ana, now() - interval '2 days'
        ),
        (
            gen_random_uuid(), v_proc3_id, 'documento', v_unid_poguba, v_unid_poguba,
            'Documento juntado: Formulário de Triagem e Solicitação de Convênio Digital.',
            v_u_ana, now() - interval '2 days'
        ),
        (
            gen_random_uuid(), v_proc3_id, 'tramitacao', v_unid_poguba, v_unid_aurora,
            'Encaminhado à equipe de Engenharia de Software no Hub da Vila Aurora para homologação técnica e emissão de credenciais.',
            v_u_ana, now() - interval '4 hours'
        );
    END IF;

    -- =========================================================================
    -- PROCESSO 4: Sede Centro (Marketing & Comunicação Exposul 2026 - Concluído)
    -- =========================================================================
    IF NOT EXISTS (SELECT 1 FROM tramite_processos WHERE id = v_proc4_id) THEN
        INSERT INTO tramite_processos (
            id, numero, tipo_id, assunto, interessado, descricao, sigilo, status,
            unidade_origem_id, unidade_atual_id, created_by, created_at, updated_at, concluido_at
        ) VALUES (
            v_proc4_id, '000004/2026', v_tipo_memorando,
            'Aprovação de Campanha Institucional e Stand da Nova na Exposul 2026',
            'Departamento de Marketing & Comunicação Institucional',
            'Planejamento de comunicação, identidade visual, estande corporativo e divulgação de soluções agro-digitais na maior feira agropecuária do sul do estado.',
            'publico', 'concluido',
            v_unid_centro, v_unid_centro, v_u_mariana, now() - interval '10 days', now() - interval '1 day', now() - interval '1 day'
        );

        -- Documentos
        INSERT INTO tramite_documentos (id, processo_id, tipo, titulo, origem, conteudo, status, created_by, created_at)
        VALUES (
            gen_random_uuid(), v_proc4_id, 'memorando',
            'Plano de Mídia e Projeto Arquitetônico do Estande Exposul',
            'redigido',
            '# Plano de Ação - Exposul Rondonópolis 2026\n\n- **Local:** Parque de Exposições Wilmar Peres de Farias\n- **Ações:** Demonstração interativa do aplicativo Nova AgroTech, palestras sobre segurança de dados no agronegócio e distribuição de brindes corporativos.',
            'assinado', v_u_mariana, now() - interval '10 days'
        ),
        (
            gen_random_uuid(), v_proc4_id, 'despacho',
            'Despacho de Homologação da Presidência',
            'redigido',
            '# Homologação e Conclusão\n\nPlano de marketing aprovado com louvor. Recursos liberados e contratações de cenografia concluídas.',
            'assinado', v_u_carlos, now() - interval '1 day'
        );

        -- Movimentações
        INSERT INTO tramite_movimentos (id, processo_id, acao, de_unidade_id, para_unidade_id, despacho, actor_id, created_at)
        VALUES
        (
            gen_random_uuid(), v_proc4_id, 'abertura', NULL, v_unid_centro,
            'Processo autuado por Mariana Silva na Sede Centro.',
            v_u_mariana, now() - interval '10 days'
        ),
        (
            gen_random_uuid(), v_proc4_id, 'documento', v_unid_centro, v_unid_centro,
            'Documento juntado: Plano de Mídia e Projeto Arquitetônico.',
            v_u_mariana, now() - interval '10 days'
        ),
        (
            gen_random_uuid(), v_proc4_id, 'documento', v_unid_centro, v_unid_centro,
            'Documento juntado: Despacho de Homologação da Presidência.',
            v_u_carlos, now() - interval '2 days'
        ),
        (
            gen_random_uuid(), v_proc4_id, 'conclusao', v_unid_centro, v_unid_centro,
            'Processo concluído com sucesso. Stand contratado e montagem agendada.',
            v_u_mariana, now() - interval '1 day'
        );
    END IF;

END $$;

COMMIT;
