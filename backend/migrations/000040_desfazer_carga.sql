-- +goose Up
-- Desfazer a carga (ADR 079): a tarefa simula ou executa a remoção do que
-- veio do GLPI. O registro da importação passa a guardar também as
-- categorias, as equipes, a política e os feriados, e a dizer se a carga
-- criou o destino ou só reaproveitou um que já existia.
ALTER TABLE integracoes_tarefas DROP CONSTRAINT IF EXISTS integracoes_tarefas_tipo_check;
ALTER TABLE integracoes_tarefas ADD CONSTRAINT integracoes_tarefas_tipo_check
    CHECK (tipo IN ('extracao', 'simulacao', 'carga', 'desfazer_simulacao', 'desfazer'));

ALTER TABLE atendimento_importacao DROP CONSTRAINT IF EXISTS atendimento_importacao_tipo_check;
ALTER TABLE atendimento_importacao ADD CONSTRAINT atendimento_importacao_tipo_check
    CHECK (tipo IN ('oferta', 'solicitacao', 'categoria', 'equipe', 'politica', 'feriado'));

ALTER TABLE atendimento_importacao ADD COLUMN criado BOOLEAN NOT NULL DEFAULT true;

-- A oferta reaproveitada pela tabela de-para não leva a descrição da carga.
UPDATE atendimento_importacao i SET criado = false
 WHERE i.tipo = 'oferta'
   AND NOT EXISTS (SELECT 1 FROM atendimento_ofertas o WHERE o.id = i.destino AND o.descricao = 'Importada do GLPI.');

-- As categorias que a carga já criou levam a descrição dela.
INSERT INTO atendimento_importacao (origem, tipo, id_origem, destino)
SELECT 'glpi', 'categoria', 'nome:' || lower(btrim(nome)), id FROM atendimento_categorias
 WHERE descricao = 'Importada do GLPI.'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM atendimento_importacao WHERE tipo IN ('categoria', 'equipe', 'politica', 'feriado');
ALTER TABLE atendimento_importacao DROP COLUMN IF EXISTS criado;
ALTER TABLE atendimento_importacao DROP CONSTRAINT IF EXISTS atendimento_importacao_tipo_check;
ALTER TABLE atendimento_importacao ADD CONSTRAINT atendimento_importacao_tipo_check
    CHECK (tipo IN ('oferta', 'solicitacao'));
DELETE FROM integracoes_tarefas WHERE tipo IN ('desfazer_simulacao', 'desfazer');
ALTER TABLE integracoes_tarefas DROP CONSTRAINT IF EXISTS integracoes_tarefas_tipo_check;
ALTER TABLE integracoes_tarefas ADD CONSTRAINT integracoes_tarefas_tipo_check
    CHECK (tipo IN ('extracao', 'simulacao', 'carga'));
