# 071 — Contrato de apoio: o prazo do fornecedor no chamado

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 070](070-relatorio-por-secretaria-e-ola.md) (relatório e OLA), plug-in Contratos

## Contexto

Quando o chamado depende de um fornecedor (link, energia, garantia de equipamento), ele vai para "aguardando terceiro". O prazo da central para enquanto espera, e com razão: não é a equipe que está atrasando. Mas ninguém mede o fornecedor. O contrato está cadastrado no plug-in Contratos, e o chamado não sabe dele. Sem isso, a prefeitura não tem como cobrar o fornecedor pelo que ele combinou.

## Decisão

1. **O contrato tem prazo de atendimento.** Campo opcional em horas corridas (1 a 8760), definido por quem gere contratos (`PUT /contratos/contratos/{id}/prazo`).
2. **O chamado aponta o contrato de apoio.** Quem atende escolhe, entre os contratos vigentes que servem às unidades que atende (e os institucionais), o contrato que o chamado espera (`PUT /atendimento/solicitacoes/{id}/contrato`). Nulo limpa.
3. **O chamado guarda uma cópia** do rótulo (número, título e fornecedor) e do prazo de quando o contrato foi apontado. A medição usa o combinado na hora em que o chamado foi ao fornecedor. Ela não muda se o contrato for editado ou apagado depois, e não depende do plug-in Contratos estar ligado. Não há chave estrangeira entre os plug-ins.
4. **A medição sai das passagens.** Soma-se o tempo em "aguardando terceiro", em todas as idas. O relatório por secretaria ganha a seção "Contratos de apoio": chamados, quantos voltaram do fornecedor, a % desses dentro do prazo, a mediana, quantos estão com o fornecedor agora e quantos desses já passaram do prazo.
5. **A porta fica no `app`.** O atendimento pede a lista e a confirmação por uma interface, e o `app` a liga ao serviço do plug-in Contratos. Com o plug-in desligado, a lista responde 503 e o vínculo é recusado. Os chamados já vinculados continuam medidos.
6. Quem pediu não vê o contrato: é assunto da equipe com o fornecedor.

## Consequências

A central passa a ter números para a reunião com o fornecedor, e a gestão de contratos ganha o argumento para glosa ou renovação. Com o OLA (ADR 070) e o SLA da política de prazo, a cadeia de acordos do ITIL fica completa: o que a central promete, o que cada equipe promete e o que cada fornecedor promete.

O tempo é corrido. Contrato com prazo em horas úteis precisaria do calendário do fornecedor, e isso fica para quando aparecer o primeiro caso.
