-- +goose Up
-- Quem esteve com o chamado e por quanto tempo (ADR 073): a base do OLA
-- da equipe no incidente.

-- Um trecho por pessoa que assumiu, com a equipe dela na unidade do
-- chamado. Fecha quando o chamado muda de mãos ou é concluído; a equipe
-- guarda o nome, para o trecho sobreviver à equipe apagada.
CREATE TABLE atendimento_atribuicoes (
    id BIGSERIAL PRIMARY KEY,
    solicitacao_id UUID NOT NULL REFERENCES atendimento_solicitacoes(id) ON DELETE CASCADE,
    responsavel_id UUID REFERENCES users(id) ON DELETE SET NULL,
    equipe_id UUID REFERENCES atendimento_equipes(id) ON DELETE SET NULL,
    equipe_nome TEXT NOT NULL DEFAULT '',
    inicio TIMESTAMPTZ NOT NULL,
    fim TIMESTAMPTZ,
    CHECK (fim IS NULL OR fim >= inicio)
);

-- No máximo um trecho aberto por chamado.
CREATE UNIQUE INDEX atendimento_atribuicoes_aberta_idx ON atendimento_atribuicoes (solicitacao_id) WHERE fim IS NULL;
CREATE INDEX atendimento_atribuicoes_equipe_idx ON atendimento_atribuicoes (equipe_id, inicio);

-- O passado: um trecho por chamado que já tem responsável, do primeiro
-- "em atendimento" até a conclusão. As trocas de mão de antes não ficaram
-- registradas.
INSERT INTO atendimento_atribuicoes (solicitacao_id, responsavel_id, equipe_id, equipe_nome, inicio, fim)
SELECT s.id, s.responsavel_id, eq.id, COALESCE(eq.nome, ''), ini.em,
    CASE WHEN s.estado IN ('resolvido', 'atendido', 'encerrado', 'cancelado')
        THEN GREATEST(COALESCE(s.resolvida_em, s.encerrada_em, s.updated_at), ini.em) END
FROM atendimento_solicitacoes s
CROSS JOIN LATERAL (
    SELECT COALESCE(min(p.em), s.created_at) AS em FROM atendimento_passagens p
    WHERE p.solicitacao_id = s.id AND p.para = 'em_atendimento'
) ini
LEFT JOIN LATERAL (
    SELECT e.id, e.nome FROM atendimento_equipes e
    JOIN atendimento_equipe_membros m ON m.equipe_id = e.id
    WHERE m.user_id = s.responsavel_id AND e.unidade_id IS NOT DISTINCT FROM s.unidade_id
    ORDER BY e.nome, e.id LIMIT 1
) eq ON true
WHERE s.responsavel_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS atendimento_atribuicoes;
