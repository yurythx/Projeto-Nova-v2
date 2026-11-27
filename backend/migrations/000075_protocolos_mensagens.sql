-- +goose Up
-- Conversa com o cidadão: a mensagem dele acende a novidade para a mesa.
ALTER TABLE protocolos_pedidos
    ADD COLUMN mensagem_nova BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE protocolos_pedidos
    DROP COLUMN IF EXISTS mensagem_nova;
