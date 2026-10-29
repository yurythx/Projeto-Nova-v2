-- +goose Up
-- Sincronizar com o GLPI até a virada (ADR 081). O registro da importação
-- guarda a versão do chamado no GLPI (date_mod) e quando a sincronização
-- o atualizou por último; os acompanhamentos também entram nele, para não
-- se repetirem. A conexão ganha a chave que liga a sincronização.
ALTER TABLE atendimento_importacao ADD COLUMN versao TEXT;
ALTER TABLE atendimento_importacao ADD COLUMN sincronizado_em TIMESTAMPTZ;

ALTER TABLE atendimento_importacao DROP CONSTRAINT IF EXISTS atendimento_importacao_tipo_check;
ALTER TABLE atendimento_importacao ADD CONSTRAINT atendimento_importacao_tipo_check
    CHECK (tipo IN ('oferta', 'solicitacao', 'categoria', 'equipe', 'politica', 'feriado', 'acompanhamento'));

ALTER TABLE integracoes_conexoes ADD COLUMN sincronizar BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE integracoes_conexoes DROP COLUMN IF EXISTS sincronizar;
DELETE FROM atendimento_importacao WHERE tipo = 'acompanhamento';
ALTER TABLE atendimento_importacao DROP CONSTRAINT IF EXISTS atendimento_importacao_tipo_check;
ALTER TABLE atendimento_importacao ADD CONSTRAINT atendimento_importacao_tipo_check
    CHECK (tipo IN ('oferta', 'solicitacao', 'categoria', 'equipe', 'politica', 'feriado'));
ALTER TABLE atendimento_importacao DROP COLUMN IF EXISTS sincronizado_em;
ALTER TABLE atendimento_importacao DROP COLUMN IF EXISTS versao;
