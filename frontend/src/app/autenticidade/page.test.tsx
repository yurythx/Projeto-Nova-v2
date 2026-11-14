import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { mockBackend, renderApp } from "@/test/backend";

vi.mock("qrcode", () => ({
  default: { toString: async () => "<svg xmlns='http://www.w3.org/2000/svg'></svg>" },
}));

import { Consulta } from "./Consulta";

describe("Autenticidade", () => {
  it("confirma o código e desenha o QR no navegador", async () => {
    const user = userEvent.setup();
    mockBackend({
      "GET v1/fazenda/autenticidade/abc": {
        data: {
          codigo: "abc",
          numero: "000001/2026",
          valido: true,
          mensagem: "documento confirmado",
        },
      },
    });
    renderApp(<Consulta />);
    await user.type(screen.getByLabelText("Código"), "abc");
    await user.click(screen.getByRole("button", { name: "Confirmar" }));
    expect(await screen.findByText("Documento autêntico")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "QR do código abc" })).toBeInTheDocument();
  });

  it("não desenha o QR quando o código não confere", async () => {
    const user = userEvent.setup();
    mockBackend({
      "GET v1/fazenda/autenticidade/errado": {
        data: { codigo: "errado", numero: "", valido: false, mensagem: "código não confere" },
      },
    });
    renderApp(<Consulta />);
    await user.type(screen.getByLabelText("Código"), "errado");
    await user.click(screen.getByRole("button", { name: "Confirmar" }));
    expect(await screen.findByText("Código não confirmado")).toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });
});
