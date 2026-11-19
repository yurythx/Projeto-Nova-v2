-- +goose Up
-- Um horário da grade só cabe um atendimento. Falta continua ocupando a vaga.

CREATE UNIQUE INDEX protocolos_agendamentos_vaga_key
    ON protocolos_agendamentos (servico_id, data, hora)
    WHERE status IN ('agendado', 'confirmado', 'falta');

-- +goose Down
DROP INDEX IF EXISTS protocolos_agendamentos_vaga_key;
