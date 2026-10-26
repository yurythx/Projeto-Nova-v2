import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, moduleStatus, page, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AssistenciaCalendarioPage from "./page";

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

beforeEach(() => resetNavigation({}, "/assistencia/calendario"));

describe("Calendário Unificado da Assistência", () => {
  it("renderiza eventos de mudanças, projetos e SLA e detecta conflitos operacionais", async () => {
    const user = userEvent.setup();
    const hoje = new Date();
    const isoHoje = hoje.toISOString();

    mockBackend({
      ...identityRoutes(perms("atendimento:atender", "mudancas:manage", "projetos:read"), [
        moduleStatus("atendimento"),
        moduleStatus("mudancas"),
        moduleStatus("projetos"),
      ]),
      "GET v1/mudancas": {
        data: [
          {
            id: "m1",
            numero: 401,
            titulo: "Migração do Switch Core",
            tipo: "normal",
            risco: "alto",
            estado: "aprovada",
            inicio_planejado: isoHoje,
            fim_planejado: isoHoje,
            solicitante_nome: "Admin TI",
            descricao: "Atualização de firmware e substituição de hardware",
          },
        ],
      },
      "GET v1/projetos": {
        data: [
          {
            id: "pr1",
            numero: 501,
            titulo: "Entrega Fase 1 Data Center",
            estado: "em_andamento",
            prioridade: "alta",
            data_fim_planejada: isoHoje,
            gerente_nome: "Mariana Engenheira",
            orcamento: 80000,
          },
        ],
      },
      "GET v1/atendimento/fila": page([
        {
          id: "s1",
          numero: 601,
          titulo: "Queda de Link Fibra",
          tipo: "incidente",
          estado: "aberto",
          oferta_nome: "Suporte de Redes",
          solicitante_nome: "Secretaria de Saúde",
          created_at: isoHoje,
          prazo: {
            solucao: {
              vence_em: isoHoje,
              estado: "correndo",
              meta_min: 120,
              usado_min: 30,
              percentual: 25,
              situacao: "no_prazo",
            },
          },
        },
      ]),
    });

    renderApp(<AssistenciaCalendarioPage />);

    // Cabeçalho e estrutura
    expect(
      await screen.findByRole("heading", { name: "Calendário Operacional de Serviços" }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("assistencia-calendario")).toBeInTheDocument();

    // Como m1 tem risco alto e coincide na data de pr1 e s1, um alerta de sobreposição deve surgir
    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("Alerta de Sobreposição Operacional Detectado")).toBeInTheDocument();

    // Eventos visíveis na grade
    expect(screen.getByText("M#401 Migração do Switch Core")).toBeInTheDocument();
    expect(screen.getByText("PRJ#501 Entrega Fase 1 Data Center")).toBeInTheDocument();
    expect(screen.getByText("SLA #601 Queda de Link Fibra")).toBeInTheDocument();

    // Abre detalhes ao clicar no evento
    await user.click(screen.getByText("M#401 Migração do Switch Core"));
    expect(
      await screen.findByRole("heading", { name: "M#401 Migração do Switch Core" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Aviso de Sobreposição")).toBeInTheDocument();
    expect(screen.getByText(/Atualização de firmware/)).toBeInTheDocument();

    // Fecha modal de detalhes
    await user.click(screen.getByRole("button", { name: "Fechar" }));
    await waitFor(() => {
      expect(
        screen.queryByRole("heading", { name: "M#401 Migração do Switch Core" }),
      ).not.toBeInTheDocument();
    });

    // Alterna para visualização em Lista
    await user.click(screen.getByRole("button", { name: "Ver como lista" }));
    expect(screen.getByRole("heading", { name: /Eventos e Prazos do Mês/ })).toBeInTheDocument();
    expect(screen.getByText("Mudança CAB")).toBeInTheDocument();
    expect(screen.getByText("Projeto")).toBeInTheDocument();
    expect(screen.getByText("Prazo SLA")).toBeInTheDocument();

    // Desativa o filtro de Projetos
    await user.click(screen.getByRole("button", { name: /Marcos de Projetos/i }));
    expect(screen.queryByText("PRJ#501 Entrega Fase 1 Data Center")).not.toBeInTheDocument();
    expect(screen.getByText("M#401 Migração do Switch Core")).toBeInTheDocument();
  });
});
