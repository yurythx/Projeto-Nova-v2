import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import DuaPage from "./page";

describe("Entrega da DUA", () => {
  it("mostra o recibo sem repetir o documento", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({ permissions: ["fazenda:protocolar"] }),
      "POST v1/fazenda/processos/dua": {
        data: {
          recibo: "DUA: entrega recebida. Acompanhe na área restrita.",
          multa: 20000,
          linhas: 1,
          falhas: [{ linha: 2, motivo: "a situação é Quitado ou Em curso" }],
        },
      },
    });
    renderApp(<DuaPage />);
    await user.type(
      screen.getByLabelText("Planilha"),
      "10;20;Ana;00000000000;Rua 1;01/02/2024;Em curso;",
    );
    await user.type(screen.getByLabelText("Inscrição"), "10");
    await user.click(screen.getByRole("button", { name: "Entregar" }));
    expect(await screen.findByText(/entrega recebida/)).toBeInTheDocument();
    expect(screen.getByText(/200,00/)).toBeInTheDocument();
    expect(screen.getByText(/Linha 2/)).toBeInTheDocument();
    expect(screen.queryByText(/123/)).not.toBeInTheDocument();
  });
});
