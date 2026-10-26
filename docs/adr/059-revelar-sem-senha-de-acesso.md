# 059 — Revelar sem pedir a senha de acesso de novo

- **Status:** Aceito e implementado
- **Data:** 2026-10-26
- **Relacionado:** [ADR 054](054-revelacao-e-segredo-no-chamado.md), [ADR 056](056-retirada-com-prazo.md)

## Contexto

Quem abre o cofre já passou pelo login. Pedir a senha de acesso outra
vez só para ver um item não acrescenta a sessão: a pessoa já está
autenticada e a permissão `cofre:reveal` já limita quem chega ali. A
lista continua sem a senha. Reservar o segredo (retirar) ainda trava o
item para as outras pessoas, e esse passo continua confirmando a senha
de acesso.

## Decisão

1. `POST /api/v1/cofre/segredos/{id}/revelar` não tem corpo. A sessão e
   as regras já existentes bastam: `cofre:reveal` na unidade, e, se o
   segredo está numa pasta, a marca de revelar nessa pasta. `cofre:manage`
   na unidade continua revelando. Credencial de máquina não revela.
   Segredo retirado por outra pessoa continua recusado. A auditoria
   grava quem revelou, o id e a unidade, nunca o valor.
2. O item 1 do [ADR 054](054-revelacao-e-segredo-no-chamado.md) fica
   substituído por esta decisão. O item 2 permanece: o chamado guarda só
   o id.
3. `POST /api/v1/cofre/segredos/{id}/retirar` continua exigindo
   `senha_de_acesso` e o prazo, como nos ADRs 056 e 057.

## Consequências

O cliente envia a revelação sem senha. Um corpo antigo com
`senha_de_acesso` é ignorado, porque o handler não lê o corpo. Senha
errada na revelação deixa de contar no bloqueio do login; na retirada,
continua contando.
