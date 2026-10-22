-- Equipe do Protocolo pelo AD (via Keycloak, realm rondonopolis). Proposta
-- aprovada pela gestão em 2026-10-19:
--
--   "Grupo Protocolo", "Departamento de Protocolo", "Nucleo Protocolo"
--       -> perfil Protocolo (abrir e tramitar processos)
--   Sem escopo (vale em todas as secretarias).
--
-- O perfil Protocolo é de sistema (vem no esquema base).
--
-- Idempotente; cada criação fica na trilha de auditoria.
-- Aplicar: make iam-protocolo
BEGIN;

CREATE TEMP TABLE m (ad_group text, perfil text) ON COMMIT DROP;
INSERT INTO m VALUES
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
       jsonb_build_object('ad_group', novos.ad_group, 'perfil', p.slug, 'escopo', 'global', 'origem', 'deploy/iam/protocolo-ad.sql')
  FROM novos JOIN perfis p ON p.id = novos.perfil_id;

SELECT m.ad_group, p.slug AS perfil, (m.entidade_id IS NULL AND m.unidade_id IS NULL) AS global
  FROM ad_group_mappings m JOIN perfis p ON p.id = m.perfil_id
 WHERE m.created_by = 'deploy/iam' ORDER BY p.slug, m.ad_group;

COMMIT;
