-- ============================================================================
-- PROJETO NOVA: RESET E IMPLANTAÇÃO DA ESTRUTURA ORGANIZACIONAL
-- EXCLUSIVA PARA A CIDADE DE RONDONÓPOLIS - MT
-- ============================================================================
-- Este script realiza a Opção 2 solicitada:
-- 1. Remove com segurança a malha pública desordenada (25 secretarias, 255 postos/escolas).
-- 2. Implanta a nova estrutura corporativa e de referência focada em Rondonópolis (MT):
--    - 2 Entidades (Nova Corporation e Nova AgroTech & Inovação)
--    - 5 Unidades Estratégicas em polos reais de Rondonópolis (Centro, Vila Aurora, Distrito Industrial, Sagrada Família e Complexo Ferroviário)
--    - 19 Departamentos corporativos completos com ramais, e-mails e siglas
-- ============================================================================

BEGIN;

-- 1. Limpeza de referências em tabelas que apontam para a estrutura antiga
DELETE FROM ad_group_mappings;
DELETE FROM user_scopes;
DELETE FROM directory_profiles;

-- Limpar vínculos de conteúdos existentes (não destrutivo aos conteúdos)
UPDATE blog_posts SET unidade_id = NULL;
UPDATE wiki_pages SET unidade_id = NULL;
UPDATE catalog_services SET responsible_unidade_id = NULL;
UPDATE contact_messages SET unidade_id = NULL;
UPDATE calendar_rooms SET unidade_id = NULL;
UPDATE files_folders SET unidade_id = NULL;
UPDATE mercurio_rooms SET departamento_id = NULL;

-- 2. Remoção da estrutura antiga na ordem correta das FKs
DELETE FROM departamentos;
DELETE FROM unidades;
DELETE FROM entidades;

-- 3. Inserção das Novas Entidades
-- UUIDs determinísticos
INSERT INTO entidades (id, nome, sigla, slug, documento, ativo)
VALUES
  (
    '10000000-0000-0000-0000-000000000001',
    'Nova Corporation Matriz',
    'NOVA',
    'nova-matriz',
    '12.345.678/0001-90',
    true
  ),
  (
    '10000000-0000-0000-0000-000000000002',
    'Nova AgroTech & Soluções Digitais',
    'AGROTECH',
    'nova-agrotech',
    '12.345.678/0002-71',
    true
  );

-- 4. Inserção das Unidades Estruturadas em Rondonópolis - MT
INSERT INTO unidades (id, entidade_id, parent_id, nome, sigla, slug, ad_group, email, telefone, endereco, ativo)
VALUES
  -- 4.1. Sede Corporativa Central (Centro de Rondonópolis)
  (
    '20000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    NULL,
    'Sede Corporativa Central - Centro',
    'SEDE-CENTRO',
    'sede-centro-rondonopolis',
    'NOVA-SEDE',
    'sede.centro@nova.corp',
    '(66) 3411-1000',
    'Av. Marechal Rondon, 1250 - Centro, Rondonópolis - MT, CEP 78700-075',
    true
  ),

  -- 4.2. Hub de Inovação & Tecnologia (Vila Aurora)
  (
    '20000000-0000-0000-0000-000000000002',
    '10000000-0000-0000-0000-000000000002',
    NULL,
    'Hub de Inovação & P&D - Vila Aurora',
    'HUB-AURORA',
    'hub-vila-aurora',
    'NOVA-TECH',
    'hub.aurora@nova.corp',
    '(66) 3411-2000',
    'Av. Lions Internacional, 1500 - Vila Aurora, Rondonópolis - MT, CEP 78740-000',
    true
  ),

  -- 4.3. Complexo Operacional & Logística (Distrito Industrial)
  (
    '20000000-0000-0000-0000-000000000003',
    '10000000-0000-0000-0000-000000000001',
    NULL,
    'Complexo Operacional - Distrito Industrial',
    'POLO-IND',
    'polo-distrito-industrial',
    'NOVA-LOG',
    'operacoes.ind@nova.corp',
    '(66) 3411-3000',
    'Rodovia BR-364, km 205 - Distrito Industrial Veterinário, Rondonópolis - MT, CEP 78705-800',
    true
  ),

  -- 4.4. Centro Integrado de Atendimento (Sagrada Família)
  (
    '20000000-0000-0000-0000-000000000004',
    '10000000-0000-0000-0000-000000000001',
    NULL,
    'Centro Integrado de Atendimento - Sagrada Família',
    'UNID-POGUBA',
    'unidade-sagrada-familia',
    'NOVA-ATD',
    'atendimento.roo@nova.corp',
    '(66) 3411-4000',
    'Av. Poguba, 850 - Sagrada Família, Rondonópolis - MT, CEP 78735-000',
    true
  ),

  -- 4.5. Terminal Intermodal Agro & Ferrovia (Anel Viário)
  (
    '20000000-0000-0000-0000-000000000005',
    '10000000-0000-0000-0000-000000000002',
    NULL,
    'Terminal Intermodal & Agronegócios - Anel Viário',
    'TERM-AGRO',
    'terminal-intermodal-agro',
    'NOVA-AGRO',
    'terminal.ferrovia@nova.corp',
    '(66) 3411-5000',
    'Anel Viário Conrado Sales Brito, s/n - Terminal Ferroviário, Rondonópolis - MT, CEP 78750-000',
    true
  );

