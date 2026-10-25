# 053 — Cofre de senhas

- **Status:** Aceito e implementado
- **Data:** 2026-10-24
- **Relacionado:** [ADR 013](013-permissao-com-escopo-e-heranca.md), [ADR 016](016-camada-de-aplicacao-e-transacoes.md)

## Contexto

A equipe precisa guardar senhas de serviço no escopo da unidade. O
Keycloak já autentica a pessoa e entrega os grupos do AD. A senha em si
não pode morar no token, em atributo de usuário nem numa role por
segredo. A chave que cifra a configuração do Keycloak também não pode
abrir o cofre.

## Decisão

1. **Plugin.** `cofre` nasce desligado. Rota `/cofre`. Nenhum outro
   plugin o importa. Um chamado, se um dia apontar para um segredo,
   guarda o id — não a senha.
2. **Permissões com escopo.** `cofre:read` vê nome e usuário.
   `cofre:reveal` revela. `cofre:manage` guarda, troca e exclui.
   `cofre:grant` move o segredo de unidade. O administrador também
   revela, e essa revelação entra na auditoria.
3. **Contrato.** `GET /api/v1/cofre/segredos` devolve só metadado.
   `POST /api/v1/cofre/segredos` cria. `PUT` e `DELETE` em
   `/api/v1/cofre/segredos/{id}` alteram e excluem.
   `POST /api/v1/cofre/segredos/{id}/revelar` devolve a senha. A busca
   não indexa o valor.
4. **Envelope.** Cada segredo tem a própria chave, embrulhada por
   `COFRE_MASTER_KEY`. O texto cifrado está amarrado ao id, à versão e
   à unidade. A chave mestra vem do processo e não é a
   `CONFIG_ENCRYPTION_KEY`.
5. **Auditoria.** Criar, alterar, excluir e revelar gravam o ator, o id
   e a unidade na mesma transação do outbox. O valor não entra no
   payload.

Ficam para uma leva seguinte: retirada com prazo, rotação automática,
pasta com acesso próprio e uma nova autenticação antes de revelar.

## Consequências

A migration `000009_cofre` cria `cofre_segredos`. Em produção o processo
recusa a chave pública de desenvolvimento e recusa repetir a chave da
configuração. Sem `cofre:reveal` ou `cofre:manage` na unidade, a senha
não sai.
