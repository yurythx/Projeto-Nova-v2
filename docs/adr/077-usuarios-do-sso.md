# 077 — Usuários do SSO trazidos de uma vez, e não só no primeiro login

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 074](074-perfis-pelos-grupos-do-sso.md), [ADR 076](076-modulo-de-integracoes.md)

## Contexto

A conta de quem usa o SSO nasce no primeiro login. Até lá, a pessoa não existe no Projeto Nova: não pode ser responsável por um chamado, aprovadora nem membro de equipe, e a migração do GLPI não tem com quem casar os 515 nomes dos chamados.

## Decisão

1. **Sincronizar usuários do SSO** é uma ação do IAM (`users:manage`), sob demanda e periódica no worker.
2. **O que lê:** a API de administração do Keycloak, com uma conta de serviço do client do Projeto Nova. Essa conta só tem a função `view-users` do realm. A configuração do SSO já existente traz o endereço, o client e o segredo.
3. **O que grava:** a mesma linha que o primeiro login gravaria, com o mesmo `keycloak_subject`, usuário, e-mail, nome e grupos. Quem entra depois cai na conta que já existe. Quem foi desativado no SSO é desativado aqui e volta quando o SSO o reativar (`desativada_pelo_sso`). Federado ao AD, o Keycloak às vezes informa um estado desatualizado: das contas desativadas na segunda sincronização da produção, uma amostra achou ativa no SSO uma em cada oito. Quem um administrador desativou continua desativado. A lista resumida do Keycloak oscila (na terceira rodada, 427 desativações e 355 reativações), provavelmente porque cada réplica guarda o seu cache. Por isso, a mudança de estado, desativar ou reativar, é confirmada na pessoa completa (`GET /users/{id}`), que o Keycloak valida no AD. Só a mudança custa essa leitura, e sem confirmação o estado não muda. Nem a pessoa completa é estável: na rodada seguinte ainda houve 134 desativações e 192 reativações. Por isso a fonte final é o **login**. Quem entra pelo SSO está ativo nele, e a conta que a sincronização desativou volta na hora. Assim a desativação por estado velho do Keycloak não tira o acesso de ninguém; no máximo a pessoa some das listas até entrar ou até a próxima rodada. Ninguém é apagado: a autoria dos registros continua.
4. **Conta local com o mesmo usuário ou e-mail não é vinculada**, como no primeiro login. Ela aparece no relatório para um administrador resolver.
5. **Em segundo plano.** Um realm federado ao AD, com milhares de pessoas, leva minutos para ler, e a Cloudflare corta a requisição em 100 s. Por isso:
   - o pedido só começa a sincronização (202) e devolve 409 se já há uma em andamento, de qualquer processo;
   - cada página lida é gravada na hora e atualiza a linha da sincronização em `iam_sso_sincronizacoes` (relatório parcial e batida). Uma interrupção não perde o que já foi lido;
   - a tela consulta `GET /iam/sso/sincronizacao`, que traz a sincronização mais recente, da tela ou do worker, com o andamento, o fim e o erro. Sem batida há 20 minutos, ela é dada como abandonada (o processo caiu) e não segura a próxima;
   - o fim da leitura é o total de `/users/count`, porque o Keycloak federado devolve páginas curtas no meio da lista. A primeira sincronização na produção parou em 1.299 de 7.740 por isso;
   - a leitura usa a representação resumida, em páginas de 50, com os grupos lidos em paralelo e o token renovado a cada página.
   - O total também oscila, e a página além do fim real volta 400: contado de novo, se a lista acabou antes, o fim é normal.
   - Uma página que o Keycloak devolve com o JSON emendado, sempre no mesmo ponto, é lida pessoa por pessoa, e cada uma vem certa.
   - Fora do cache do Keycloak, algumas páginas fazem o provedor LDAP varrer o AD inteiro e levam uns 4 minutos, até o aborto de transação do próprio Keycloak. As páginas seguintes saem do cache. Por isso cada chamada tem até 10 minutos, e a sincronização tem até 6 horas. Uma tentativa interrompida não se perde, porque deixa o cache do Keycloak mais quente para a próxima.

## Consequências

As equipes, os aprovadores e a migração passam a contar com todos os servidores desde o primeiro dia. É preciso habilitar a conta de serviço do client no Keycloak, com `view-users`.
