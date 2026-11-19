-- +goose Up
-- A sugestão de resolução técnica usa o mesmo registro das outras tarefas da IA.

ALTER TABLE atendimento_ia_sugestoes DROP CONSTRAINT atendimento_ia_sugestoes_tipo_check;
ALTER TABLE atendimento_ia_sugestoes ADD CONSTRAINT atendimento_ia_sugestoes_tipo_check
    CHECK (tipo IN ('classificacao', 'resumo', 'resolucao'));

-- +goose Down
DELETE FROM atendimento_ia_sugestoes WHERE tipo = 'resolucao';
ALTER TABLE atendimento_ia_sugestoes DROP CONSTRAINT atendimento_ia_sugestoes_tipo_check;
ALTER TABLE atendimento_ia_sugestoes ADD CONSTRAINT atendimento_ia_sugestoes_tipo_check
    CHECK (tipo IN ('classificacao', 'resumo'));
