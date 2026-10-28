# ADR 080 — Lista de chamados como a do GLPI

## Contexto

Com 15 mil chamados vindos do GLPI, a vista "Lista" da fila ficou estreita para quem atende:

- filtrava só por responsável, tipo e estado;
- não buscava;
- não ordenava pelas colunas;
- agrupava em faixas por estado.

A prefeitura trabalha há anos na lista do GLPI: busca no topo, critérios recolhidos, colunas que ordenam e a mais recém-atualizada primeiro.

A "última atualização" dos chamados importados também estava errada. O gatilho `set_updated_at` grava `now()` em todo UPDATE, então todos ficaram com a hora da carga.

## Decisão

1. **A vista "Lista" da fila segue o GLPI:**
   - a busca (número, título, descrição) fica sempre visível;
   - os demais filtros ficam num painel recolhido por padrão, com o botão "Filtros (n)" e "Limpar filtros": situação, status, tipo, técnico, unidade, categoria do catálogo, prioridade e período de abertura;
   - a tabela é única, sem faixas, com as colunas ID, Título, Unidade, Status, Prioridade, Última atualização, Abertura, Requerente, Técnico, Categoria e Prazo;
   - um clique no título da coluna ordena, outro inverte. Sem escolha, a mais recém-atualizada vem primeiro;
   - 20, 50 ou 100 por página, com "Mostrando a–b de N".
2. **Os filtros moram na URL.** O link leva à mesma lista, e o voltar do navegador funciona. Só o painel aberto ou fechado fica no navegador de quem usa.
3. **O quadro (kanban) não muda.** Continua com responsável e tipo, sem as encerradas.
4. **API** (`GET /atendimento/fila` e `/atendimento/minhas`):
   - novos parâmetros `q`, `unidade`, `categoria`, `prioridade`, `aberta_de` e `aberta_ate` (instantes RFC 3339, período `[de, até)`), `ordem` (uma das colunas) e `sentido` (`asc`/`desc`);
   - o que não se lê é 422;
   - a unidade continua limitada à cobertura de quem pede;
   - a resposta ganha `unidade_nome` e `categoria_nome`.
5. **Última atualização:**
   - o gatilho de `atendimento_solicitacoes` passa a respeitar o `updated_at` dado de propósito (`set_updated_at_se_omitido`); sem ele, continua `now()`;
   - a carga grava a data da última mudança no GLPI (`date_mod`);
   - a migração corrige os chamados já importados que ninguém mexeu desde a carga, pela versão guardada na área de preparo.

## Consequências

- O período é calculado no fuso de quem olha: o navegador manda o começo do dia como instante.
- A ordem por nome (requerente, técnico, unidade, categoria) não diferencia maiúsculas. A ordem por prioridade vai de crítica a baixa, e sem prioridade fica por último.
- As faixas coloridas da lista e a personalização de cores por cartão continuam só no quadro.

## Implementação

- **Migração 41:** a função, o gatilho e a correção dos importados.
- **Backend:**
  - `domain.FiltroSolicitacao` (recortes e `OrdensDaLista`);
  - `infrastructure/solicitacoes.go` (`ordemDaLista`, joins de unidade e categoria);
  - `application.FiltroFila.recortes`;
  - `transport.filtroFila`.
- **Frontend:** `components/atendimento/ListaDeChamados.tsx` (`FiltrosDaLista`, `TabelaDeChamados`, `consultaDaLista`) na página da fila.
