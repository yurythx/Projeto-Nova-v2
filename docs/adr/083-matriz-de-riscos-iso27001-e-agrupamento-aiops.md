# ADR 083: Matriz de Riscos de TI conforme a ISO/IEC 27001 (GRC) e Agrupamento AIOps de Incidentes

## Status
Aceito e Implementado

## Contexto
Grandes organizações e prefeituras operam ambientes críticos de TI com milhares de servidores, links de fibra, prontuários de saúde (SMS), plataformas de educação (SEMECEL) e sistemas tributários/financeiros da Prefeitura.

Para atender aos mais altos padrões de conformidade corporativa e governamental:
1. **Norma ISO/IEC 27001 e ISO/IEC 27005 (GRC)**: Exigem a identificação formal, avaliação de severidade (Probabilidade × Impacto), amarração com os controles do Anexo A (A.5 a A.18), definição de estratégias de mitigação e acompanhamento contínuo de riscos técnicos e de segurança da informação.
2. **AIOps & Agrupamento de Incidentes (ITIL v5)**: Incidentes massivos (ex.: rompimento de cabo de fibra ou pane elétrica no Data Center) geram dezenas de chamados correlatos em poucos minutos. A operação necessita da capacidade de agrupar chamados sob um "Incidente Pai", evitando retrabalho e permitindo a resolução sincronizada de todos os chamados dependentes.

## Decisão de Arquitetura

### 1. Módulo Independente de Riscos (`internal/modules/riscos`)
- Criado seguindo Clean Architecture e Domain-Driven Design (DDD):
  - `domain`: Entidade `Risco`, validações de severidade 1 a 25 (`NivelSeveridade`: Baixo, Médio, Alto, Crítico) e cálculo automático da matriz 5x5.
  - `application`: Casos de uso de listagem filtrada, criação, atualização com auditoria e agregação do Heatmap.
  - `infrastructure`: Persistência com PostgreSQL via `pgx/v5` e transações seguras.
  - `transport`: Handlers HTTP REST com rotas autenticadas sob `/api/v1/riscos` e `/api/v1/riscos/heatmap`.
  - Kernel: Registrado no catálogo de módulos do sistema com permissões granulares (`riscos:read`, `riscos:manage`) e ícone `shield-alert`.

### 2. Frontend com Heatmap Interativo ISO/IEC 27005
- Desenvolvido em `frontend/src/app/(protected)/riscos/page.tsx`:
  - Heatmap 5x5 interativo (Probabilidade 1-5 × Impacto 1-5) com quadrantes Verde (1-6), Amarelo (7-12), Laranja (13-19) e Vermelho (20-25).
  - Filtro por clique direto em qualquer quadrante da matriz para exibição instantânea dos riscos afetados.
  - Integração com `useEntityFilter` para isolamento de riscos entre Prefeitura, Saúde (SMS) e Educação (SEMECEL).
  - Modal de cadastro e edição com sugestões oficiais dos controles do Anexo A da ISO 27001 (A.5 a A.18).

### 3. Agrupamento de Incidentes no Atendimento (AIOps)
- Coluna `incidente_pai_id` e índice `idx_atendimento_solicitacoes_incidente_pai` adicionados na tabela `atendimento_solicitacoes` via Migration 47.
- Permite vincular chamados de mesma causa raiz, agilizando o atendimento e a gestão de crises operacionais.

## Consequências e Benefícios
- **Conformidade Legal & GRC**: Preparação total para auditorias ISO 27001, LGPD e Tribunal de Contas.
- **Redução do Tempo de Resposta (MTTR)**: Visão consolidada da criticidade dos serviços essenciais municipais.
- **Integração Completa**: O módulo de Riscos trabalha em conjunto com a Gestão de Mudanças (CAB), Problemas (Causa Raiz) e CMDB (Ativos).

