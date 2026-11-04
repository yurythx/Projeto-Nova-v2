-- +goose Up
-- A fila lê o que ainda não encerrou. A busca por trecho usa o pg_trgm
-- que a base já instala. Sem o índice, o texto varre a tabela inteira.
CREATE INDEX idx_atendimento_solicitacoes_abertas
    ON atendimento_solicitacoes (estado, updated_at DESC, id)
    WHERE estado <> 'encerrado' AND estado <> 'cancelado';

CREATE INDEX idx_atendimento_solicitacoes_titulo_trgm
    ON atendimento_solicitacoes USING gin (titulo gin_trgm_ops);

CREATE INDEX idx_atendimento_solicitacoes_descricao_trgm
    ON atendimento_solicitacoes USING gin (descricao gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS idx_atendimento_solicitacoes_descricao_trgm;
DROP INDEX IF EXISTS idx_atendimento_solicitacoes_titulo_trgm;
DROP INDEX IF EXISTS idx_atendimento_solicitacoes_abertas;
