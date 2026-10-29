-- +goose Up
-- Item de configuração (CMDB) vinculado à solicitação de atendimento.
-- Sem chave estrangeira para ativos: são plug-ins separados. O rótulo é
-- copiado quando o vínculo é gravado para que a leitura não dependa do
-- plug-in Ativos estar ligado.
ALTER TABLE atendimento_solicitacoes
    ADD COLUMN ativo_id UUID,
    ADD COLUMN ativo_nome TEXT NOT NULL DEFAULT '' CHECK (length(ativo_nome) <= 260),
    ADD CONSTRAINT atendimento_solicitacoes_ativo_chk CHECK (ativo_id IS NOT NULL OR ativo_nome = '');

CREATE INDEX idx_atendimento_solicitacoes_ativo ON atendimento_solicitacoes (ativo_id, created_at DESC) WHERE ativo_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_atendimento_solicitacoes_ativo;
ALTER TABLE atendimento_solicitacoes
    DROP CONSTRAINT IF EXISTS atendimento_solicitacoes_ativo_chk,
    DROP COLUMN IF EXISTS ativo_nome,
    DROP COLUMN IF EXISTS ativo_id;

