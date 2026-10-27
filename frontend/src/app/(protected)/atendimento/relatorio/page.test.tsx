import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { LinhaSecretaria, MedidaXLA } from "@/lib/atendimento/atendimento";
import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import RelatorioPage from "./page";

const atende = {
  permissions: ["atendimento:atender"],
  scopes: [
    {
      perfil: "p",
      origem: "manual" as const,
      unidade_id: "u1",
      permissions: ["atendimento:atender"],
    },
  ],
};

const m = (x: Partial<MedidaXLA> = {}): MedidaXLA => ({
  valor: null,
  meta: null,
  atingida: null,
  amostra: 0,
  maior_melhor: false,
  ...x,
});

const linha = (x: Partial<LinhaSecretaria>): LinhaSecretaria => ({
  nome: "Saúde",
  total: 0,
  abertas: 0,
  concluidas: 0,
  canceladas: 0,
  mttr_horas: null,
  sla_percentual: null,
  fcr_percentual: null,
  experiencia: {
    csat: m(),
    facilidade: m(),
    reabertura: m(),
    idas_e_voltas: m(),
    espera_horas: m(),
  },
  ...x,
});

const relatorio = {
  desde: "2026-07-29T00:00:00Z",
  dias: 90,
  truncado: true,
  total: linha({ nome: "Total", total: 12, concluidas: 9, abertas: 3 }),
  secretarias: [
    linha({
      unidade_id: "s1",
      nome: "Saúde",
      sigla: "SMS",
      total: 10,
      concluidas: 8,
      abertas: 2,
      sla_percentual: 87.5,
      experiencia: {
        csat: m({ valor: 4.5, meta: 4, atingida: true, amostra: 6, maior_melhor: true }),
        facilidade: m(),
        reabertura: m({ valor: 12.5, meta: 10, atingida: false, amostra: 8 }),
        idas_e_voltas: m(),
        espera_horas: m({ valor: 30, amostra: 8 }),
      },
    }),
    linha({ nome: "Institucional", total: 2 }),
  ],
  equipes: [
    {
      equipe_id: "e1",
      nome: "Rede",
      ola_horas: 24,
      concluidas: 4,
      no_prazo: 3,
      no_prazo_percentual: 75,
      mediana_horas: 10,
      pendentes: 1,
      atrasadas: 1,
      chamados: 6,
      chamados_no_prazo: 5,
      chamados_no_prazo_percentual: 83.3,
      chamados_mediana_horas: 3.5,
      com_a_equipe: 2,
      chamados_atrasados: 1,
    },
    {
      equipe_id: "e2",
      nome: "Patrimônio",
      ola_horas: null,
      concluidas: 0,
      no_prazo: 0,
      no_prazo_percentual: null,
      mediana_horas: null,
      pendentes: 2,
      atrasadas: 0,
      chamados: 0,
      chamados_no_prazo: 0,
      chamados_no_prazo_percentual: null,
      chamados_mediana_horas: null,
      com_a_equipe: 0,
      chamados_atrasados: 0,
    },
  ],
  contratos: [
    {
      contrato_id: "k1",
      rotulo: "12/2026 — Link (Provedor)",
      prazo_horas: 8,
      pedidos: 3,
      medidos: 2,
      no_prazo: 1,
      no_prazo_percentual: 50,
      mediana_horas: 9.5,
      com_fornecedor: 1,
      atrasados: 1,
    },
    {
      contrato_id: "k2",
      rotulo: "Energia (Concessionária)",
      prazo_horas: null,
      pedidos: 1,
      medidos: 0,
      no_prazo: 0,
      no_prazo_percentual: null,
      mediana_horas: null,
      com_fornecedor: 1,
      atrasados: 0,
    },
  ],
};

afterEach(() => vi.restoreAllMocks());

