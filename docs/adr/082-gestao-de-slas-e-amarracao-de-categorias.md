# ADR 082 — Gestão de SLAs Independentes e Amarração por Categoria (ITIL v5 / GLPI)

## Contexto

Originalmente, o cálculo de prazos no atendimento do Projeto Nova utilizava exclusivamente uma matriz bidimensional de Urgência × Impacto vinculada à política da unidade/instituição.
No entanto, no GLPI e nas melhores práticas da ITIL v5, os Acordos de Nível de Serviço (SLAs) são configurados de maneira independente e vinculados diretamente às **Categorias de Serviço** (ITIL Categories) ou Ofertas do catálogo. Cada categoria pode possuir seu tempo de Primeira Resposta (TTO - Time To Own) e tempo de Solução (TTR - Time To Resolve), com regras de pausa em espera e calendário (expediente útil vs. corrido 24x7).

Para garantir paridade operacional com o GLPI e permitir governança granular por tipo de demanda, foi implementado o módulo de Gestão de SLAs Independentes e a amarração direta nas categorias de atendimento.

## Decisão

1. **Entidade SLA Independente (`atendimento_slas`):**
   - Suporte a nome, descrição, tempo de primeira resposta (minutos), tempo de solução (minutos), pausa em espera (`pausar_em_espera`) e tipo de calendário (`expediente` ou `corrido_24x7`).
   - Escopo institucional ou por unidade administrativa (`unidade_id`).

2. **Amarração nas Categorias (`atendimento_categorias.sla_id`):**
   - Cada categoria de atendimento pode apontar para uma SLA específica através de `sla_id` (com chave estrangeira e deleção `ON DELETE SET NULL`).
   - A gestão do atendimento pode alterar a SLA vinculada via endpoint `PUT /v1/atendimento/categorias/{id}/sla` ou pelo formulário de edição da categoria.

3. **Cálculo de Prazo na Abertura e Classificação do Chamado:**
   - Ao abrir um chamado (`abrirNa`) ou reclassificá-lo (`Classificar`), se a categoria vinculada à oferta possuir uma SLA definida (`cat.SLAID != nil`), os tempos de primeira resposta e solução da solicitação (`sol.MetaRespostaMin` e `sol.MetaSolucaoMin`) são determinados diretamente pela SLA associada, garantindo conformidade com os níveis de serviço contratados para aquela categoria de serviço.
   - Chamados sem SLA específica na categoria continuam usando o cálculo dinâmico da matriz de Urgência × Impacto da política da unidade.

4. **Interface Administrativa Unificada:**
   - A tela `/atendimento/prazos` consolida:
     - Catálogo de SLAs Oficiais (com criação, edição e exclusão de SLAs).
     - Matriz de Amarração de SLAs nas Categorias de Chamado em tabela interativa com seletor `<Select>`.
     - Política de Expediente (turnos de trabalho) e Feriados / Pontos Facultativos.

## Consequências

- **Positivas**: Flexibilidade total para definir prazos específicos por tipo de serviço (ex.: falha de link crítico com SLA de 2 horas vs. solicitação de toner com SLA de 24 horas) sem distorcer a matriz genérica de impacto/urgência.
- **Interoperabilidade**: Compatibilidade nativa com as importações e sincronizações contínuas do GLPI.
- **Auditoria**: Todas as alterações de SLA e amarrações geram eventos de domínio auditáveis (`atendimento.sla.criada`, `atendimento.sla.atualizada`, `atendimento.categoria.sla_vinculada`).
