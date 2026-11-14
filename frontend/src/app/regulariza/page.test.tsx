import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { mockBackend, renderApp } from "@/test/backend";

import { Simulador } from "./Simulador";

describe("Simulador do Regulariza", () => {
  it("consulta a inscrição sem abrir processo", async () => {
    const user = userEvent.setup();
    mockBackend({
      "POST v1/fazenda/simulador": { data: { devido: 120000, remitido: 10, itbi: 500 } },
    });
    renderApp(<Simulador />);
    await user.type(screen.getByLabelText("Inscrição"), "10");
    await user.click(screen.getByRole("button", { name: "Simular" }));
    expect(await screen.findByText(/1\.200,00/)).toBeInTheDocument();
    expect(screen.getByText(/5,00/)).toBeInTheDocument();
    expect(screen.getByText(/0,10/)).toBeInTheDocument();
  });
});
