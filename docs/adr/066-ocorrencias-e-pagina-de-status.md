# 066 — Incidente grave e página de status pública

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 048](048-roteiro-itsm.md), [ADR 065](065-ia-assistiva-no-atendimento.md), [ADR 014](014-publico-alvo.md)

## Contexto

Quando um sistema da prefeitura cai, o que mais dói não é a falha, é ninguém saber o que está acontecendo: cada setor abre o seu chamado, a central responde a mesma coisa dezenas de vezes e a gestão sabe pelo telefone. O ITIL trata isso como incidente grave (coordenação e comunicação próprias) e o mercado resolve com uma página de status pública.

## Decisão

Um plug-in próprio, **Ocorrências** (`ocorrencias`, painel em `/ocorrencias`), nasce desligado. A página pública fica em `/status`. Ele não depende do atendimento: uma ocorrência pode ser declarada sem chamado, e aponta para o chamado principal quando há um.

1. **Ocorrência.** Quem tem `ocorrencias:manage` declara uma ocorrência com título público, serviços afetados e impacto (`degradado`, `parcial`, `fora`). Ela começa em `investigando` e segue para `identificado`, `monitorando` e `resolvido`. Quem declara é o coordenador; o coordenador pode ser trocado.
2. **Atualização.** Cada mudança de estado é uma atualização com texto, gravada na linha do tempo. A última atualização é o que a página pública mostra. Resolver exige texto e registra o fim.
3. **Página pública.** `GET /api/v1/status`, sem login e com limite por IP, devolve a situação geral (o pior impacto em aberto, ou `operacional`), as ocorrências abertas e as resolvidas nos últimos 7 dias. Só os campos públicos saem: título, serviços, impacto, estado, horários e o texto das atualizações. O nome de quem atuou, o chamado e a revisão interna não saem.
4. **O texto público é escrito para o público.** A ocorrência não copia título nem descrição do chamado, que podem ter nome, CPF ou detalhe interno. Quem declara escreve o título público.
5. **Revisão pós-incidente.** Depois de resolvida, a ocorrência recebe a revisão interna (causa, o que funcionou, o que fazer) por quem a coordena. A revisão não é pública e fica na trilha de auditoria.
6. **Auditoria e eventos.** Declaração, atualização, troca de coordenador e revisão gravam auditoria e evento no outbox na mesma transação (`ocorrencias.declarada`, `ocorrencias.atualizada`, `ocorrencias.resolvida`, `ocorrencias.revisada`). Os eventos são a entrada para o aviso automático e para o monitoramento (ADR 067).

## Consequências

A central aponta para `/status` em vez de responder caso a caso, e quem abre chamado vê que a falha já é conhecida. A página pública não passa pelo login nem pelo módulo de atendimento: continua de pé quando o atendimento é desligado.

A página só diz o que alguém escreveu. Uma ocorrência esquecida em aberto continua aparecendo: o painel interno lista as abertas com o tempo desde a última atualização, para a coordenação não deixar o público sem notícia.
