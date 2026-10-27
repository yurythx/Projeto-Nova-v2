# 075 — Tabela de-para do GLPI: categoria, unidade, prioridade e o prazo da época

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 072](072-importador-do-glpi.md) (importador), [ADR 070](070-relatorio-por-secretaria-e-ola.md)

## Contexto

No GLPI, a categoria não é só um rótulo: as regras de negócio usam a categoria para dar o SLA, e cada entidade (secretaria) tem a sua própria árvore de categorias, com os mesmos nomes. No GLPI da prefeitura, só a PREFEITURA tem SLA (N1 4 h, N2 24 h e N3 36 h, no horário comercial). A SMS e a SEMECEL não têm, e só 42% dos chamados levam SLA. O ADR 072 criava uma oferta inativa por id de categoria, numa unidade só e sem prioridade: "Computadores" viraria três ofertas iguais, e nenhum chamado importado teria prazo.

A API do GLPI não expõe as regras de negócio. O que dá para ler é o resultado delas: o SLA que cada chamado recebeu.

## Decisão

1. **Uma tabela de-para, revisada pela organização antes da carga** (`--mapa`, CSV com `;`). Cada linha é uma secretaria do GLPI (o 2º nível da árvore de entidades) e uma categoria. A linha diz a unidade, a oferta, o tipo e a prioridade daqui. A categoria `*` é o destino padrão da secretaria: é o caminho dos chamados sem categoria e das categorias apagadas. Duas categorias com a mesma oferta viram uma oferta só.
2. **O rascunho vem dos dados.** Ele traz o volume de cada linha e o SLA que os chamados receberam. A prioridade sugerida segue o nível que a PREFEITURA aplicou à categoria com o mesmo nome: N1 é alta, N2 é média e N3 é baixa. A tabela tem dados da prefeitura e fica fora do repositório.
3. **A oferta é da unidade e nasce ativa**, porque a tabela já foi revisada. Se a unidade já tem uma oferta com o mesmo nome, ela é reaproveitada. A oferta nova ganha uma **regra da fila** "oferta → prioridade": os pedidos novos seguem a regra que a categoria dava no GLPI. A regra respeita o limite de 30 por unidade.
4. **A prioridade da tabela vale para todos os chamados importados.** As **metas de resposta e de solução** entram só nos chamados que tiveram SLA no GLPI, convertidas dos tempos do próprio GLPI. Assim, o histórico mostra o que foi prometido na época e não inventa prazo para quem nunca teve um. Os pedidos novos seguem a política da unidade.
5. **A simulação mostra o que falta.** O relatório traz as unidades da tabela que ainda não existem, as secretarias e categorias que a tabela não cobre (com quantos chamados cada uma tem) e quantos chamados entram com prazo. Gravar com uma unidade que falta é recusado.

Sem `--mapa`, o comportamento é o do ADR 072.

## Consequências

A amarração do GLPI passa a ser explícita e revisável: unidade, oferta, regra da fila e política. O prazo dos chamados novos depende da política da unidade (turnos, feriados e metas por prioridade), que é configurada antes da virada. No GLPI da prefeitura, a política segue os níveis N1, N2 e N3 e o horário comercial.

Um dia de SLA do GLPI é convertido como 24 horas corridas. Os SLAs da prefeitura são em minutos e horas. Os grupos técnicos do GLPI não entram: os chamados vão para o técnico, e as equipes daqui são cadastradas.
