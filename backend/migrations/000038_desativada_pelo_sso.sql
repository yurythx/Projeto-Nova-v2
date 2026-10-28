-- +goose Up
-- Quem desativou a conta (ADR 077): a sincronização com o SSO ou um
-- administrador. Federado ao AD, o Keycloak às vezes informa um estado
-- desatualizado; a conta que o SSO desativou volta quando ele a der como
-- ativa. A que um administrador desativou fica desativada.
ALTER TABLE users ADD COLUMN desativada_pelo_sso BOOLEAN NOT NULL DEFAULT false;

-- A conta do SSO que está inativa sem nenhuma alteração de um
-- administrador na auditoria foi desativada pela sincronização.
UPDATE users u SET desativada_pelo_sso = true
 WHERE u.keycloak_subject IS NOT NULL AND NOT u.active
   AND NOT EXISTS (SELECT 1 FROM audit_logs a
                    WHERE a.action = 'user.updated' AND a.resource_type = 'user' AND a.resource_id = u.id::text);

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS desativada_pelo_sso;
