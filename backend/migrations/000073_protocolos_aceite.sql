-- +goose Up
-- Aceite, encaminhamento e aprovação do protocolo.
-- O serviço pode exigir que alguém aceite o pedido antes de tratá-lo e pode
-- mandar a conclusão para aprovação. O encaminhamento a outra unidade fica
-- pendente no próprio pedido até a unidade de destino aceitar ou recusar.
ALTER TABLE protocolos_servicos
    ADD COLUMN exige_aceite    BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN exige_aprovacao BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE protocolos_pedidos
    ADD COLUMN encaminhar_para          UUID REFERENCES unidades (id),
    ADD COLUMN encaminhar_justificativa TEXT NOT NULL DEFAULT '',
    ADD COLUMN encaminhar_prazo_dias    INT NOT NULL DEFAULT 0,
    ADD COLUMN encaminhado_por          UUID,
    ADD COLUMN encaminhado_em           TIMESTAMPTZ;

CREATE INDEX idx_protocolos_pedidos_encaminhar ON protocolos_pedidos (encaminhar_para)
    WHERE encaminhar_para IS NOT NULL;
CREATE INDEX idx_protocolos_pedidos_responsavel ON protocolos_pedidos (responsavel_id)
    WHERE responsavel_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_protocolos_pedidos_responsavel;
DROP INDEX IF EXISTS idx_protocolos_pedidos_encaminhar;
ALTER TABLE protocolos_pedidos
    DROP COLUMN IF EXISTS encaminhado_em,
    DROP COLUMN IF EXISTS encaminhado_por,
    DROP COLUMN IF EXISTS encaminhar_prazo_dias,
    DROP COLUMN IF EXISTS encaminhar_justificativa,
    DROP COLUMN IF EXISTS encaminhar_para;
ALTER TABLE protocolos_servicos
    DROP COLUMN IF EXISTS exige_aprovacao,
    DROP COLUMN IF EXISTS exige_aceite;
