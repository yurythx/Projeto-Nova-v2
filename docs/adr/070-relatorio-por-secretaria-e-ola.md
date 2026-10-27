# 070 — Relatório por secretaria e OLA das equipes

- **Status:** Aceito e implementado (o contrato de apoio está no [ADR 071](071-contrato-de-apoio.md))
- **Data:** 2026-10-27
- **Relacionado:** [ADR 068](068-aprovacao-em-etapas-e-tarefas.md) (tarefas por equipe), [ADR 069](069-experiencia-de-quem-pede-xla.md) (experiência)

## Contexto

Os acordos de nível do ITIL se encadeiam. O **SLA** é o que a central promete a quem pede; ele já existe na política de prazo por unidade e prioridade. O **OLA** é o que cada equipe interna promete à central para que o SLA se cumpra. O **contrato de apoio** (UC) é o que o fornecedor promete. Na prefeitura, a pergunta que chega à reunião de resultados é por secretaria: quanto a Saúde pediu, quanto ficou no prazo, como foi para quem pediu. A governança atual responde para o escopo inteiro de quem consulta, não por secretaria.

## Decisão

1. **Secretaria é a unidade de topo da árvore.** O pedido aberto para um departamento conta para a secretaria de que ele faz parte. O pedido sem unidade, ou com unidade fora da árvore, vai para "Institucional".
2. **Relatório por secretaria** (`GET /atendimento/relatorio/secretarias?dias=`, padrão 90, teto 366), no escopo de quem atende. Cada secretaria, e o total, traz: volume (abertos, concluídos, cancelados), resolução, prazo cumprido, primeiro contato e as medidas de experiência com as metas da instituição (ADR 069). A amostra tem o mesmo teto da governança; acima dele ficam os pedidos mais recentes, e a resposta avisa.
3. **OLA da equipe.** A equipe ganha um prazo opcional, em horas corridas, para as tarefas que recebe (ADR 068). O relatório mostra, por equipe, as tarefas concluídas, quantas no prazo, a mediana e as pendentes atrasadas. Tarefa dispensada não conta, porque não houve trabalho a medir. Equipe sem prazo aparece com os números, mas sem percentual no prazo.
4. A tela **Relatório** do atendimento mostra as duas tabelas, baixa em CSV e imprime. As metas de experiência aparecem marcadas com ✓ ou ✗, para que a folha impressa também mostre o que ficou abaixo.

## Consequências

A central leva à reunião um número por secretaria em vez de um total que esconde as diferenças. O OLA torna visível qual equipe segura o pedido de admissão ou de desligamento.

O OLA mede só as tarefas da requisição. A passagem do chamado entre equipes no incidente não tem histórico por equipe hoje. Se precisar, entra depois, a partir das passagens de responsável.

O **contrato de apoio** fica para a entrega seguinte. Ele precisa ligar o pedido em "aguardando terceiro" a um contrato do plug-in Contratos, com o prazo do fornecedor, e isso pede uma porta entre os dois plug-ins.
