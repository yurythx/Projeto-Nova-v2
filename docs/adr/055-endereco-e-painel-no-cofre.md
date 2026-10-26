# 055 — Endereço e painel no cofre

- **Status:** Aceito e implementado
- **Data:** 2026-10-25
- **Relacionado:** [ADR 053](053-cofre-de-senhas.md), [ADR 054](054-revelacao-e-segredo-no-chamado.md)

## Contexto

O cofre é da organização, não da pessoa. A TI guarda a senha de uma
máquina, de um administrador de aplicação e de um painel que administra
essas máquinas. O endereço precisa aparecer na lista para achar o item
certo. A senha continua selada.

## Decisão

1. Cada segredo tem tipo (`maquina`, `aplicacao`, `painel` ou `outro`)
   e endereço (IP, host ou URL). Os dois saem na lista. Não entram no
   envelope.
2. Um painel aponta máquinas pelo identificador, na tabela
   `cofre_administra`. A senha da máquina não é copiada. Máquina fora
   do alcance de quem consulta não aparece no vínculo. Gravar de novo
   não apaga um vínculo que essa pessoa não vê.
3. A credencial do agente de inventário continua fora do cofre.

## Consequências

A migration `000011_cofre_endereco_e_painel` acrescenta as colunas e a
tabela. Itens já gravados ficam como `outro`, sem endereço.
