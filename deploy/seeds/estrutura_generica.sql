-- ============================================================================
-- NEXUS: ESTRUTURA ORGANIZACIONAL GENÉRICA / CORPORATIVA DE REFERÊNCIA
-- ============================================================================
-- Este script define a modelagem canônica corporativa agnóstica para o Nexus,
-- substituindo a dependência de dados específicos do setor público (secretarias/postos/escolas)
-- por uma estrutura empresarial/institucional moderna de 3 níveis:
--   1. Entidades      -> Organizações / Holdings / Empresas do Grupo
--   2. Unidades       -> Sedes / Filiais / Polos / Centros Operacionais
--   3. Departamentos  -> Áreas Funcionais Corporativas Universais
-- ============================================================================

-- 1. ENTIDADES (ORGANIZAÇÕES)
-- UUIDs determinísticos para consistência em testes e deploys
INSERT INTO entidades (id, nome, sigla, slug, documento, ativo)
VALUES 
  ('10000000-0000-0000-0000-000000000001', 'Nexus Corporation', 'NEXUS', 'nexus-corp', '00.000.000/0001-00', true),
  ('10000000-0000-0000-0000-000000000002', 'Nexus Tecnologia & Inovação', 'NEXUSTECH', 'nexus-tech', '00.000.000/0002-00', true)
ON CONFLICT (id) DO UPDATE SET 
  nome = EXCLUDED.nome, sigla = EXCLUDED.sigla, ativo = true;

-- 2. UNIDADES (SEDES E POLOS REGIONAIS)
-- Locais físicos/operacionais onde tramitam processos e fluxos
INSERT INTO unidades (id, entidade_id, parent_id, nome, sigla, slug, ad_group, email, telefone, endereco, ativo)
VALUES
  -- Unidades da Nexus Corporation
  ('20000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', NULL, 'Sede Corporativa Central', 'SEDE', 'sede-central', 'NEXUS-SEDE', 'contato@nexus.corp', '(11) 3000-1000', 'Av. Paulista, 1000 - Bela Vista, São Paulo - SP', true),
  ('20000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001', NULL, 'Polo Operacional & Logística', 'POLO-LOG', 'polo-logistica', 'NEXUS-LOG', 'logistica@nexus.corp', '(11) 3000-2000', 'Rodovia Anhanguera, km 30 - Cajamar - SP', true),
  ('20000000-0000-0000-0000-000000000003', '10000000-0000-0000-0000-000000000001', NULL, 'Filial Regional Sul', 'FILIAL-SUL', 'filial-sul', 'NEXUS-SUL', 'sul@nexus.corp', '(41) 3000-3000', 'Rua XV de Novembro, 500 - Centro, Curitiba - PR', true),
  
  -- Unidades da Nexus Tecnologia
  ('20000000-0000-0000-0000-000000000004', '10000000-0000-0000-0000-000000000002', NULL, 'Hub de Engenharia & P&D', 'HUB-DEV', 'hub-engenharia', 'TECH-DEV', 'tech@nexus.corp', '(11) 3000-4000', 'Av. Brigadeiro Faria Lima, 2500 - Itaim Bibi, São Paulo - SP', true)
ON CONFLICT (id) DO UPDATE SET 
  nome = EXCLUDED.nome, sigla = EXCLUDED.sigla, endereco = EXCLUDED.endereco, ativo = true;

