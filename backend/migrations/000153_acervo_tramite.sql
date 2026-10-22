-- +goose Up
-- Acervo, ponte com o Trâmite (ADR 039): o processo encerrado vira unidade
-- de arquivamento. Um processo gera no máximo uma unidade; reabrir o
-- processo não apaga o registro (o ingresso é fato arquivístico). Sem série
-- da TTDD o consumer não grava — a coluna fica nula no cadastro pela tela.

-- Referência por valor (o mesmo padrão do ADR 026): o Acervo não importa o
-- Trâmite e a migration não cria FK entre plug-ins.
ALTER TABLE acervo_unidades
    ADD COLUMN tramite_processo_id UUID UNIQUE;

-- +goose Down
ALTER TABLE acervo_unidades DROP COLUMN IF EXISTS tramite_processo_id;
