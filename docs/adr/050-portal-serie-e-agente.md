# 050 — Portal de quem pede, série diária e agente da própria máquina

- **Status:** Aceito e implementado
- **Data:** 2026-10-23
- **Relacionado:** [ADR 048](048-roteiro-itsm.md) (o que ficou para depois), [ADR 049](049-correio-contratos-e-cadastros.md), [ADR 016](016-camada-de-aplicacao-e-transacoes.md)

## Contexto

As três levas do roteiro já estavam no produto. Faltavam o acompanhamento
em linguagem de quem pede, um histórico dos indicadores que não invente
número e um agente de inventário que não varra a rede. Plataforma de
analytics e atendente virtual continuam de fora.

## Decisão

1. **Portal.** Quem abre o pedido acompanha em `/atendimento`. Cada
   pedido diz o próximo passo em português comum: a equipe está
   cuidando, pediu uma resposta, cabe avaliar, encerrou ou cancelou.
   Não há uma segunda interface nem um índice público de chamados.
2. **Série.** O worker grava, de hora em hora, o retrato do dia (UTC)
   por unidade e uma linha institucional. O mesmo dia é regravado. Os
   dias anteriores não mudam. `GET /api/v1/atendimento/governanca/serie`
   exige `atendimento:atender` e devolve no máximo 366 dias. Sem
   amostra, o campo fica nulo. Quem cobre várias unidades não recebe
   um MTBF misturado: esse campo fica nulo, e as outras médias saem
   das somas.
3. **Agente.** `POST /api/v1/ativos/inventario`, permissão
   `ativos:reportar`. O programa `cmd/agente` envia o nome da máquina,
   um identificador estável e o sistema operacional. Não lista a rede
   nem lê arquivos. Patrimônio cadastrado à mão não é alterado.
   Desativado e em manutenção conservam o estado. O mesmo relato não
   gera nova auditoria.

## Consequências

A migration `000006_serie_e_agente` cria a série e a coluna `origem`
do ativo. Sem o worker, a série fica vazia: a tela diz que ainda não
há dia medido.
