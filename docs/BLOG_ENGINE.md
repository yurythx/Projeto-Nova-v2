# Projeto-nova-v2 Blog Engine — Arquitetura Editorial, SEO Técnico e Gestão de Mídia

> **Status:** Ativo / Em Produção  
> **Padrão Arquitetural:** Go Clean Architecture (Backend) + Next.js SSR / React 19 (Frontend) + MinIO S3 Object Storage  
> **Conformidade:** WCAG 2.1 AA / e-MAG, LGPD, W3C TraceContext, Schema.org

---

## 1. Visão Geral

O módulo **Blog & Novidades** do Projeto-nova-v2 foi projetado para operar como um motor de autoridade técnica, aquisição orgânica de tráfego (SEO técnico) e conversão de serviços gerenciados, inspirado nos padrões editoriais e de engenharia de plataformas como **Cloudflare, GitHub Engineering, Stripe e Vercel**.

### Pilares Fundamentais:
1. **Desempenho e Segurança:** Todo conteúdo é armazenado como Markdown puro no PostgreSQL, sem execução de HTML inseguro (zero vetor XSS) e com carregamento via Server-Side Rendering (SSR).
2. **Zero-Load Backend para Mídia:** O upload e a recuperação de imagens operam diretamente contra o **MinIO Object Storage** usando **URLs Pré-Assinadas (Presigned URLs)**, mantendo os processos Go livres de contenção de CPU e memória.
3. **Experiência do Autor de Classe Mundial:** Painel administrativo equipado com o editor visual **TipTap (ProseMirror)** com barra de ferramentas rica, upload de imagens no meio do texto e exportação nativa em Markdown.
4. **Experiência de Leitura Imersiva e Acessível:** Sumário lateral dinâmico (*Table of Contents com ScrollSpy*), barra de progresso de leitura, callouts semânticos, fallbacks elegantes e compartilhamento social em 1 clique.
5. **SEO & Distribuição:** Metadados estruturados **Schema.org (JSON-LD)** e sindicação pública via **Feed RSS 2.0 (`/blog/rss.xml`)**.

---

## 2. Arquitetura de Mídia & MinIO (S3)

```
┌─────────────────┐       1. POST /api/v1/blog/uploads       ┌─────────────────┐
│                 ├─────────────────────────────────────────►│                 │
│  Navegador Web  │◄─────────────────────────────────────────┤   Backend Go    │
│  (TipTap/Admin) │        2. UploadTicket (Presigned PUT)   │  (Porta 8443)   │
│                 │                                          └────────┬────────┘
└────────┬────────┘                                                   │
         │                                                            │
         │ 3. PUT direto do arquivo                                   │ Gera chave única
         │    (https://host:9443/nova/blog/covers/...)               │
         ▼                                                            ▼
┌────────────────────────────────────────────────────────┐   ┌─────────────────┐
│                    MinIO Storage                       │   │   PostgreSQL    │
│  • Bucket: nova / WORM audit lock                     │   │   (Chave S3 e   │
│  • Proxy seguro Caddy TLS na porta 9443                │   │    Markdown)    │
└────────────────────────────────────────────────────────┘   └─────────────────┘
```

### 2.1. O Fluxo de Upload Pré-Assinado (*Presigned PUT*)
Para evitar ataques de negação de serviço e sobrecarga de memória (Slowloris/DoS), nenhum arquivo binário atravessa a API Go ou a camada BFF:
1. O autor seleciona uma imagem no editor (`.png`, `.jpeg`, `.webp` ou `.gif`).
2. O frontend requisita um tíquete de upload:
   ```http
   POST /api/v1/blog/uploads
   Content-Type: application/json

   {"filename": "diagrama-rede.png", "content_type": "image/png"}
   ```
3. O backend invoca o método `PresignedPutURL` da SDK MinIO, gerando uma chave única não previsível (`blog/covers/AAAA/MM/{uuid}.ext`) e retornando o `UploadTicket`.
4. O navegador envia os bytes brutos do arquivo diretamente ao MinIO através da URL pré-assinada (`putToTicket`).
5. A imagem é inserida instantaneamente no documento do TipTap ou cadastrada como `cover_object_key`.

