# ADR 078 — Central do solicitante e canais de comunicação

## Contexto

Quem não atende nem administra só precisa pedir, acompanhar e avaliar. Hoje essa pessoa entra na mesma casca de quem trabalha no sistema, escolhe a oferta numa lista solta e não recebe confirmação nenhuma ao abrir o pedido. O modelo que a prefeitura conhece é a central do GLPI:

- blocos para fazer um pedido, relatar um problema, ver os chamados e ler a base de conhecimento;
- o catálogo também em blocos;
- o chamado como uma linha do tempo;
- a pesquisa de satisfação no fim.

O e-mail já existe ([ADR 049](049-correio-contratos-e-cadastros.md)), mas só se configura por variável de ambiente, sem ligar e desligar pela tela. Não há WhatsApp. E o contato é o ponto fraco: no SSO, só 3 de cada 50 pessoas têm e-mail.

## Decisão

1. **Central do solicitante, no atendimento.** Quem tem `atendimento:abrir` e nenhuma permissão de trabalho entra direto em `/atendimento`, numa casca enxuta, sem o menu do sistema. A API já recusa o resto: a casca só não oferece o que a pessoa não pode usar.
   - **Início em blocos:** "Fazer um pedido" (requisições), "Relatar um problema" (incidentes), "Meus chamados" e "Base de conhecimento" (a wiki publicada).
   - **Catálogo em blocos:** categorias com ícone, descrição e ordem, e as ofertas de cada categoria também em blocos. Categoria e ícone são do atendimento: `atendimento_categorias`, mais `categoria_id`, `icone` e `ordem` na oferta. Oferta sem categoria aparece em "Outros".
   - **Chamado como linha do tempo:** abertura, comentários públicos, passagens de estado, solução e avaliação, em ordem. O comentário interno nunca aparece para quem pediu.
   - **Avaliação:** depois de concluído, nota e facilidade, como hoje ([ADR 069](069-experiencia-de-quem-pede-xla.md)).
2. **Canais de comunicação, na plataforma** (`platform/canais`, tela em Configurações > Canais). Ficam na plataforma porque mudanças, contratos e outros módulos também vão usar.
   - O atendimento só pede "avise quem pediu sobre este fato". O canal decide como a mensagem sai.
   - Cada canal liga e desliga ali, com modelo de mensagem e teste de envio. As credenciais ficam cifradas (`secretcrypto`).
   - **E-mail:** o SMTP passa a ser configurado pela tela. A variável de ambiente fica só como valor inicial. Sem servidor, o canal fica desligado.
   - **WhatsApp:** dois provedores, a API oficial da Meta (Cloud API) e a Evolution API. O administrador escolhe qual fica ativo. A Evolution usa um número comum e fere os termos da Meta, e a tela avisa disso.
3. **O que quem pediu recebe**, por cada canal ligado e pelo sino: a confirmação de abertura com o link de acompanhamento, os comentários públicos, as mudanças de estado, a solução e o pedido de avaliação.
4. **A avaliação pede login**, como no GLPI. O link leva à página do chamado, e a pessoa entra pelo SSO. Não há link de nota sem login.
5. **Contato, na conta da pessoa** (colunas em `users`, rotas `/me/contato` em `platform/canais`). O perfil ganha e-mail de contato e WhatsApp, com o consentimento para receber avisos (LGPD). A central pede esses dados no primeiro acesso de quem não os tem, e a pessoa pode recusar.
6. **Fases:**
   - (1) a central do solicitante;
   - (2) os canais com o e-mail e o contato;
   - (3) o WhatsApp de saída;
   - (4) a entrada pelo WhatsApp, como já é com o IMAP: responder vira comentário.

## Implementação

- **Fase 1:** `atendimento_categorias`, mais `categoria_id`, `icone` e `ordem` na oferta (migração 33); a central em `/atendimento`, com as visões na URL; a casca enxuta para quem só tem `atendimento:abrir`.
- **Fase 2:**
  - `platform/canais` (migração 34) guarda o SMTP numa linha só (`canais_email`), com a senha cifrada, e implementa o correio do sino;
  - a configuração vigente fica 30 s em memória, então a troca vale nas outras réplicas em até esse tempo;
  - rotas `GET/PUT /canais/email` e `POST /canais/email/testar` (`canais:manage`, com auditoria sem a senha) e `GET/PUT /me/contato` (a própria pessoa);
  - o e-mail vai para o contato que a pessoa deixou ou, sem ele, para o e-mail do login, nunca para o marcador `.invalid` de quem veio do SSO sem e-mail;
  - o corpo leva o link do chamado e o caminho para desligar os avisos;
  - quem pediu recebe também a confirmação de abertura e o aviso de que o chamado tem um responsável.

## Consequências

Quem pede tem uma porta só e não se perde no sistema. Os avisos passam a chegar fora do sistema, para quem deixou contato. Ligar o e-mail exige um servidor SMTP da prefeitura ou um serviço transacional. A API oficial do WhatsApp exige verificar a empresa na Meta e aprovar os modelos de mensagem, e cobra por conversa.
