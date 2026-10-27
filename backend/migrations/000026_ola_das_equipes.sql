-- +goose Up
-- OLA (ADR 070): o tempo que cada equipe se compromete a levar, em horas
-- corridas, para fazer a tarefa que recebe no pedido. Nulo = sem meta.
ALTER TABLE atendimento_equipes ADD COLUMN ola_horas INTEGER CHECK (ola_horas BETWEEN 1 AND 8760);

CREATE INDEX atendimento_tarefas_equipe_idx ON atendimento_tarefas (equipe_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS atendimento_tarefas_equipe_idx;
ALTER TABLE atendimento_equipes DROP COLUMN IF EXISTS ola_horas;
