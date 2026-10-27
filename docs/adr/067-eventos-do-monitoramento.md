# 067 — Alertas do monitoramento viram incidente

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 065](065-ia-assistiva-no-atendimento.md), [ADR 066](066-ocorrencias-e-pagina-de-status.md), [docs/DEPLOY.md](../DEPLOY.md) (Monitoramento)

## Contexto

O Alertmanager dispara quando a API para, o disco enche ou o backup atrasa (`deploy/observability/alert_rules.yml`), mas os alertas iam para uma rota que não existia e se perdiam. Na gestão de eventos do ITIL, o evento que pede ação vira incidente sem esperar que alguém perceba.

## Decisão

1. **Entrada.** `POST /api/v1/atendimento/eventos/alertmanager` recebe o webhook do Alertmanager (formato versão 4). Sem sessão de pessoa: exige o token de `ATENDIMENTO_EVENTOS_TOKEN` no cabeçalho `Authorization: Bearer`, comparado em tempo constante. Sem o token configurado, a rota responde 404: a entrada nasce desligada.
2. **Um alerta, um chamado.** Cada alerta tem uma *fingerprint*. Enquanto o chamado dela estiver aberto, o mesmo alerta disparando de novo só acrescenta uma nota interna; não abre outro chamado. Fechado o chamado, um novo disparo abre um novo.
3. **Onde cai.** O incidente é aberto na oferta escolhida por quem gere o atendimento em `PUT /atendimento/eventos/configuracao`, com a unidade dela. Sem oferta escolhida, o evento é recusado com 409 e o Alertmanager tenta de novo; nada se perde em silêncio.
4. **Quem abre.** A conta de sistema `monitoramento` (criada pela migration, sem senha, sem login) aparece como solicitante. Assim o chamado diz de onde veio, e ninguém recebe aviso de "seu chamado" por engano.
5. **Classificação.** `severity=critical` abre com impacto e urgência 3; o resto, com 2. A política de prazo da unidade vale como em qualquer abertura.
6. **Normalizou não é resolvido.** Quando o alerta volta ao normal (`status=resolved`), o chamado ganha a nota interna "o alerta normalizou às …". Quem atende confirma e conclui, como manda o ITIL: o sintoma sumir não prova que a causa foi tratada.
7. **Só o que o alerta diz.** Título e descrição vêm de `summary`, `description` e dos rótulos do alerta, cortados nos limites do chamado. Nada de fora entra.

## Consequências

O Alertmanager passa a mandar os alertas para a entrada, com o token num arquivo gerado na VM, como o de `/metrics`. Um alerta que dispara sem parar não enche a fila. Ligar a página de status (ADR 066) a partir de um incidente continua sendo decisão de quem coordena: um alerta interno não vira aviso público sozinho.
