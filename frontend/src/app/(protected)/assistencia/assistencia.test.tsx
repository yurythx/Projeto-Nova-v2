import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, moduleStatus, page, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AssistenciaDashboardPage from "./page";

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

beforeEach(() => resetNavigation({}, "/assistencia"));

describe("Dashboard da Assistência", () => {
  it("renderiza métricas consolidadas de atendimento, problemas, mudanças e projetos", async () => {
    mockBackend({
      ...identityRoutes(
        perms("atendimento:atender", "problemas:read", "mudancas:manage", "projetos:read"),
        [
          moduleStatus("atendimento"),
          moduleStatus("problemas"),
          moduleStatus("mudancas"),
          moduleStatus("projetos"),
        ],
      ),
      "GET v1/atendimento/fila": page([
        { id: "s1", numero: 1, estado: "aberto", tipo: "incidente" },
        { id: "s2", numero: 2, estado: "em_atendimento", tipo: "requisicao" },
        { id: "s3", numero: 3, estado: "resolvido", tipo: "incidente" },
      ]),
      "GET v1/problemas": {
        data: [
          { id: "p1", numero: 1, estado: "em_analise" },
          { id: "p2", numero: 2, estado: "erro_conhecido" },
          { id: "p3", numero: 3, estado: "resolvido" },
        ],
      },
      "GET v1/mudancas": {
        data: [
          { id: "m1", numero: 1, estado: "em_avaliacao" },
          { id: "m2", numero: 2, estado: "em_execucao" },
          { id: "m3", numero: 3, estado: "concluida" },
        ],
      },
      "GET v1/projetos": {
        data: [
          { id: "pr1", numero: 1, estado: "planejado", orcamento: 50000 },
          { id: "pr2", numero: 2, estado: "em_andamento", orcamento: 120000 },
          { id: "pr3", numero: 3, estado: "concluido", orcamento: 30000 },
        ],
      },
    });

    renderApp(<AssistenciaDashboardPage />);

    expect(await screen.findByRole("heading", { name: "Dashboard de Serviços" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Atendimento" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Problemas" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Mudanças" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Projetos" })).toBeInTheDocument();

    // Verificação de KPIs no topo
    expect(screen.getByText("Fila de Atendimento")).toBeInTheDocument();
    expect(screen.getByText("Erros Conhecidos")).toBeInTheDocument();
    expect(screen.getByText("Janelas de Mudança")).toBeInTheDocument();
    expect(screen.getByText("Projetos Ativos")).toBeInTheDocument();

    // Verificação de Gráficos Visuais (SVG nativo)
    expect(screen.getByTestId("assistencia-graficos")).toBeInTheDocument();
    expect(screen.getByText("Demandas de Atendimento")).toBeInTheDocument();
    expect(screen.getByText("Matriz de Risco das Mudanças")).toBeInTheDocument();
    expect(screen.getByText("Progresso Estratégico")).toBeInTheDocument();

    // Ações de exportação, calendário, impressão e relatório executivo
    expect(screen.getByRole("link", { name: /Calendário/i })).toHaveAttribute(
      "href",
      "/assistencia/calendario",
    );
    expect(screen.getByRole("button", { name: /Relatório Executivo/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Imprimir Painel/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Exportar CSV/i })).toBeInTheDocument();

    // Abrir Relatório Executivo e verificar KPIs de Governança ITIL
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /Relatório Executivo/i }));

    expect(
      await screen.findByRole("heading", { name: "Relatório Executivo de Governança de TI" }),
    ).toBeInTheDocument();
    expect(screen.getByText("MTTR (Resolução)")).toBeInTheDocument();
    expect(screen.getByText("MTBF (Confiabilidade)")).toBeInTheDocument();
    expect(screen.getByText("Conformidade de SLA")).toBeInTheDocument();
    expect(screen.getByText("Sucesso em Mudanças")).toBeInTheDocument();
    expect(screen.getByText("Balanço Integrado de Processos e Serviços")).toBeInTheDocument();
    expect(screen.getByText("Parecer e Diretrizes de Governança")).toBeInTheDocument();

    // Fechar modal
    await user.click(screen.getByRole("button", { name: "Concluir" }));

    // Links de atalho
    expect(screen.getByRole("link", { name: /Abrir fila e quadro de atendimento/ })).toHaveAttribute(
      "href",
      "/atendimento/fila",
    );
    expect(screen.getByRole("link", { name: /Gerenciar problemas e contornos/ })).toHaveAttribute(
      "href",
      "/problemas",
    );
    expect(screen.getByRole("link", { name: /Planejar e acompanhar mudanças/ })).toHaveAttribute(
      "href",
      "/mudancas",
    );
    expect(screen.getByRole("link", { name: /Gerenciar projetos e tarefas/ })).toHaveAttribute(
      "href",
      "/projetos",
    );
  });
});
