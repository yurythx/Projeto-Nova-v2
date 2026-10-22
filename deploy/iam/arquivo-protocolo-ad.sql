-- Equipes do Arquivo Público e do Protocolo pelo AD (via Keycloak, realm
-- rondonopolis). Proposta aprovada pela gestão em 2026-10-19:
--
--   "Arquivo Publico", "Departamento Arquivo Publico"
--       -> perfil Arquivo Público (órgão central: acervo completo e Atlas/TTDD)
--   "Grupo Protocolo", "Departamento de Protocolo", "Nucleo Protocolo"
--       -> perfil Protocolo (abrir e tramitar processos)
--   Ambos sem escopo (valem em todas as secretarias).
--
-- Fora daqui, de propósito: "Grupo Administração Arquivo Municipal" e
-- "Grupo Administracao Divisao Arquivo" são grupos de pasta de rede, com
-- gente de toda a prefeitura. A aprovação da SEMAD (acervo:semad) e os
-- agentes setoriais por secretaria ficam para quando houver os grupos.
--
-- Idempotente; cada criação fica na trilha de auditoria.
-- Aplicar: make iam-arquivo-protocolo
BEGIN;

WITH novo AS (
    INSERT INTO perfis (slug, nome, descricao, permissoes)
    VALUES ('arquivo-publico', 'Arquivo Público',
            'Órgão central (Lei 10.301/2019): acervo completo — consulta, gestão, avaliação/eliminação (CSPAD), bloqueio judicial, recolhimentos — e gestão do Atlas/TTDD.',
            ARRAY['acervo:read', 'acervo:manage', 'acervo:central', 'acervo:avaliacao', 'acervo:bloqueio', 'atlas:read', 'atlas:manage'])
    ON CONFLICT (slug) DO NOTHING
    RETURNING id, slug
)
INSERT INTO audit_logs (action, resource_type, resource_id, actor_subject, metadata)
SELECT 'iam.perfil.saved', 'perfil', novo.id::text, 'deploy/iam',
       jsonb_build_object('slug', novo.slug, 'origem', 'deploy/iam/arquivo-protocolo-ad.sql')
  FROM novo;

CREATE TEMP TABLE m (ad_group text, perfil text) ON COMMIT DROP;
INSERT INTO m VALUES
    ('Arquivo Publico', 'arquivo-publico'),
    ('Departamento Arquivo Publico', 'arquivo-publico'),
    ('Grupo Protocolo', 'protocolo'),
    ('Departamento de Protocolo', 'protocolo'),
    ('Nucleo Protocolo', 'protocolo');

WITH novos AS (
    INSERT INTO ad_group_mappings (ad_group, perfil_id, descricao, created_by)
    SELECT m.ad_group, p.id, p.nome || ' — todas as secretarias (decisão da gestão, 2026-10-19)', 'deploy/iam'
      FROM m JOIN perfis p ON p.slug = m.perfil
    ON CONFLICT DO NOTHING
    RETURNING id, ad_group, perfil_id
)
INSERT INTO audit_logs (action, resource_type, resource_id, actor_subject, metadata)
SELECT 'iam.ad_mapping.created', 'ad_group_mapping', novos.id::text, 'deploy/iam',
       jsonb_build_object('ad_group', novos.ad_group, 'perfil', p.slug, 'escopo', 'global', 'origem', 'deploy/iam/arquivo-protocolo-ad.sql')
  FROM novos JOIN perfis p ON p.id = novos.perfil_id;

SELECT m.ad_group, p.slug AS perfil, (m.entidade_id IS NULL AND m.unidade_id IS NULL) AS global
  FROM ad_group_mappings m JOIN perfis p ON p.id = m.perfil_id
 WHERE m.created_by = 'deploy/iam' ORDER BY p.slug, m.ad_group;

COMMIT;
