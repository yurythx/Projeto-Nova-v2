import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import ProcessoPage from "./page";

describe("Processo da Fazenda", () => {
  beforeEach(() => resetNavigation({ id: "p1" }, "/fazenda/p1"));

  it("mostra a pendência e registra a resposta da exigência", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({ permissions: [] }),
      "GET v1/fazenda/processos/p1": {
        data: {
          id: "p1",
          numero: "000001/2026",
          programa: "regulariza",
          situacao: "em_exigencia",
          pendencia: "exigência aberta",
          recibo_em: "2026-06-01T15:00:00Z",
          codigo: "000001202620260601110000",
          exigencia_aberta: "2026-06-10T15:00:00Z",
          exigencia_prazo: "2026-06-20T03:59:59Z",
          ciencia_recibo: "2026-06-11T15:00:00Z",
        },
      },
      "POST v1/fazenda/processos/p1/resposta": {
        data: { id: "p1", situacao: "em_analise" },
      },
    });
    renderApp(<ProcessoPage />);
    expect(await screen.findByText("exigência aberta")).toBeInTheDocument();
    expect(screen.getByText("Regulariza Rondonópolis")).toBeInTheDocument();
    expect(screen.getByText("Em exigência")).toBeInTheDocument();
    expect(screen.getByText("Protocolo registrado")).toBeInTheDocument();
    expect(screen.getByText("Exigência aberta")).toBeInTheDocument();
    expect(screen.getByText("Prazo 19/06/2026, 23:59")).toBeInTheDocument();
    expect(screen.getByText("Ciência do recibo em 11/06/2026, 11:00")).toBeInTheDocument();
    expect(screen.getByText("000001202620260601110000")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Confirmar autenticidade" })).toHaveAttribute(
      "href",
      "/autenticidade",
    );
    await user.click(screen.getByRole("checkbox", { name: "A exigência foi respondida" }));
    await user.click(screen.getByRole("button", { name: "Registrar resposta" }));
    expect(await screen.findByText("exigência aberta")).toBeInTheDocument();
  });

  it("a equipe abre a exigência no processo protocolado", async () => {
    const user = userEvent.setup();
    const backend = mockBackend({
      ...identityRoutes({ permissions: ["fazenda:analisar"] }),
      "GET v1/fazenda/processos/p1": {
        data: {
          id: "p1",
          numero: "000001/2026",
          programa: "avanca",
          situacao: "protocolado",
          recibo_em: "2026-11-14T01:03:00Z",
        },
      },
      "POST v1/fazenda/processos/p1/exigencia": { data: { id: "p1", situacao: "em_exigencia" } },
    });
    renderApp(<ProcessoPage />);
    await user.click(await screen.findByRole("button", { name: "Abrir exigência" }));
    expect(backend.to("POST v1/fazenda/processos/p1/exigencia")[0]?.body).toEqual({
      fato_novo: false,
    });
  });

  it("calcula o parcelamento em centavos e não chama o débito de parcelado", async () => {
    const user = userEvent.setup();
    const backend = mockBackend({
      ...identityRoutes({ permissions: ["fazenda:analisar"] }),
      "GET v1/fazenda/processos/p1": {
        data: {
          id: "p1",
          numero: "000001/2026",
          programa: "regulariza",
          situacao: "protocolado",
          recibo_em: "2026-11-14T01:03:00Z",
        },
      },
      "POST v1/fazenda/processos/p1/parcelamento": { data: { parcelas: 2, valor: 60000 } },
    });
    renderApp(<ProcessoPage />);
    await user.type(await screen.findByLabelText("Valor do débito"), "1.200,00");
    await user.type(screen.getByLabelText("Parcelas"), "2");
    await user.click(screen.getByRole("button", { name: "Calcular parcelamento" }));
    expect(backend.to("POST v1/fazenda/processos/p1/parcelamento")[0]?.body).toEqual({
      total: 120000,
      parcelas: 2,
    });
  });
});
