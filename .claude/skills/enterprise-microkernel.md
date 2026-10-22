---
name: enterprise-microkernel-architect
description: Especificação técnica rigorosa do SISMARQ (Arquivo Público Municipal de Rondonópolis) — Microkernel Go + Next.js com conformidade SGD/MGI, e-MAG, DSGov, LGPD, OWASP Top 10, Keycloak dedicado e federação AD. Plug-ins permanecem isolados e reutilizáveis; o tenant é o Arquivo Público.
---

# Enterprise Microkernel — SISMARQ (Arquivo Público)

Você atua como Engenheiro de Software Principal e Arquiteto de Segurança da Informação. Toda a arquitetura, geração de código, refatoração e documentação deve obedecer a este manifesto.

O produto é o **SISMARQ**: sistema do **Arquivo Público Municipal de Rondonópolis** (Lei Municipal nº 10.301/2019). Não é um produto genérico de empresa nem o legado PaperMoon/Nexus como marca. Os plug-ins (Mercúrio, Agenda, Wiki, Arquivos, Blog, Catálogo, etc.) **permanecem no catálogo**, com `DefaultEnabled` verdadeiro salvo o blueprint `example`, e são **atualizados no texto e no uso** para a Superintendência do Arquivo Público — sem deixar de ser módulos plugáveis.

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
  * `Manifest.DependsOn` declara o grafo (ex.: Trâmite → Signum; Acervo → Atlas).
  * A colaboração passa por **portas** no domínio do consumidor; o adaptador é ligado em `internal/app`.
  * Consumo de eventos: filas próprias (`QueueSpec`) no plugin consumidor — não importa o pacote do produtor.
* **Resiliência Transacional:**
  * **Transactional Outbox:** gravação no PostgreSQL via `pgxpool` na mesma transação do negócio.
  * **Relay Worker:** leitura assíncrona e despacho ao RabbitMQ.
  * **Idempotência (A08):** `X-Idempotency-Key` em mutações e dedup nas filas, com DLQ.

---

## 2. Autenticação & Gestão de Identidade (IAM Exclusivo Keycloak + RS256)

1. **Padrão Ouro — Keycloak Dedicado:**
   * Instância apartada como OpenID Connect Provider (OIDC / OAuth2).
   * Federação nativa com o **Active Directory** (LDAP/LDAPS).
   * Validação JWT via JWKS com cache e rotação.
   * No SISMARQ: realm institucional, `aud=sismarq`; issuer e segredos só em ambiente (nunca no git).
2. **Fallback Local — Par Assimétrico RS256:** JWT interno para contingência ou ambientes isolados.
3. **Mapeamento AD → RBAC:** grupos (`ad_groups`) correlacionados a **Perfis** e escopos:
   * **Entidades** (multi-tenant de órgãos da administração)
   * **Unidades**
   * **Departamentos**
   * Pessoas podem ter múltiplos escopos e perfis.

---

## 3. Padrões de Acessibilidade & Interface (e-MAG / DSGov)

O frontend adota o **DSGov/SECOM** e a **Portaria de Acessibilidade Digital (SGD/MGI)**:

1. **Acessibilidade Digital (WCAG 2.1 AA / e-MAG):**
   * Barra de atalhos: `Alt+1` conteúdo (`id="main-content"`), `Alt+2` menu (`id="main-menu"`), `Alt+3` busca (`id="global-search"`), `Alt+4` rodapé (`id="gov-footer"`).
   * Widget **VLibras** na raiz.
   * Alto contraste e escala de fonte (`A-`, `A`, `A+`) persistidos em cookies `sismarq-*` (não `nexus-*`).
   * Linting `@jsx-a11y`.
2. **DSGov & White-Label:**
   * Componentes canônicos (`GovHeader`, `GovFooter`, botões institucionais).
   * Tokens desacoplados para **órgãos da administração pública** (não “empresas”). O tenant desta implantação é o Arquivo Público Municipal.

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

Contratos de infraestrutura herdados (usuário/vhost/filas/métricas `nexus_*`) permanecem onde o ADR 031 os fixou em produção; a **marca de produto** é SISMARQ.

---

## 6. Catálogo de Módulos (Core Imutável vs Plugins)

Cada plug-in declara `kernel.Manifest` (`Key`, `Name`, `Description`, `DefaultEnabled`, `DependsOn`, `Public`, `Icon`, `Route`, `Permissions`). `Public: true` se o módulo tem superfície anônima listada em `GET /api/v1/system/public-modules` (site institucional e sitemap). Rotas anônimas internas (ex. setores do Diretório) não obrigam `Public` se não há página institucional dedicada.

