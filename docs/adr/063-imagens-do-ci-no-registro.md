# 063 — Imagens do CI no registro; o servidor só baixa

- **Status:** Aceito e implementado
- **Data:** 2026-10-27
- **Relacionado:** [ADR 010](010-revisao-da-plataforma-e-cobertura-por-pacote.md) (CI), [docs/DEPLOY.md](../DEPLOY.md)

## Contexto

O deploy fazia `git pull` e compilava as imagens na própria VM de produção. A imagem que rodava não era a que o CI testou, o build ocupava a máquina que atende, e voltar uma versão significava compilar de novo um commit antigo.

## Decisão

O workflow `Release` roda depois do CI verde num push na `main` e gera as imagens do commit testado, não da `main` do momento. Elas vão para o GHCR:

| Imagem | Tag |
|---|---|
| `ghcr.io/yurythx/projeto-nova-v2/api` | `sha-<commit>` e `main` |
| `ghcr.io/yurythx/projeto-nova-v2/worker` | `sha-<commit>` e `main` |
| `ghcr.io/yurythx/projeto-nova-v2/frontend` | `<ambiente>-sha-<commit>` e `<ambiente>` |

O frontend embute as URLs públicas no build (`NEXT_PUBLIC_*`), então sai uma imagem por ambiente (`producao`, `homologacao`). As URLs de cada uma ficam fixadas na matriz do workflow, ao lado do `app_domain` do Ansible: uma variável ausente nunca leva as URLs de produção para a homologação. API e worker são os mesmos em todos os ambientes.

Cada imagem leva proveniência (SLSA, `mode=max`), SBOM e assinatura keyless do cosign, feita com o OIDC do GitHub, sem chave guardada.

No servidor, `app_image_tag` do Ansible grava `NOVA_REGISTRY`, `NOVA_IMAGE_TAG` e `NOVA_FRONTEND_TAG` no `.env`. Com elas, `scripts/deploy.sh` faz `docker compose pull` e sobe com `--no-build`. Sem elas, compila no servidor como antes, o que vale para o primeiro deploy, antes de existir release, e para laboratório.

Produção instala sempre uma tag `sha-`. As tags móveis (`main`, `producao`) servem para olhar, não para instalar.

## Consequências

A versão em produção é um commit que passou por testes, análise de segurança e build. Rollback é trocar `app_image_tag` pela tag anterior e rodar `make infra-deploy`; nada compila. As migrations continuam indo só para a frente: voltar uma versão que já migrou o banco pede uma migration de correção, não um `down`.

O repositório é público, e as imagens também; a VM baixa sem credencial. Se o repositório fechar, a VM precisa de `docker login ghcr.io` com um token só de leitura.

Uma imagem de frontend com o domínio errado sobe sem erro e não fala com a API. O ambiente do Ansible (`app_frontend_ambiente`) e o domínio (`app_domain`) precisam bater com a matriz do workflow `Release`.
