-- +goose Up
-- Identidade do órgão no branding: Superintendência de Arquivo Público
-- (Fase 0 / ADR 043). Só troca o padrão; um nome gravado na tela de
-- Branding permanece.
ALTER TABLE branding_settings ALTER COLUMN org_name SET DEFAULT 'Superintendência de Arquivo Público';
UPDATE branding_settings SET org_name = 'Superintendência de Arquivo Público'
 WHERE org_name IN ('Organização', '');

-- +goose Down
ALTER TABLE branding_settings ALTER COLUMN org_name SET DEFAULT 'Organização';
UPDATE branding_settings SET org_name = 'Organização'
 WHERE org_name = 'Superintendência de Arquivo Público';
