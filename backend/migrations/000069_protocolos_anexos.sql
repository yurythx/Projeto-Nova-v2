-- +goose Up
-- Anexo da resposta do cidadão à complementação. A chave fica no histórico
-- da mesa. A consulta pública não devolve a chave.

ALTER TABLE protocolos_historico
    ADD COLUMN anexos JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE protocolos_historico DROP COLUMN IF EXISTS anexos;
