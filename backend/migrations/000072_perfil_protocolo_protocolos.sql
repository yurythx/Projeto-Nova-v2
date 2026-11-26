-- +goose Up
-- O perfil Protocolo abre e tramita: no plug-in protocolos (ADR 085) isso é
-- tratar o pedido (complemento, avanço, conclusão, cancelamento, falta) e
-- abrir no balcão. Desenhar serviço e formulário (protocolos:gerir) fica com
-- quem administra a secretaria. Só acrescenta o que falta: a edição que o
-- administrador fez no perfil continua.
UPDATE perfis
   SET permissoes = permissoes || ARRAY(
           SELECT p FROM unnest(ARRAY['protocolos:tratar', 'protocolos:abrir']) p
            WHERE p <> ALL (permissoes)),
       updated_at = now()
 WHERE slug = 'protocolo'
   AND NOT (permissoes @> ARRAY['protocolos:tratar', 'protocolos:abrir']);

-- +goose Down
UPDATE perfis
   SET permissoes = array_remove(array_remove(permissoes, 'protocolos:tratar'), 'protocolos:abrir'),
       updated_at = now()
 WHERE slug = 'protocolo';
