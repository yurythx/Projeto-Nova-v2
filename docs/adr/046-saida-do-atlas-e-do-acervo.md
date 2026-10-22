# 046 — Saída do Atlas e do Acervo; esquema base para um banco novo

- **Status:** Aceito e implementado
- **Data:** 2026-10-22
- **Relacionado:** [ADR 013](013-permissao-com-escopo-e-heranca.md) (escopo), [ADR 020](020-conexoes-de-ia.md) (IA), [ADR 026](026-tramite-e-atlas-publico.md) (Trâmite e Atlas), [ADR 031](031-sismarq-e-typesense.md) (Typesense)

## Contexto

O produto deixou de ser o sistema do arquivo municipal. Os plug-ins
**Atlas** (procedimentos, TTDD e assistente) e **Acervo** (unidades de
arquivamento, eliminação, recolhimento, digitalização e consulta pública)
saíram do código. No lugar entrou a gestão de serviços: Produtos digitais,
Atendimento, Ativos, Problemas e Mudanças (regras em
[REGRAS_DE_NEGOCIO.md](../REGRAS_DE_NEGOCIO.md)).

O que ficou para trás:

- 67 migrations com o histórico do produto anterior: tabelas criadas e
  depois abandonadas (40 do Atlas e do Acervo), conteúdo do arquivo
  semeado no catálogo e no blog, renomeações de marca;
- a classificação do processo no Trâmite (procedimento do Atlas e série
  da TTDD), sem rota, sem tela e sem quem a lesse;
- configuração que nada usava (limitador do assistente, `ACERVO_*`) e
  variáveis de IA com o prefixo `ATLAS_`;
- ADRs e documentação de operação que descreviam telas e roteiros que não
  existem mais.

A aplicação nasce com **banco novo**: não há instalação a migrar, então
nada disso precisa ser mantido por compatibilidade.

## Decisão

1. **Esquema base.** As migrations viram uma só,
   `backend/migrations/000001_base.sql`: o esquema atual, sem as tabelas,
   funções e sequências do Atlas e do Acervo, mais os dados de partida
   (aparência, perfis de sistema, sala geral do chat, tipos de processo e
   a flag do Módulo Modelo). O conteúdo do arquivo municipal no catálogo e
   no blog não entra. A base foi gerada de um banco migrado
   (`pg_dump --schema-only`) e conferida contra ele: o esquema que ela
   produz é o mesmo. O `Down` esvazia o esquema. A base não se edita;
   mudanças entram em migrations novas.
2. **Trâmite sem classificação.** Saem `atlas_procedimento_id` e
   `codigo_ttdd` do processo, da API e do evento.
3. **IA com nome próprio.** As variáveis passam a `IA_ENDPOINT`,
   `IA_API_KEY`, `IA_MODELO` e `IA_TIMEOUT`. A configuração e as telas
   seguem, mas nenhum módulo atual consulta a IA.
4. **Índice da busca** (ADR 031) segue pronto, mas nenhum módulo atual
   implementa `search.Indexer`; a busca roda no Postgres.
5. **ADRs são históricos.** Os do Atlas e do Acervo (015, 017–019,
   021–025, 028, 029, 032–039, 041, 042, 044 e 045) ficam como registro,
   marcados como superados por este; o 026, o 040 e o 043, como
   parcialmente superados. A documentação de operação descreve só o que
   existe.

## Consequências

- Um banco criado pelas migrations antigas não sobe com a base (o goose
  veria a versão 165 sem os arquivos). Como não há instalação, todo
  ambiente é recriado do zero (`make test-db-reset` no teste; volume novo
  no desenvolvimento).
- O histórico das 67 migrations fica no git, no commit anterior a este.
- O VPAT foi avaliado com as telas do Acervo; as telas da gestão de
  serviços ainda precisam de avaliação.
