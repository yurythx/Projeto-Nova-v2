-- +goose Up
-- Foto da pessoa, trocada no perfil e mostrada nas listas e nos quadros.
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_key text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS avatar_key;
