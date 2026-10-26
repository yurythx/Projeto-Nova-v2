# 057 — Rotação da senha guardada

- **Status:** Aceito e implementado
- **Data:** 2026-10-25
- **Relacionado:** [ADR 055](055-endereco-e-painel-no-cofre.md), [ADR 056](056-retirada-com-prazo.md)

## Contexto

Depois da retirada, a senha que a pessoa viu não deve continuar valendo
no cofre. O cofre não entra na máquina, no painel nem na aplicação: não
há conector para trocar a senha lá.

## Decisão

1. O cadastro marca `rotaciona`. Sem a marca, devolver não troca a senha.
2. Com a marca, a senha guardada muda ao devolver, ao vencer a retirada
   e a cada 30 dias. Não muda enquanto alguém está com a retirada aberta.
   Uma rotação já avisada e ainda não aplicada também espera.
3. A lista ganha `rotacao_pendente`. A senha nova só sai na revelação,
   com a senha de acesso de novo. A auditoria grava a versão, nunca o
   valor. `POST /api/v1/cofre/segredos/{id}/rotacao/aplicada` tira o
   aviso depois que alguém aplica no endereço.
4. Trocar a senha à mão vale como a senha atual e zera o aviso.

Pasta com acesso próprio fica para a leva seguinte.

## Consequências

A migration `000013_cofre_rotacao` acrescenta as colunas. Itens já
gravados não rotacionam até alguém marcar. O worker `rotacao-do-cofre`
só corre com o módulo ligado.
