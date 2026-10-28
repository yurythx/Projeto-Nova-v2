-- +goose Up
-- A área de preparo guarda também o calendário (turnos e feriados) e os
-- grupos técnicos do GLPI (ADR 078).
ALTER TABLE integracoes_itens DROP CONSTRAINT IF EXISTS integracoes_itens_tipo_check;
ALTER TABLE integracoes_itens ADD CONSTRAINT integracoes_itens_tipo_check
    CHECK (tipo IN ('chamado', 'pessoa', 'categoria', 'entidade', 'sla', 'calendario', 'grupo'));

-- +goose Down
DELETE FROM integracoes_itens WHERE tipo IN ('calendario', 'grupo');
ALTER TABLE integracoes_itens DROP CONSTRAINT IF EXISTS integracoes_itens_tipo_check;
ALTER TABLE integracoes_itens ADD CONSTRAINT integracoes_itens_tipo_check
    CHECK (tipo IN ('chamado', 'pessoa', 'categoria', 'entidade', 'sla'));
