# 📋 Documento de Handover: Gestão de Serviços TIC (Projeto Nova v2)

> **Data de Registro:** 23 de Outubro de 2026  
> **Branch Git Ativa:** `feat/gestao-de-servicos`  
> **Status do Repositório:** Árvore limpa (`working tree clean`), sem arquivos não versionados ou modificações pendentes  
> **Suíte de Testes do Frontend:** **108/108 arquivos** e **591/591 testes aprovados** (100% verde)  
> **Next.js Production Build:** Compilação bem-sucedida em todas as 59 rotas estáticas e dinâmicas  
> **Ambiente Docker:** 8 containers operacionais e saudáveis (`backend-api:8012`, `frontend:3012`, `postgres`, `rabbitmq`, `redis`, `typesense`, `minio`)

---

## 1. Contexto e Objetivo da Demanda

O usuário solicitou uma grande reformulação e expansão na área de **Gestão de Serviços TIC (ITSM / ITIL v4)** do **Projeto Nova v2**, alinhando a experiência ao padrão consolidado de ferramentas como o **GLPI**:

1. **Reestruturação do Menu:** Agrupar **Atendimento**, **Problemas**, **Mudanças** e o novo módulo de **Projetos** sob o grupo pai **"Assistência"**.
2. **Padronização de Visualizações:** Padronizar os módulos de **Problemas** e **Mudanças** com alternância idêntica entre **Lista** e **Quadro Kanban** (com drag-and-drop e diálogos de transição de estado).
3. **Módulo de Projetos:** Criação do módulo fim-a-fim (banco de dados, Go backend, Next.js frontend, metas, tarefas, vínculos com Mudanças/Problemas/Ativos/Incidentes, timesheet de apontamento de horas e cronograma Gantt).
4. **Central de Assistência Unificada (`/assistencia`):** Dashboard central com KPIs consolidados, gráficos em SVG nativo, calendário de janelas operacionais com detecção de conflitos e Relatório Executivo de Governança TIC (MTTR, MTBF, conformidade SLA).
5. **Automações e Recursos ITIL:**
   - Automações em cascata no encerramento de RFCs no CAB (resolvendo incidentes e problemas associados).
   - Vínculo de ativos do CMDB no chamado (`/ativos`).
   - Deflexão de chamados inteligente com busca prévia na Base de Conhecimento (`/wiki`).
   - Calculadora de Matriz de Risco do CAB (Impacto x Complexidade x Rollback com mapa de calor 3x3).
   - Pesquisa de Satisfação pós-atendimento (CSAT) de 1 a 5 estrelas com comentário.

---

## 2. Histórico de Commits na Branch `feat/gestao-de-servicos`

```text
* 1b4e254 feat(itil): calculadora de matriz de risco do cab e pesquisa de satisfacao csat
* 5d5578b feat(projetos): diagrama de gantt interativo com marcos, escala temporal e progresso no detalhe do projeto
* 14945f6 feat(itil): automacoes em cascata, vinculo cmdb, campos complementares e relatorio executivo
* 97c514e feat(assistencia): central de assistencia, projetos, kanban, timesheet e deflexao de chamados
```

---

## 3. Módulos e Arquitetura de Componentes

### A. Menu Lateral & Navegação
- **Arquivo:** `frontend/src/components/layout/Sidebar.tsx`
- **Detalhes:** Submenu "Assistência" com rotas retráteis:
  - Central de Assistência: `/assistencia`
  - Atendimento: `/atendimento`
  - Problemas: `/problemas`
  - Mudanças: `/mudancas`
  - Projetos: `/projetos`

### B. Gestão de Problemas (ITIL)
- **Arquivo:** `frontend/src/app/(protected)/problemas/page.tsx`
- **Detalhes:** Alternância entre Lista e Kanban por estado (`registrado`, `em_analise`, `erro_conhecido`, `resolvido`, `encerrado`). Card de Solução de Contorno e Erro Conhecido destacados com links para incidentes relacionados.

