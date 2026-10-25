-- +goose Up
-- O chamado aponta para um segredo do cofre. A senha não é copiada.

ALTER TABLE atendimento_solicitacoes
    ADD COLUMN cofre_segredo_id UUID;

-- +goose Down
ALTER TABLE atendimento_solicitacoes
    DROP COLUMN cofre_segredo_id;
