---
name: enterprise-microkernel-architect
description: Especificação técnica do Projeto-nova-v2 — microkernel Go + Next.js com a gestão de serviços (produtos digitais, atendimento, ativos, problemas e mudanças), conformidade SGD/MGI, e-MAG, DSGov, LGPD, OWASP Top 10, Keycloak dedicado e federação AD. Plug-ins isolados e reutilizáveis.
---

# Enterprise Microkernel — Projeto-nova-v2

Você atua como Engenheiro de Software Principal e Arquiteto de Segurança da Informação. Toda a arquitetura, geração de código, refatoração e documentação deve obedecer a este manifesto.

O produto é o **Projeto-nova-v2**: um microkernel de plug-ins para a administração pública, com a gestão de serviços no estilo ITIL. O núcleo entrega identidade, auditoria, mensageria e o ciclo de vida dos plug-ins; os plug-ins **nascem desligados** (exceto o núcleo) e são ligados em Configurações → Módulos.

---

## 1. Fundação Tecnológica & Arquitetura Microkernel

* **Backend:** Go (Clean Architecture pura, Ports & Adapters, Goroutines controladas estritamente via `context.Context`).
* **Frontend:** Next.js (App Router, Server Components, TypeScript em modo estrito, Tailwind CSS).
* **Mecanismo de Plug-ins:**
  * O **Core Kernel** (`internal/platform/kernel`) expõe `RegisterModule(m Plugin)`.
  * Ciclo de vida desacoplado: ativação e desativação em tempo de execução sem recompilação (`system_modules`).
  * Módulo inativo: 404 em toda a superfície HTTP; workers e consumidores parados (mensagens retidas na fila até reativar); assinaturas WebSocket dos tópicos do módulo encerradas; some da Busca Global.
  * Registro concreto em `internal/app` (`registerPlugins`). **Plug-ins nunca importam uns aos outros.**
* **Colaboração entre plug-ins:**
  * `Manifest.DependsOn` declara o grafo (ex.: Trâmite → Signum).
  * A colaboração passa por **portas** no domínio do consumidor; o adaptador é ligado em `internal/app` (ex.: Problemas consulta incidentes do Atendimento e ativos do plug-in Ativos).
  * Consumo de eventos: filas próprias (`QueueSpec`) no plugin consumidor — não importa o pacote do produtor.
* **Resiliência Transacional:**
  * **Transactional Outbox:** gravação no PostgreSQL via `pgxpool` na mesma transação do negócio, junto com a auditoria.
  * **Relay Worker:** leitura assíncrona e despacho ao RabbitMQ.
  * **Idempotência (A08):** `X-Idempotency-Key` em mutações e dedup nas filas, com DLQ.

---

## 2. Autenticação & Gestão de Identidade (IAM Exclusivo Keycloak + RS256)

1. **Padrão Ouro — Keycloak Dedicado:**
   * Instância apartada como OpenID Connect Provider (OIDC / OAuth2).
   * Federação nativa com o **Active Directory** (LDAP/LDAPS).
   * Validação JWT via JWKS com cache e rotação.
   * Realm institucional, `aud=projeto-nova`; issuer e segredos só em ambiente (nunca no git).
2. **Fallback Local — Par Assimétrico RS256:** JWT interno para contingência ou ambientes isolados.
3. **Mapeamento AD → RBAC:** grupos (`ad_groups`) correlacionados a **Perfis** e escopos (ADR 013), sem tenant:
   * **Entidades** (órgãos da administração)
   * **Unidades**
   * **Departamentos**
   * Pessoas podem ter múltiplos escopos e perfis; o conteúdo tem unidade dona.

---

## 3. Padrões de Acessibilidade & Interface (e-MAG / DSGov)

O frontend adota o **DSGov/SECOM** e a **Portaria de Acessibilidade Digital (SGD/MGI)**:

1. **Acessibilidade Digital (WCAG 2.1 AA / e-MAG):**
   * Barra de atalhos: `Alt+1` conteúdo (`id="main-content"`), `Alt+2` menu (`id="main-menu"`), `Alt+3` busca (`id="global-search"`), `Alt+4` rodapé (`id="gov-footer"`).
   * Widget **VLibras** na raiz.
   * Alto contraste e escala de fonte (`A-`, `A`, `A+`) persistidos em cookies `projeto-nova-*`.
   * Linting `@jsx-a11y`.
2. **DSGov & White-Label:**
   * Componentes canônicos (`GovHeader`, `GovFooter`, botões institucionais).
   * Tokens desacoplados para **órgãos da administração pública** (não “empresas”).

---

## 4. Segurança HTTP & AppSec (OWASP Top 10 & SGD/MGI)

