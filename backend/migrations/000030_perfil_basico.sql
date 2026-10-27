-- +goose Up
-- O perfil "servidor" é o básico de quem entra sem grupo de administração
-- (ADR 074): abre e acompanha as próprias solicitações. Só muda se ninguém
-- o editou depois de criado.
UPDATE perfis
   SET permissoes = ARRAY['atendimento:abrir']
 WHERE slug = 'servidor' AND permissoes = '{}';

-- +goose Down
UPDATE perfis
   SET permissoes = '{}'
 WHERE slug = 'servidor' AND permissoes = ARRAY['atendimento:abrir'];
