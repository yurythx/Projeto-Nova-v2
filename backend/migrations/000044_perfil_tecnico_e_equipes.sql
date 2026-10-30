-- +goose Up
-- Perfil de Técnico de Atendimento (ADR 013, ADR 074)
-- Permite atendimento da fila, gestão de solicitações, ativos, problemas e mudanças na unidade vinculada.
INSERT INTO perfis (slug, nome, descricao, permissoes, sistema)
VALUES (
    'tecnico',
    'Técnico de Atendimento',
    'Atende a fila de chamados da unidade, assume e resolve solicitações, gerencia ativos, problemas e mudanças.',
    ARRAY['atendimento:abrir','atendimento:atender','atendimento:manage','problemas:read','problemas:manage','mudancas:manage','ativos:read','ativos:manage','projetos:read'],
    true
)
ON CONFLICT (slug) DO UPDATE
SET permissoes = ARRAY['atendimento:abrir','atendimento:atender','atendimento:manage','problemas:read','problemas:manage','mudancas:manage','ativos:read','ativos:manage','projetos:read'];

-- Equipes de atendimento para PREFEITURA e SEMECEL
INSERT INTO atendimento_equipes (nome, unidade_id)
SELECT 'Grupo Técnico - Prefeitura', u.id
FROM unidades u WHERE u.sigla = 'PREFEITURA'
AND NOT EXISTS (SELECT 1 FROM atendimento_equipes WHERE nome = 'Grupo Técnico - Prefeitura');

INSERT INTO atendimento_equipes (nome, unidade_id)
SELECT 'Grupo Sistemas - Prefeitura', u.id
FROM unidades u WHERE u.sigla = 'PREFEITURA'
AND NOT EXISTS (SELECT 1 FROM atendimento_equipes WHERE nome = 'Grupo Sistemas - Prefeitura');

INSERT INTO atendimento_equipes (nome, unidade_id)
SELECT 'Grupo Técnico - SEMECEL', u.id
FROM unidades u WHERE u.sigla = 'SEMECEL'
AND NOT EXISTS (SELECT 1 FROM atendimento_equipes WHERE nome = 'Grupo Técnico - SEMECEL');

INSERT INTO atendimento_equipes (nome, unidade_id)
SELECT 'Grupo Sistemas - SEMECEL', u.id
FROM unidades u WHERE u.sigla = 'SEMECEL'
AND NOT EXISTS (SELECT 1 FROM atendimento_equipes WHERE nome = 'Grupo Sistemas - SEMECEL');

-- Mapeamentos de grupos do AD para Técnico nas respectivas unidades (ADR 013, ADR 074)
-- 1. PREFEITURA
INSERT INTO ad_group_mappings (ad_group, perfil_id, entidade_id, unidade_id, descricao, created_by)
SELECT g.grupo, p.id, u.entidade_id, u.id, 'Técnico de TI - Prefeitura', 'sistema'
FROM (VALUES ('grupo ti  - helpdesk'), ('g-ultravnc-suporte'), ('grupo tecnologia da informação')) AS g(grupo)
CROSS JOIN (SELECT id FROM perfis WHERE slug = 'tecnico') p
CROSS JOIN (SELECT id, entidade_id FROM unidades WHERE sigla = 'PREFEITURA') u
WHERE NOT EXISTS (
    SELECT 1 FROM ad_group_mappings m
    WHERE lower(m.ad_group) = lower(g.grupo)
      AND m.perfil_id = p.id
      AND m.unidade_id = u.id
);

-- 2. SMS (Saúde)
INSERT INTO ad_group_mappings (ad_group, perfil_id, entidade_id, unidade_id, descricao, created_by)
SELECT g.grupo, p.id, u.entidade_id, u.id, 'Técnico de TI - Saúde', 'sistema'
FROM (VALUES ('grupo saúde tecnologia'), ('saude.tecnologia'), ('help desk sáude')) AS g(grupo)
CROSS JOIN (SELECT id FROM perfis WHERE slug = 'tecnico') p
CROSS JOIN (SELECT id, entidade_id FROM unidades WHERE sigla = 'SMS') u
WHERE NOT EXISTS (
    SELECT 1 FROM ad_group_mappings m
    WHERE lower(m.ad_group) = lower(g.grupo)
      AND m.perfil_id = p.id
      AND m.unidade_id = u.id
);

-- 3. SEMECEL (Educação)
INSERT INTO ad_group_mappings (ad_group, perfil_id, entidade_id, unidade_id, descricao, created_by)
SELECT g.grupo, p.id, u.entidade_id, u.id, 'Técnico de TI - SEMECEL', 'sistema'
FROM (VALUES ('ti semed'), ('help desk educacao')) AS g(grupo)
CROSS JOIN (SELECT id FROM perfis WHERE slug = 'tecnico') p
CROSS JOIN (SELECT id, entidade_id FROM unidades WHERE sigla = 'SEMECEL') u
WHERE NOT EXISTS (
    SELECT 1 FROM ad_group_mappings m
    WHERE lower(m.ad_group) = lower(g.grupo)
      AND m.perfil_id = p.id
      AND m.unidade_id = u.id
);

-- Membros das equipes de atendimento (técnicos do GLPI/AD cadastrados em users)
-- Equipe Prefeitura
INSERT INTO atendimento_equipe_membros (equipe_id, user_id)
SELECT eq.id, u.id
FROM atendimento_equipes eq
CROSS JOIN users u
WHERE eq.nome = 'Grupo Técnico - Prefeitura'
  AND lower(u.username) IN ('yuri.menezes', 'lucas.sete', 'davi.fogaca', 'raphael.sodre', 'watila.barbosa', 'marco.caetano', 'vinicius.moura', 'enzo.vale', 'jhalys.wunsche', 'arthur.sousa', 'matheus.ferreira', 'victor.parmejane')
ON CONFLICT DO NOTHING;

-- Equipe SEMECEL
INSERT INTO atendimento_equipe_membros (equipe_id, user_id)
SELECT eq.id, u.id
FROM atendimento_equipes eq
CROSS JOIN users u
WHERE eq.nome = 'Grupo Técnico - SEMECEL'
  AND lower(u.username) IN ('eduardo.costa', 'raffael.schuh', 'pedro.soares', 'marcos.jacovozzi')
ON CONFLICT DO NOTHING;

-- Equipe SMS
INSERT INTO atendimento_equipe_membros (equipe_id, user_id)
SELECT eq.id, u.id
FROM atendimento_equipes eq
CROSS JOIN users u
WHERE eq.nome = 'Grupo Técnico - SMS'
  AND lower(u.username) IN ('clauber.inacio', 'vitor.moresco', 'valdir.vastor')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM ad_group_mappings WHERE created_by = 'sistema' AND descricao LIKE 'Técnico de TI%';
DELETE FROM atendimento_equipes WHERE nome IN ('Grupo Técnico - Prefeitura', 'Grupo Sistemas - Prefeitura', 'Grupo Técnico - SEMECEL', 'Grupo Sistemas - SEMECEL');
DELETE FROM perfis WHERE slug = 'tecnico' AND sistema = true;
