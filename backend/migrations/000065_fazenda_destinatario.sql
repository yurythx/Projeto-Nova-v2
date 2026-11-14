-- +goose Up
-- Quem protocola o próprio processo recebe o aviso na caixa, no e-mail e no WhatsApp.
ALTER TABLE fazenda_processos
    ADD COLUMN destinatario UUID REFERENCES users (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE fazenda_processos DROP COLUMN IF EXISTS destinatario;
