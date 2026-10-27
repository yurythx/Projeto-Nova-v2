# Handover: gestão de serviços (Projeto Nova v2)

- **Atualizado em:** 27/10/2026
- **Branch:** `feat/gestao-de-servicos` (publicada; PR para a `main` pendente)
- **Para quem:** quem continuar o trabalho, pessoa ou IA. O que já está decidido está nos ADRs; aqui fica o mapa e o que falta.

## 1. O que a branch entrega

A branch leva o atendimento do Projeto Nova ao padrão do ITIL 4 e 5. As entregas e o estado de cada uma estão em [ROADMAP_GESTAO_DE_SERVICOS.md](ROADMAP_GESTAO_DE_SERVICOS.md). A última leva (Leva 4) trouxe:

| Entrega | ADR | Onde mexer |
|---|---|---|
| IA assistiva para quem atende (classificação e resumo) | [065](adr/065-ia-assistiva-no-atendimento.md) | `atendimento/application/ia.go`, `app/ia_porta.go` |
| Incidente grave e página de status pública | [066](adr/066-ocorrencias-e-pagina-de-status.md) | plug-in `modules/ocorrencias`, `frontend/src/app/status` |
| Alertas do monitoramento viram incidente | [067](adr/067-eventos-do-monitoramento.md) | `atendimento/application/eventos.go`, `deploy/observability/alertmanager.yml` |
| Aprovação em etapas e tarefas por equipe (admissão, desligamento) | [068](adr/068-aprovacao-em-etapas-e-tarefas.md) | `atendimento/*/aprovacao.go`, `atendimento/*/tarefa.go`, tela `/atendimento/aprovacoes` |
| Experiência de quem pede (XLA) com metas | [069](adr/069-experiencia-de-quem-pede-xla.md) | `atendimento/domain/experiencia.go`, `ExperienciaCard.tsx` |
| Relatório por secretaria e OLA das equipes | [070](adr/070-relatorio-por-secretaria-e-ola.md) | `atendimento/*/relatorio.go`, tela `/atendimento/relatorio` |
| Contrato de apoio (prazo do fornecedor) | [071](adr/071-contrato-de-apoio.md) | `atendimento/*/contrato_apoio.go`, `app/contrato_porta.go`, plug-in `contratos` |
| Importador do GLPI | [072](adr/072-importador-do-glpi.md) | `atendimento/glpi`, `cmd/importarglpi`, [DEPLOY.md](DEPLOY.md#migrar-do-glpi) |
| OLA no chamado (histórico de atribuição) | [073](adr/073-ola-no-chamado.md) | `atendimento/*/atribuicao.go`, `domain/relatorio.go` |
| Perfis pelos grupos do SSO | [074](adr/074-perfis-pelos-grupos-do-sso.md) | `platform/iam/resolver.go`, `iam_mapeamentos` no Ansible |
| Tabela de-para do GLPI | [075](adr/075-de-para-do-glpi.md) | `atendimento/glpi/mapa.go` |
| Módulo de integrações e conector | [076](adr/076-modulo-de-integracoes.md) | plug-in `modules/integracoes`, `app/integracoes_porta.go`, `cmd/conector`, tela `/integracoes` |

A infraestrutura de produção (Proxmox, Terraform, Ansible, domínio `projetonova.cloud`, backup e monitoramento) está em [DEPLOY.md](DEPLOY.md). As imagens do CI estão no [ADR 063](adr/063-imagens-do-ci-no-registro.md), e a catraca de cobertura no [ADR 064](adr/064-catraca-de-cobertura.md).

## 2. Como o código está organizado

- **Backend** (`backend/`, Go): microkernel. Cada plug-in fica em `internal/modules/<nome>` com as camadas `domain`, `application`, `infrastructure` e `transport`. Um plug-in não importa outro: as portas entre eles ficam em `internal/app` (exemplos: `cofre_porta.go`, `contrato_porta.go`). Toda gravação de negócio leva evento no outbox e auditoria na mesma transação.
- **Migrações:** `backend/migrations/0000NN_*.sql` (goose). A última é a 000029.
- **Contrato da API:** `backend/docs/openapi.json` é a fonte, e `docs/openapi.yaml` é gerado. Rota nova sem documentação quebra o teste `TestOpenAPIMatchesRouter`. Para regenerar, rode com `UPDATE_OPENAPI=1` e escreva à mão o `summary` e a `description` da rota nova.
- **Frontend** (`frontend/`, Next.js 16 e React 19): as telas ficam em `src/app/(protected)`, os componentes em `src/components`, e os tipos da API em `src/lib/<plug-in>`.

## 3. Como verificar

Backend: a suíte completa precisa de Postgres, RabbitMQ, MinIO e Typesense. É o que o CI roda (`.github/workflows/ci.yml`).

```bash
cd backend
gofmt -l ./internal ./pkg ./cmd     # nada a listar
go vet ./...
go test -race -p 1 -count=1 -coverpkg=./internal/...,./pkg/... -coverprofile=coverage.out ./...
cd .. && bash scripts/coverage-gate.sh backend/coverage.out
```

O portão de cobertura ([ADR 064](adr/064-catraca-de-cobertura.md)) exige 100% de módulo novo e não deixa os outros caírem abaixo do piso em `scripts/coverage-floor.txt`. Quando a cobertura sobe, ele sugere subir o piso.

Frontend:

```bash
cd frontend
npx tsc --noEmit && npx eslint --max-warnings 0 . && npx prettier --check . && npx vitest run
```

Na última execução, tudo passou: a suíte completa do backend com o portão, e 684 testes no frontend.

## 4. O que falta

Depende de quem opera:

1. **Produção no ar** (VM 240, `projetonova.cloud`): falta o destino do backup fora da VM (rclone e age) e o bucket R2 para o estado do Terraform. Ver [DEPLOY.md](DEPLOY.md).
2. **Migração do GLPI** pela tela de Integrações, com o conector na rede da prefeitura. A simulação real (14.994 chamados) passou com a tabela de-para; antes da carga: limpar a base de teste, cadastrar a estrutura real e trazer os usuários do SSO ([ADR 077](adr/077-usuarios-do-sso.md), que pede a conta de serviço no Keycloak).
3. **PR para a `main`**, proteção do branch e Renovate.

Melhorias registradas nos ADRs, sem pressa:

- Prazo do fornecedor em horas úteis. Hoje é corrido ([ADR 071](adr/071-contrato-de-apoio.md)).
- Anexos do GLPI na importação ([ADR 072](adr/072-importador-do-glpi.md)).
