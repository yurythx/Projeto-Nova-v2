-- +goose Up
-- Processo SEI, prazo da complementação, aprovador do serviço, detalhes da
-- carta e feriados municipais do protocolo.
ALTER TABLE protocolos_servicos
    ADD COLUMN aprovador_id UUID,
    ADD COLUMN detalhes     JSONB NOT NULL DEFAULT '{}';

ALTER TABLE protocolos_pedidos
    ADD COLUMN processo_sei        TEXT NOT NULL DEFAULT '',
    ADD COLUMN complemento_prazo   TIMESTAMPTZ,
    ADD COLUMN complemento_avisado BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX idx_protocolos_pedidos_complemento_prazo ON protocolos_pedidos (complemento_prazo)
    WHERE complemento_prazo IS NOT NULL;
CREATE INDEX idx_protocolos_pedidos_prazo ON protocolos_pedidos (prazo_em)
    WHERE prazo_em IS NOT NULL;

-- Feriado municipal: não conta como dia útil no prazo nem abre horário.
CREATE TABLE protocolos_feriados (
    data DATE PRIMARY KEY,
    nome TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS protocolos_feriados;
DROP INDEX IF EXISTS idx_protocolos_pedidos_prazo;
DROP INDEX IF EXISTS idx_protocolos_pedidos_complemento_prazo;
ALTER TABLE protocolos_pedidos
    DROP COLUMN IF EXISTS complemento_avisado,
    DROP COLUMN IF EXISTS complemento_prazo,
    DROP COLUMN IF EXISTS processo_sei;
ALTER TABLE protocolos_servicos
    DROP COLUMN IF EXISTS detalhes,
    DROP COLUMN IF EXISTS aprovador_id;
