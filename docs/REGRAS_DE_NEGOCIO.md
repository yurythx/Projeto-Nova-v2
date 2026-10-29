# Estrutura, funcionamento e regras de negócio

O nome público do produto é **Projeto Nova**. A operação de serviços segue o
ITIL (Version 5) nas práticas que o código implementa: incidente e requisição,
problema, mudança e projeto. O repositório e os pacotes técnicos conservam o
identificador histórico.

Referência funcional da plataforma. Serve para quem **implanta e administra**
(como modelar a organização, quem pode o quê) e para quem **desenvolve**
(quais regras cada módulo garante). Tudo aqui reflete o código em
`backend/internal`. Como programar um módulo está em
[GUIDE_DESENVOLVEDOR.md](GUIDE_DESENVOLVEDOR.md) e
[GUIDE_MODULOS.md](GUIDE_MODULOS.md). Como implantar está em
[DEPLOY.md](DEPLOY.md).

---

## 1. Como a plataforma funciona

### 1.1. O caminho de uma requisição

1. **Login.** O login principal é pelo Keycloak (OIDC), que pode federar o
   Active Directory. Se o Keycloak ficar indisponível, há o login local de
   contingência. O token traz quem é a pessoa e os **grupos** dela.
2. **Identidade efetiva.** A cada requisição, o IAM da API:
   - cria ou atualiza o usuário na primeira vez que ele entra
     (provisionamento automático, com nome, e-mail e grupos);
   - soma as **permissões** de todos os perfis que a pessoa recebeu, seja
     por **lotação manual** ou por **mapeamento de grupo**;
   - monta a lista de **escopos** (onde a pessoa está lotada).

   O resultado fica em cache por poucos segundos. Quando um administrador
   muda perfis, lotações ou mapeamentos, o cache é invalidado em todas as
   instâncias e a mudança vale **na hora**.
3. **Módulo.** A rota exige a permissão necessária. Depois, o próprio
   módulo aplica as regras dele: unidade, sigilo, dono, ACL.
4. **Efeitos.** Toda alteração grava, na mesma transação:
   - o **registro de auditoria**, imutável e encadeado por hash;
   - o **evento de domínio** no outbox.

   O worker publica esse evento no RabbitMQ, e daí ele segue para o tempo
   real (WebSocket), para os webhooks (Egress) e para a busca.

### 1.2. Módulos (Kernel)

| Tipo | Módulos | Pode desligar? |
|---|---|---|
| Núcleo | IAM & Usuários, Auditoria | Não |
| Plug-ins | Mercúrio, Egress, Blog, Catálogo, Contato, Diretório, Agenda, Arquivos, Wiki, Busca, Signum, Trâmite, Produtos digitais, Atendimento, Ativos, Problemas, Mudanças, Projetos, Módulo Modelo | Sim, em Configurações → Módulos (`modules:manage`) |

- Desligar um módulo tem efeito imediato, sem reiniciar:
  - as rotas dele respondem `404 MODULE_DISABLED`;
  - ele some do menu, do site público, do sitemap e da busca;
  - as assinaturas de tempo real dele caem;
  - os workers e consumidores de fila dele param.
- **Dependências:** o Trâmite depende do Signum. Não é possível desligar o
  Signum com o Trâmite ligado, nem ligar o Trâmite com o Signum desligado.
  A tela mostra esse grafo.
- **Módulos públicos** (Catálogo, Contato, Diretório, Agenda e Signum) têm
  páginas no site institucional, que funcionam sem login.

### 1.3. Garantias transversais

