-- +goose Up
-- O projeto passa a se chamar SISMARQ (ADR 031). Só troca o nome de quem
-- ainda está no padrão antigo: um nome escolhido na tela de Branding fica.
ALTER TABLE branding_settings ALTER COLUMN app_name SET DEFAULT 'SISMARQ';
UPDATE branding_settings SET app_name = 'SISMARQ' WHERE app_name = 'Projeto Nexus';

-- +goose Down
ALTER TABLE branding_settings ALTER COLUMN app_name SET DEFAULT 'Projeto Nexus';
UPDATE branding_settings SET app_name = 'Projeto Nexus' WHERE app_name = 'SISMARQ';
