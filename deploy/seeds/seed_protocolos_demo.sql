-- ==============================================================================
-- PROJETO NOVA — DADOS FICTÍCIOS DO PROTOCOLO DIGITAL (ADR 085)
-- ==============================================================================
-- Seis serviços com formulário próprio, responsáveis técnicos e 46 pedidos
-- espalhados pelos estados da mesa (novo, etapas do workflow, complementação,
-- resposta, concluído favorável e desfavorável, cancelado, aprovado sozinho,
-- agendamento e falta). Cobre cada recurso do plug-in:
--
--   Alvará de funcionamento ............ workflow de 3 etapas, taxa, CPF único
--   Poda ou remoção de árvore .......... sem workflow, mapa, denúncia anônima
--   Certidão negativa de débitos ....... aprova sozinho
--   Aprovação de projeto arquitetônico . só responsável técnico, área no mapa
--   Vacinação em domicílio (SMS) ....... agenda com grade de horários e faltas
--   Matrícula na rede municipal ........ workflow de 2 etapas, campo revelado
--
-- Nomes, CPFs, CNPJs, telefones e endereços são inventados. Os CPFs têm
-- dígito verificador válido, mas foram gerados; os e-mails são @example.com.
--
-- Unidades: usa as de sigla PREFEITURA, SMS e SEMECEL; sem elas, a primeira
-- unidade ativa. Os números dos pedidos terminam em 90001..90046.
--
-- Idempotente: serviços e formulários são atualizados; os pedidos do seed são
-- apagados e recriados com datas relativas a agora. Pedidos que alguém abriu
-- à mão nos serviços de teste ficam.
--
-- Remover tudo: deploy/seeds/remover_protocolos_demo.sql
-- ==============================================================================

BEGIN;

-- Dias úteis em Cuiabá, como domain.SomarDiasUteis (sábado e domingo fora).
CREATE FUNCTION pg_temp.du(inicio timestamptz, dias int) RETURNS timestamptz
LANGUAGE plpgsql AS $$
DECLARE
    atual timestamptz := inicio;
    restantes int := dias;
BEGIN
    WHILE restantes > 0 LOOP
        atual := atual + interval '1 day';
        IF extract(isodow FROM atual AT TIME ZONE 'America/Cuiaba') < 6 THEN
            restantes := restantes - 1;
        END IF;
    END LOOP;
    RETURN atual;
END $$;

-- CPF com dígito verificador válido a partir de uma semente.
CREATE FUNCTION pg_temp.cpf(semente int) RETURNS text
LANGUAGE plpgsql AS $$
DECLARE
    base text := lpad(((semente::bigint * 7919 + 104729) % 1000000000)::text, 9, '0');
    soma int; d1 int; d2 int; i int;
BEGIN
    soma := 0;
    FOR i IN 1..9 LOOP soma := soma + substr(base, i, 1)::int * (11 - i); END LOOP;
    d1 := (soma * 10) % 11; IF d1 = 10 THEN d1 := 0; END IF;
    soma := 0;
    FOR i IN 1..9 LOOP soma := soma + substr(base, i, 1)::int * (12 - i); END LOOP;
    soma := soma + d1 * 2;
    d2 := (soma * 10) % 11; IF d2 = 10 THEN d2 := 0; END IF;
    RETURN base || d1 || d2;
END $$;

-- Iniciais do número, como domain.Iniciais: a sigla (só letras e dígitos, até
-- quatro); sem sigla, a primeira letra de cada palavra do nome.
CREATE FUNCTION pg_temp.iniciais(sigla text, nome text) RETURNS text
LANGUAGE sql AS $$
    SELECT CASE
        WHEN coalesce(regexp_replace(upper(sigla), '[^[:alnum:]]', '', 'g'), '') <> ''
            THEN left(regexp_replace(upper(sigla), '[^[:alnum:]]', '', 'g'), 4)
        ELSE coalesce(nullif(left((SELECT string_agg(upper(left(p, 1)), '')
                                     FROM regexp_split_to_table(nome, '\s+') p WHERE p <> ''), 4), ''), 'PROT')
    END
$$;