-- 3. DEPARTAMENTOS (ÁREAS FUNCIONAIS CORPORATIVAS)
-- Divisões internas dentro de cada unidade
INSERT INTO departamentos (id, unidade_id, nome, sigla, slug, ad_group, email, telefone, ativo)
VALUES
  -- Departamentos na Sede Corporativa
  ('30000000-0000-0000-0000-000000000001', '20000000-0000-0000-0000-000000000001', 'Diretoria Executiva (Presidência)', 'DIR-EXEC', 'diretoria-executiva', 'NEXUS-DIR', 'presidencia@nexus.corp', 'Ramal 101', true),
  ('30000000-0000-0000-0000-000000000002', '20000000-0000-0000-0000-000000000001', 'Gente & Gestão (Recursos Humanos)', 'RH', 'recursos-humanos', 'NEXUS-RH', 'rh@nexus.corp', 'Ramal 102', true),
  ('30000000-0000-0000-0000-000000000003', '20000000-0000-0000-0000-000000000001', 'Financeiro & Controladoria', 'FIN', 'financeiro', 'NEXUS-FIN', 'financeiro@nexus.corp', 'Ramal 103', true),
  ('30000000-0000-0000-0000-000000000004', '20000000-0000-0000-0000-000000000001', 'Jurídico, Compliance & Governança', 'JUR', 'juridico', 'NEXUS-JUR', 'juridico@nexus.corp', 'Ramal 104', true),
  ('30000000-0000-0000-0000-000000000005', '20000000-0000-0000-0000-000000000001', 'Comercial & Novos Negócios', 'COM', 'comercial', 'NEXUS-COM', 'vendas@nexus.corp', 'Ramal 105', true),
  ('30000000-0000-0000-0000-000000000006', '20000000-0000-0000-0000-000000000001', 'Marketing & Comunicação Institucional', 'MKT', 'marketing', 'NEXUS-MKT', 'marketing@nexus.corp', 'Ramal 106', true),
  ('30000000-0000-0000-0000-000000000007', '20000000-0000-0000-0000-000000000001', 'Compras, Suprimentos & Facilities', 'SUP', 'suprimentos', 'NEXUS-SUP', 'compras@nexus.corp', 'Ramal 107', true),

  -- Departamentos no Polo de Logística
  ('30000000-0000-0000-0000-000000000008', '20000000-0000-0000-0000-000000000002', 'Operações & Logística', 'OPS', 'operacoes-logistica', 'LOG-OPS', 'operacoes@nexus.corp', 'Ramal 201', true),
  ('30000000-0000-0000-0000-000000000009', '20000000-0000-0000-0000-000000000002', 'Armazenagem & Almoxarifado Central', 'ALMOX', 'almoxarifado', 'LOG-ALM', 'almoxarifado@nexus.corp', 'Ramal 202', true),
  ('30000000-0000-0000-0000-000000000010', '20000000-0000-0000-0000-000000000002', 'Controle de Qualidade & Segurança', 'QUAL', 'qualidade', 'LOG-QUAL', 'qualidade@nexus.corp', 'Ramal 203', true),

  -- Departamentos na Filial Sul
  ('30000000-0000-0000-0000-000000000011', '20000000-0000-0000-0000-000000000003', 'Atendimento & Sucesso do Cliente (CS)', 'CS', 'customer-success', 'SUL-CS', 'atendimento.sul@nexus.corp', 'Ramal 301', true),
  ('30000000-0000-0000-0000-000000000012', '20000000-0000-0000-0000-000000000003', 'Administração Regional Sul', 'ADM-SUL', 'administracao-sul', 'SUL-ADM', 'admin.sul@nexus.corp', 'Ramal 302', true),

  -- Departamentos no Hub de Tecnologia
  ('30000000-0000-0000-0000-000000000013', '20000000-0000-0000-0000-000000000004', 'Engenharia de Software & Arquitetura', 'DEV', 'engenharia-software', 'TECH-DEV', 'dev@nexus.corp', 'Ramal 401', true),
  ('30000000-0000-0000-0000-000000000014', '20000000-0000-0000-0000-000000000004', 'Infraestrutura Cloud & DevOps', 'OPS-TI', 'devops-cloud', 'TECH-OPS', 'cloud@nexus.corp', 'Ramal 402', true),
  ('30000000-0000-0000-0000-000000000015', '20000000-0000-0000-0000-000000000004', 'Segurança da Informação & SOC', 'SEC', 'seguranca-informacao', 'TECH-SEC', 'sec@nexus.corp', 'Ramal 403', true),
  ('30000000-0000-0000-0000-000000000016', '20000000-0000-0000-0000-000000000004', 'Suporte Técnico & Help Desk', 'SUPORTE', 'suporte-ti', 'TECH-SUP', 'helpdesk@nexus.corp', 'Ramal 404', true)
ON CONFLICT (id) DO UPDATE SET 
  nome = EXCLUDED.nome, sigla = EXCLUDED.sigla, ativo = true;