### 2.2. Exibição de Capas e Fallbacks Inteligentes
- **Imagens Cadastradas:** O backend gera uma URL pré-assinada temporária de leitura (`PresignedGetURL`) válida por 1 hora, entregue no campo `cover_url`.
- **Artigos Sem Imagem (Fallback Tecnológico):** O componente [`BlogCoverImage`](frontend/src/components/blog/BlogCoverImage.tsx) detecta o serviço correspondente e gera um card editorial nobre com gradientes de alta densidade, o ícone SVG correspondente da ferramenta (ex.: Terraform, Ansible, Proxmox, WireGuard) e badge da categoria técnica.

---

## 3. Editor Visual TipTap (`TipTapEditor.tsx`)

O editor visual no painel de gestão (`/gestao/blog`) utiliza a suíte modular **TipTap** sobre o motor **ProseMirror**:

### 3.1. Recursos Integrados:
- **Títulos Semânticos:** H2 e H3 para hierarquia clara.
- **Formatação de Texto:** Negrito (`Ctrl+B`), Itálico (`Ctrl+I`), Riscado, Código inline.
- **Listas & Citações:** Listas com marcadores, listas ordenadas, citações em bloco e blocos de código pré-formatados.
- **Inserção de Imagens no Corpo:** Botão de imagem integrado com input invisível que dispara o fluxo de upload para o MinIO e injeta a tag `![legenda](minio_url)` na posição do cursor.
- **Modo Duplo (Visual vs. Markdown Puro):** Botão na barra de ferramentas permite alternar entre o editor WYSIWYG e a área de texto em Markdown puro para desenvolvedores que preferem escrever sintaxe direta.
- **Persistência Limpa:** O editor lê Markdown na inicialização e exporta Markdown limpo no evento `onChange`, mantendo 100% de conformidade com a coluna `body` do PostgreSQL.

---

## 4. Parser Markdown Seguro & Recursos Avançados

O componente [`Markdown.tsx`](frontend/src/components/plataforma/Markdown.tsx) renderiza texto sem recorrer a `dangerouslySetInnerHTML`, criando diretamente a árvore de componentes React com segurança contra XSS:

### 4.1. Âncoras Automáticas nos Cabeçalhos
Todo `##` e `###` gera um identificador unificado via [`slugifyHeading`](frontend/src/lib/blog/slugify.ts):
```html
<h2 id="como-surgiu" class="group relative ...">
  Como Surgiu (Origem e História)
  <a href="#como-surgiu" class="... opacity-0 group-hover:opacity-100">#</a>
</h2>
```

### 4.2. Suporte a Callouts / Admonitions Semânticos
Suporta marcações de alertas inspiradas no padrão GitHub / Obsidian:
```markdown
> [!NOTE] Informações operacionais importantes.
> [!TIP] Dica de melhor prática para otimizar o ambiente.
> [!WARNING] Atenção a regras de firewall e portas de rede.
> [!IMPORTANT] Requisitos mandatórios antes de publicar.
> [!CAUTION] Alerta crítico de segurança e risco de indisponibilidade.
```
Os blocos são renderizados com ícones Lucide específicos, bordas em destaque e contraste ajustado para os modos claro e escuro.

### 4.3. Imagens Inline no Texto
A regex inline processa a sintaxe `![alt](url)`:
- Validação da URL via `safeResourceUrl` (rejeitando esquemas perigosos como `javascript:` ou `data:` maliciosos).
- Renderização em `<figure>` centralizado com cantos arredondados, bordas de contraste e legenda (`<figcaption>`).

---

## 5. Experiência do Leitor no Blog Público

