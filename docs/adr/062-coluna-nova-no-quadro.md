# 062 — Coluna nova no quadro

- **Status:** Aceito e implementado
- **Data:** 2026-10-26
- **Relacionado:** [ADR 061](061-catalogo-de-colunas.md), [ADR 016](016-camada-de-aplicacao-e-transacoes.md)

## Contexto

O catálogo já guarda nome, cor e ordem. A chave continua o estado do registro. Faltava nascer uma coluna quando um quadro precisa de um passo que os outros não têm. Copiar a mesma coluna em todos os quadros misturaria fluxos diferentes.

## Decisão

`POST /api/v1/quadros/{quadro}` cria a coluna só naquele quadro. A chave sai no formato `c` e oito caracteres hexadecimais e não muda com o nome. O pedido traz nome, cor, posição a partir de zero, de onde o card entra e para onde pode sair. Sem ao menos uma entrada, a coluna não nasce.

A passagem nova fica em `quadro_passagens`. O fluxo de fábrica de cada módulo continua no código e soma com essas passagens. Quem gere conduz a coluna nova na mudança; aprovar e rejeitar continuam com quem aprova. Na fila, quem atende conduz; quem abriu o pedido não ganha caminho novo. A coluna inicial do fim do fluxo não origina passagem. A aprovação da mudança não entra na coluna nova.

Entrar num estado que já exige causa, contorno, janela ou parecer continua exigindo. A coluna nova não pede campo próprio. Tarefa e ativo aceitam a chave quando ela está ativa e a passagem existe; entre os estados de fábrica o movimento continua o de hoje.

A permissão é a que já gere o módulo. A criação grava outbox e auditoria na mesma transação.

## Consequências

Uma coluna de mudanças não aparece em problemas. O estado gravado aceita a chave nova além da lista de fábrica. Retirar continua exigindo coluna vazia, e a coluna inicial do fluxo não sai.
