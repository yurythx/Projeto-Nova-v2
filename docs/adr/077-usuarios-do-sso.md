# 077 — Usuários do SSO trazidos de uma vez, e não só no primeiro login

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 074](074-perfis-pelos-grupos-do-sso.md), [ADR 076](076-modulo-de-integracoes.md)

## Contexto

A conta de quem usa o SSO nasce no primeiro login. Até lá, a pessoa não existe no Projeto Nova: não pode ser responsável por um chamado, aprovadora nem membro de equipe, e a migração do GLPI não tem com quem casar os 515 nomes dos chamados.

## Decisão

1. **Sincronizar usuários do SSO** é uma ação do IAM (`users:manage`), sob demanda e periódica no worker.
2. **O que lê:** a API de administração do Keycloak, com uma conta de serviço do client do Projeto Nova. Essa conta só tem a função `view-users` do realm. A configuração do SSO já existente traz o endereço, o client e o segredo.
3. **O que grava:** a mesma linha que o primeiro login gravaria, com o mesmo `keycloak_subject`, usuário, e-mail, nome e grupos. Quem entra depois cai na conta que já existe. Quem foi desativado no SSO é desativado aqui. Ninguém é apagado: a autoria dos registros continua.
4. **Conta local com o mesmo usuário ou e-mail não é vinculada**, como no primeiro login. Ela aparece no relatório para um administrador resolver.

## Consequências

As equipes, os aprovadores e a migração passam a contar com todos os servidores desde o primeiro dia. É preciso habilitar a conta de serviço do client no Keycloak, com `view-users`.