-- 5. Inserção dos Departamentos Estruturados
INSERT INTO departamentos (id, unidade_id, nome, sigla, slug, ad_group, email, telefone, ativo)
VALUES
  -- 5.1. Departamentos na Sede Central (Centro)
  (
    '30000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    'Diretoria Executiva & Presidência',
    'DIR-EXEC',
    'diretoria-executiva',
    'NOVA-DIR',
    'presidencia@nova.corp',
    'Ramal 101',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000002',
    '20000000-0000-0000-0000-000000000001',
    'Gente & Gestão (Recursos Humanos)',
    'RH',
    'recursos-humanos',
    'NOVA-RH',
    'rh@nova.corp',
    'Ramal 102',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000003',
    '20000000-0000-0000-0000-000000000001',
    'Financeiro, Fiscal & Controladoria',
    'FIN',
    'financeiro',
    'NOVA-FIN',
    'financeiro@nova.corp',
    'Ramal 103',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000004',
    '20000000-0000-0000-0000-000000000001',
    'Jurídico, Compliance & Governança',
    'JUR',
    'juridico',
    'NOVA-JUR',
    'juridico@nova.corp',
    'Ramal 104',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000005',
    '20000000-0000-0000-0000-000000000001',
    'Comercial & Novos Negócios',
    'COM',
    'comercial',
    'NOVA-COM',
    'comercial@nova.corp',
    'Ramal 105',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000006',
    '20000000-0000-0000-0000-000000000001',
    'Marketing & Comunicação Institucional',
    'MKT',
    'marketing',
    'NOVA-MKT',
    'marketing@nova.corp',
    'Ramal 106',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000007',
    '20000000-0000-0000-0000-000000000001',
    'Compras, Suprimentos & Facilities',
    'SUP',
    'suprimentos',
    'NOVA-SUP',
    'suprimentos@nova.corp',
    'Ramal 107',
    true
  ),

  -- 5.2. Departamentos no Hub Vila Aurora
  (
    '30000000-0000-0000-0000-000000000008',
    '20000000-0000-0000-0000-000000000002',
    'Engenharia de Software & Arquitetura',
    'DEV',
    'engenharia-software',
    'TECH-DEV',
    'dev@nova.corp',
    'Ramal 201',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000009',
    '20000000-0000-0000-0000-000000000002',
    'Infraestrutura Cloud & DevOps',
    'DEVOPS',
    'devops-cloud',
    'TECH-OPS',
    'cloud@nova.corp',
    'Ramal 202',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000010',
    '20000000-0000-0000-0000-000000000002',
    'Segurança da Informação & SOC',
    'SEC',
    'seguranca-informacao',
    'TECH-SEC',
    'seguranca@nova.corp',
    'Ramal 203',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000011',
    '20000000-0000-0000-0000-000000000002',
    'Suporte Técnico & Help Desk Corporativo',
    'SUPORTE',
    'suporte-ti',
    'TECH-SUP',
    'suporte@nova.corp',
    'Ramal 204',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000012',
    '20000000-0000-0000-0000-000000000002',
    'Pesquisa & Inovação AgroTech (P&D)',
    'PED',
    'inovacao-ped',
    'TECH-PED',
    'inovacao@nova.corp',
    'Ramal 205',
    true
  ),

  -- 5.3. Departamentos no Complexo Distrito Industrial
  (
    '30000000-0000-0000-0000-000000000013',
    '20000000-0000-0000-0000-000000000003',
    'Operações & Logística Regional',
    'OPS',
    'operacoes-logistica',
    'LOG-OPS',
    'operacoes@nova.corp',
    'Ramal 301',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000014',
    '20000000-0000-0000-0000-000000000003',
    'Armazenagem & Almoxarifado Central',
    'ALMOX',
    'almoxarifado-central',
    'LOG-ALM',
    'almoxarifado@nova.corp',
    'Ramal 302',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000015',
    '20000000-0000-0000-0000-000000000003',
    'Gestão de Frotas & Transporte Rodoviário',
    'FROTAS',
    'gestao-frotas',
    'LOG-FROTA',
    'frotas@nova.corp',
    'Ramal 303',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000016',
    '20000000-0000-0000-0000-000000000003',
    'Segurança do Trabalho, Saúde & Qualidade',
    'SMS',
    'seguranca-trabalho',
    'LOG-SMS',
    'qualidade@nova.corp',
    'Ramal 304',
    true
  ),

  -- 5.4. Departamentos na Unidade Sagrada Família
  (
    '30000000-0000-0000-0000-000000000017',
    '20000000-0000-0000-0000-000000000004',
    'Atendimento ao Cliente & Sucesso (Customer Success)',
    'CS',
    'customer-success',
    'ATD-CS',
    'atendimento@nova.corp',
    'Ramal 401',
    true
  ),
  (
    '30000000-0000-0000-0000-000000000018',
    '20000000-0000-0000-0000-000000000004',
    'Protocolo Central & Triagem de Documentos',
    'PROT',
    'protocolo-central',
    'ATD-PROT',
    'protocolo@nova.corp',
    'Ramal 402',
    true
  ),

  -- 5.5. Departamentos no Terminal Ferroviário
  (
    '30000000-0000-0000-0000-000000000019',
    '20000000-0000-0000-0000-000000000005',
    'Operações Intermodais & Balança',
    'INTERMODAL',
    'operacoes-intermodais',
    'AGRO-OPS',
    'intermodal@nova.corp',
    'Ramal 501',
    true
  );

-- 6. Configurar perfil e lotação do Administrador na Sede Central
INSERT INTO directory_profiles (user_id, job_title, phone, extension, unidade_id, departamento_id, bio, visible, updated_at)
SELECT 
  id,
  'Administrador de Sistemas & TI',
  '(66) 3411-1000',
  '201',
  '20000000-0000-0000-0000-000000000001'::uuid,
  '30000000-0000-0000-0000-000000000008'::uuid,
  'Gestor da infraestrutura tecnológica e sistemas do Projeto-nova-v2 em Rondonópolis.',
  true,
  now()
FROM users
WHERE username = 'admin'
ON CONFLICT (user_id) DO UPDATE SET
  job_title = EXCLUDED.job_title,
  phone = EXCLUDED.phone,
  extension = EXCLUDED.extension,
  unidade_id = EXCLUDED.unidade_id,
  departamento_id = EXCLUDED.departamento_id,
  bio = EXCLUDED.bio,
  visible = true,
  updated_at = now();

COMMIT;