describe("Relatório por secretaria", () => {
  it("mostra as secretarias com o total, as metas e o OLA das equipes", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(atende),
      "GET v1/atendimento/relatorio/secretarias*": { data: relatorio },
    });
    renderApp(<RelatorioPage />);
    const saude = await screen.findByRole("row", { name: /Saúde \(SMS\)/ });
    expect(within(saude).getByText("87,5%")).toBeInTheDocument();
    expect(within(saude).getByText("4,5 ✓")).toBeInTheDocument();
    expect(within(saude).getByText("12,5% ✗")).toBeInTheDocument();
    expect(within(saude).getByText("30 h")).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /^Total/ })).toHaveTextContent("12");
    expect(screen.getByText(/entram os pedidos mais recentes/)).toBeInTheDocument();
    const rede = screen.getByRole("row", { name: /^Rede/ });
    expect(within(rede).getByText("24 h")).toBeInTheDocument();
    expect(within(rede).getByText("75%")).toBeInTheDocument();
    expect(within(rede).getByText("83,3%")).toBeInTheDocument();
    expect(within(rede).getByText("3,5 h")).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /^Patrimônio/ })).toHaveTextContent("sem OLA");
    const link = screen.getByRole("row", { name: /Link \(Provedor\)/ });
    expect(within(link).getByText("9,5 h")).toBeInTheDocument();
    expect(within(link).getByText("8 h")).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /Energia/ })).toHaveTextContent("sem prazo");

    await user.selectOptions(screen.getByLabelText("Período"), "30");
    await waitFor(() =>
      expect(
        api
          .to("GET v1/atendimento/relatorio/secretarias")
          .some((c) => c.query.get("dias") === "30"),
      ).toBe(true),
    );
  });

  it("baixa o CSV e imprime", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(atende),
      "GET v1/atendimento/relatorio/secretarias*": { data: relatorio },
    });
    const criar = vi.fn(() => "blob:x");
    const soltar = vi.fn();
    Object.assign(URL, { createObjectURL: criar, revokeObjectURL: soltar });
    const imprimir = vi.spyOn(window, "print").mockImplementation(() => {});
    const clique = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    renderApp(<RelatorioPage />);
    await user.click(await screen.findByRole("button", { name: /Baixar CSV/ }));
    expect(criar).toHaveBeenCalled();
    expect(clique).toHaveBeenCalled();
    expect(soltar).toHaveBeenCalledWith("blob:x");
    const blob = (criar.mock.calls[0] as unknown as [Blob])[0];
    const texto = await blob.text();
    expect(texto).toContain('"Saúde (SMS)"');
    expect(texto).toContain('"Rede";"24 h"');
    expect(texto).toContain('"12/2026 — Link (Provedor)";"8 h"');
    await user.click(screen.getByRole("button", { name: /Imprimir/ }));
    expect(imprimir).toHaveBeenCalled();
  });

  it("sem pedidos no período, e falha com nova tentativa", async () => {
    mockBackend({
      ...identityRoutes(atende),
      "GET v1/atendimento/relatorio/secretarias*": {
        data: { ...relatorio, truncado: false, total: linha({ nome: "Total" }), secretarias: [] },
      },
    });
    const { unmount } = renderApp(<RelatorioPage />);
    expect(await screen.findByText("Nenhum pedido no período")).toBeInTheDocument();
    unmount();
    mockBackend({
      ...identityRoutes(atende),
      "GET v1/atendimento/relatorio/secretarias*": fail(500),
    });
    renderApp(<RelatorioPage />);
    expect(await screen.findByRole("button", { name: /tentar/i })).toBeInTheDocument();
  });

  it("quem não atende não vê o relatório", async () => {
    mockBackend(identityRoutes({ permissions: [], scopes: [] }));
    renderApp(<RelatorioPage />);
    expect(await screen.findByText("Seu perfil não atende")).toBeInTheDocument();
  });

  it("equipe sem tarefa no período", async () => {
    mockBackend({
      ...identityRoutes(atende),
      "GET v1/atendimento/relatorio/secretarias*": {
        data: { ...relatorio, equipes: [], contratos: [] },
      },
    });
    renderApp(<RelatorioPage />);
    expect(
      await screen.findByText(/Nenhuma equipe com chamado ou tarefa no período/),
    ).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Contratos de apoio" })).not.toBeInTheDocument();
  });
});