| Garantia | Como |
|---|---|
| Auditoria de toda alteração | Tabela só de inclusão (o banco impede alterar ou apagar), cadeia SHA-256 verificável, exportação LAI (CSV/JSON/XML) e cópia diária imutável (WORM) |
| Consentimento LGPD | Aceite registrado com versão do termo e IP (anônimo ou da conta); direitos do titular: acesso, portabilidade e pedido de exclusão |
| Limites de abuso | Por identidade na API (600/min), por IP nas rotas públicas e no login, e bloqueio progressivo de login local (ver [DEPLOY.md](DEPLOY.md#segurança-do-login-local)) |
| Idempotência | Repetir uma alteração com a mesma chave devolve a resposta original, sem duplicar |
| Dados pessoais nos logs | Mascarados (CPF, e-mail, telefone) |

---

## 2. Estrutura organizacional

```
Entidade (ex.: Prefeitura, Secretaria de Saúde, Empresa A)
└── Unidade (ex.: UPA 24h, Escola, Filial) — pode ter unidade-mãe
    └── Departamento (ex.: Administração, RH, TI)
```

### 2.1. Entidade

É o órgão ou a organização de nível mais alto: prefeitura, secretaria,
autarquia, empresa.

- Campos: nome, sigla, slug (endereço), documento (CNPJ ou outro) e ativo.
- O slug é gerado a partir do nome quando não é informado, e é único.

### 2.2. Unidade

É o local ou a subdivisão onde as pessoas trabalham: escola, posto de
saúde, filial, sede.

- Pertence a **uma** entidade e **não pode mudar de entidade** depois de
  criada, porque as lotações gravam a entidade da unidade.
- Pode ter uma **unidade-mãe**, o que forma uma hierarquia. A mãe precisa
  ser da **mesma entidade**, e ciclos são recusados.
- Campos: nome, sigla, slug (único dentro da entidade), grupo do AD de
  referência, e-mail, telefone, endereço e ativo.
- **É o nível que o Trâmite usa.** Processos são abertos, ficam e são
  tramitados entre **unidades**.

### 2.3. Departamento

É o setor dentro de uma unidade: Administração, Atendimento, RH.

- Pertence a **uma** unidade e **não pode mudar de unidade**.
- Campos: nome, sigla, slug (único dentro da unidade), grupo do AD de
  referência, e-mail, telefone e ativo.
- **Todo departamento pertence a uma unidade.** Uma organização que só tem
  departamentos (como a Prefeitura no ambiente de teste) usa uma unidade
  "Sede" para agrupá-los.

### 2.4. Regras comuns

- **Exclusão protegida:** excluir entidade, unidade ou departamento **em
  uso** responde `409`. Nada é apagado em cascata; primeiro se remove o que
  depende, depois o item.

  **Bloqueiam a exclusão:**
  - unidades de uma entidade, departamentos de uma unidade e subunidades
    de uma unidade-mãe;
  - lotações e mapeamentos de grupo que apontam para o item;
  - processos do Trâmite (origem, unidade atual e histórico).

  **Não bloqueiam; a referência é apenas limpa:**
  - unidade responsável de um serviço do Catálogo;
  - lotação exibida no Diretório;
  - departamento de uma sala do Mercúrio.
- **"Ativo" — desativar é tirar de operação sem apagar:**
  - lotações e mapeamentos de grupo cujo escopo tenha entidade, unidade ou
    departamento desativado **deixam de conceder** o perfil, na hora
    (salvar a estrutura invalida o cache do IAM); reativar devolve;
  - desativar a entidade vale para tudo que está abaixo dela;
  - o Trâmite recusa (`422`) abrir processo em unidade desativada, ou de
    entidade desativada, e tramitar para ela — inclusive para quem tem
    `tramite:manage`. Processos que já estão lá continuam consultáveis;
  - o Diretório e as telas do Trâmite deixam de oferecer o que está
    desativado.

  A regra do escopo ativo é única, `nova_scope_active` (função do banco),
  usada tanto pelo cálculo de permissões por requisição quanto pela tela de
  usuários.
- Toda alteração é auditada com o antes e o depois.
- Quem administra: permissão `iam:manage`.

---

## 3. Perfis e permissões

### 3.1. Permissões

- Formato `recurso:ação` (ex.: `tramite:create`). Há dois curingas:
  `recurso:*` (todas as ações do recurso) e `*` (tudo).
- Cada módulo declara as próprias permissões. A lista completa, com
  descrições, sai em `GET /api/v1/iam/permissions` e aparece na tela de
  perfis.

| Módulo | Permissões |
|---|---|
| IAM | `users:read`, `users:manage`, `iam:manage`, `modules:manage`, `keycloak:manage`, `ia:manage`, `branding:manage`, `monitoring:read`, `monitoring:manage` |
| Auditoria | `audit:read`, `audit:verify` |
| Trâmite | `tramite:create`, `tramite:route`, `tramite:manage` |
| Blog / Catálogo / Wiki | `blog:manage`, `catalog:manage`, `wiki:manage` |
| Contato | `contact:read`, `contact:manage` |
| Agenda / Arquivos / Diretório | `calendar:manage`, `files:manage`, `directory:manage` |
| Mercúrio / Signum / Egress | `mercurio:manage`, `signum:manage`, `egress:manage` |
| Produtos digitais | `produtos:manage` |
| Atendimento | `atendimento:abrir`, `atendimento:atender`, `atendimento:manage` |
| Ativos / Problemas | `ativos:manage`, `problemas:read`, `problemas:manage` |
| Mudanças | `mudancas:manage`, `mudancas:aprovar` |
| Módulo Modelo | `example:manage` |

O que **não** exige permissão, só estar autenticado:
- ler Blog e Wiki, criar e editar páginas da Wiki;
- usar o Mercúrio, a Agenda (eventos próprios), os Arquivos (pastas
  próprias e compartilhadas) e o Diretório;
- criar envelopes no Signum e usar a Busca.

### 3.2. Perfis

Um perfil é um **conjunto nomeado de permissões**. Perfis de sistema (vêm
com a plataforma):

| Perfil | Permissões | Para quem |
|---|---|---|
| `administrador` | `*` | Administração da plataforma |
| `auditor` | `audit:read`, `audit:verify`, `users:read`, `monitoring:read` | Controle interno |
| `gestor-iam` | `users:read`, `users:manage`, `iam:manage` | Quem cadastra estrutura, usuários e lotações |
| `gestor-conteudo` | `blog:manage`, `catalog:manage`, `wiki:manage`, `contact:read`, `contact:manage` | Comunicação e ouvidoria |
| `protocolo` | `tramite:create`, `tramite:route` | Quem abre e tramita processos |
| `servidor` | *(nenhuma)* | Colaborador comum, só o que não exige permissão |

Regras:
- É possível criar **perfis customizados** com qualquer combinação de
  permissões válidas. Permissão malformada é recusada.
- Perfis de sistema não são excluídos e mantêm o slug. O `administrador`
  **sempre** mantém `*` e fica ativo, para que um erro de edição não tranque
  a plataforma.
- **Desativar um perfil** retira na hora as permissões dele de todos que o
  têm. Reativar devolve.
- A role de realm `nova-admin` no Keycloak equivale a `*`.

### 3.3. Ninguém concede o que não tem (anti-escalada)

Criar ou editar um perfil, lotar alguém ou mapear um grupo exige que
**quem concede** tenha todas as permissões envolvidas. Retirar também exige.
Na administração delegada (seção 5), vale **no escopo da lotação**: o
delegado só concede uma permissão que ele próprio tem ali.
Exemplo: o `gestor-iam` (que não tem `audit:read`) não cria um perfil com
`audit:read` nem lota alguém num perfil com essa permissão; recebe `403`.
Só quem tem acesso total concede ou retira `nova-admin`.

---

## 4. Lotações e mapeamentos de grupo

Há dois jeitos de alguém receber um perfil. Os dois resultam na mesma coisa:
**perfil + escopo**.

| | Lotação manual | Mapeamento de grupo |
|---|---|---|
| O que é | "Usuário X tem o perfil P no escopo E" | "Quem está no grupo G do Keycloak/AD tem o perfil P no escopo E" |
| Onde se cadastra | Configurações → Usuários → Lotações | Configurações → Mapeamento AD |
| Quando vale | Enquanto existir | Enquanto a pessoa estiver no grupo (lido do token a cada login ou renovação) |
| Uso típico | Exceções e acúmulos | Regra geral: departamento do AD → perfil + departamento |

- **Escopo:** entidade, unidade e/ou departamento. Tudo vazio significa a
  plataforma inteira. O escopo é **completado e validado**:
  - informar só o departamento preenche a unidade e a entidade;
  - informar unidade e departamento de unidades diferentes é recusado.
- Uma pessoa pode ter **várias** lotações, por exemplo trabalhar em duas
  unidades.
- O grupo é comparado sem diferenciar maiúsculas e aceita o caminho do
  Keycloak (`/SEMSA/SEMSA-UPA/SEMSA-UPA-ADM` ou `SEMSA-UPA-ADM`).
- Cada lotação é uma **concessão** separada: as permissões do perfil
  valem no escopo daquela lotação, não somadas para a plataforma toda
  (seção 5). O menu mostra o que a pessoa pode fazer em **algum** lugar.
- Mudanças valem na próxima requisição da pessoa, sem novo login.

---

## 5. Onde o escopo vale (e onde não vale)

> **Regra central (ADR 013):** cada lotação é uma **concessão**: um perfil
> num lugar. As permissões do perfil valem **só onde a concessão foi dada**
> e **para baixo** — a unidade cobre as subunidades e os departamentos
> dela; a entidade cobre todas as unidades. Concessão sem escopo é
> **global**.

**Permissões com escopo** (valem onde foram concedidas):

| Permissão | Onde o recurso "está" |
|---|---|
| `tramite:create`, `tramite:route`, `tramite:manage` | unidade de origem ou atual do processo |
| `catalog:manage` | unidade responsável do serviço |
| `blog:manage` | unidade dona do post |
| `wiki:manage` | unidade dona da página (herdada da página-mãe) |
| `calendar:manage` | unidade dona da sala (e dos eventos reservados nela) |
| `mercurio:manage` | departamento da sala |
| `directory:manage` | lotação exibida da pessoa |
| `contact:read`, `contact:manage` | setor para onde a triagem encaminhou a mensagem |
| `audit:read` | lotação do autor da ação, gravada no registro |
| `files:manage` | unidade dona da pasta (ou de uma pasta acima) |
| `iam:manage`, `users:read`, `users:manage` | estrutura, lotações e contas da área (ver IAM, seção 7) |
| `produtos:manage` | unidade provedora do produto digital |
| `atendimento:manage`, `atendimento:atender` | unidade da oferta, da equipe e da solicitação |
| `atendimento:abrir` | vale em qualquer concessão: abre pelo catálogo inteiro |
| `cofre:read`, `cofre:reveal`, `cofre:manage`, `cofre:grant` | unidade dona do segredo |

**Permissões de plataforma** (`modules:manage`, `keycloak:manage`, `ia:manage`,
`branding:manage`, `monitoring:*`, `egress:manage`, `audit:verify`,
`signum:manage`): **só valem com concessão global**. Dadas com escopo, não
valem em lugar nenhum.

**Institucional:** recurso sem unidade dona (tudo o que existia antes do
ADR 013, e o que a gestão global criar sem dona). Só a gestão global o
gerencia.

**A leitura:** o escopo restringe a **gestão**. Para restringir a
**leitura**, Blog, Wiki e Agenda têm o **público-alvo** (ADR 014): o
conteúdo pode ser só para uma ou mais secretarias e/ou unidades. Sem
público-alvo, tudo é visível a todos como antes; o Catálogo publicado
segue sempre aberto (é do cidadão).

**Público-alvo (ADR 014):**
- **Quem pertence:** quem tem lotação (manual ou por grupo do AD, de
  qualquer perfil) numa secretaria do público, ou numa unidade dele **ou
  abaixo dela** (a lotação num departamento conta na unidade dele).
  Concessão só global não é pertencimento.
- **Quem mais vê:** o autor (organizador, no evento) e a gestão que cobre
  o conteúdo — a do módulo na unidade dona — e a gestão global.
- Conteúdo com público-alvo **nunca** vai ao site público; não é
  difundido a todos em tempo real; a Busca Global respeita.
- Excluir uma secretaria ou unidade que é público de algum conteúdo é
  recusado (senão o conteúdo ficaria aberto a todos em silêncio).
- Público-alvo direciona a leitura; **não é sigilo** (o sigilo de
  processos é regra do Trâmite).

Além disso, **Arquivos** usa as lotações nas ACLs das pastas (`unidade`,
`departamento`, `perfil`) e o **Trâmite** usa a lotação para abrir e
tramitar.

Consequência prática: lotar alguém como `gestor-conteudo` **na UPA** dá
`blog:manage` e `catalog:manage` **só na UPA** (e nas subunidades dela).
Para gerir a plataforma inteira, a concessão precisa ser global. O
relatório `make iam-scope-report` mostra, por lotação, o que vale e o que
não vale.

---

## 6. Como modelar uma implantação

1. **Entidades:** uma por órgão (ou empresa).
2. **Unidades:** os locais de trabalho, que no Trâmite são quem "recebe"
   processos. Use unidade-mãe se houver hierarquia.
3. **Departamentos:** os setores de cada unidade.
4. **Grupos no Keycloak/AD:** idealmente um por departamento, com nome
   único (ex.: `SEMSA-UPA-ADM`), e grupos de papel (`NOVA-ADMINS`,
   `NOVA-AUDITORES`…).
5. **Mapeamentos:**
   - cada grupo de departamento → `servidor` no departamento;
   - quem protocola (ex.: a Administração da unidade) → `protocolo` na
     unidade;
   - grupos de papel → perfis de gestão, sem escopo.
6. **Exceções:** lotações manuais.

O ambiente de teste segue exatamente essa receita
(`scripts/demo-data/generate.mjs`), com 4 entidades, 15 unidades e 76
departamentos.

**Várias organizações independentes** (ex.: duas empresas): a estrutura se
modela sem código, com uma entidade por empresa. Mas a plataforma **não
isola** as empresas entre si. Blog, Wiki, Catálogo, Agenda pública, a sala
"Geral" do Mercúrio, o Diretório, a identidade visual e o site são
compartilhados, e o administrador vê tudo. Para empresas independentes, use
**uma implantação por empresa**.

---

## 7. Regras por módulo

### Trâmite (processos administrativos)
- **Tipos:** Contratação, Memorando, Ofício, Requerimento, Outros. Tipo
  desativado não abre processo novo.
- **Numeração:** `NNNNNN/AAAA`, sequencial por ano.
- **Estados:** `aberto` → `em_tramitacao` → `concluido` → `arquivado`.
  Reabrir (`tramite:manage` na unidade atual) volta para `em_tramitacao`. Só se arquiva o
  que está **concluído**.
- **Abrir:** exige `tramite:create` e lotação na unidade de origem, que
  precisa estar ativa (e a entidade dela também).
- **Tramitar:** exige `tramite:route`, o processo na sua unidade e um
  destino ativo, diferente da unidade atual.
- **Sigilo:**

  | Sigilo | Quem vê |
  |---|---|
  | público | qualquer autenticado |
  | restrito | autor, credenciados, lotados na unidade de origem ou na atual, e `tramite:manage` que cubra uma delas |
  | sigiloso | **somente** o autor e os credenciados nominalmente (nem a unidade nem o gestor) |

  Credenciar ou revogar acesso nominal só existe para processo não público
  e dá **leitura**.
- **Documentos:**
  - nascem em rascunho;
  - são editáveis só pela unidade que está com o processo;
  - anexos vão para o MinIO por URL pré-assinada;
  - **pedido de assinatura** abre um envelope no Signum, e o documento
    assinado não pode mais ser editado;
  - não se conclui processo com assinatura pendente.
- Processo concluído ou arquivado não recebe alterações. Cada movimento
  (abertura, tramitação, conclusão, arquivamento, reabertura) fica no
  histórico.

### Signum (assinatura eletrônica)
- Qualquer autenticado abre um envelope com o **hash SHA-256** do
  documento e de 1 a 50 signatários ativos, em ordem sequencial ou livre.
- **Cerimônia:**
  1. o signatário pede um **desafio** (uso único, expira em
     `SIGNUM_CHALLENGE_TTL`, padrão 5 min);
  2. confirma o hash do documento;
  3. **reautentica com a senha**: conta local contra o banco, conta do
     Keycloak via Keycloak/AD, pelo client `nova-backend`.
- **Sequencial:** ninguém assina antes dos anteriores.
- **Estados:** `pending` → `completed` (todos assinaram) | `refused`
  (alguém recusou com motivo) | `cancelled` (dono ou `signum:manage`
  cancelou). Envelope não pendente não aceita ação.
- Tentativas de senha erradas são limitadas por usuário.
- **Verificação pública** pelo id do envelope, sem login.
- Veem o envelope: o dono, os signatários e `signum:manage`.

### Arquivos
- Pastas em árvore e arquivos no MinIO. Upload e download são feitos
  **direto** pelo navegador, por URL pré-assinada (15 min); o limite é
  `UPLOAD_MAX_FILE_MB` (100 MB).
- **Acesso a uma pasta:**
  - `files:manage` global tem acesso total;
  - `files:manage` com escopo tem acesso total às pastas cuja **unidade
    dona** (da própria pasta ou de uma acima) ele cobre;
  - o **dono** da pasta, ou de qualquer pasta acima dela, tem acesso total;
  - a **ACL** de qualquer nível acima (herança) dá leitura, e escrita se
    `can_write`.
- Sujeitos da ACL: `everyone`, `user`, `perfil`, `ad_group`, `unidade`,
  `departamento`.
- Alterar a ACL, renomear ou excluir a pasta exige acesso total.
- **Unidade dona** (opcional): marca quem está lotado na unidade ou tem
  `files:manage` cobrindo-a; trocar exige poder marcar a antiga e a nova.
  Levar para a raiz uma pasta cuja dona era herdada exige ser o dono dela
  ou marcá-la antes.
- Nomes são únicos dentro da pasta. Pasta com conteúdo só é excluída com
  confirmação recursiva. Não se move uma pasta para dentro de si mesma.

### Mercúrio (chat)
- **Salas:**
  - **global** (todos);
  - **de departamento**: acesso por grupo do AD ou lotação no
    departamento, e `mercurio:manage`;
  - **direta** (duas pessoas; não se abre conversa consigo mesmo).
- Criar ou arquivar sala e moderar mensagens: `mercurio:manage` cobrindo
  o departamento da sala (sala global e direta: só a gestão global). Sala
  arquivada é só leitura.
- Mensagens de 1 a 4.000 caracteres. O autor **edita nos primeiros 15
  minutos** e pode excluir. Tudo chega em **tempo real**.
- Não lidas por sala; "marcar como lida" zera o contador.

### Agenda
- Eventos com visibilidade:
  - **público:** aparece no site;
  - **interno:** todos os autenticados;
  - **privado:** só o organizador.
- Duração de até 31 dias.
- Salas com capacidade, recursos e **unidade dona**: criar, editar e
  excluir exigem `calendar:manage` cobrindo a dona (sala sem dona é
  institucional). Salas inativas aparecem só para a gestão que as cobre. **Sem conflito:**
  uma sala não aceita duas reservas no mesmo horário. Sala inativa não
  aceita reservas, e sala com reservas futuras não é excluída (desative).
- Só o organizador ou `calendar:manage` na unidade dona da **sala
  reservada** altera, cancela ou vê um evento privado de terceiros. Evento
  sem sala é institucional. Evento cancelado não se
  altera, cria-se outro. Cancelar libera a sala.
- **Público-alvo** (seção 5): o evento interno só aparece para o público
  escolhido; com público, o evento não pode ser "público" (site), e na
  ocupação da sala aparece como "Reservado".

### Blog
- `blog:manage` cria, edita, publica, arquiva e exclui os posts da
  unidade dona que ele cobre. Quem tem só escopo escolhe a dona entre as
  suas unidades; post institucional (sem dona) é da gestão global.
- **Estados:** `draft` → `published` → `archived`, com volta a rascunho.
  Não se publica sem texto.
- Leitores veem só o publicado. Rascunho e arquivado ficam só para
  a gestão que cobre a dona. Slug único. Tipos: notícia ou comunicado.
- **Público-alvo** (seção 5): o post pode ser só para secretarias e/ou
  unidades; fora do público some da lista, do detalhe e da busca.

### Wiki
- Qualquer autenticado cria e edita. Excluir exige `wiki:manage` cobrindo a
  **unidade dona** da página (sem dona: só a gestão global).
- **Unidade dona:** vazia na criação, herda a da página-mãe. Marca uma
  unidade como dona quem está lotado nela ou tem `wiki:manage` cobrindo-a;
  trocar a dona exige poder marcar a antiga e a nova.
- **Público-alvo** (seção 5), com herança: a página é lida por quem
  satisfaz o público de **cada** página da cadeia até a raiz que tem um —
  a subpágina restringe mais, nunca abre o que a mãe fechou. Trocar o
  público de uma página: quem a criou ou a gestão da dona (qualquer um
  edita, mas não esconde dos colegas o que é de todos).
- **Controle de concorrência:** a edição informa a versão aberta. Se outra
  pessoa salvou antes, a edição é recusada (`409`) em vez de sobrescrever.
- Histórico de revisões com restauração. Árvore de páginas: não se exclui
  página com subpáginas, e uma página não fica dentro de si mesma.

### Catálogo de Serviços (Carta de Serviços)
- `catalog:manage` cria, publica e arquiva os serviços da unidade
  responsável que ele cobre (sem unidade: gestão global).
- **Publicar exige** resumo e ao menos um **canal de atendimento** (Lei
  13.460/2017).
- Publicado aparece no site sem login. Publicado **não é excluído**
  (arquive antes). A unidade responsável precisa existir.

### Produtos digitais
- `produtos:manage` cadastra o produto da unidade provedora que cobre
  (sem unidade: gestão global): nome, utilidade, garantia e o estágio do
  ciclo de vida — descoberta, desenho, aquisição, construção, transição,
  operação, entrega e suporte.
- Mudar o estágio é uma ação à parte da edição; repetir o mesmo estágio
  não gera evento nem auditoria.

### Atendimento (central de serviços)
- **Oferta** é o que o servidor pode pedir, em linguagem simples ("não
  consigo acessar o e-mail"). Não é a Carta de Serviços do site. O
  **tipo** da oferta decide o fluxo: **incidente** (algo parou) ou
  **requisição** (um pedido). Pode apontar para um produto da mesma
  unidade. **Equipe** reúne servidores ativos lotados na unidade.
  Cadastrar os dois: `atendimento:manage`.
- **Abrir** (`atendimento:abrir`): o servidor escolhe uma oferta ativa e
  descreve o pedido. A solicitação herda da oferta o tipo, a unidade que
  atende e o produto, e ganha um número sequencial.
- **Triagem:** até 30 regras por unidade, em ordem. A primeira que casa
  (oferta, palavra no título, unidade de quem pede — o que estiver
  preenchido) define a prioridade pela matriz já gravada e avisa só a
  equipe escolhida. Não marca responsável. Sem regra, o chamado segue
  com impacto e urgência médios.
- **Resposta pronta:** uma resposta pública e uma nota interna por
  oferta. Quem cadastra vê o texto já gravado da oferta. A equipe cola
  o texto na caixa. O envio não é automático.
- **Chamado mensal:** no dia 1 a 28, no fuso da política (ou Brasília),
  uma vez naquele mês. Quem criou a marca é o solicitante. O prazo
  começa na abertura, como em qualquer chamado.
- **Quem vê:** quem abriu e quem tem `atendimento:atender` na unidade da
  solicitação. Para os demais, ela não existe (404).
- **Responsável:** só um membro ativo de uma equipe da unidade.
  Assumir uma solicitação aberta já a põe em atendimento; não se atende
  sem responsável.
- **Estados:** `aberto`, `em_atendimento`, `aguardando_servidor`,
  `aguardando_terceiro`, a conclusão (`resolvido` no incidente,
  `atendido` na requisição), `encerrado` e `cancelado`. A equipe move
  entre eles; quem abriu cancela enquanto ninguém começou e, depois da
  conclusão, confirma (encerra) ou reabre. Quando a equipe aguarda o
  servidor, a resposta dele devolve a solicitação ao atendimento.
  Encerrada ou cancelada não muda mais nem recebe comentário.
- **Conversa:** a resposta é vista pelas duas pontas; a **nota interna**
  é só da equipe e não chega a quem abriu, nem no aviso.
- Cada passagem de estado fica registrada com horário e autor (base do
  prazo, na fase seguinte); toda mudança gera evento e auditoria. Os
  avisos vão para o sino de quem abriu, do responsável e, na abertura, da
  equipe da unidade — nunca para quem fez a ação.
- **Fila em lista ou quadro:** as duas vistas leem a mesma consulta e
  mudam o estado pela mesma passagem. No quadro, o cartão só vai para uma
  coluna que a regra permite; se a API recusar (ex.: atender sem
  responsável), ele fica onde estava. Arrastar é atalho: todo cartão tem
  o seletor "Mover para…" (teclado, toque e leitor de tela). Cancelar,
  só pela tela da solicitação, com confirmação.
- **Tempo real:** quem está com a fila, o quadro ou a solicitação aberta
  vê a mudança de um colega sem recarregar. O aviso não leva conteúdo —
  só diz o que mudou, e a tela busca de novo pela API. A fila de uma
  unidade só é assinada por quem atende ali; a nota interna só chega à
  equipe.
- **Prazo (tempo útil):** cada unidade tem uma **política** —
  fuso, turnos de expediente (até 4 por dia), a matriz **impacto x
  urgência → prioridade** (as 9 combinações) e as **metas** de resposta
  e de solução de cada prioridade usada (resposta nunca maior que a
  solução). Unidade sem política própria usa a da unidade acima e, no
  topo, a institucional; sem nenhuma, a solicitação fica **sem prazo**.
  Cadastrar: `atendimento:manage`.
- **Feriados e pontos facultativos** não contam como tempo útil; o da
  unidade vale para as subunidades, o institucional para todas. Uma data
  por unidade.
- A solicitação **nasce classificada** com impacto e urgência médios; a
  equipe **reclassifica** na triagem. A prioridade e as metas ficam
  gravadas na solicitação: mudar a política depois **não altera** o
  compromisso já assumido.
- **Relógios:** a **resposta** corre enquanto a solicitação está aberta e
  ninguém começou; a **solução** corre aberta ou em atendimento. As duas
  esperas (`aguardando_servidor`, `aguardando_terceiro`) **pausam** a
  solução; a conclusão a para, e a reabertura volta a contar.
- **Semáforo:** no prazo, atenção (75%), crítico (90%), vencido (100%);
  com o relógio parado, cumprido ou descumprido. Só a equipe vê o prazo.
- **Alertas** aos 75%, 90% e no vencimento. Aos 75%, o sino é do
  responsável ou, sem ele, da equipe da unidade. Aos 90%, nas duas horas
  finais e no vencimento, a equipe da unidade recebe o mesmo aviso junto
  com o responsável. O responsável não é trocado. Cada limiar avisa uma
  vez por pessoa, solicitação e relógio, mesmo com várias réplicas do
  worker.
- **Busca:** número, título e descrição entram na Busca Global para quem
  abriu a solicitação e para quem atende a unidade. O resultado mostra
  número, título, oferta e estado. A descrição não aparece no trecho, e
  a solicitação não vai para o índice público.
- **Aviso de 2 horas:** com o relógio correndo, o vencimento marcado e a
  meta maior que 2 horas, o sino avisa de novo o responsável e a equipe
  da unidade quando falta no máximo 2 horas. A chave é outra (`…:2h`),
  então o aviso de percentual continua valendo. Relógio parado, já
  vencido ou com meta de até 2 horas não dispara esse aviso. O chamado
  que nasce dentro da janela não alerta na abertura.
- **Avaliação (CSAT):** só quem abriu, e só com a solicitação
  `resolvido`, `atendido` ou `encerrado`. Uma nota de 1 a 5, comentário
  opcional de até 4.000 caracteres, uma vez: a nota não se altera. A
  avaliação não vira comentário. O responsável recebe o aviso. Entra no
  "meus dados"; na eliminação fica como registro do serviço, ligada à
  conta anonimizada, no mesmo critério dos comentários.
- **Indicadores da central** (`GET /api/v1/atendimento/governanca`,
  `atendimento:atender`, no escopo de quem atende): MTTR é a média do
  tempo útil do relógio de solução dos incidentes concluídos com o
  relógio parado; MTBF é o intervalo médio entre as aberturas de
  incidentes não cancelados (precisa de pelo menos dois); SLA é a
  parcela cumprida entre as soluções concluídas com prazo; FCR conta a
  concluída que nunca esperou e entrou em atendimento no máximo uma vez.
  Sem amostra o número vem nulo. A amostra para em 5.000 e a resposta
  marca o truncamento. A média de satisfação usa todas as notas do
  escopo e devolve as cinco mais recentes.
- **Visita (atendimento externo):** quem atende a unidade registra a
  **chegada** (check-in) a partir da solicitação; junta **fotos** e a
  **assinatura** de quem recebeu; registra a **saída** (check-out) com o
  nome de quem recebeu. Uma visita aberta por técnico e solicitação; só o
  técnico da visita a altera. Visita aberta por engano é **cancelada**.
- **Localização (LGPD):** lida **só** na chegada e na saída, nunca em
  segundo plano. Base legal: execução de política pública (art. 7º, III,
  e art. 23), gravada em cada visita e mostrada ao técnico antes da
  leitura. Sem permissão de localização, a visita não é registrada.
  Com destino informado, a chegada mede a distância e marca **fora da
  área** — sem recusar, porque o GPS erra.
- **Retenção:** as coordenadas são apagadas **um ano** depois da última
  leitura (worker de hora em hora; a auditoria registra quantas visitas
  foram tocadas). Horário, distância, "fora da área", fotos e assinatura
  ficam. A **auditoria e os eventos nunca levam coordenadas** (a trilha é
  imutável e não poderia ser apagada).
- **Quem vê a posição:** só a **supervisão** (`atendimento:atender` na
  unidade) e só das visitas **em andamento**; o mapa abre no
  OpenStreetMap, fora do sistema. O detalhe da solicitação mostra a
  visita sem coordenadas, para a equipe e para quem abriu. Fotos e
  assinatura abrem por URL temporária, para quem vê a solicitação.
- **Sem rede:** chegada, fotos, assinatura, saída e cancelamento ficam
  guardados no aparelho e sobem, em ordem, quando a conexão volta. O
  horário vale o do aparelho (até 7 dias de atraso; não antes da
  abertura da solicitação). Reenviar não duplica.
- **Direitos do titular:** "meus dados" inclui as solicitações abertas,
  os comentários escritos e as visitas feitas (com as coordenadas ainda
  guardadas). A eliminação apaga as coordenadas das visitas da pessoa;
  solicitações, comentários e visitas ficam como registro da prestação do
  serviço, ligados à conta anonimizada.
- **Perguntas da oferta:** até 20, cada uma com rótulo e tipo texto, número
  ou lista. A lista exige opções. A chave sai do rótulo quando fica em
  branco. A resposta entra na abertura e não muda depois. Chave
  desconhecida, obrigatório vazio, número com letra ou opção fora da
  lista recusam a abertura.
- **Acompanhamento de quem pediu:** a lista diz o próximo passo em
  português comum. A equipe está cuidando, pediu uma resposta, cabe
  avaliar, o pedido encerrou ou foi cancelado. Não há outra tela para
  a mesma pessoa.
- **Série diária:** de hora em hora o worker grava o retrato do dia
  civil de cada unidade que já teve chamado e uma linha de todo o
  escopo. O dia segue o fuso da política da unidade; sem política, vale
  Brasília. Quem cobre várias unidades lê a janela no fuso
  institucional. O dia corrente é atualizado. Os dias anteriores ficam,
  inclusive os que já foram gravados em UTC.
  `atendimento:atender` lê até 366 dias. Sem amostra, o indicador fica
  vazio. Quem cobre mais de uma unidade não recebe um intervalo entre
  falhas misturado.
- **Correio:** o aviso do atendimento (passagem e pedido de avaliação)
  sai por e-mail depois do sino, se o SMTP estiver configurado. Falha
  de envio não desfaz a solicitação. Os outros módulos continuam só no
  sino. A resposta recebida vira comentário quando o assunto traz o
  número do chamado e o remetente é o e-mail de quem abriu ou de quem
  atende. A mesma mensagem não entra duas vezes. Remetente estranho,
  chamado encerrado ou número ausente não geram comentário.
- **Separação das 3 entidades provedoras (ADR 075 / ADR 078):** o catálogo de
  ofertas enriquecido (`GET /api/v1/atendimento/catalogo`) identifica a
  secretaria/entidade provedora de cada oferta (no cenário importado do GLPI:
  **Prefeitura/SECITI**, **Saúde/SMS** e **Educação/SEMED**). Na central do
  solicitante (`/atendimento`), quando há mais de uma entidade provedora, o
  catálogo pré-seleciona automaticamente a central correspondente à lotação
  do servidor e exibe abas para alternar entre as centrais ou consultar todas,
  evitando colisão de ofertas homônimas entre secretarias.
- **Integrações e carga do GLPI (ADR 072, 075, 076, 078, 079, 080, 081):**
  `integracoes:manage` gerencia conexões, área de preparo, inventário e a
  tabela De-Para; `integracoes:conector` autentica o conector de máquina
  (`nova_con_`). A sugestão da tabela De-Para vincula automaticamente as 3
  entidades do GLPI (`PREFEITURA` → `SECITI`, `SMS` → `SMS`, `SEMECEL` →
  `SEMED`), importa acompanhamentos e soluções técnicas (`ITILSolution`),
  expediente, feriados, grupos técnicos por entidade e regras de triagem, e
  preserva `takeintoaccountdate` e `waiting_duration` nas passagens de estado
  para não distorcer TTO, pausa de SLA, FCR e experiência (XLA).
  Com a sincronização ligada até a virada (ADR 081), chamados espelhados do
  GLPI ficam somente leitura no atendimento local (`ErrEspelho`) e recebem as
  atualizações incrementais da origem.
- **Vínculo persistente Solicitação ↔ Ativo (CMDB) e Rastreabilidade Reversa:**
  quem atende a unidade pode vincular ou desvincular um **Item de Configuração**
  (`PUT /api/v1/atendimento/solicitacoes/{id}/ativo`, gravando `ativo_id` e
  `ativo_nome` na solicitação e auditando a ação). O ativo precisa pertencer à
  mesma unidade da solicitação e o módulo de ativos precisa estar ligado;
  chamados encerrados, cancelados ou espelhados do GLPI recusam a alteração.
  A fila (`GET /api/v1/atendimento/fila`) aceita `?ativo_id=` e a ficha da
  solicitação exibe os **Problemas**, **Mudanças** e **Projetos** vinculados.
  No relatório por secretaria (`GET /api/v1/atendimento/relatorio/secretarias`),
  cada linha de `Equipes (OLA)` inclui o `unidade_id` da secretaria de topo para
  acompanhar o filtro por entidade (`SECITI`, `SMS`, `SEMED`).

### Ativos
- `ativos:read` consulta os itens e relações no escopo da unidade (concedido
  também aos perfis operacionais da cadeia ITIL v5 para vínculo cruzado);
  `ativos:manage` cadastra, edita e relaciona os itens da unidade que cobre: equipamento,
  sistema ou licença, serviço contratado, rede ou outro. Estados:
  planejado, em uso, em manutenção, desativado. Não há exclusão: o ativo
  que sai de uso fica **desativado**, para que o histórico de chamados, problemas,
  mudanças e projetos siga apontando para ele.
- **Ficha do Item de Configuração (`/ativos/[id]`):** reúne dados patrimoniais,
  inventário coletado pelo agente, relações na CMDB (`depende`, `composto`,
  `conectado`) e rastreabilidade reversa nas 4 práticas ITIL v5 (**Chamados**,
  **Problemas**, **Mudanças** e **Projetos** vinculados ao ativo).
- O **identificador** (patrimônio, número de série) é opcional, mas,
  quando informado, não se repete.
- Pode apontar para o **produto digital** que sustenta, desde que o
  produto seja da mesma unidade (produto institucional, só com ativo
  institucional). A tela de produtos leva
  aos ativos de cada produto.
- **Relações:** um ativo aponta para outro como dependência, composição
  ou conexão. Os dois precisam estar no escopo de quem cadastra. A
  dependência não pode fechar ciclo. Não há varredura de rede.
- **Agente:** `ativos:reportar` deixa a própria máquina informar
  identificador, nome, sistema, processadores, memória, discos locais e
  até 100 programas (nome e versão, sem caminho). Acima disso a lista é
  parcial. A unidade é a do escopo quando há uma só; com várias, o
  relato traz a unidade. Patrimônio cadastrado à mão não muda, e a
  edição manual não apaga a medida já gravada. Desativado e em
  manutenção conservam o estado. O mesmo relato não gera nova
  auditoria. O programa não lista a rede nem compartilhamento.
- **Credencial da máquina:** quem gerencia ativos emite um segredo para
  a unidade. Ele aparece uma vez. O bearer `nova_agt_` só tem
  `ativos:reportar` nessa unidade e não é uma conta de pessoa. Revogar
  invalida o segredo e conserva o registro.

### Problemas e erros conhecidos
- **Problema** é a causa por trás de um ou mais incidentes. `problemas:read`
  consulta, `problemas:manage` registra, investiga e liga. A unidade dona
  é a do escopo de quem registra.
- **Estados:** `registrado` → `em_analise` → `erro_conhecido` (opcional)
  → `resolvido` → `encerrado`. Erro conhecido e resolvido podem voltar à
  análise; encerrado não sai. **Erro conhecido exige solução de contorno**; **resolver e encerrar
  exigem causa raiz**. O banco também recusa essas combinações. Editar não
  apaga o que o estado atual exige.
- **Vínculos:** incidentes (só solicitações do tipo incidente) e ativos,
  da mesma unidade do problema.
  O vínculo guarda número e título do incidente no momento da ligação.
  Ligar de novo o que já está ligado não duplica. Problema encerrado não
  aceita nem desfaz vínculos. A listagem (`GET /api/v1/problemas`) aceita
  `?solicitacao_id=` e `?ativo_id=` para consulta reversa a partir do chamado
  ou do item de configuração.
- **No incidente:** o técnico vê o problema relacionado e, se já for erro
  conhecido, o contorno a aplicar.
- **Sugestões de recorrência:** uma oferta com **3 ou mais incidentes em
  30 dias** na mesma unidade aparece como sugestão, sem contar os
  incidentes já ligados a algum problema. "Abrir problema" cria o problema
  e liga todos esses incidentes. Sem o módulo de atendimento ligado, não
  há sugestões.

### Mudanças
- **Tipos:** **padrão** (rotineira, pré-aprovada), **normal** e
  **emergencial**. Risco baixo, médio ou alto. `mudancas:manage` registra
  e conduz; `mudancas:aprovar` decide.
- **Estados:** `rascunho` → `em_avaliacao` → `aprovada` | `rejeitada`;
  `aprovada` → `em_execucao` → `concluida` | `falhou`. Rascunho,
  rejeitada, em avaliação e aprovada podem ser **canceladas**. A rejeitada
  é corrigida e reenviada; só rascunho e rejeitada são editáveis.
  Concluída, falhou e cancelada não mudam mais de estado.
- **Enviar para aprovação exige** a janela planejada (início antes do
  fim) e, fora da padrão, o **plano de reversão**. A padrão é aprovada no
  envio, com a decisão registrada como automática.
- **Segregação:** quem pediu a mudança **não a aprova nem a rejeita**,
  mesmo tendo `mudancas:aprovar`. Rejeitar exige **parecer**; aprovar o
  aceita. Toda decisão fica registrada com autor, data e parecer.
- **Execução:** iniciar grava o horário real de início; concluir e
  falhar gravam o fim. **Falhar exige o resultado** (o que foi revertido).
- **Vínculos:** problemas, incidentes e ativos da mesma unidade da
  mudança (institucional com institucional). O rótulo é guardado no momento da ligação. Mudança
  cancelada não aceita nem desfaz vínculos; a concluída e a que falhou
  aceitam (para ligar a mudança ao problema que ela resolveu).
- **Concluir vínculos junto** (`resolver_vinculos` na passagem para
  `concluida`): na mesma transação, cada incidente vinculado vai para a
  conclusão do tipo dele e cada problema vinculado vai para `resolvido`,
  se a máquina de estados já permitir. Vínculo já concluído é ignorado.
  Ativo não é resolvido. Se algum vínculo não puder ser concluído
  (sem responsável, cancelado, fora do estado, problema sem causa raiz,
  ou o módulo desligado), a mudança inteira permanece como estava. Os
  avisos do sino saem só depois do commit. A folha da RFC imprime o que
  está gravado (risco, justificativa, plano de reversão, decisões e
  vínculos); o navegador gera o PDF.
- **Nota de risco:** opcional, de 5 a 15. Até 7 é baixo, até 11 é médio,
  acima é alto. Nota fora da faixa do risco escolhido é recusada. A
  folha da RFC mostra a nota quando ela está gravada.
- **Congelamento:** um período, da unidade ou de toda a instituição, em
  que mudança nova não é criada com janela sobreposta nem enviada para
  avaliação. A mudança já agendada não é cancelada.

### Contratos e licenças
- `contratos:manage` cadastra fornecedor, contrato e licença no escopo
  da unidade. A licença guarda nome, fornecedor, quantidade de assentos,
  vencimento e, se informado, o nome do programa. A chave secreta não é
  gravada. A conta de instalações lê ativos em uso ou manutenção. Sem
  inventário, a tela diz que nenhum ativo informou programas. Se o
  agente cortou a lista em 100, a conta fica marcada como parcial.
- O sino avisa uma vez em cada marco: 30 dias, 7 dias, no dia e depois
  do vencimento. O aviso vai para quem administra ou tem
  `contratos:manage` no alcance da unidade.

### Cofre de senhas
- O Keycloak autentica a pessoa. O grupo do AD vira um perfil com escopo,
  como nas outras concessões. A senha do cofre não fica em atributo do
  Keycloak nem numa role por segredo.
- `cofre:read` vê nome, usuário, tipo e endereço. `cofre:reveal` revela a senha.
  `cofre:manage` guarda, troca e exclui. `cofre:grant` move o segredo
  para outra unidade que também alcança. Quem administra revela, e essa
  revelação também fica na auditoria.
- A lista e a busca não devolvem a senha nem o texto cifrado. Revelar é
  um POST. A auditoria grava quem revelou, o segredo e a unidade — nunca
  o valor.
- Cada segredo tem a própria chave, embrulhada por `COFRE_MASTER_KEY`.
  Essa chave não é a `CONFIG_ENCRYPTION_KEY`. O texto cifrado está
  amarrado ao segredo, à versão e à unidade.
- Revelar usa a sessão de quem já entrou e tem `cofre:reveal`. Não pede
  a senha de acesso de novo. A auditoria grava quem revelou, nunca a
  senha. Credencial de máquina não revela. Retirar o segredo, que
  reserva o item, continua pedindo a senha de acesso.
- O item é da organização: máquina, aplicação, painel ou outro. Máquina
  é VM ou container: o endereço é o IP e o acesso mostrado é SSH
  (IP, usuário e senha). Aplicação mostra o usuário e a senha de
  administrador. A senha não aparece na lista. Um painel aponta as
  máquinas que administra, pelo identificador, sem copiar a senha delas.
- Quem revela pode retirar o segredo por um prazo, de 15 minutos a 8
  horas. Enquanto a retirada vale, outra pessoa não revela nem retira.
  Quem retirou, ou quem administra, devolve antes do prazo. A lista
  mostra quem retirou e até quando, sem a senha.
- Um segredo marcado rotaciona ao devolver, ao vencer a retirada e a
  cada 30 dias. A senha nova fica só no cofre. O endereço não muda
  sozinho: a lista avisa até alguém com `cofre:manage` registrar que
  aplicou. Segredo sem a marca, e senha trocada à mão, não rotacionam.
- A pasta é da unidade e tem a própria lista de pessoas. Quem não está
  nela não vê o segredo, mesmo com `cofre:read` na unidade. Revelar
  continua pedindo `cofre:reveal` na unidade e a marca de revelar na
  pasta. Quem tem `cofre:manage` na unidade vê a
  pasta, troca a lista de pessoas e coloca o segredo nela na criação
  ou depois, sem mudar a senha. Apagar a pasta só vale quando ela está
  vazia: senha dentro dela impede a exclusão, e a senha não sai junto.
  Mover de unidade tira o segredo da pasta. Não é role do
  Keycloak.
- O chamado pode guardar o id do segredo. A senha não é copiada. Quem
  abriu o pedido não vê o id. O cofre desligado e a solicitação
  encerrada recusam o vínculo.

### Projetos
- **Finalidade:** Gestão e acompanhamento de iniciativas, planos de ação, cronogramas e entregas estratégicas ou operacionais.
- **Permissões e escopo:** `projetos:read` para consulta e `projetos:manage` para criação, edição, transição de estado, tarefas, vínculos e apontamentos no escopo da unidade dona do projeto (sem unidade: gestão institucional). Projeto `cancelado` não aceita edições nem novos vínculos.
- **Ciclo de vida do projeto:**
  - `planejado` → `em_andamento` | `cancelado`
  - `em_andamento` → `suspenso` | `concluido` | `cancelado`
  - `suspenso` → `em_andamento` | `cancelado`
  - `concluido` e `cancelado` encerram o ciclo de vida.
- **Tarefas e marcos:** cada projeto possui tarefas ordenadas com estados `a_fazer`, `em_andamento`, `concluida` e `bloqueada`.
- **Vínculos da Cadeia de Valor (ITIL v5):** cada projeto pode apontar para um **produto digital** (`produto_id`) e vincular itens de **mudanças**, **problemas**, **ativos** e **solicitações**, validados quanto à existência no módulo de origem, módulo ligado no Kernel e compatibilidade de unidade. Quando o rótulo é omitido, o sistema preenche o rótulo oficial do item vinculado. A listagem aceita `?produto_id=` e `?alvo_tipo=&alvo_id=` para rastreabilidade reversa nas fichas de problemas, mudanças e produtos.
- **Visões integradas:** interface em lista, quadro Kanban e Cronograma / Linha do Tempo (Gantt) de 90 dias.
- **Apontamento de Horas & Timesheet:** registro de esforço técnico por tarefa ou projeto (data, horas entre 0 e 24h, custo/hora e descrição das atividades). Cálculo automático do total de horas realizadas, custo acumulado e acompanhamento visual da taxa de queima do orçamento previsto.

### Quadros (kanban e lista)
- **O mesmo catálogo:** problemas, mudanças, projetos, tarefas, ativos e a fila leem nome, cor e ordem em `GET /api/v1/quadros/{quadro}`. O quadro e a lista usam essa ordem, a partir de zero. O nome é rótulo. A chave da coluna é o estado do registro e não muda quando o nome muda.
- **Quem altera:** a permissão de gestão que o módulo já tem (`problemas:manage`, `mudancas:manage`, `projetos:manage`, `ativos:manage`, `atendimento:manage`). Quem só consulta lê o catálogo e não grava. Não há permissão nova.
- **Onde se configura:** cada quadro tem a página Colunas (`/problemas/colunas`, `/mudancas/colunas`, `/projetos/colunas`, `/projetos/tarefas/colunas`, `/ativos/colunas`, `/atendimento/fila/colunas`). Lá ficam o nome (até 80), a cor (`#rrggbb`) e a posição a partir de zero. A posição define o lugar da coluna no quadro e na lista. A barra da coluna não sobe, não desce e não retira. Recolher a coluna continua neste navegador.
- **Retirar:** só nessa página, e só quando a coluna está vazia. A coluna inicial do fluxo (por onde o registro nasce: registrado, rascunho, planejado, a fazer, planejado do ativo, aberto) não sai. Coluna com registro não sai — a mensagem pede para esvaziar. Retirar não apaga a chave: o estado antigo continua legível e devolver coloca a coluna no fim da ordem.
- **Coluna nova:** nasce só no quadro em que foi pedida, em `POST /api/v1/quadros/{quadro}`. A chave é gerada e não muda com o nome. É preciso dizer de quais colunas daquele quadro o card entra e para quais pode sair. Sem entrada, a coluna não nasce. O fluxo de fábrica continua e soma com essas passagens. Em mudanças, quem gere conduz a coluna nova; aprovar e rejeitar continuam com quem aprova. Na fila, quem atende conduz; quem abriu o pedido não ganha caminho novo. Entrar num estado que já exige causa, contorno, janela ou parecer continua exigindo. A coluna nova não pede campo próprio.
- **O que não muda na tela:** mover o card continua na API de estado do módulo. Na mudança, a coluna fechada conta rejeitada, falhou e cancelada. Na fila, concluídas conta resolvido e atendido; encerrada e cancelada ficam na lista, fora do quadro.
- **Auditoria:** cada alteração grava, na mesma transação, quem mudou, a chave e a ordem anterior e a nova. O nome novo não é escrito no lugar do estado.

### Seção Assistência (Central de Serviços estilo ITIL/GLPI)
- **Agrupamento unificado:** no menu lateral, a seção **Assistência** consolida:
  - **Dashboard:** fila, problemas, mudanças e projetos do escopo, mais a satisfação e os indicadores de `GET /api/v1/atendimento/governanca`. Sem amostra, o relatório escreve "sem amostra" — não preenche MTTR, MTBF, SLA nem FCR com constante. A deflexão, no mesmo retorno, conta pessoas que viram artigo e não abriram pedido em 30 dias; sem vista, o campo não vem. O sucesso das mudanças é concluídas sobre concluídas e falhas da lista carregada.
  - **Calendário Operacional:** visão cronológica integrada de janelas planejadas de mudanças (CAB), marcos de projetos e vencimentos de SLA de chamados em grade mensal e lista de prazos.
  - **Detecção de Sobreposições e Conflitos:** alerta proativo visual quando janelas de Mudanças de risco médio ou alto coincidem com outros eventos ou marcos na mesma data, prevenindo colisões operacionais em infraestrutura.
  - **Atendimento:** filas, chamados e quadro kanban de solicitações e incidentes.
  - **Problemas:** catálogo de erros conhecidos, causas-raiz e contornos.
  - **Mudanças:** planejamento de janelas, avaliação do CAB e reversões.
  - **Projetos:** planos de ação, cronogramas, marcos e entregas.

### Contato (ouvidoria)
- Formulário público sem login.
- Exige **consentimento LGPD**. Há um campo-armadilha contra robôs e um
  limite de **5 mensagens por hora por IP**.
- Ler: `contact:read` (contém dados pessoais). Triar: `contact:manage`.
- **Setor:** toda mensagem chega na **caixa geral** (só a gestão global).
  A triagem encaminha a um setor (unidade ativa); `contact:*` com escopo
  vê e trata só o que chegou à área dele e pode devolver à caixa geral,
  mas não encaminhar para fora da área.
- Estados: `new`, `in_progress`, `answered`, `archived`. Pode haver um
  responsável, que precisa ser um usuário ativo.

### Diretório
- Pessoas e setores. Os dados de identidade (nome, e-mail, grupos) vêm do
  Keycloak/AD.
- Cada pessoa edita o próprio cargo, telefone, ramal e bio. A **lotação
  exibida** e o perfil de terceiros exigem `directory:manage` cobrindo a
  lotação exibida atual e a nova. Perfis ocultos aparecem só para essa
  gestão.
- **Setores públicos** no site listam só entidades, unidades e
  departamentos **ativos**.

### Busca Global
- Busca em Blog, Wiki, Arquivos, Catálogo, Diretório e Trâmite, nos
  módulos ativos, **respeitando as regras de cada um**: rascunho não
  aparece, a ACL das pastas vale, o sigilo do processo vale.
- Mínimo de 2 caracteres. Acha por **prefixo** ("vacina" acha
  "vacinação") e aceita aspas, `-termo` e `or`.

### Egress (webhooks de saída)
- `egress:manage` cadastra destinos:
  - tipos webhook, n8n, Zabbix, Grafana;
  - padrões de evento (ex.: `tramite.*`);
  - segredo HMAC opcional, guardado cifrado e nunca devolvido.
- **Anti-SSRF:** destinos em rede interna, loopback, endereço de metadados
  de nuvem ou nome de container são recusados. Em produção não dá para
  desligar a proteção.
- Entregas assíncronas, com **retentativa e espera crescente** até
  `EGRESS_MAX_ATTEMPTS` (8). Há reenvio manual e teste de conexão.

### Auditoria
- `audit:read` consulta (com filtros) e exporta (LAI: CSV, JSON, XML).
  `audit:verify` confere a cadeia de hash (só global). A própria
  exportação é auditada.
- **Com escopo**, `audit:read` vê as ações de quem estava **lotado na
  área quando agiu** (a lotação gravada no registro, protegida pelo hash).
  Ações sem lotação registrada (sistema, contas só globais) ficam com a
  gestão global.

### IAM (administração delegada)
- **Só a administração global:** entidades, perfis e mapeamentos do AD.
- **Estrutura:** com `iam:manage` numa unidade, cria e exclui as
  subunidades dela, edita a própria unidade e as de baixo (mudar de mãe
  exige cobrir o destino) e cuida dos departamentos. Numa entidade, também
  cria e exclui as unidades de 1º nível.
- **Lotações:** só com escopo dentro da área e com perfil cujas permissões
  o delegado tem naquele escopo. Ele vê só as lotações que cobre.
- **Contas:** `users:read` com escopo vê as contas com alguma lotação na
  área (manual ou por grupo do AD) e as sem lotação, para lotá-las.
  `users:manage` com escopo edita, redefine a senha e desbloqueia só
  conta **inteiramente** na área (todas as lotações cobertas) e sem o papel
  de administrador.

### LGPD
- **Aceite do termo:** anônimo (identificador do navegador) ou da conta,
  com versão e IP.
- **Direitos do titular:** "meus dados" (acesso e portabilidade, incluindo
  os dados pessoais dos módulos) e pedido de exclusão, processado com
  retentativas.

### Transparência
- Endpoints públicos, sem login: dados da plataforma, módulos ativos,
  conjuntos de dados e estatística de ações da auditoria.

---

## 8. Pontos de atenção (comportamento atual)

Comportamentos do código hoje que merecem decisão de quem administra:

1. **Permissões de plataforma exigem concessão global** (seção 5): um
   `monitoring:read` ou `modules:manage` dado numa unidade não vale.
2. **Processo sigiloso não é visto nem por `tramite:manage`.** É
   intencional (só autor e credenciados), mas quem administra precisa
   saber.
3. **Reabrir processo exige `tramite:manage`** (na unidade atual),
   que nenhum perfil de sistema tem além do administrador. O Protocolo conclui e arquiva, mas
   não reabre.
4. **Sem isolamento entre entidades** (seção 6): uma implantação serve a
   uma organização (que pode ter várias entidades), não a várias
   organizações independentes.

Corrigidos (antes eram pontos de atenção): perfis de gestão com escopo
valiam para a plataforma toda (ADR 013); estrutura desativada seguia
dando acesso e recebendo processos, e excluir uma unidade-mãe deixava as
subunidades sem mãe em silêncio — ver a seção 2.4.

### 9. Catálogo de Atendimento e Alinhamento Kanban vs. Lista (Pós-Migração GLPI)

1. **Central de Atendimento ("Abrir um chamado"):**
   - O portal principal adota a nomenclatura oficial "Abrir um chamado" (em vez de "Fazer um pedido") para requisições e "Relatar um problema" para incidentes.
   - **Resolução de Entidade Provedora:** Usuários com lotação (seja em nível de departamento, unidade ou entidade) têm sua secretaria raiz detectada automaticamente através da árvore hierárquica (`v1/iam/org-tree`). O catálogo inicializa filtrado para a central da secretaria correspondente do usuário.
   - **Deduplicação de Serviços:** Ofertas com nomes idênticos importadas de múltiplas entidades (ex.: SECITI, SMS, SEMED) ou recargas de GLPI são deduplicadas por entidade/chave. Ao alternar para "Todas as centrais", as ofertas exibem distintivo explícito da secretaria provedora (`Badge`), evitando confusão ao solicitante.

2. **Paridade Rigorosa entre Kanban e Listas Operacionais:**
   - **Fila de Atendimento (`/atendimento/fila`):** Os parâmetros de busca (`q`), recorte de unidade (`unidade`), categoria, prioridade, período e situação (`situacao`) são compartilhados integralmente entre o modo Lista e o modo Quadro (Kanban).
   - **Chamados Concluídos e Migrados:** Por padrão operacional, o quadro foca no fluxo em andamento (`abertas`). Quando a situação selecionada for `"todas"`, o quadro incorpora as colunas de "Encerradas" e "Canceladas", exibindo os chamados importados do GLPI e concluídos.
   - **Problemas e Projetos (`/problemas`, `/projetos`):** A busca textual rápida (`busca`) e o filtro por unidade (`unidadeFiltro`) operam de forma bidirecional e simétrica tanto na visualização em lista quanto no quadro Kanban.
