# 074 — Perfis pelos grupos do SSO: administração por grupo e básico para todo mundo

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 013](013-permissao-com-escopo-e-heranca.md) (escopo das concessões)

## Contexto

Na primeira entrada pelo SSO da prefeitura, quem entrou não tinha perfil nenhum: as telas de administração respondiam 403 (`users:read`, `audit:read`). O IAM já cruzava grupos do SSO com perfis (`ad_group_mappings`), mas não havia como dar um perfil a quem não está em grupo mapeado, e o perfil básico (`servidor`) não tinha permissão nenhuma.

A regra pedida: quem vem dos grupos de administração ou de TI do SSO é administrador do sistema e vê tudo; o resto entra com o básico, sem administração.

## Decisão

1. **O grupo `*` casa com qualquer pessoa autenticada.** O resolvedor do IAM soma `*` aos grupos de toda identidade, e um mapeamento de `*` em `ad_group_mappings` vale para todo mundo. Continua sendo um mapeamento comum: aparece e se edita na tela de IAM.
2. **O perfil `servidor` passa a abrir e acompanhar as próprias solicitações** (`atendimento:abrir`), pela migração 000030. Só muda se ninguém editou o perfil.
3. **Os mapeamentos da implantação ficam no Ansible** (`iam_mapeamentos` em `group_vars/all.yml`): os grupos de administração e TI do AD (`Administrators`, `Domain Admins`, `Departamento de Tecnologia da Informação` e `Grupo Seciti Tecnologia`) para o perfil `administrador` (`*`), e `*` para `servidor`, todos com escopo global. O deploy os grava depois de subir a stack e não duplica o que já existe. Os nomes dos grupos são da prefeitura, não do produto: por isso não entram numa migração.

## Consequências

Quem entra pelo SSO sempre tem pelo menos o básico, e um grupo novo de administração é uma linha no Ansible ou um cadastro na tela de IAM. Tirar um grupo do Ansible não apaga o mapeamento já gravado: isso é feito na tela. O resolvedor guarda as permissões por 60 segundos, então um mapeamento gravado direto no banco leva até um minuto para valer.
