# 056 — Retirada com prazo

- **Status:** Aceito e implementado
- **Data:** 2026-10-25
- **Relacionado:** [ADR 053](053-cofre-de-senhas.md), [ADR 054](054-revelacao-e-segredo-no-chamado.md), [ADR 055](055-endereco-e-painel-no-cofre.md)

## Contexto

Duas pessoas não devem usar a mesma senha de máquina ao mesmo tempo sem
que a outra saiba. A revelação continua existindo para quem só precisa
olhar. A retirada reserva o item por um prazo curto.

## Decisão

1. `POST /api/v1/cofre/segredos/{id}/retirar` pede a senha de acesso e
   uma duração de 15 minutos a 8 horas. A resposta traz a senha. A
   auditoria grava quem, o segredo, a unidade e o prazo — nunca a senha.
2. Enquanto a retirada está no prazo, outra pessoa não revela e não
   retira. Quem retirou pode revelar de novo e renovar o prazo.
3. `POST /api/v1/cofre/segredos/{id}/devolver` encerra antes do prazo.
   Quem retirou devolve. Quem administra também. Devolver não troca a
   senha. Trocar a senha ou mover o segredo encerra a retirada.
4. A lista mostra quem retirou e até quando. O prazo vencido não
   aparece: a próxima retirada encerra a linha vencida.

Rotação automática e pasta com acesso próprio ficam para a leva seguinte.

## Consequências

A migration `000012_cofre_retirada` cria `cofre_retiradas`. Há no máximo
uma retirada em aberto por segredo. Credencial de máquina não retira.