### C. Gestão de Mudanças & CAB (ITIL)
- **Listagem e Kanban:** `frontend/src/app/(protected)/mudancas/page.tsx`
- **Detalhe da Mudança:** `frontend/src/app/(protected)/mudancas/[id]/page.tsx`
- **Calculadora de Risco:** `frontend/src/components/servicos/MatrizRiscoCalculadora.tsx`
- **Detalhes:** Formulário de avaliação do CAB, transições com motivo/rollback, cálculo do score de risco ponderado e automação em cascata (ao concluir a RFC, permite resolver em lote os incidentes e problemas vinculados).

### D. Módulo de Projetos
- **Tipos & Schemas:** `frontend/src/lib/projetos/projetos.ts`
- **Backend Go:**
  - Migração SQL: `backend/internal/database/migrations/000002_projetos.sql`
  - Domínio e Repositório: `backend/internal/domain/projetos/...`
  - Rotas e Handlers: `backend/internal/interfaces/http/handlers/projetos_handler.go`
- **Listagem & Kanban:** `frontend/src/app/(protected)/projetos/page.tsx`
- **Página de Detalhes:** `frontend/src/app/(protected)/projetos/[id]/page.tsx`
- **Gantt Chart:** `frontend/src/components/projetos/ProjetoGanttChart.tsx`
  - Linha do tempo dinâmica, linha vertical "Hoje", marcação de marcos vinculados ao CAB, atrasos em vermelho e filtros de estado.
- **Timesheet:** `frontend/src/components/atendimento/ApontamentosCard.tsx`
  - Registro de horas trabalhadas por técnico/desenvolvedor e queima de orçamento.

### E. Central de Assistência & Governança de TI
- **Dashboard:** `frontend/src/app/(protected)/assistencia/page.tsx`
- **Gráficos SVG:** `frontend/src/components/servicos/AssistenciaGraficos.tsx`
- **Calendário Operacional:** `frontend/src/app/(protected)/assistencia/calendario/page.tsx`
- **Relatório Executivo:** `frontend/src/components/servicos/RelatorioExecutivoModal.tsx`
  - Métricas de MTTR, MTBF, conformidade de SLA e taxa de sucesso do CAB com exportação CSV e CSS `@media print`.

### F. Atendimento & CSAT
- **Vínculo CMDB:** `frontend/src/components/atendimento/AtivoVinculadoCard.tsx`
- **Deflexão Wiki:** `frontend/src/components/atendimento/DeflexaoConhecimento.tsx`
- **Pesquisa de Satisfação (CSAT):** `frontend/src/components/atendimento/AvaliacaoAtendimentoCard.tsx`
  - Coleta estrelas (1 a 5) e comentário após chamado resolvido/encerrado.

---

## 4. Como Executar os Testes e Comprovar a Estabilidade

```powershell
# 1. Executar todos os testes do Frontend (108 arquivos, 591 testes):
cd frontend
npm test

# 2. Executar testes específicos dos novos componentes:
npm test "src/components/servicos/MatrizRiscoCalculadora.test.tsx"
npm test "src/components/atendimento/AvaliacaoAtendimentoCard.test.tsx"
npm test "src/components/projetos/ProjetoGanttChart.test.tsx"

# 3. Validar build do Next.js (59 rotas):
npm run build

# 4. Validar os containers Docker:
cd ..
docker compose ps
```

---

## 5. Próximos Passos Sugeridos para a Próxima IA

1. **Alertas Proativos de Risco de Estouro de SLA:**
   - Adicionar checagem ou job que emita notificações via `RealtimeProvider` e `NotificationBell` para chamados com menos de 2 horas para vencer o SLA.
2. **Dashboard de CSAT Agregado na Central de Assistência:**
   - Exibir na Central de Assistência (`/assistencia`) a nota média geral de satisfação e as avaliações recentes coletadas pelo `AvaliacaoAtendimentoCard`.
3. **Exportação Formal de RFC do CAB em PDF:**
   - Adicionar na tela de Mudanças (`/mudancas/[id]`) um botão para emitir documento formal da RFC pronto para impressão/assinatura digital do comitê com parecer técnico, matriz de risco e plano de contingência.
4. **Merge da branch `feat/gestao-de-servicos` na `main`:**
   - Com todos os 591 testes passando e containers rodando saudáveis, realizar o merge quando solicitado pelo usuário.
