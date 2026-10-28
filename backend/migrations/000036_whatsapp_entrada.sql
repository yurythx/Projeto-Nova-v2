-- +goose Up
-- Entrada pelo WhatsApp (ADR 078, fase 4): a resposta de quem pediu vira
-- comentário, como o e-mail pelo IMAP. O webhook é público e só aceita o
-- token secreto do endereço; na API da Meta, também a assinatura feita com
-- o App Secret. Os dois ficam cifrados.
ALTER TABLE canais_whatsapp
    ADD COLUMN receber BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN webhook_token_cifrado TEXT NOT NULL DEFAULT '',
    ADD COLUMN meta_app_secret_cifrado TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE canais_whatsapp
    DROP COLUMN IF EXISTS meta_app_secret_cifrado,
    DROP COLUMN IF EXISTS webhook_token_cifrado,
    DROP COLUMN IF EXISTS receber;
