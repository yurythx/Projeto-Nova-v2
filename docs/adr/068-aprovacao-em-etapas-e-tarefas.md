# 068 — Aprovação em etapas e tarefas por equipe na requisição

- **Status:** Aceito; aprovação em etapas implementada, tarefas por equipe a seguir
- **Data:** 2026-10-27
- **Relacionado:** [ADR 049](049-correio-contratos-e-cadastros.md) (perguntas da oferta), [ADR 058](058-fila-curta-e-pasta-do-cofre.md) (triagem), [ADR 016](016-camada-de-aplicacao-e-transacoes.md)

## Contexto

A requisição mais pedida numa prefeitura é a entrada e a saída de servidores: conta no AD, e-mail, equipamento, acessos aos sistemas e, na saída, recolher tudo. Hoje ela é um chamado único, sem aprovação, que uma equipe só resolve. Na prática, a chefia precisa autorizar, e o trabalho se divide entre equipes (rede, suporte, patrimônio). A mudança já tem aprovação (CAB); a requisição não tem.

## Decisão

Duas peças opcionais na oferta de **requisição**. A oferta sem elas continua exatamente como hoje.

### Aprovação em etapas

1. A oferta define etapas em ordem, cada uma com nome e aprovadores (pessoas). Basta um aprovador de cada etapa.
2. A requisição de uma oferta com etapas abre em `aguardando_aprovacao`. A equipe vê, mas não atende: não há passagem para `em_atendimento` daí.
3. Aprovar passa à próxima etapa; a última aprovação entrega a requisição à fila (`aberto`), e o relógio de prazo começa ali. Recusar exige motivo e cancela a requisição, com o motivo como resposta a quem pediu.
4. Quem pediu não aprova o próprio pedido, mesmo que seja aprovador da etapa.
5. Cada decisão é gravada (`atendimento_aprovacoes`: etapa, quem, decisão, motivo, quando), com auditoria e evento no outbox na mesma transação, e avisa os aprovadores da etapa seguinte ou quem pediu.

### Tarefas por equipe

1. A oferta define um modelo de tarefas: título e equipe responsável de cada uma. É a lista da admissão (criar conta, entregar notebook, liberar sistemas) ou do desligamento (revogar acessos, recolher equipamento).
2. Quando a requisição chega à fila (aprovada, ou na abertura quando não há etapas), as tarefas são criadas a partir do modelo, uma por linha, e cada equipe é avisada da sua.
3. Quem é da equipe da tarefa a conclui, com observação opcional. A requisição só pode ser dada como atendida quando todas as tarefas estão concluídas ou dispensadas (dispensar exige motivo).
4. O modelo vale para as requisições novas; mudar o modelo não mexe nas tarefas já criadas.

## Consequências

A máquina de estados da requisição ganha `aguardando_aprovacao`, que só existe para ofertas com etapas; o incidente não muda. O prazo não corre enquanto se espera aprovação: o que a central promete começa quando ela pode agir.

O desligamento vira um controle de segurança: a requisição não fecha com acesso por revogar. O registro de quem aprovou e de quem fez cada tarefa responde à auditoria sem planilha à parte.

## Implementação da aprovação

- Migração `000023`: estado `aguardando_aprovacao` só para requisição, `etapa_atual`, `atendimento_oferta_etapas`, `atendimento_aprovacoes` e a coluna do quadro na ordem 0. Desfazer cancela os pedidos que ainda esperam.
- `GET`/`PUT /atendimento/ofertas/{id}/etapas` (gestão da unidade da oferta; até 5 etapas, até 20 aprovadores ativos cada).
- `GET /atendimento/aprovacoes` e `POST /atendimento/solicitacoes/{id}/aprovacao` não pedem permissão do atendimento: a chefia aprova sem atender. Só vê e decide quem é aprovador da etapa atual.
- O aviso ao aprovador abre a tela **Aprovações** (`/atendimento/aprovacoes`), e não o chamado, que a chefia não enxerga.
- A recusa cabe em 1000 caracteres, para que a resposta a quem pediu ("Pedido recusado na etapa …: motivo") nunca passe do limite do comentário.
- O detalhe do chamado mostra a etapa que falta e as decisões tomadas.