DO $$
DECLARE
    -- Unidades reais pela sigla; sem elas, a primeira ativa.
    u_pref uuid := coalesce((SELECT id FROM unidades WHERE sigla = 'PREFEITURA' LIMIT 1),
                            (SELECT id FROM unidades WHERE ativo ORDER BY created_at LIMIT 1));
    u_sms  uuid := coalesce((SELECT id FROM unidades WHERE sigla = 'SMS' LIMIT 1), u_pref);
    u_edu  uuid := coalesce((SELECT id FROM unidades WHERE sigla = 'SEMECEL' LIMIT 1), u_pref);

    -- Formulários e serviços: ids fixos, reconhecíveis (d0c0de00-...).
    f_alvara    uuid := 'd0c0de00-0000-4000-8000-00000000f001';
    f_poda      uuid := 'd0c0de00-0000-4000-8000-00000000f002';
    f_certidao  uuid := 'd0c0de00-0000-4000-8000-00000000f003';
    f_projeto   uuid := 'd0c0de00-0000-4000-8000-00000000f004';
    f_vacina    uuid := 'd0c0de00-0000-4000-8000-00000000f005';
    f_matricula uuid := 'd0c0de00-0000-4000-8000-00000000f006';
    s_alvara    uuid := 'd0c0de00-0000-4000-8000-000000005e01';
    s_poda      uuid := 'd0c0de00-0000-4000-8000-000000005e02';
    s_certidao  uuid := 'd0c0de00-0000-4000-8000-000000005e03';
    s_projeto   uuid := 'd0c0de00-0000-4000-8000-000000005e04';
    s_vacina    uuid := 'd0c0de00-0000-4000-8000-000000005e05';
    s_matricula uuid := 'd0c0de00-0000-4000-8000-000000005e06';

    nomes text[] := ARRAY[
        'Ana Paula Ribeiro', 'Bruno Henrique Costa', 'Carla Fernanda Souza', 'Daniel Augusto Lima',
        'Eliane Cristina Barbosa', 'Fábio Rodrigues Nunes', 'Gabriela Martins Rocha', 'Heitor Almeida Pires',
        'Isabela Duarte Moreira', 'João Victor Cardoso', 'Karina Lopes Teixeira', 'Leonardo Batista Freitas',
        'Mariana Gomes Araújo', 'Nelson Pereira Campos', 'Olívia Santana Reis', 'Paulo César Mendes',
        'Quitéria Alves Monteiro', 'Rafael Oliveira Prado', 'Sandra Regina Fonseca', 'Thiago Moura Vieira',
        'Úrsula Machado Dias', 'Vinícius Castro Leal', 'Wanessa Ferreira Brito', 'Xavier Antunes Rezende',
        'Yasmin Correia Pacheco', 'Zeca Honório Bastos', 'Adriana Melo Sampaio', 'Benedito Silva Arruda',
        'Cláudia Ramos Siqueira', 'Diego Tavares Coelho', 'Edna Marques Queiroz', 'Francisco Lemos Paiva',
        'Glória Nogueira Valente', 'Hugo Rangel Cavalcante', 'Irene Sales Bittencourt', 'Jorge Amaral Fontes',
        'Lúcia Helena Macedo', 'Márcio Peixoto Galvão', 'Neide Andrade Furtado', 'Otávio Brandão Serra',
        'Patrícia Viana Lacerda', 'Renato Cunha Espíndola', 'Simone Paixão Toledo', 'Tadeu Morais Quintela',
        'Valéria Rios Albuquerque', 'Wellington Dantas Moraes'
    ];
    bairros text[] := ARRAY['Centro', 'Vila Aurora', 'Jardim Atlântico', 'Parque Sagrada Família',
                            'Vila Operária', 'Jardim Iguaçu', 'Cidade Salmen', 'Jardim Primavera',
                            'Vila Birigui', 'Parque São Jorge'];
    ruas text[] := ARRAY['Rua das Acácias', 'Avenida dos Ipês', 'Rua Projetada 12', 'Rua do Cerrado',
                         'Avenida das Palmeiras', 'Rua Bom Jardim', 'Travessa do Comércio', 'Rua Pantanal'];
    empresas text[] := ARRAY['Mercadinho Bom Preço Ltda', 'Oficina Rota Sul ME', 'Padaria Pão Dourado Ltda',
                             'Studio Bela Forma Eireli', 'Escritório Contábil Exato', 'Lanchonete Sabor do Cerrado',
                             'Ótica Visão Clara Ltda', 'Pet Shop Amigo Fiel ME', 'Distribuidora Vale Verde',
                             'Clínica Sorriso Pleno'];
    escolas text[] := ARRAY['EMEB Professora Aurora Lima', 'EMEB Monteiro Lobato', 'CMEI Pequeno Príncipe',
                            'EMEB Cecília Meireles', 'CMEI Sementinha'];

    -- Pedidos: n, serviço, dias atrás, cenário, prioridade.
    r record;
    v_id uuid; v_servico uuid; v_form uuid; v_unidade uuid; v_campos jsonb; v_wf jsonb;
    v_prazo_servico int; v_iniciais text; v_nome text; v_cpf text; v_email text; v_tel text;
    v_resp jsonb; v_criado timestamptz; v_t timestamptz; v_status text; v_etapa text;
    v_prazo timestamptz; v_decisao text; v_just text; v_comp text; v_etapas jsonb; v_e jsonb;
    v_passos int; v_i int; k int; v_rt text[];
    v_hist jsonb;  -- lista de [acao, descricao, status, momento]
    v_ag_data date; v_ag_hora text; v_ag_status text; v_hoje date;
