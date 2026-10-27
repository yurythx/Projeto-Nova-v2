# 073 — OLA também no chamado: quem esteve com ele e por quanto tempo

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 070](070-relatorio-por-secretaria-e-ola.md) (relatório e OLA pelas tarefas), [ADR 071](071-contrato-de-apoio.md)

## Contexto

O OLA do ADR 070 media só as tarefas da requisição. A maior parte do volume é incidente, que passa de mão em mão sem tarefa. O chamado guardava só o responsável atual: não se sabia quem esteve com ele antes nem por quanto tempo, e por isso não havia como dizer qual equipe segurou o chamado.

## Decisão

1. **Histórico de atribuição** (`atendimento_atribuicoes`). Assumir o chamado abre um trecho com a pessoa e a equipe dela na unidade do chamado. Se ela estiver em mais de uma equipe da unidade, vale a primeira pelo nome. O trecho fecha quando o chamado muda de mãos, é concluído ou é cancelado. Reabrir o chamado abre um trecho novo para o responsável. Há no máximo um trecho aberto por chamado. A equipe guarda o nome, para o trecho sobreviver se a equipe for apagada.
2. **O tempo da equipe desconta as esperas.** O tempo em que o chamado esteve aguardando o servidor ou o fornecedor sai da conta: não é atraso da equipe, e o fornecedor já é medido pelo contrato de apoio (ADR 071).
3. **O relatório por equipe soma chamados e tarefas.** Em cada equipe aparecem os chamados que passaram por ela, a % dentro do prazo da equipe, a mediana, os que estão com ela agora e os que já passaram do prazo. Os números das tarefas continuam ao lado. Trecho que passou para outra equipe conta como encerrado para a primeira: ela fez a sua parte.
4. **O passado é reconstruído pela migração.** Cada chamado que já tem responsável ganha um trecho, do primeiro "em atendimento" até a conclusão. As trocas de mão anteriores não foram registradas e não aparecem.

## Consequências

A central passa a enxergar onde o incidente para: na triagem, numa equipe sobrecarregada ou na passagem entre equipes. O prazo da equipe vale para os dois tipos de trabalho, o chamado e a tarefa.

Os chamados importados do GLPI (ADR 072) não têm trechos: o GLPI não entrega o histórico de atribuição pela API. Pessoa sem equipe na unidade (por exemplo, gestão que assumiu um chamado) gera trecho sem equipe, e esse trecho não entra no OLA.