Na rota `/blog/[ref]`:
1. **Barra de Progresso de Leitura (`ReadingProgressBar`):** Barra horizontal de 4px no topo do viewport (`z-50`) que preenche suavemente conforme a rolagem do usuário.
2. **Sumário Lateral Flutuante (`TableOfContents`):** Menu lateral fixo (`sticky`) no desktop que monitora os títulos do artigo via `IntersectionObserver` (*ScrollSpy*) e permite saltar suavemente para qualquer seção com clique direto.
3. **Barra de Compartilhamento Social Ativa (`SocialShareBar`):**
   - 🟢 **WhatsApp:** Compartilha título + URL formatada.
   - 💼 **LinkedIn:** Abre janela oficial de compartilhamento.
   - 🐦 **X (Twitter):** Compartilha com citação e link.
   - 📋 **Copiar Link:** Salva na área de transferência com feedback visual ("Copiado!").
4. **Artigos Relacionados (`RelatedPosts`):** Sugere automaticamente até 3 artigos no rodapé da página para estender o tempo de permanência no portal.
5. **Ponte Contextual com o Catálogo:** Banner no rodapé vinculando a leitura técnica ao serviço correspondente com links para contratação e atendimento.

---

## 6. SEO Técnico e Distribuição

### 6.1. Dados Estruturados Schema.org (JSON-LD)
Toda página de artigo público injeta automaticamente o esquema `TechArticle`:
```json
{
  "@context": "https://schema.org",
  "@type": "TechArticle",
  "headline": "Título da Publicação",
  "description": "Resumo...",
  "datePublished": "2026-10-11T17:09:11Z",
  "dateModified": "2026-10-11T17:09:11Z",
  "author": {
    "@type": "Organization",
    "name": "Projeto-nova-v2 Editorial"
  },
  "publisher": {
    "@type": "Organization",
    "name": "Projeto-nova-v2",
    "url": "https://..."
  }
}
```

### 6.2. Feed RSS 2.0 Público
Disponível publicamente no endpoint:
```
GET /blog/rss.xml
```
- Content-Type: `application/xml; charset=utf-8`
- Cabeçalhos de cache: `public, s-maxage=3600, stale-while-revalidate=86400`
- Contém todos os artigos publicados ordenados cronologicamente para consumo em leitores RSS (Feedly, Miniflux, Thunderbird) e automações corporativas.

---

## 7. Guia Operacional para Redatores e Gestores

1. **Acessar o Painel de Gestão:**
   - Navegue para `/gestao/blog` com usuário que possua a permissão `blog:manage`.
2. **Criar Nova Matéria (Página Dedicada):**
   - Clique em **"Nova publicação"** (direciona para a tela de autoria completa em [`/gestao/blog/novo`](frontend/src/app/(protected)/gestao/blog/novo/page.tsx)).
   - Preencha o título, selecione se deseja fixar no topo (`pinned`) e defina a unidade dona caso pertença a uma unidade específica.
   - **Público-Alvo Otimizado:** O seletor de audiência inicia recolhido por padrão configurado para **"Todos os usuários (sem restrição)"**. Se for um comunicado restrito a unidades ou departamentos de Rondonópolis, expanda o acordeão e marque os setores desejados (com botão para "Restaurar para Todos" a qualquer momento).
   - Envie uma **Imagem de capa** de alta resolução (recomendado: 1200x630px, `.webp` ou `.png`).
3. **Escrever o Conteúdo com TipTap:**
   - Utilize a barra de ferramentas do **TipTap WYSIWYG** para estruturar títulos, listas e citações.
   - Para ilustrar com diagramas de arquitetura ou prints, clique no ícone de imagem da barra de ferramentas e selecione o arquivo local (enviado diretamente via MinIO S3).
   - Alternância entre o modo visual e Markdown puro disponível na barra.
4. **Editar Matéria Existente:**
   - Na listagem de gestão ou no detalhe do artigo (`/gestao/blog/[ref]`), clique em **"Editar matéria"**, que abre a tela dedicada em [`/gestao/blog/[ref]/editar`](frontend/src/app/(protected)/gestao/blog/[ref]/editar/page.tsx).
5. **Publicação e Distribuição:**
   - Salve como rascunho para revisão interna ou marque **"Publicar"**.
   - No momento da publicação, o artigo entra imediatamente no feed público (`/blog`), no RSS 2.0 (`/blog/rss.xml`) e nas notificações de usuários contemplados pelo público-alvo.
