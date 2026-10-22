# 046 — Produto novo: o sistema anterior sai por inteiro

- **Status:** Aceito e implementado
- **Data:** 2026-10-22
- **Relacionado:** [ADR 020](020-conexoes-de-ia.md) (IA), [ADR 027](027-notificacoes.md) (notificações), [ADR 031](031-typesense-na-busca-global.md) (Typesense)

## Contexto

Esta base serviu antes ao SISMARQ, o sistema do arquivo público municipal,
com os plug-ins **Atlas** (procedimentos, tabela de temporalidade e
assistente) e **Acervo** (unidades de arquivamento, eliminação,
recolhimento, digitalização e consulta pública). O Projeto-nova-v2 é outro
produto: o microkernel com a gestão de serviços (Produtos digitais,
Atendimento, Ativos, Problemas e Mudanças; regras em
[REGRAS_DE_NEGOCIO.md](../REGRAS_DE_NEGOCIO.md)). Ele nasce com banco novo
e sem instalação a migrar, então nada do sistema anterior precisa ficar
por compatibilidade.

## Decisão

1. **Esquema base.** As migrations do sistema anterior viram uma só,
   `backend/migrations/000001_base.sql`: o esquema atual, sem as tabelas,
   funções e sequências do Atlas e do Acervo, mais os dados de partida
   (aparência, perfis de sistema, sala geral do chat, tipos de processo e
   a flag do Módulo Modelo). A base foi gerada de um banco migrado
   (`pg_dump --schema-only`) e conferida contra ele. O `Down` esvazia o
   esquema. A base não se edita; mudanças entram em migrations novas.
2. **Sem resto funcional:** saem a classificação do processo no Trâmite,
   o limitador do assistente, as configurações e o bucket do acervo, o
   perfil "Arquivo Público", o conteúdo do arquivo no catálogo e no blog,
   e a camada que lia cookies com os nomes antigos.
3. **Nome do produto.** O módulo Go é `github.com/yurythx/projeto-nova-v2`;
   as pastas do frontend são `lib/plataforma` e `components/plataforma`
   (`PlataformaProvider`, `usePlataforma`). Cookies, `localStorage`,
   cabeçalhos do Egress (`X-Projeto-Nova-*`, `Projeto-Nova-Egress/1.0`) e a
   origem dos eventos (`projeto-nova.platform`) usam o nome novo. A
   variável de IA é `IA_*`.
4. **Nomes de infraestrutura com `nova`:** banco, usuário e vhost `nova`,
   exchange `nova.events`, filas `nova.*`, coleção do Typesense, métricas
   e funções SQL `nova_*`, chaves `nova:` do Redis, papéis `nova-*` do
   Keycloak (realm `nova`), domínio de teste `nova.test` e o projeto
   Compose `projeto-nova`. Os nomes herdados da base anterior (`nexus`)
   saem junto.
5. **Documentação.** Saem os 26 ADRs do Atlas e do Acervo e os registros
   do servidor e da VM do sistema anterior (`SERVIDOR_TESTE.md`,
   `INFRA_PROXMOX.md`). O histórico fica no git.

## Consequências

- Um banco criado pelas migrations antigas não sobe com a base. Todo
  ambiente é recriado do zero (`make test-db-reset` no teste; banco novo
  no desenvolvimento e na produção).
- Quem consumia webhooks do Egress precisa ler os cabeçalhos novos.
- O ambiente local muda de projeto Compose: os volumes `projeto-nexus_*`
  ficam órfãos e o banco nasce de novo no projeto `projeto-nova`.
- O Terraform usa `moved` para que o state troque o endereço do módulo da
  VM sem recriá-la.
- O VPAT segue valendo para o casco e os componentes; as telas da gestão
  de serviços ainda precisam de avaliação.
