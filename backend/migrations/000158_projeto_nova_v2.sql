-- +goose Up
-- O produto passa a ser o microkernel Projeto-nova-v2.
-- IAM e Auditoria continuam no núcleo. Os demais plug-ins nascem desligados.
-- Acervo e Atlas saem do catálogo (o código deles foi removido).

ALTER TABLE branding_settings ALTER COLUMN app_name SET DEFAULT 'Projeto-nova-v2';
ALTER TABLE branding_settings ALTER COLUMN app_description SET DEFAULT 'Microkernel de plug-ins: identidade, auditoria e módulos que nascem desligados.';
ALTER TABLE branding_settings ALTER COLUMN org_name SET DEFAULT 'Projeto-nova-v2';

UPDATE branding_settings
   SET app_name = 'Projeto-nova-v2'
 WHERE app_name IN ('SISMARQ', 'Projeto Nexus', 'sismarq');

UPDATE branding_settings
   SET app_description = 'Microkernel de plug-ins: identidade, auditoria e módulos que nascem desligados.'
 WHERE app_description IN (
   'Sistema Municipal de Arquivos — Arquivo Público Municipal de Rondonópolis',
   'Plataforma corporativa modular'
 );

UPDATE branding_settings
   SET org_name = 'Projeto-nova-v2'
 WHERE org_name IN ('Superintendência de Arquivo Público', 'Organização', '');

UPDATE system_modules
   SET enabled = false, updated_at = now(), updated_by = 'projeto-nova-v2'
 WHERE key NOT IN ('iam', 'auditoria');

DELETE FROM system_modules WHERE key IN ('atlas', 'acervo');

ALTER TABLE ia_uso DROP CONSTRAINT IF EXISTS ia_uso_funcao_check;
UPDATE ia_uso SET funcao = 'plataforma.ia' WHERE funcao = 'atlas.assistente';
ALTER TABLE ia_uso ADD CONSTRAINT ia_uso_funcao_check CHECK (funcao IN ('plataforma.ia'));

-- +goose Down
ALTER TABLE branding_settings ALTER COLUMN app_name SET DEFAULT 'SISMARQ';
ALTER TABLE branding_settings ALTER COLUMN app_description SET DEFAULT 'Sistema Municipal de Arquivos — Arquivo Público Municipal de Rondonópolis';
ALTER TABLE branding_settings ALTER COLUMN org_name SET DEFAULT 'Superintendência de Arquivo Público';

UPDATE branding_settings SET app_name = 'SISMARQ' WHERE app_name = 'Projeto-nova-v2';
UPDATE branding_settings
   SET app_description = 'Sistema Municipal de Arquivos — Arquivo Público Municipal de Rondonópolis'
 WHERE app_description = 'Microkernel de plug-ins: identidade, auditoria e módulos que nascem desligados.';
UPDATE branding_settings SET org_name = 'Superintendência de Arquivo Público' WHERE org_name = 'Projeto-nova-v2';

ALTER TABLE ia_uso DROP CONSTRAINT IF EXISTS ia_uso_funcao_check;
UPDATE ia_uso SET funcao = 'atlas.assistente' WHERE funcao = 'plataforma.ia';
ALTER TABLE ia_uso ADD CONSTRAINT ia_uso_funcao_check CHECK (funcao IN ('atlas.assistente'));
