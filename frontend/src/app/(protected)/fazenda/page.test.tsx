import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import FazendaPage from "./page";

describe("Protocolo da Fazenda", () => {
  it("lista o processo sem dado de débito", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({ permissions: ["fazenda:analisar"] }),
      "POST v1/fazenda/processos/alertas": { data: { publicadas: 1 } },
      "GET v1/fazenda/processos/painel": {
        data: [
          {
            nome: "relatório semestral",
            limite: "2026-07-31T23:59:59-04:00",
            situacao: "atrasado",
          },
        ],
      },
      "GET v1/fazenda/processos": {
        data: [
          {
            id: "p1",
            numero: "000001/2026",
            programa: "regulariza",
            situacao: "em_exigencia",
            titular: "ana",
            pendencia: "exigência aberta",
            recibo_em: "2026-06-01T15:00:00Z",
            exigencia_prazo: "2026-06-20T03:59:59Z",
          },
        ],
      },
    });
    renderApp(<FazendaPage />);
    expect(await screen.findByText("000001/2026")).toBeInTheDocument();
    expect(screen.getByText("exigência aberta")).toBeInTheDocument();
    expect(screen.getAllByText("Em exigência").length).toBeGreaterThan(1);
    expect(screen.getAllByText("Regulariza Rondonópolis").length).toBeGreaterThan(1);
    expect(screen.getByText(/relatório semestral/)).toBeInTheDocument();
    expect(screen.getByText(/atrasado/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Publicar alertas" }));
    expect(await screen.findByText("1 alerta(s) publicado(s) para o canal.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Abrir protocolo" })).toHaveAttribute(
      "href",
      "/fazenda/novo",
    );
    expect(screen.getByRole("link", { name: "Entregar DUA" })).toHaveAttribute(
      "href",
      "/fazenda/dua",
    );
    expect(screen.getByRole("link", { name: "Solicitar Regulariza Rondonópolis" })).toHaveAttribute(
      "href",
      "/fazenda/novo?programa=regulariza",
    );
    expect(screen.getByRole("link", { name: "Ver detalhes de Regulariza Rondonópolis" })).toHaveAttribute(
      "href",
      "/fazenda/novo?programa=regulariza&etapa=detalhe",
    );
    await user.type(screen.getByLabelText("Número do protocolo"), "999");
    expect(screen.queryByText("000001/2026")).not.toBeInTheDocument();
    expect(screen.getByText("Nenhum protocolo com esse número.")).toBeInTheDocument();
  });

  it("filtra a lista pela situação", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({ permissions: [] }),
      "GET v1/fazenda/processos": {
        data: [
          {
            id: "p1",
            numero: "000001/2026",
            programa: "regulariza",
            situacao: "em_exigencia",
            recibo_em: "2026-06-01T15:00:00Z",
          },
        ],
      },
    });
    renderApp(<FazendaPage />);
    expect(await screen.findByText("000001/2026")).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Situação"), "decidido");
    expect(screen.queryByText("000001/2026")).not.toBeInTheDocument();
    expect(screen.getByText("Nenhum protocolo nessa situação.")).toBeInTheDocument();
  });
});