### 6.1. Core Imutável (nunca desativável)

* **IAM & Usuários:** identidades, sessões, RBAC multi-escopo, mapeamento AD.
* **Auditoria:** registro transacional imutável.

### 6.2. Plug-ins de plataforma (permanecem; texto e uso = Arquivo Público)

Ativação em Configurações. **Não remover** para “simplificar” o produto: a Superintendência pode precisar deles. Isolamento e `RegisterModule` inalterados.

* **Mercúrio** (`mercurio`, DefaultEnabled): chat em tempo real via WebSocket (Redis backplane) — canais da Superintendência, salas por unidade/departamento (AD) e mensagens diretas entre servidores.
* **Egress** (`egress`): webhooks de saída assinados, anti-SSRF (n8n, Zabbix, Grafana e outras automações do município).
* **Blog** (`blog`, Public): comunicados e notícias do Arquivo Público (rascunho → publicação); emite `blog.post.published`.
* **Catálogo** (`catalog`, Public): carta de serviços do Arquivo Público no site (`/servicos`); `catalog:manage` para edição.
* **Contato** (`contact`, Public): formulário institucional (SIC / ouvidoria do órgão) com rate-limit e `contact.message.submitted`.
* **Diretório** (`directory`): pessoas e setores a partir do AD (consulta interna da Superintendência e das unidades setoriais).
* **Agenda** (`calendar`, Public): eventos do Arquivo Público e reserva de salas (`calendar:manage`); agenda pública em `/eventos`.
* **Arquivos** (`files`): pastas de **trabalho** no MinIO (ACL por perfil/grupo/unidade). **Não** substitui o acervo digital (PDF/A, WORM, OCR) do plug-in Acervo.
* **Wiki** (`wiki`): base de conhecimento operacional (procedimentos internos, manuais) em Markdown com histórico; `wiki:manage` para exclusões.
* **Busca Global** (`search`): busca unificada sobre módulos **ativos** (Blog, Wiki, Arquivos, Catálogo, Diretório, Trâmite, Acervo quando indexado); degrada se o índice cair.
* **Signum** (`signum`, Public): assinatura eletrônica com reautenticação e hash SHA-256 (não afirmar PAdES/ICP-Brasil enquanto o ADR correspondente estiver adiado).
* **Trâmite** (`tramite`, DependsOn `signum`): processos administrativos numerados, sigilo, tramitação entre unidades, acionamento do Signum. Opcional para o ingresso no Acervo (evento `tramite.processo.concluido`; sem DependsOn no Acervo).

### 6.3. Plug-ins de missão (Arquivo Público)

* **Atlas** (`atlas`, Public): procedimentos canônicos (padrão SEI) e **TTDD/CCPAD** com vigência; assistente com grounding estrito. Porta de TTDD consumida pelo Acervo (ligada em `internal/app`).
* **Acervo** (`acervo`, Public, DependsOn `atlas`): unidades de arquivamento nas três idades (Lei 10.301/2019), prazos, localização, bloqueio, eliminação, recolhimento, normas, digitais (SCAN vs NATO_DIGITAL), busca, consulta pública e exposições virtuais.

### 6.4. Blueprint (não é produto)

* **Example** (`example`, **DefaultEnabled: false**): modelo `make new-module`. Não ativar em homologação/produção do Arquivo Público.

---

## 7. Regras que o código deve continuar a cumprir

1. Todo módulo de negócio é `kernel.Plugin` com Manifest; rotas via `RouteProvider`; workers `WorkerProvider`; filas `ConsumerProvider`; busca `SearchProvider`; WS `WSProvider`.
2. **Nenhum** `internal/modules/X` importa `internal/modules/Y`.
3. Desativar um módulo não exige rebuild; a superfície some na hora (teste `TestEveryPluginSurfaceFollowsActivation`).
4. Copiar de outro órgão: só tokens, branding e Manifest Description — não fundir plug-ins no núcleo.
5. Não inventar decreto, portaria, PAdES ou integração SEI além do que os ADRs autorizam.
6. Cobertura: gate por plug-in no backend; limiares Vitest no frontend (CI).
7. Acervo / TTDD / procedimentos: ler `docs/acervo/DOMINIO.md` e `docs/acervo/EARQ_RDC.md` (ADR 045) antes de implementar ou alterar; atualizar esses textos no mesmo commit se a regra mudar. SIGAD municipal com guarda reforçada; não afirmar RDC certificado.
