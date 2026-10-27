# 065 — IA assistiva no atendimento: sugere, quem atende decide

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 020](020-conexoes-de-ia.md) (conexões de IA), [ADR 048](048-roteiro-itsm.md) (sem atendente virtual), [ADR 016](016-camada-de-aplicacao-e-transacoes.md)

## Contexto

O ITIL 5 trata a IA como parte normal da operação e pede governança sobre ela. As conexões de IA existem desde o ADR 020 (local por padrão, mascaramento de dados pessoais, autorização para fornecedor externo), mas nenhum módulo as usava. O ADR 048 descartou o atendente virtual: a IA não conversa com quem abre o chamado.

## Decisão

A IA ajuda **quem atende**, em duas tarefas, e nunca muda o chamado sozinha.

| Tarefa | O que a IA devolve | Quem pede | Efeito |
|---|---|---|---|
| Classificação | impacto e urgência (1 a 3) e o motivo | quem atende a unidade | nenhum até a pessoa aplicar pela classificação de sempre |
| Resumo | o problema, o que já foi feito e o que falta, em até 5 linhas | quem atende a unidade | nenhum; fica na tela |

1. **Função própria.** A IA do atendimento é a função `atendimento.assistente` em Configurações > Inteligência artificial, com principal e reserva, separada da função da plataforma. Sem conexão, os botões não aparecem (`GET /atendimento/ia`).
2. **Só o necessário sai.** A classificação leva título, descrição e nome da oferta. O resumo leva também a conversa, inclusive as notas internas, porque só a equipe o vê. Com fornecedor externo, o mascaramento de CPF, CNPJ, e-mail e telefone do ADR 020 vale aqui também. Quem abriu não é identificado: a pergunta não leva nomes.
3. **Cada sugestão é registrada** (`atendimento_ia_sugestoes`): o chamado, a tarefa, a conexão que respondeu, o que foi sugerido, quem pediu e quando. Não guarda a pergunta enviada, que está no próprio chamado.
4. **A decisão também.** Quem atende marca a sugestão como aceita ou recusada; uma sugestão só é decidida uma vez. Aceitar a classificação não a aplica: a classificação continua pela rota de sempre, com prazo, evento e auditoria próprios.
5. **Resposta fora do formato não vira sugestão.** Impacto ou urgência fora de 1 a 3, ou texto sem o JSON pedido, devolvem 503, e nada é gravado.
6. **Medição.** `GET /atendimento/ia/indicadores` dá, por tarefa e nos últimos 30 dias do escopo de quem consulta, quantas sugestões houve, quantas foram aceitas, recusadas e ficaram sem decisão. Uma taxa de aceite baixa é o sinal para trocar o modelo, e não para confiar mais nele.
7. **Auditoria.** Pedido e decisão entram na trilha (`atendimento.ia.sugerida`, `atendimento.ia.decidida`) na mesma transação do registro.

## Consequências

Nenhum campo do chamado muda por causa da IA: o que muda passa pelas regras e pela auditoria de sempre, com o nome de quem decidiu. A IA local (Ollama) mantém os dados na rede da prefeitura; a externa precisa da autorização registrada do ADR 020.

As sugestões guardam texto gerado a partir do chamado e seguem a retenção dele: saem junto com o chamado.
