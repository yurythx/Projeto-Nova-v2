# 030 — Autoria Dedicada do Blog, Público-Alvo Recolhível e Estrutura Corporativa de Rondonópolis

- **Status:** Aceito e implementado
- **Data:** 2026-10-12
- **Escopo:** `frontend/src/app/(protected)/gestao/blog`, `frontend/src/components/blog`, `frontend/src/components/iam/PublicoAlvo.tsx`, `deploy/seeds/reset_estrutura_rondonopolis.sql`, infraestrutura Docker e documentação

## Contexto

A evolução do ecossistema editorial e organizacional do Projeto Nexus identificou quatro pontos de fricção operacional e arquitetural:

1. **Autoria em Modais Restritos:** O cadastro e a edição de matérias eram executados dentro de caixas de diálogo modais (`ModalShell` / `dialog`). Em monitores convencionais ou com textos longos, isso gerava barras de rolagem duplas, cortava a visualização da barra de ferramentas rica do TipTap (WYSIWYG) e prejudicava o fluxo de escrita de artigos técnicos profundos.
2. **Sobrecarga Visual no Seletor de Público-Alvo:** O componente `PublicoAlvoPicker` (introduzido no ADR 014) renderizava todas as secretarias e unidades abertas e marcáveis por padrão logo no início do formulário, poluindo a tela quando a imensa maioria dos artigos tem escopo amplo/institucional ("Todos os usuários").
3. **Massa de Dados Legada Desalinhada:** O banco continha sementes históricas com mais de 310 registros de órgãos públicos municipais (escolas, postos de saúde, unidades de atenção básica) incompatíveis com a proposta corporativa do Nexus e gerando lentidão e poluição na navegação de setores e no diretório.
4. **Fragmentação de Nomenclatura (`v2`):** Existiam menções e contêineres nomeados como `projeto-nexus-v2`, divergindo da identidade canônica do projeto (`Projeto-Nexus` / `projeto-nexus`).

## Decisões

### 1. Páginas Dedicadas de Criação e Edição do Blog
A autoria editorial foi migrada integralmente para rotas dedicadas de página inteira:
- **Nova Matéria:** Rota canônica [`/gestao/blog/novo`](frontend/src/app/(protected)/gestao/blog/novo/page.tsx).
- **Editar Matéria:** Rota canônica [`/gestao/blog/[ref]/editar`](frontend/src/app/(protected)/gestao/blog/[ref]/editar/page.tsx).

Ambas utilizam o layout administrativo protegido, com navegação estruturada via breadcrumb, botão de retorno intuitivo e espaço visual ampliado para o editor TipTap com alternância instantânea entre modo visual (WYSIWYG) e Markdown puro, além de upload de imagens via MinIO S3 presigned URLs.

### 2. Seletor de Público-Alvo Recolhível por Padrão
O componente [`PublicoAlvoPicker`](frontend/src/components/iam/PublicoAlvo.tsx) passa a:
- Iniciar recolhido dentro de uma tag nativa `<details open={temRestricao || defaultOpen}>`, exibindo um sumário com selo informativo: `Todos (Padrão)`.
- Manter como regra padrão zero restrições ("Todos os usuários").
- Disponibilizar botão direto "Restaurar para Todos", permitindo limpar seleções com um único clique.
- Expandir-se automaticamente apenas se o registro já possuir restrições previamente salvas.

### 3. Estrutura Organizacional Exclusiva de Rondonópolis (MT)
Foi executada uma reestruturação atômica no banco de dados via script idempotente [`deploy/seeds/reset_estrutura_rondonopolis.sql`](deploy/seeds/reset_estrutura_rondonopolis.sql), substituindo os 310 registros legados por uma hierarquia corporativa limpa, moderna e geolocalizada em Rondonópolis - MT:
- **2 Entidades:**
  1. `Nexus Corporation Matriz` (Sigla: `NEXUS`)
  2. `Nexus AgroTech & Soluções Digitais` (Sigla: `AGROTECH`)
- **5 Unidades Corporativas:**
  1. `SEDE-CENTRO`: Sede Corporativa Central - Centro (Av. Marechal Rondon, 1250)
  2. `HUB-AURORA`: Hub de Inovação & P&D - Vila Aurora (Av. Lions Internacional, 1500)
  3. `POLO-IND`: Complexo Operacional - Distrito Industrial Veterinário (BR-364, km 205)
  4. `UNID-POGUBA`: Centro Integrado de Atendimento - Sagrada Família (Av. Poguba, 850)
  5. `TERM-AGRO`: Terminal Intermodal & Agronegócios - Anel Viário (Complexo Ferroviário)
- **19 Departamentos:** TI, Engenharia, Suporte, Finanças, RH, Logística, etc.
- O usuário inicial `admin` foi associado à `SEDE-CENTRO` / Tecnologia da Informação.

### 4. Unificação Canônica de Nomenclatura e Contêineres
- Repositório Git local e servidor unificados com remote único `origin` apontando para `https://github.com/yurythx/Projeto-Nexus.git`.
- Diretório de implantação no servidor fixado como `/root/Projetos/Projeto-Nexus`.
- Nome do projeto no Docker Compose padronizado para `projeto-nexus`, com contêineres e volumes canônicos (`projeto-nexus-*`).

## Consequências

- **Experiência Editorial Superior:** Autores contam com tela dedicada e espaço amplo para redigir artigos longos, diagramas e mídias sem limitações de diálogos modais.
- **Formulários Limpos:** A tela de criação não sofre com sobrecarga de dezenas de caixas de seleção abertas sem necessidade.
- **Diretório Eficiente:** O módulo Diretório (`/diretorio`) agora opera com resposta instantânea e filtros claros entre Entidade, Unidade e Departamento.
- **Infraestrutura Consistente:** Eliminação de qualquer confusão entre branches ou nomes de contêineres legados no host de produção.