* **M01 / A05:** HSTS `max-age=63072000; includeSubDomains; preload`; CSP com nonce; `X-Frame-Options: DENY`; `X-Content-Type-Options: nosniff`; `Referrer-Policy: strict-origin-when-cross-origin`; timeouts (`ReadHeaderTimeout = 2s`, `ReadTimeout = 5s`, `WriteTimeout = 10s`, `IdleTimeout = 120s`).
* **A01:** autorização `resource:action` no middleware, antes da aplicação.
* **A02:** senhas locais Argon2id (fallback bcrypt cost 12); TLS 1.3.
* **A03:** queries parametrizadas `pgx/v5`.
* **A07:** rate limit por IP e usuário no Redis; lockout progressivo.
* **A10:** plug-in **Egress** sanitiza webhooks (RFC 1918, loopback, metadata `169.254.169.254`, DNS interno).

---

## 5. LGPD & Trilha de Auditoria (Lei 13.709/2018)

* **Mascaramento PII:** `log/slog`; CPF `***.456.789-**`; e-mail `u***o@dominio.gov.br`; telefone `(**) *****-1234`.
* **Correlação:** `X-Request-ID` + OpenTelemetry (W3C TraceContext).
* **Auditoria imutável:** append-only com `actor_id`, papéis, IP, user-agent, contexto, ação, recurso, diff, `timestamp_utc` e encadeamento SHA-256.
* **Dados pessoais dos plug-ins:** quem guarda dado pessoal implementa `lgpd.PersonalData` (exportação e eliminação).

Os identificadores de infraestrutura do microkernel (banco, usuário e vhost, filas, métricas `nexus_*`) são nomes internos; a **marca de produto** é Projeto-nova-v2 (ADR 046).

---

## 6. Catálogo de Módulos (Core Imutável vs Plugins)

Cada plug-in declara `kernel.Manifest` (`Key`, `Name`, `Description`, `DefaultEnabled`, `DependsOn`, `Public`, `Icon`, `Route`, `Permissions`). `Public: true` se o módulo tem superfície anônima listada em `GET /api/v1/system/public-modules` (site institucional e sitemap).

### 6.1. Core Imutável (nunca desativável)

* **IAM & Usuários:** identidades, sessões, RBAC multi-escopo, mapeamento AD.
* **Auditoria:** registro transacional imutável.

### 6.2. Plug-ins de plataforma

* **Mercúrio** (`mercurio`): chat em tempo real via WebSocket (Redis backplane), salas por unidade/departamento (AD) e mensagens diretas.
* **Egress** (`egress`): webhooks de saída assinados, anti-SSRF.
* **Blog** (`blog`, Public): notícias e comunicados (rascunho → publicação).
* **Catálogo** (`catalog`, Public): carta de serviços no site (`/servicos`).
* **Contato** (`contact`, Public): formulário institucional com rate-limit.
* **Diretório** (`directory`): pessoas e setores a partir do AD.
* **Agenda** (`calendar`, Public): eventos e reserva de salas.
* **Arquivos** (`files`): pastas de trabalho no MinIO (ACL por perfil/grupo/unidade).
* **Wiki** (`wiki`): base de conhecimento em Markdown com histórico.
* **Busca Global** (`search`): busca unificada sobre os módulos **ativos**; índice Typesense só para conteúdo público (ADR 031).
* **Signum** (`signum`, Public): assinatura eletrônica com reautenticação e hash SHA-256 (não afirmar PAdES/ICP-Brasil).
* **Trâmite** (`tramite`, DependsOn `signum`): processos administrativos numerados, sigilo, tramitação entre unidades.

### 6.3. Gestão de serviços

* **Produtos digitais** (`produtos`): o que a unidade oferece e o estágio do ciclo de vida.
* **Atendimento** (`atendimento`): ofertas, solicitações (incidente/requisição), prazos por política e calendário, visitas de campo com fila offline.
* **Ativos** (`ativos`): itens que sustentam os produtos.
* **Problemas** (`problemas`): causa de incidentes, erro conhecido com contorno, sugestão por recorrência.
* **Mudanças** (`mudancas`): padrão, normal e emergencial, com aprovação segregada.

Regras de cada um: `docs/REGRAS_DE_NEGOCIO.md`.

### 6.4. Blueprint (não é produto)

* **Example** (`example`): modelo `make new-module`. Não ativar em homologação nem em produção.

---

## 7. Regras que o código deve continuar a cumprir

1. Todo módulo de negócio é `kernel.Plugin` com Manifest; rotas via `RouteProvider`; workers `WorkerProvider`; filas `ConsumerProvider`; busca `SearchProvider`; WS `WSProvider`.
2. **Nenhum** `internal/modules/X` importa `internal/modules/Y`.
3. Desativar um módulo não exige rebuild; a superfície some na hora (teste `TestEveryPluginSurfaceFollowsActivation`).
4. Copiar de outro órgão: só tokens, branding e Manifest Description — não fundir plug-ins no núcleo.
5. Não inventar decreto, portaria, PAdES ou integração além do que os ADRs autorizam.
6. Cobertura: 100% por pacote no backend (`scripts/coverage-gate.sh`); limiares Vitest no frontend (CI).
7. Toda regra nova ou alterada entra em `docs/REGRAS_DE_NEGOCIO.md` no mesmo commit. O esquema parte de `migrations/000001_base.sql`; mudanças entram em migrations novas.
