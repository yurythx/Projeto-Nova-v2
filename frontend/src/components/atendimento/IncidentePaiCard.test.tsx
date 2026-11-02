import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { mockBackend, renderApp } from "@/test/backend";
import { IncidentePaiCard } from "./IncidentePaiCard";
import type { Solicitacao } from "@/lib/atendimento/atendimento";

const CHAMADO_PAI: Solicitacao = {
  id: "pai-1",
  numero: 100,
  tipo: "incidente",
  estado: "em_atendimento",
  titulo: "Queda Geral de Link Fibra",
  descricao: "Rompimento de fibra da operadora.",
  oferta_id: "of-1",
  oferta_nome: "Suporte de Redes",
  solicitante_id: "usr-1",
  solicitante_nome: "Maria Servidora",
  impacto: 3,
  urgencia: 3,
  created_at: "2026-11-01T10:00:00Z",
  updated_at: "2026-11-01T10:00:00Z",
};

const CHAMADO_FILHO: Solicitacao = {
  id: "filho-1",
  numero: 101,
  tipo: "incidente",
  estado: "aberto",
  titulo: "Sem internet na UBS Centro",
  descricao: "Queda na conexão.",
  oferta_id: "of-1",
  oferta_nome: "Suporte de Redes",
  solicitante_id: "usr-2",
  solicitante_nome: "João Médico",
  incidente_pai_id: "pai-1",
  impacto: 2,
  urgencia: 2,
  created_at: "2026-11-01T10:05:00Z",
  updated_at: "2026-11-01T10:05:00Z",
};

describe("IncidentePaiCard (AIOps)", () => {
  it("exibe informações do chamado pai quando o chamado atual for filho", async () => {
    mockBackend({
      "GET v1/atendimento/solicitacoes/pai-1": {
        data: { solicitacao: CHAMADO_PAI },
      },
      "GET v1/atendimento/solicitacoes/filho-1/filhos": {
        data: [],
      },
    });

    renderApp(
      <IncidentePaiCard
        solicitacao={CHAMADO_FILHO}
        podeEditar={true}
        onAtualizado={vi.fn()}
      />,
    );

    expect(await screen.findByText(/Chamado dependente vinculado ao Incidente Pai/i)).toBeInTheDocument();
    expect(screen.getByText(/#100 — Queda Geral de Link Fibra/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Desvincular/i })).toBeInTheDocument();
  });

  it("exibe lista de filhos quando o chamado for pai", async () => {
    mockBackend({
      "GET v1/atendimento/solicitacoes/pai-1/filhos": {
        data: [CHAMADO_FILHO],
      },
    });

    renderApp(
      <IncidentePaiCard
        solicitacao={CHAMADO_PAI}
        podeEditar={true}
        onAtualizado={vi.fn()}
      />,
    );

    expect(await screen.findByText(/Chamados filhos correlatos \(1\)/i)).toBeInTheDocument();
    expect(screen.getByText("#101")).toBeInTheDocument();
    expect(screen.getByText("Sem internet na UBS Centro")).toBeInTheDocument();
    expect(screen.getByText("João Médico")).toBeInTheDocument();
  });

  it("permite vincular novo chamado filho por número", async () => {
    let payloadVinculo: any = null;

    mockBackend({
      "GET v1/atendimento/solicitacoes/pai-1/filhos": {
        data: [],
      },
      "GET v1/atendimento/fila": {
        data: {
          items: [CHAMADO_FILHO],
        },
      },
      "PUT v1/atendimento/solicitacoes/filho-1/incidente-pai": (req) => {
        payloadVinculo = req.body;
        return {
          status: 200,
          data: { solicitacao: { ...CHAMADO_FILHO, incidente_pai_id: "pai-1" } },
        };
      },
    });

    const user = userEvent.setup();
    renderApp(
      <IncidentePaiCard
        solicitacao={CHAMADO_PAI}
        podeEditar={true}
        onAtualizado={vi.fn()}
      />,
    );

    const btnAbrirModal = await screen.findByRole("button", { name: /Vincular chamado filho/i });
    await user.click(btnAbrirModal);

    expect(screen.getByText("Vincular Chamado Filho (AIOps)")).toBeInTheDocument();

    const inputNumero = screen.getByPlaceholderText(/Ex.: 601 ou 00000000/i);
    await user.type(inputNumero, "101");

    const btnSalvar = screen.getByRole("button", { name: "Vincular Chamado" });
    await user.click(btnSalvar);

    await waitFor(() => {
      expect(payloadVinculo).not.toBeNull();
      expect(payloadVinculo.incidente_pai_id).toBe("pai-1");
    });
  });
});

