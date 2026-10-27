-- +goose Up
-- Contrato de apoio (ADR 071): o prazo que o fornecedor assume e o
-- chamado que espera por ele.

-- Prazo de atendimento do fornecedor, em horas corridas. Nulo = sem prazo.
ALTER TABLE contratos_contratos ADD COLUMN prazo_horas INTEGER CHECK (prazo_horas BETWEEN 1 AND 8760);

-- O chamado guarda o contrato e uma cópia do rótulo e do prazo de quando
-- foi vinculado: a medição não depende do plug-in Contratos estar ligado,
-- nem muda se o contrato for editado ou apagado depois. Sem chave
-- estrangeira: são plug-ins separados.
ALTER TABLE atendimento_solicitacoes
    ADD COLUMN contrato_apoio_id UUID,
    ADD COLUMN contrato_apoio TEXT NOT NULL DEFAULT '' CHECK (length(contrato_apoio) <= 400),
    ADD COLUMN contrato_prazo_horas INTEGER CHECK (contrato_prazo_horas BETWEEN 1 AND 8760),
    ADD CONSTRAINT atendimento_solicitacoes_contrato_chk CHECK (contrato_apoio_id IS NOT NULL OR (contrato_apoio = '' AND contrato_prazo_horas IS NULL));

CREATE INDEX atendimento_solicitacoes_contrato_idx ON atendimento_solicitacoes (contrato_apoio_id, created_at) WHERE contrato_apoio_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS atendimento_solicitacoes_contrato_idx;
ALTER TABLE atendimento_solicitacoes
    DROP CONSTRAINT IF EXISTS atendimento_solicitacoes_contrato_chk,
    DROP COLUMN IF EXISTS contrato_prazo_horas,
    DROP COLUMN IF EXISTS contrato_apoio,
    DROP COLUMN IF EXISTS contrato_apoio_id;
ALTER TABLE contratos_contratos DROP COLUMN IF EXISTS prazo_horas;