BEGIN
    IF u_pref IS NULL THEN
        RAISE EXCEPTION 'nenhuma unidade ativa: aplique a estrutura organizacional antes';
    END IF;
    v_hoje := (now() AT TIME ZONE 'America/Cuiaba')::date;

    -- ---------------------------------------------------------------- formulários
    INSERT INTO protocolos_formularios (id, unidade_id, titulo, descricao, instrucao, campos) VALUES
    (f_alvara, u_pref, 'Requerimento de alvará de funcionamento',
     'Dados da empresa e do local onde a atividade vai funcionar.',
     'Tenha em mãos o CNPJ, o endereço completo e a área utilizada. O contrato social pode ser anexado agora ou quando a fiscalização pedir.',
     '[{"id":"pagina_empresa","tipo":"page","rotulo":"Dados da empresa","obrigatorio":false},
       {"id":"cnpj","tipo":"cnpj","rotulo":"CNPJ","obrigatorio":true,"placeholder":"00.000.000/0000-00"},
       {"id":"razao_social","tipo":"text","rotulo":"Razão social","obrigatorio":true},
       {"id":"atividade","tipo":"select","rotulo":"Atividade principal","obrigatorio":true,
        "opcoes":["Comércio varejista","Prestação de serviços","Alimentação","Indústria","Escritório administrativo"]},
       {"id":"pagina_local","tipo":"page","rotulo":"Local de funcionamento","obrigatorio":false},
       {"id":"endereco","tipo":"text","rotulo":"Endereço completo","obrigatorio":true},
       {"id":"cep","tipo":"cep","rotulo":"CEP","obrigatorio":true,"placeholder":"78700-000"},
       {"id":"area_m2","tipo":"number","rotulo":"Área utilizada (m²)","obrigatorio":true},
       {"id":"localizacao","tipo":"map","rotulo":"Localização no mapa","obrigatorio":false},
       {"id":"tem_placa","tipo":"boolean","rotulo":"Terá placa ou letreiro na fachada?","obrigatorio":false,"revela":["tamanho_placa"]},
       {"id":"tamanho_placa","tipo":"text","rotulo":"Dimensões da placa (largura x altura)","obrigatorio":true},
       {"id":"contrato_social","tipo":"file","rotulo":"Contrato social (PDF)","obrigatorio":false}]'),
    (f_poda, u_pref, 'Solicitação de poda ou remoção de árvore',
     'Árvore em via pública, praça ou calçada.',
     'Marque no mapa onde fica a árvore. Se preferir, o seu nome não aparece na consulta pública.',
     '[{"id":"endereco","tipo":"text","rotulo":"Endereço de referência","obrigatorio":true},
       {"id":"localizacao","tipo":"map","rotulo":"Onde fica a árvore","obrigatorio":true},
       {"id":"servico_desejado","tipo":"radio","rotulo":"O que precisa ser feito","obrigatorio":true,
        "opcoes":["Poda","Remoção","Avaliação técnica"]},
       {"id":"motivo","tipo":"checkbox_group","rotulo":"Motivo","obrigatorio":true,
        "opcoes":["Risco de queda","Conflito com a fiação","Danos à calçada","Árvore doente","Outro"]},
       {"id":"descricao","tipo":"textarea","rotulo":"Descreva a situação","obrigatorio":true},
       {"id":"foto","tipo":"file","rotulo":"Foto da árvore","obrigatorio":false},
       {"id":"anonimo","tipo":"boolean","rotulo":"Não quero que meu nome apareça na consulta pública","obrigatorio":false}]'),
    (f_certidao, u_pref, 'Pedido de certidão negativa de débitos municipais',
     'Certidão emitida para pessoa física ou jurídica.',
     'A certidão sai na hora quando não há débito. Ela chega pelo e-mail informado.',
     '[{"id":"tipo_pessoa","tipo":"radio","rotulo":"Contribuinte","obrigatorio":true,"opcoes":["Pessoa física","Pessoa jurídica"]},
       {"id":"documento","tipo":"text","rotulo":"CPF ou CNPJ do contribuinte","obrigatorio":true},
       {"id":"inscricao","tipo":"text","rotulo":"Inscrição imobiliária ou municipal","obrigatorio":false},
       {"id":"finalidade","tipo":"select","rotulo":"Finalidade","obrigatorio":true,
        "opcoes":["Licitação","Financiamento","Venda de imóvel","Outros"]},
       {"id":"email_envio","tipo":"email","rotulo":"E-mail para envio da certidão","obrigatorio":true}]'),
    (f_projeto, u_pref, 'Aprovação de projeto arquitetônico',
     'Projeto de construção, ampliação ou reforma.',
     'Só o responsável técnico cadastrado na Prefeitura pode protocolar. Informe o registro no conselho e a ART ou RRT.',
     '[{"id":"tipo_obra","tipo":"select","rotulo":"Tipo de obra","obrigatorio":true,
        "opcoes":["Residencial unifamiliar","Residencial multifamiliar","Comercial","Ampliação","Reforma"]},
       {"id":"area_construida","tipo":"number","rotulo":"Área construída (m²)","obrigatorio":true},
       {"id":"endereco","tipo":"text","rotulo":"Endereço da obra","obrigatorio":true},
       {"id":"area_lote","tipo":"map_area","rotulo":"Lote no mapa (dois cantos opostos)","obrigatorio":false},
       {"id":"registro_profissional","tipo":"text","rotulo":"Registro no CREA ou CAU","obrigatorio":true},
       {"id":"art_rrt","tipo":"text","rotulo":"Número da ART ou RRT","obrigatorio":true},
       {"id":"projeto","tipo":"file","rotulo":"Projeto (PDF)","obrigatorio":false},
       {"id":"observacoes","tipo":"textarea","rotulo":"Observações","obrigatorio":false}]'),
    (f_vacina, u_sms, 'Agendamento de vacinação em domicílio',
     'Para quem não consegue ir até a unidade de saúde.',
     'Escolha o horário. A equipe vai até o endereço informado; tenha o cartão de vacina em mãos.',
     '[{"id":"cartao_sus","tipo":"text","rotulo":"Cartão Nacional de Saúde (CNS)","obrigatorio":true},
       {"id":"data_nascimento","tipo":"date","rotulo":"Data de nascimento de quem vai ser vacinado","obrigatorio":true},
       {"id":"vacina","tipo":"select","rotulo":"Vacina","obrigatorio":true,
        "opcoes":["Influenza","COVID-19","Pneumocócica 23","Hepatite B","Dupla adulto (dT)"]},
       {"id":"endereco","tipo":"text","rotulo":"Endereço do atendimento","obrigatorio":true},
       {"id":"acamado","tipo":"boolean","rotulo":"A pessoa está acamada?","obrigatorio":false,"revela":["condicao"]},
       {"id":"condicao","tipo":"textarea","rotulo":"Descreva a condição de saúde","obrigatorio":true},
       {"id":"contato_alternativo","tipo":"phone","rotulo":"Telefone de outro contato","obrigatorio":false}]'),
    (f_matricula, u_edu, 'Pré-matrícula na rede municipal de ensino',
     'Educação infantil e anos iniciais do ensino fundamental.',
     'Preencha um pedido por criança. A vaga é confirmada pela Secretaria conforme a disponibilidade da escola.',
     '[{"id":"pagina_aluno","tipo":"page","rotulo":"Dados da criança","obrigatorio":false},
       {"id":"nome_aluno","tipo":"text","rotulo":"Nome completo da criança","obrigatorio":true},
       {"id":"data_nascimento_aluno","tipo":"date","rotulo":"Data de nascimento","obrigatorio":true},
       {"id":"cpf_aluno","tipo":"cpf","rotulo":"CPF da criança","obrigatorio":false},
       {"id":"pagina_vaga","tipo":"page","rotulo":"Vaga pretendida","obrigatorio":false},
       {"id":"serie","tipo":"select","rotulo":"Etapa","obrigatorio":true,
        "opcoes":["Berçário","Maternal","Pré I","Pré II","1º ano","2º ano","3º ano","4º ano","5º ano"]},
       {"id":"escola_pretendida","tipo":"select","rotulo":"Escola pretendida","obrigatorio":true,
        "opcoes":["EMEB Professora Aurora Lima","EMEB Monteiro Lobato","CMEI Pequeno Príncipe","EMEB Cecília Meireles","CMEI Sementinha"]},
       {"id":"turno","tipo":"checkbox_group","rotulo":"Turnos possíveis","obrigatorio":true,"opcoes":["Matutino","Vespertino","Integral"]},
       {"id":"possui_deficiencia","tipo":"boolean","rotulo":"A criança tem deficiência ou necessidade específica?","obrigatorio":false,"revela":["necessidade"]},
       {"id":"necessidade","tipo":"textarea","rotulo":"Qual necessidade e que apoio a escola precisa oferecer","obrigatorio":true},
       {"id":"certidao","tipo":"file","rotulo":"Certidão de nascimento","obrigatorio":false}]')
    ON CONFLICT (id) DO UPDATE SET
        unidade_id = EXCLUDED.unidade_id, titulo = EXCLUDED.titulo, descricao = EXCLUDED.descricao,
        instrucao = EXCLUDED.instrucao, campos = EXCLUDED.campos, ativo = true, updated_at = now();

    -- ------------------------------------------------------------------- serviços
    INSERT INTO protocolos_servicos (id, unidade_id, formulario_id, nome, descricao, categoria, prazo_dias,
        taxa_centavos, destaque, ativo, aprova_sozinho, exige_agendamento, so_responsavel_tecnico,
        impedir_cpf_aberto, campo_anonimo, workflow, agenda, requisitos, documentos) VALUES
    (s_alvara, u_pref, f_alvara, 'Alvará de funcionamento',
     'Licença para a empresa funcionar no endereço informado. Passa por triagem, vistoria e parecer da fiscalização.',
     'Empresas e tributos', 15, 18750, true, true, false, false, false, true, '',
     '{"ligado":true,"etapas":[
        {"id":"triagem","ordem":1,"status":"em_analise","prazo_dias":3,"responsavel_tipo":"","responsavel_id":""},
        {"id":"vistoria","ordem":2,"status":"em_andamento","prazo_dias":7,"responsavel_tipo":"","responsavel_id":""},
        {"id":"parecer","ordem":3,"status":"pending_approval","prazo_dias":5,"responsavel_tipo":"","responsavel_id":""}]}',
     '{}',
     '[{"titulo":"Inscrição municipal ativa","texto":"A empresa precisa estar inscrita no cadastro mobiliário.","obrigatorio":true},
       {"titulo":"Imóvel regular","texto":"O imóvel precisa ter habite-se ou estar em regularização.","obrigatorio":true}]',
     '[{"titulo":"Contrato social ou requerimento de empresário","texto":"Última alteração registrada na Junta Comercial.","obrigatorio":true},
       {"titulo":"Documento do responsável legal","texto":"RG e CPF, ou CNH.","obrigatorio":true},
       {"titulo":"Laudo do Corpo de Bombeiros","texto":"Só para área acima de 200 m² ou atividade de risco.","obrigatorio":false}]'),
    (s_poda, u_pref, f_poda, 'Poda ou remoção de árvore',
     'Avaliação e poda ou remoção de árvore em área pública. Pode ser feita sem expor o nome de quem pede.',
     'Meio ambiente', 20, 0, false, true, false, false, false, false, 'anonimo',
     '{"ligado":false,"etapas":[]}', '{}',
     '[{"titulo":"Árvore em área pública","texto":"Árvores dentro de imóvel particular pedem autorização ambiental, outro serviço.","obrigatorio":true}]',
     '[]'),
    (s_certidao, u_pref, f_certidao, 'Certidão negativa de débitos municipais',
     'Certidão de que o contribuinte não tem débito com o Município. Sai na hora quando não há pendência.',
     'Certidões', 2, 0, true, true, true, false, false, false, '',
     '{"ligado":false,"etapas":[]}', '{}', '[]', '[]'),
    (s_projeto, u_pref, f_projeto, 'Aprovação de projeto arquitetônico',
     'Análise do projeto de construção, ampliação ou reforma antes do alvará de obra.',
     'Obras e urbanismo', 30, 32000, false, true, false, false, true, false, '',
     '{"ligado":true,"etapas":[
        {"id":"analise_documental","ordem":1,"status":"em_analise","prazo_dias":5,"responsavel_tipo":"","responsavel_id":""},
        {"id":"analise_tecnica","ordem":2,"status":"em_andamento","prazo_dias":15,"responsavel_tipo":"","responsavel_id":""},
        {"id":"aprovacao","ordem":3,"status":"pending_approval","prazo_dias":5,"responsavel_tipo":"","responsavel_id":""}]}',
     '{}',
     '[{"titulo":"Responsável técnico cadastrado","texto":"O profissional precisa estar no cadastro de responsáveis técnicos da Prefeitura.","obrigatorio":true}]',
     '[{"titulo":"Projeto arquitetônico","texto":"Plantas, cortes, fachadas e situação, em PDF.","obrigatorio":true},
       {"titulo":"ART ou RRT","texto":"Anotação ou registro de responsabilidade técnica quitado.","obrigatorio":true},
       {"titulo":"Matrícula do imóvel","texto":"Atualizada nos últimos 90 dias.","obrigatorio":true}]'),
    (s_vacina, u_sms, f_vacina, 'Vacinação em domicílio',
     'A equipe de saúde vai até a casa de quem não consegue se deslocar: idosos, acamados e pessoas com mobilidade reduzida.',
     'Saúde', 5, 0, true, true, false, true, false, true, '',
     '{"ligado":false,"etapas":[]}',
     '{"dias":[1,2,3,4,5],"inicio":"08:00","fim":"12:00","duracao_min":30,"vagas_dia":8}',
     '[{"titulo":"Morar em Rondonópolis","texto":"O atendimento é só dentro do perímetro do município.","obrigatorio":true}]',
     '[{"titulo":"Cartão de vacina","texto":"Apresente à equipe no dia.","obrigatorio":true},
       {"titulo":"Laudo ou receita médica","texto":"Para quem está acamado, se tiver.","obrigatorio":false}]'),
    (s_matricula, u_edu, f_matricula, 'Pré-matrícula na rede municipal',
     'Pedido de vaga na educação infantil e no ensino fundamental (1º ao 5º ano).',
     'Educação', 10, 0, true, true, false, false, false, false, '',
     '{"ligado":true,"etapas":[
        {"id":"conferencia","ordem":1,"status":"em_analise","prazo_dias":3,"responsavel_tipo":"","responsavel_id":""},
        {"id":"alocacao","ordem":2,"status":"em_andamento","prazo_dias":5,"responsavel_tipo":"","responsavel_id":""}]}',
     '{}',
     '[{"titulo":"Criança em idade escolar","texto":"Berçário a partir de 4 meses; 1º ano com 6 anos completos até 31 de março.","obrigatorio":true}]',
     '[{"titulo":"Certidão de nascimento","texto":"Da criança.","obrigatorio":true},
       {"titulo":"Comprovante de endereço","texto":"Dos últimos 3 meses, em nome do responsável.","obrigatorio":true},
       {"titulo":"Cartão de vacina","texto":"Atualizado.","obrigatorio":true}]')
    ON CONFLICT (id) DO UPDATE SET
        unidade_id = EXCLUDED.unidade_id, formulario_id = EXCLUDED.formulario_id, nome = EXCLUDED.nome,
        descricao = EXCLUDED.descricao, categoria = EXCLUDED.categoria, prazo_dias = EXCLUDED.prazo_dias,
        taxa_centavos = EXCLUDED.taxa_centavos, destaque = EXCLUDED.destaque, ativo = true,
        aprova_sozinho = EXCLUDED.aprova_sozinho, exige_agendamento = EXCLUDED.exige_agendamento,
        so_responsavel_tecnico = EXCLUDED.so_responsavel_tecnico, impedir_cpf_aberto = EXCLUDED.impedir_cpf_aberto,
        campo_anonimo = EXCLUDED.campo_anonimo, workflow = EXCLUDED.workflow, agenda = EXCLUDED.agenda,
        requisitos = EXCLUDED.requisitos, documentos = EXCLUDED.documentos, updated_at = now();

    -- --------------------------------------------------------- responsáveis técnicos
    v_rt := ARRAY[pg_temp.cpf(501), pg_temp.cpf(502), pg_temp.cpf(503), pg_temp.cpf(504)];
    INSERT INTO protocolos_responsaveis (id, cpf, nome, ativo) VALUES
        (md5('protocolo-demo-rt-1')::uuid, v_rt[1], 'Eng. Roberto Siqueira Lopes', true),
        (md5('protocolo-demo-rt-2')::uuid, v_rt[2], 'Arq. Fernanda Ayres Molina', true),
        (md5('protocolo-demo-rt-3')::uuid, v_rt[3], 'Eng. Cássio Prates Viana', true),
        (md5('protocolo-demo-rt-4')::uuid, v_rt[4], 'Arq. Juliana Torres Bueno', false)
    ON CONFLICT (cpf) DO NOTHING;

    -- --------------------------------------------- pedidos (recriados a cada rodada)
    DELETE FROM protocolos_pedidos
     WHERE id IN (SELECT md5('protocolo-demo-pedido-' || g)::uuid FROM generate_series(1, 200) g);

    FOR r IN SELECT * FROM (VALUES
        -- alvará: workflow triagem -> vistoria -> parecer
        ( 1, 'alvara',     1, 'novo',        'normal'),
        ( 2, 'alvara',     2, 'novo',        'alta'),
        ( 3, 'alvara',     6, 'etapa2',      'normal'),
        ( 4, 'alvara',    13, 'etapa2',      'normal'),   -- vistoria atrasada
        ( 5, 'alvara',    14, 'etapa3',      'alta'),
        ( 6, 'alvara',     5, 'complemento', 'normal'),
        ( 7, 'alvara',     9, 'respondido',  'normal'),
        ( 8, 'alvara',    25, 'favoravel',   'normal'),
        ( 9, 'alvara',    20, 'desfavoravel','normal'),
        (10, 'alvara',     8, 'cancelado',   'baixa'),
        -- poda: sem workflow
        (11, 'poda',       0, 'novo',        'normal'),
        (12, 'poda',       3, 'novo',        'urgente'),
        (13, 'poda',      33, 'novo',        'alta'),     -- atrasado
        (14, 'poda',       7, 'complemento', 'normal'),
        (15, 'poda',      10, 'respondido',  'normal'),
        (16, 'poda',      18, 'favoravel',   'alta'),
        (17, 'poda',      15, 'desfavoravel','normal'),
        (18, 'poda',       4, 'cancelado',   'normal'),
        (19, 'poda',       2, 'novo',        'normal'),   -- denúncia anônima
        -- certidão: aprova sozinho
        (20, 'certidao',   0, 'aprovado',    'normal'),
        (21, 'certidao',   1, 'aprovado',    'normal'),
        (22, 'certidao',   3, 'aprovado',    'normal'),
        (23, 'certidao',   6, 'aprovado',    'normal'),
        (24, 'certidao',  11, 'aprovado',    'normal'),
        -- projeto: só responsável técnico, workflow de 3 etapas
        (25, 'projeto',    2, 'novo',        'normal'),
        (26, 'projeto',    9, 'etapa2',      'normal'),
        (27, 'projeto',   22, 'etapa3',      'normal'),
        (28, 'projeto',   12, 'complemento', 'normal'),
        (29, 'projeto',   35, 'favoravel',   'normal'),
        (30, 'projeto',   28, 'desfavoravel','normal'),
        -- vacinação: agenda e faltas
        (31, 'vacina',     1, 'novo',        'alta'),
        (32, 'vacina',     2, 'novo',        'normal'),
        (33, 'vacina',     3, 'complemento', 'normal'),
        (34, 'vacina',    12, 'favoravel',   'normal'),
        (35, 'vacina',     9, 'falta',       'normal'),
        (36, 'vacina',    21, 'falta_cancel','normal'),
        (37, 'vacina',    10, 'falta',       'normal'),   -- mesmo CPF do 36: 2 faltas, bloqueado
        (38, 'vacina',     6, 'favoravel',   'urgente'),
        -- matrícula: conferência -> alocação
        (39, 'matricula',  1, 'novo',        'normal'),
        (40, 'matricula',  4, 'novo',        'normal'),
        (41, 'matricula',  7, 'etapa2',      'normal'),
        (42, 'matricula',  5, 'complemento', 'normal'),
        (43, 'matricula', 16, 'favoravel',   'normal'),
        (44, 'matricula', 12, 'favoravel',   'alta'),
        (45, 'matricula', 19, 'desfavoravel','normal'),
        (46, 'matricula',  6, 'cancelado',   'normal')
    ) AS t(n, svc, dias, cenario, prioridade)
    LOOP
        k := r.n;
        v_id := md5('protocolo-demo-pedido-' || k)::uuid;
        SELECT s.id, s.formulario_id, s.unidade_id, f.campos, s.workflow, s.prazo_dias
          INTO v_servico, v_form, v_unidade, v_campos, v_wf, v_prazo_servico
          FROM protocolos_servicos s JOIN protocolos_formularios f ON f.id = s.formulario_id
         WHERE s.id = CASE r.svc WHEN 'alvara' THEN s_alvara WHEN 'poda' THEN s_poda
                                 WHEN 'certidao' THEN s_certidao WHEN 'projeto' THEN s_projeto
                                 WHEN 'vacina' THEN s_vacina ELSE s_matricula END;
        SELECT pg_temp.iniciais(u.sigla, u.nome) INTO v_iniciais FROM unidades u WHERE u.id = v_unidade;

        v_nome := nomes[k];
        v_cpf := CASE WHEN r.svc = 'projeto' THEN v_rt[1 + (k % 3)]
                      WHEN k = 37 THEN pg_temp.cpf(1036)
                      ELSE pg_temp.cpf(1000 + k) END;
        IF r.svc = 'projeto' THEN
            v_nome := (ARRAY['Eng. Roberto Siqueira Lopes', 'Arq. Fernanda Ayres Molina', 'Eng. Cássio Prates Viana'])[1 + (k % 3)];
        ELSIF k = 37 THEN
            v_nome := nomes[36];
        END IF;
        v_email := translate(lower(split_part(regexp_replace(v_nome, '^(Eng|Arq)\. ', ''), ' ', 1) || '.' ||
                   reverse(split_part(reverse(v_nome), ' ', 1))),
                   'áàãâéêíóôõúüç', 'aaaaeeiooouuc') || '@example.com';
        v_tel := format('(66) 9%s-%s', lpad(((k * 7919) % 10000)::text, 4, '0'), lpad(((k * 104729) % 10000)::text, 4, '0'));

        -- Abertura em horário de expediente (8h às 17h em Cuiabá), dias atrás.
        v_criado := ((v_hoje - r.dias)::timestamp + interval '8 hours' + ((k * 53) % 540) * interval '1 minute')
                    AT TIME ZONE 'America/Cuiaba';
        IF v_criado > now() THEN
            v_criado := now() - interval '20 minutes';
        END IF;

        -- Respostas do formulário, válidas contra os campos.
        v_resp := CASE r.svc
          WHEN 'alvara' THEN jsonb_build_object(
              'cnpj', format('%s.%s.%s/0001-%s', lpad(((k * 37) % 100)::text, 2, '0'), lpad(((k * 911) % 1000)::text, 3, '0'),
                             lpad(((k * 577) % 1000)::text, 3, '0'), lpad(((k * 13) % 100)::text, 2, '0')),
              'razao_social', empresas[1 + (k % 10)],
              'atividade', (ARRAY['Comércio varejista','Prestação de serviços','Alimentação','Indústria','Escritório administrativo'])[1 + (k % 5)],
              'endereco', format('%s, %s - %s', ruas[1 + (k % 8)], 100 + k * 17, bairros[1 + (k % 10)]),
              'cep', format('787%s-%s', lpad(((k * 3) % 100)::text, 2, '0'), lpad(((k * 17) % 1000)::text, 3, '0')),
              'area_m2', 40 + (k * 13) % 380,
              'localizacao', format('%s,%s', round(-16.4700 - k * 0.0031, 6), round(-54.6350 - k * 0.0023, 6)),
              'tem_placa', k % 2 = 0)
              || CASE WHEN k % 2 = 0 THEN jsonb_build_object('tamanho_placa', '3,00 x 1,20 m') ELSE '{}'::jsonb END
          WHEN 'poda' THEN jsonb_build_object(
              'endereco', format('%s, em frente ao nº %s - %s', ruas[1 + (k % 8)], 50 + k * 11, bairros[1 + (k % 10)]),
              'localizacao', format('%s,%s', round(-16.4650 - k * 0.0027, 6), round(-54.6400 + k * 0.0019, 6)),
              'servico_desejado', (ARRAY['Poda','Remoção','Avaliação técnica'])[1 + (k % 3)],
              'motivo', CASE k % 4 WHEN 0 THEN '["Risco de queda"]'::jsonb
                                   WHEN 1 THEN '["Conflito com a fiação"]'::jsonb
                                   WHEN 2 THEN '["Danos à calçada","Risco de queda"]'::jsonb
                                   ELSE '["Árvore doente"]'::jsonb END,
              'descricao', (ARRAY[
                  'Galhos grandes encostando nos fios da rede elétrica, soltam faísca quando venta.',
                  'Raiz levantou a calçada inteira e já houve queda de idoso no local.',
                  'Árvore com tronco oco e inclinada sobre a rua depois da última chuva.',
                  'Galhos secos caindo em cima dos carros estacionados.',
                  'Árvore com cupim, a casca está se soltando e há serragem na base.'])[1 + (k % 5)],
              'anonimo', k = 19)
          WHEN 'certidao' THEN jsonb_build_object(
              'tipo_pessoa', CASE WHEN k % 2 = 0 THEN 'Pessoa física' ELSE 'Pessoa jurídica' END,
              'documento', CASE WHEN k % 2 = 0 THEN pg_temp.cpf(1000 + k)
                                ELSE format('%s.%s.%s/0001-%s', lpad(((k * 41) % 100)::text, 2, '0'), lpad(((k * 733) % 1000)::text, 3, '0'),
                                            lpad(((k * 419) % 1000)::text, 3, '0'), lpad(((k * 7) % 100)::text, 2, '0')) END,
              'inscricao', format('01.%s.%s.0000', lpad(((k * 29) % 100)::text, 2, '0'), lpad(((k * 61) % 1000)::text, 3, '0')),
              'finalidade', (ARRAY['Licitação','Financiamento','Venda de imóvel','Outros'])[1 + (k % 4)],
              'email_envio', v_email)
          WHEN 'projeto' THEN jsonb_build_object(
              'tipo_obra', (ARRAY['Residencial unifamiliar','Residencial multifamiliar','Comercial','Ampliação','Reforma'])[1 + (k % 5)],
              'area_construida', 70 + (k * 23) % 600,
              'endereco', format('%s, lote %s, quadra %s - %s', ruas[1 + (k % 8)], k % 20 + 1, k % 12 + 1, bairros[1 + (k % 10)]),
              'area_lote', format('%s,%s;%s,%s', round(-16.4800 - k * 0.002, 6), round(-54.6200 - k * 0.002, 6),
                                                 round(-16.4803 - k * 0.002, 6), round(-54.6196 - k * 0.002, 6)),
              'registro_profissional', CASE WHEN k % 3 = 1 THEN 'CAU A' || (120000 + k * 37) ELSE 'CREA-MT ' || (10000 + k * 41) || '/D' END,
              'art_rrt', 'ART ' || (1220260000000 + k * 7919)::text,
              'observacoes', CASE WHEN k % 2 = 0 THEN 'Lote de esquina; recuo frontal pelas duas testadas.' ELSE '' END)
          WHEN 'vacina' THEN jsonb_build_object(
              'cartao_sus', '7' || lpad(((k::bigint * 982451653) % 100000000000000)::text, 14, '0'),
              'data_nascimento', format('19%s-%s-%s', 30 + (k % 25), lpad((1 + k % 12)::text, 2, '0'), lpad((1 + (k * 7) % 28)::text, 2, '0')),
              'vacina', (ARRAY['Influenza','COVID-19','Pneumocócica 23','Hepatite B','Dupla adulto (dT)'])[1 + (k % 5)],
              'endereco', format('%s, %s - %s', ruas[1 + (k % 8)], 200 + k * 9, bairros[1 + (k % 10)]),
              'acamado', k % 2 = 1)
              || CASE WHEN k % 2 = 1 THEN jsonb_build_object('condicao',
                     (ARRAY['Pós-AVC, acamada há dois anos, alimentação por sonda.',
                            'Fratura de fêmur em recuperação, não pode se levantar.',
                            'Alzheimer em estágio avançado, não sai de casa.'])[1 + (k % 3)])
                 ELSE '{}'::jsonb END
              || CASE WHEN k % 3 = 0 THEN jsonb_build_object('contato_alternativo', format('(66) 3%s-%s', lpad(((k * 31) % 1000)::text, 3, '0'), lpad(((k * 97) % 10000)::text, 4, '0')))
                 ELSE '{}'::jsonb END
          ELSE jsonb_build_object(  -- matrícula
              'nome_aluno', (ARRAY['Miguel','Helena','Arthur','Alice','Théo','Laura','Davi','Manuela'])[1 + (k % 8)] || ' ' ||
                            reverse(split_part(reverse(v_nome), ' ', 1)),
              'data_nascimento_aluno', format('20%s-%s-%s', 18 + (k % 6), lpad((1 + k % 12)::text, 2, '0'), lpad((1 + (k * 5) % 28)::text, 2, '0')),
              'serie', (ARRAY['Berçário','Maternal','Pré I','Pré II','1º ano','2º ano','3º ano','4º ano','5º ano'])[1 + (k % 9)],
              'escola_pretendida', escolas[1 + (k % 5)],
              'turno', CASE k % 3 WHEN 0 THEN '["Matutino"]'::jsonb WHEN 1 THEN '["Vespertino","Matutino"]'::jsonb ELSE '["Integral"]'::jsonb END,
              'possui_deficiencia', k = 41 OR k = 44)
              || CASE WHEN k = 41 OR k = 44 THEN jsonb_build_object('necessidade',
                     'Transtorno do espectro autista; precisa de profissional de apoio em sala.') ELSE '{}'::jsonb END
              || CASE WHEN k % 2 = 0 THEN jsonb_build_object('cpf_aluno', pg_temp.cpf(3000 + k)) ELSE '{}'::jsonb END
        END;

        -- Estado inicial, como domain.Abrir.
        v_hist := '[]'::jsonb;
        v_etapa := ''; v_decisao := ''; v_just := ''; v_comp := '';
        v_etapas := CASE WHEN (v_wf ->> 'ligado')::boolean
                         THEN (SELECT jsonb_agg(e ORDER BY (e ->> 'ordem')::int) FROM jsonb_array_elements(v_wf -> 'etapas') e)
                         ELSE NULL END;
        v_t := v_criado;
        IF v_etapas IS NOT NULL THEN
            v_e := v_etapas -> 0;
            v_status := v_e ->> 'status';
            v_etapa := v_e ->> 'id';
            v_prazo := pg_temp.du(v_t, (v_e ->> 'prazo_dias')::int);
        ELSIF r.cenario = 'aprovado' THEN
            v_status := 'aprovado';
            v_prazo := pg_temp.du(v_t, v_prazo_servico);
        ELSE
            v_status := 'novo';
            v_prazo := CASE WHEN v_prazo_servico > 0 THEN pg_temp.du(v_t, v_prazo_servico) END;
        END IF;
        v_hist := v_hist || jsonb_build_array(jsonb_build_array('criado', 'Pedido aberto', v_status, v_t));

        -- Avanços de etapa (etapa2, etapa3 e os concluídos com workflow).
        v_passos := CASE r.cenario WHEN 'etapa2' THEN 1 WHEN 'etapa3' THEN 2
                                   WHEN 'favoravel' THEN 99 WHEN 'desfavoravel' THEN 99 ELSE 0 END;
        IF v_etapas IS NOT NULL AND v_passos > 0 THEN
            FOR v_i IN 1 .. least(v_passos, jsonb_array_length(v_etapas) - 1) LOOP
                v_t := v_t + interval '2 days' + ((k * 17) % 180) * interval '1 minute';
                v_e := v_etapas -> v_i;
                v_status := v_e ->> 'status';
                v_etapa := v_e ->> 'id';
                v_prazo := pg_temp.du(v_t, (v_e ->> 'prazo_dias')::int);
                v_hist := v_hist || jsonb_build_array(jsonb_build_array('avanco', 'pedido avançou de etapa', v_status, v_t));
            END LOOP;
        END IF;

        -- Nota interna em parte dos pedidos (não aparece na consulta pública).
        IF k % 4 = 0 AND r.cenario NOT IN ('aprovado') THEN
            v_t := v_t + interval '3 hours';
            v_hist := v_hist || jsonb_build_array(jsonb_build_array('nota',
                (ARRAY['Conferido com o cadastro, dados batem.',
                       'Liguei para o requerente, confirmou o endereço.',
                       'Aguardando a equipe de campo encaixar na rota da semana.'])[1 + (k % 3)], v_status, v_t));
        END IF;

        -- Complementação pedida (e respondida, no cenário respondido).
        IF r.cenario IN ('complemento', 'respondido') THEN
            v_t := v_t + interval '1 day';
            v_comp := CASE r.svc
                WHEN 'alvara' THEN 'Envie o contrato social atualizado: o enviado não mostra a última alteração de endereço.'
                WHEN 'poda' THEN 'Não localizamos a árvore pelo ponto do mapa. Informe uma referência (cor da casa, comércio próximo).'
                WHEN 'projeto' THEN 'A ART informada não está quitada no sistema do CREA. Envie o comprovante de pagamento.'
                WHEN 'vacina' THEN 'Confirme se a pessoa já tomou alguma dose desta vacina neste ano e em qual unidade.'
                ELSE 'Envie o comprovante de endereço em nome do responsável: o anexado está em nome de terceiro.' END;
            v_status := 'em_complementacao';
            v_hist := v_hist || jsonb_build_array(jsonb_build_array('complemento', v_comp, v_status, v_t));
            IF r.cenario = 'respondido' THEN
                v_t := v_t + interval '2 days';
                v_status := 'em_andamento';
                v_hist := v_hist || jsonb_build_array(jsonb_build_array('resposta',
                    CASE r.svc WHEN 'alvara' THEN 'Segue o contrato com a 3ª alteração, registrada em setembro.'
                               ELSE 'É a casa amarela de esquina, em frente à farmácia. A árvore fica na calçada lateral.' END,
                    v_status, v_t));
                v_comp := '';
            END IF;
        END IF;

        -- Encerramentos.
        IF r.cenario IN ('favoravel', 'desfavoravel', 'falta', 'falta_cancel', 'cancelado') THEN
            v_t := v_t + interval '3 days' + ((k * 7) % 240) * interval '1 minute';
        END IF;
        IF r.cenario = 'favoravel' THEN
            v_decisao := 'favoravel';
            v_just := CASE r.svc
                WHEN 'alvara' THEN 'Vistoria sem pendências. Alvará emitido e disponível para impressão.'
                WHEN 'poda' THEN 'Poda executada pela equipe de arborização. Galhos recolhidos no mesmo dia.'
                WHEN 'projeto' THEN 'Projeto aprovado conforme o Código de Obras. Pode pedir o alvará de construção.'
                WHEN 'vacina' THEN 'Vacina aplicada no domicílio. Registro lançado na caderneta e no sistema.'
                ELSE 'Vaga confirmada. Comparecer à escola em até 5 dias úteis com os documentos originais.' END;
            v_status := 'concluido';
            v_hist := v_hist || jsonb_build_array(jsonb_build_array('conclusao', v_just, v_status, v_t));
        ELSIF r.cenario = 'desfavoravel' THEN
            v_decisao := 'desfavoravel';
            v_just := CASE r.svc
                WHEN 'alvara' THEN 'Atividade não permitida na zona do imóvel (zona residencial exclusiva).'
                WHEN 'poda' THEN 'Avaliação técnica: árvore saudável e sem risco. Poda não indicada nesta época.'
                WHEN 'projeto' THEN 'Taxa de ocupação acima da permitida para o lote. Ajuste o projeto e protocole de novo.'
                ELSE 'Sem vaga na escola e no turno escolhidos. O pedido entrou na lista de espera da região.' END;
            v_status := 'concluido';
            v_hist := v_hist || jsonb_build_array(jsonb_build_array('conclusao', v_just, v_status, v_t));
        ELSIF r.cenario = 'falta' THEN
            v_decisao := 'desfavoravel';
            v_just := 'Equipe foi ao endereço no horário marcado e ninguém atendeu.';
            v_status := 'concluido';
            v_hist := v_hist || jsonb_build_array(jsonb_build_array('conclusao', v_just, v_status, v_t));
        ELSIF r.cenario IN ('cancelado', 'falta_cancel') THEN
            v_status := 'cancelado';
            v_hist := v_hist || jsonb_build_array(jsonb_build_array('cancelado', 'pedido cancelado', v_status, v_t));
        END IF;

        INSERT INTO protocolos_pedidos (id, numero, servico_id, formulario_id, unidade_id, departamento_id, nome, cpf,
            email, telefone, respostas, campos, status, etapa_id, prioridade, prazo_em, decisao, justificativa,
            complemento, usuario_id, responsavel_id, created_at, updated_at)
        VALUES (v_id, v_iniciais || extract(year FROM v_criado AT TIME ZONE 'America/Cuiaba')::int || lpad((90000 + k)::text, 5, '0'),
            v_servico, v_form, v_unidade, NULL, v_nome, v_cpf, v_email, v_tel, v_resp, v_campos, v_status, v_etapa,
            r.prioridade, v_prazo, v_decisao, v_just, v_comp, NULL, NULL, v_criado, v_t);

        INSERT INTO protocolos_historico (pedido_id, acao, descricao, status, created_at)
        SELECT v_id, h ->> 0, h ->> 1, h ->> 2, (h ->> 3)::timestamptz
          FROM jsonb_array_elements(v_hist) h;

        -- Agendamento da vacinação: horário da grade, sem repetir a vaga do dia.
        IF r.svc = 'vacina' THEN
            v_ag_hora := (ARRAY['08:00','08:30','09:00','09:30','10:00','10:30','11:00','11:30'])[1 + (k % 8)];
            IF r.cenario IN ('novo', 'complemento') THEN
                v_ag_data := (pg_temp.du(now(), 1 + (k % 4)) AT TIME ZONE 'America/Cuiaba')::date;
                v_ag_status := 'agendado';
            ELSE
                v_ag_data := (pg_temp.du(v_criado, 2) AT TIME ZONE 'America/Cuiaba')::date;
                v_ag_status := CASE WHEN r.cenario IN ('falta', 'falta_cancel') THEN 'falta' ELSE 'agendado' END;
            END IF;
            INSERT INTO protocolos_agendamentos (id, pedido_id, servico_id, data, hora, status)
            VALUES (md5('protocolo-demo-agenda-' || k)::uuid, v_id, v_servico, v_ag_data, v_ag_hora, v_ag_status);
        END IF;
    END LOOP;

    -- Faltas por CPF: uma no pedido 35; duas no CPF dos pedidos 36 e 37
    -- (bloqueado por 30 dias para novos horários).
    INSERT INTO protocolos_faltas (cpf, quantidade, ultima)
    SELECT p.cpf, count(*), max((a.data + time '12:00') AT TIME ZONE 'America/Cuiaba')
      FROM protocolos_agendamentos a JOIN protocolos_pedidos p ON p.id = a.pedido_id
     WHERE a.status = 'falta' AND p.servico_id = s_vacina
       AND p.id IN (SELECT md5('protocolo-demo-pedido-' || g)::uuid FROM generate_series(1, 200) g)
     GROUP BY p.cpf
    ON CONFLICT (cpf) DO UPDATE SET quantidade = EXCLUDED.quantidade, ultima = EXCLUDED.ultima;

    RAISE NOTICE 'protocolo (demo): 6 serviços, % pedidos, % agendamentos',
        (SELECT count(*) FROM protocolos_pedidos WHERE servico_id IN (s_alvara, s_poda, s_certidao, s_projeto, s_vacina, s_matricula)),
        (SELECT count(*) FROM protocolos_agendamentos WHERE servico_id = s_vacina);
END $$;

COMMIT;
