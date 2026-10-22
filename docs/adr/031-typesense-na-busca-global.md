# 031 — Typesense como índice da Busca Global

- **Status:** Aceito e implementado
- **Data:** 2026-10-15
- **Relacionado:** [ADR 011](011-borda-unica-ip-confiavel-e-busca-por-prefixo.md) (busca por prefixo no Postgres)

## Contexto

A Busca Global consulta cada módulo no Postgres (full-text com prefixo,
respeitando as permissões de quem pergunta). Para conteúdo público e em
volume, um motor dedicado dá busca tolerante a erro de digitação e
facetas. A integração precisa evitar os defeitos clássicos: coleção que
nunca é criada, `filter_by` montado por interpolação (injeção de filtro) e
chave com valor padrão fixo no `docker-compose.yml`.

## Decisão

- **Cliente** (`internal/platform/typesense`), escrito sobre `net/http`:
  - a coleção é garantida pelo próprio cliente (`EnsureCollection`);
  - todo valor de filtro passa por `Quote`/`In` (entre crases, crase
    escapada);
  - o import confere o resultado de cada linha.
- **Chave obrigatória, sem padrão:**
  - `TYPESENSE_URL` liga o índice e então `TYPESENSE_API_KEY` (ou `_FILE`)
    precisa ter 16+ caracteres;
  - produção recusa a chave de `.env.example`;
  - o `deploy.sh` e o deploy contínuo do CI geram a chave quando falta.
- **Só conteúdo público vai para o índice.** O índice não conhece
  permissões. Só implementa `search.Indexer` o módulo cujo conteúdo é
  visível a qualquer um. Os módulos com ACL, sigilo ou público-alvo (Blog,
  Wiki, Arquivos, Trâmite…) ficam na busca do Postgres; indexar um deles
  exige antes decidir como filtrar por permissão no índice. Hoje nenhum
  módulo indexa.
- **Indexação no worker** (`internal/platform/searchindex`):
  - carga completa de cada módulo indexado no boot e a cada
    `TYPESENSE_SYNC_INTERVAL` (padrão 10 min);
  - cada evento do módulo (fila `nexus.search.index`, ligada a
    `<módulo>.#`) pede nova carga dele, agrupada por 2 s;
  - cada carga grava uma geração nova e apaga a anterior: o que saiu do
    módulo sai do índice.
- **Busca com fallback:**
  - a Busca Global consulta o índice pelos módulos indexados e o Postgres
    pelos demais;
  - com o índice fora do ar, ou sem resultado (ainda sem carga), esses
    módulos voltam ao Postgres;
  - a métrica `nexus_search_index_fallback_total{reason}` conta cada volta.
- **Esquema versionado pelo nome** (`TYPESENSE_COLLECTION`, padrão
  `nexus_busca_v1`): mudar um campo pede outro nome. A coleção nova nasce
  vazia e recebe a carga completa.

## Consequências

- O `docker-compose.yml` tem o serviço `typesense` (volume
  `typesense_data`, sem porta publicada, healthcheck pelo `/dev/tcp` do
  bash, porque a imagem não tem curl). API e worker dependem só da ordem de
  subida, não da saúde dele.
- O CI sobe o Typesense para os testes do cliente (`TEST_TYPESENSE_URL`).
