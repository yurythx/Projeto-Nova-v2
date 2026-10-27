# Roteiro da gestão de serviços

Decisão registrada no [ADR 048](adr/048-roteiro-itsm.md), a partir da
comparação com o GLPI e com o ServiceNow. O que já funciona permanece.
O que entra segue as regras do repositório: plug-in não importa plug-in,
escrita com auditoria e outbox na mesma transação, texto de interface em
português, indicador sem amostra fica vazio, e mudança de contrato de
API ganha ADR.

## Não se constrói

- Plataforma de analytics no estilo Performance Analytics.
- Atendente virtual ou chatbot.

## Permanece

Fluxo de incidente e requisição, relógio em horário de expediente,
problema e erro conhecido, mudança com CAB e RFC para impressão, ativo
cadastrado à mão, projeto com Gantt e apontamento, escopo por unidade,
Signum, trâmite com sigilo, LGPD, visita de campo com fila offline,
carta de serviços e ouvidoria, webhooks de saída.

## Leva 1 — nesta linha de trabalho

| Entrega | Por que assim |
|---|---|
| Artigo da wiki antes de abrir o chamado | A busca do módulo wiki, com a permissão de quem lê a página. A lista fixa de tutoriais no navegador saiu. |
| Chamado na Busca Global | Quem abriu e quem atende a unidade. Não entra no Typesense: chamado não é conteúdo público (ADR 031). O trecho não mostra a descrição. |
| Aviso à equipe aos 90%, nas 2 horas e no vencimento | O responsável continua o mesmo. Aos 75% só ele é avisado, ou a equipe se ninguém assumiu. |

As migrations `000004`, `000005` e `000006` estão no banco da aplicação,
e as imagens da API, do worker e do frontend foram geradas de novo. Sem
`SMTP_HOST` e `IMAP_HOST`, o correio fica desligado e o sino continua
valendo. O plug-in Contratos nasce desligado: a gestão o ativa em
Configurações.

## Leva 2 — correio

Implementada. A porta SMTP envia o aviso do atendimento e o pedido de
avaliação depois do sino. A caixa IMAP, quando configurada, transforma
a resposta de quem abriu ou de quem atende em comentário. Sem host, o
correio fica desligado e o sino continua valendo.

## Leva 3 — cadastros que o GLPI já tem

Implementada. Contrato, licença e fornecedor avisam o vencimento.
Ativos se relacionam sem varredura de rede. A oferta tem perguntas
próprias, respondidas na abertura. A mudança guarda a nota da
calculadora e respeita a janela de congelamento.

## Depois

Implementado, com o limite que a comparação pediu.

- O portal é a página de quem abre o pedido. Cada item diz o próximo
  passo em português comum. Não há uma segunda interface.
- A série grava o dia civil no fuso da unidade e não recalcula o
  passado. Sem amostra, o número fica vazio. Várias unidades no mesmo
  dia não produzem um intervalo entre falhas.
- O agente informa a capacidade da máquina em que está: sistema,
  processadores, memória, discos locais e programas (nome e versão).
  Não varre a rede e não altera patrimônio cadastrado à mão. A
  credencial da máquina substitui a senha de uma pessoa.

## Leva 4 — ITIL 5

A comparação com o ITIL 5 e com o mercado (ServiceNow, Jira Service
Management, Freshservice, GLPI) pediu IA com governança, incidente grave
com comunicação, eventos do monitoramento, aprovação em etapas,
experiência medida e nível de serviço por secretaria.

| Entrega | Estado |
|---|---|
| IA assistiva: classificação e resumo para quem atende, cada sugestão e decisão registrada ([ADR 065](adr/065-ia-assistiva-no-atendimento.md)) | Implementada |
| Incidente grave e página de status pública ([ADR 066](adr/066-ocorrencias-e-pagina-de-status.md)) | Implementada |
| Eventos do monitoramento viram incidente | A seguir |
| Aprovação em etapas; admissão e desligamento de servidores | A seguir |
| Experiência medida além do CSAT (XLA) | A seguir |
| SLA, OLA e contrato de apoio; relatório por secretaria | A seguir |
| Importador do GLPI | A seguir |

A IA continua sem conversar com quem abre o chamado: o atendente virtual
segue fora (acima, "Não se constrói").
