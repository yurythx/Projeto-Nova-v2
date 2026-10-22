# 031 — O projeto passa a se chamar SISMARQ; Typesense volta como índice da Busca Global

- **Status:** Aceito e implementado
- **Data:** 2026-10-15
- **Relacionado:** [ADR 015](015-atlas-padronizado.md) (que removeu o Typesense), [ADR 018](018-atlas-paginas-e-busca-unificada.md) (busca do Atlas), [ADR 026](026-tramite-e-atlas-publico.md) (Atlas público), [ADR 030](030-blog-autoria-dedicada-publico-alvo-e-estrutura-rondonopolis.md) (nome Projeto-Nexus)

## Contexto

A base "Projeto Nexus" passou a ser o **SISMARQ**: o Sistema Municipal de
Arquivos de Rondonópolis (Lei Municipal nº 10.301/2019), descrito no artigo
em `docs/SISMARQ/`. O repositório já se chama SISMARQ, mas o código, a
interface e a documentação ainda diziam "Nexus".

O artigo prevê busca facetada e tolerante a erros de digitação sobre o
acervo (com OCR), num motor dedicado: o Typesense. O ADR 015 tinha tirado o
Typesense do Atlas, mas por defeitos da integração, não do motor:

- a coleção nunca era criada;
- o `filter_by` era montado por interpolação (injeção de filtro);
- a chave tinha default fixo no `docker-compose.yml`.

## Decisão

### 1. Renome (código e marca)

- Módulo Go `github.com/yurythx/sismarq`. Pastas do frontend
  `lib/sismarq` e `components/sismarq`, com `SismarqProvider` e
  `useSismarq`. Textos, títulos, documentação, CI, Ansible e o nome
  padrão do Branding (migration `000144`, que só troca quem ainda usa o
  padrão antigo).
- **Ficam como estão**, porque já estão gravados ou são contratos com
  sistemas de fora:
  - o projeto Compose `projeto-nexus` (os volumes de dados levam o
    prefixo);
  - banco, usuário e vhost `nexus`, o bucket do MinIO e a exchange
    `nexus.events`;
  - o realm, os clients e os papéis `nexus-*` do Keycloak, e os grupos
    `/Nexus/...`;
  - as funções SQL `nexus_*`, as migrations já aplicadas, as chaves
    `nexus:` no Redis e as métricas `nexus_*`;
  - os cookies e o `localStorage`, os cabeçalhos `X-Nexus-*` e o
    User-Agent do Egress (os receptores conferem a assinatura), e a
    origem `projeto-nexus.platform` dos eventos — **exceção (2026-10-17):**
    cookies de preferência do visitante passaram a `sismarq-*` (com
    leitura do nome legado), o Egress envia `X-SISMARQ-*` /
    `SISMARQ-Egress/1.0` e a origem dos eventos é `sismarq.platform`.
    O Arquivo Público não tem receptores PaperMoon desses cabeçalhos;
  - a CA de teste do Caddy (`nexus-ca.crt`, já instalada nos aparelhos),
    o Terraform (mudar descrições e senhas mexe na infraestrutura criada),
    o caminho `/root/Projetos/Projeto-Nexus` no servidor;
  - os ADRs 002 a 030, que são registro histórico. Os seeds
    `deploy/seeds/seed_*_rondonopolis.sql` (Nexus Corporation) **não**
    se aplicam no Arquivo Público — ver `deploy/seeds/LEIA-ME.md`.
    Catálogo/blog PaperMoon saem na migration `000157`.

### 2. Typesense como índice da Busca Global

- **Cliente** (`internal/platform/typesense`), escrito sobre `net/http`:
  - a coleção é garantida pelo próprio cliente (`EnsureCollection`);
  - todo valor de filtro passa por `Quote`/`In` (entre crases, crase
    escapada);
  - o import confere o resultado de cada linha.
- **Chave obrigatória, sem default:**
  - `TYPESENSE_URL` liga o índice e então `TYPESENSE_API_KEY` (ou `_FILE`)
    precisa ter 16+ caracteres;
  - produção recusa a chave de `.env.example`;
  - o `deploy.sh` e o deploy contínuo do CI geram a chave quando falta.
- **Só conteúdo público vai para o índice.** O índice não conhece
  permissões. Implementa `search.Indexer` apenas o módulo cujo conteúdo é
  visível a qualquer um. Hoje é o Atlas: procedimentos em vigor, séries
  vigentes da TTDD e modelos ativos, o mesmo da consulta sem login. Os
  módulos com ACL, sigilo ou público-alvo (Blog, Wiki, Arquivos, Trâmite…)
  continuam na busca do Postgres. Indexar um deles exige primeiro uma
  decisão sobre como filtrar por permissão no índice.
- **Indexação no worker** (`internal/platform/searchindex`):
  - carga completa de cada módulo no boot e a cada `TYPESENSE_SYNC_INTERVAL`
    (padrão 10 min);
  - cada evento do módulo (fila `sismarq.search.index`, ligada a
    `<módulo>.#`) pede nova carga dele, agrupada por 2 s;
  - cada carga grava uma geração nova e apaga a anterior: o que saiu do
    módulo (rascunho, revogado, desativado) sai do índice.
- **Busca com fallback:**
  - a Busca Global consulta o índice pelos módulos indexados e o Postgres
    pelos demais;
  - com o índice fora do ar, ou sem resultado (ainda sem carga), esses
    módulos voltam ao Postgres;
  - a métrica `nexus_search_index_fallback_total{reason}` conta cada volta.
- **Schema versionado pelo nome** (`TYPESENSE_COLLECTION`, padrão
  `sismarq_busca_v1`): mudar um campo pede outro nome. A coleção nova nasce
  vazia e recebe a carga completa.

## Consequências

- O `docker-compose.yml` ganha o serviço `typesense` (volume
  `typesense_data`, sem porta publicada, healthcheck pelo `/dev/tcp` do
  bash, porque a imagem não tem curl). API e worker dependem só da ordem de
  subida, não da saúde dele.
- O CI sobe o Typesense para os testes do cliente (`TEST_TYPESENSE_URL`).
- A busca dentro das páginas do Atlas (ADR 018) segue no Postgres. Passar
  para o índice fica para quando houver o acervo do SISMARQ (OCR, facetas
  por classe, fase e ano), que é o caso que justifica o motor.
- Quem tem um `.env` antigo precisa de `TYPESENSE_API_KEY`: o `deploy.sh`
  e o deploy do CI a geram; em desenvolvimento, copie a linha de
  `.env.example`.
