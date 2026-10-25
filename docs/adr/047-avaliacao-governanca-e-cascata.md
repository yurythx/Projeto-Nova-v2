# 047 — Avaliação, indicadores da central e conclusão dos vínculos

- **Status:** Aceito e implementado
- **Data:** 2026-10-23
- **Relacionado:** [ADR 016](016-camada-de-aplicacao-e-transacoes.md) (transação), [ADR 027](027-notificacoes.md) (aviso depois do commit)

## Contexto

A nota de satisfação ficava no navegador e o comentário virava mensagem
na conversa, então a central não tinha média. O relatório executivo
calculava MTTR a partir de `updated_at`, MTBF como 720 dividido pelos
incidentes da primeira página, SLA com uma fórmula e FCR com a constante
82,4. A tela de mudança disparava um POST por vínculo e engolia o erro.
O worker já avisava aos 75%, 90% e no vencimento; faltava o aviso quando
faltam duas horas e o percentual ainda não cruzou o limiar.

## Decisão

1. **Avaliação.** Tabela `atendimento_avaliacoes`, uma por solicitação.
   `POST /api/v1/atendimento/solicitacoes/{id}/avaliacao` grava nota de 1
   a 5 e comentário opcional. Só o solicitante, só nos estados
   `resolvido`, `atendido` e `encerrado`, uma vez. Evento
   `atendimento.solicitacao.avaliada` e auditoria na mesma transação. O
   detalhe devolve a avaliação. A exportação LGPD inclui as avaliações
   da pessoa; a eliminação as conserva, como os comentários.
2. **Indicadores.** `GET /api/v1/atendimento/governanca`, permissão
   `atendimento:atender`, no mesmo recorte da fila. MTTR, MTBF, SLA e FCR
   saem dos relógios e das passagens (regras em
   [REGRAS_DE_NEGOCIO.md](../REGRAS_DE_NEGOCIO.md)). Número ausente é
   `null`. A amostra para em 5.000 e marca `truncado`.
3. **Duas horas.** Com o relógio correndo, vencimento marcado e meta
   maior que 2 horas, o worker grava o aviso de chave
   `atendimento:{id}:prazo:{relogio}:2h`, independente do percentual.
4. **Vínculos da mudança.** `POST /api/v1/mudancas/{id}/estado` aceita
   `resolver_vinculos`. Na conclusão, o módulo de mudanças pede a cada
   efeito, na mesma transação e na ordem estável dos vínculos, a
   conclusão que o domínio do atendimento ou do problema já permite.
   Recusa de um vínculo desfaz a passagem da mudança. A entrega do sino
   ocorre depois do commit. Plug-in não importa plug-in: a ligação fica
   em `servicos_ports.go`.

## Consequências

O OpenAPI passa a descrever as duas rotas e o campo `resolver_vinculos`.
O painel deixa de inventar indicador quando a amostra é vazia. Uma
mudança que pede a cascata ou conclui os vínculos prontos, ou não muda
de estado.
