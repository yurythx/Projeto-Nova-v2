# 046 — Saída do Atlas e do Acervo; gestão de serviços no lugar

- **Status:** Aceito e implementado
- **Data:** 2026-10-21
- **Relacionado:** [ADR 013](013-permissao-com-escopo-e-heranca.md) (escopo), [ADR 020](020-conexoes-de-ia.md) (IA), [ADR 026](026-tramite-e-atlas-publico.md) (Trâmite e Atlas), [ADR 031](031-sismarq-e-typesense.md) (Typesense)

## Contexto

O produto deixou de ser o sistema do arquivo municipal. Os plug-ins
**Atlas** (procedimentos, TTDD e assistente) e **Acervo** (unidades de
arquivamento, eliminação, recolhimento, digitalização e consulta pública)
saíram do código. No lugar entrou a gestão de serviços: Produtos digitais,
Atendimento, Ativos, Problemas e Mudanças (regras em
[REGRAS_DE_NEGOCIO.md](../REGRAS_DE_NEGOCIO.md)).

Os ADRs dos dois módulos continuavam descrevendo telas, permissões,
variáveis e roteiros que não existem mais, e a documentação de operação
apontava para pastas apagadas (`docs/atlas`, `docs/acervo`, `scripts/ttdd`).

## Decisão

1. **ADRs são históricos.** Os do Atlas e do Acervo (015, 017–019,
   021–025, 028, 029, 032–039, 041, 042, 044 e 045) ficam como registro e
   são marcados como superados por este. O 026, o 040 e o 043 ficam
   parcialmente superados (ver abaixo).
2. **O que sobrevive aos módulos:**
   - **Classificação do processo no Trâmite** (ADR 026): o processo
     continua guardando o identificador do procedimento e o código da série
     (`atlas_procedimento_id`, `codigo_ttdd`), sem validação contra uma
     tabela. Os nomes ficam porque são contrato da API e colunas do banco.
   - **Conexões de IA** (ADR 020): a configuração e as telas seguem, mas
     nenhum módulo atual consulta a IA. As variáveis `ATLAS_AI_*` mantêm o
     nome para não quebrar instalações existentes.
   - **Índice da Busca Global** (ADR 031): segue pronto, mas nenhum módulo
     atual implementa `search.Indexer`; a busca roda no Postgres.
3. **Código morto removido:** o limitador do assistente
   (`ATLAS_CHAT_RATE_LIMIT_*`) e as configurações `ACERVO_*`, que nenhum
   módulo lia.
4. **Documentação de operação** (README, DEPLOY, guias, RIPD, Dossiê, VPAT,
   regras de negócio) passa a descrever só o que existe.

## Consequências

- Quem tinha `ACERVO_*` ou `ATLAS_CHAT_RATE_LIMIT_*` no `.env` pode
  apagá-las; ficam ignoradas.
- As tabelas do Atlas e do Acervo que as migrations antigas criaram não são
  apagadas por esta decisão. Apagar dado de arquivo é decisão da
  instituição, não da engenharia.
- O VPAT foi avaliado com as telas do Acervo; as telas da gestão de
  serviços ainda precisam de avaliação.
