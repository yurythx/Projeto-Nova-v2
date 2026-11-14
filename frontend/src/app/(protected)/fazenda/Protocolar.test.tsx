import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { mockBackend, renderApp } from "@/test/backend";
import { router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import { Protocolar } from "./Protocolar";

describe("Protocolar", () => {
  it("abre o próprio processo e segue para o recibo", async () => {
    const user = userEvent.setup();
    mockBackend({
      "POST v1/fazenda/processos": { data: { id: "p1", numero: "000001/2026" } },
    });
    renderApp(<Protocolar programa="regulariza" />);
    await user.click(screen.getByRole("checkbox", { name: "Identificação" }));
    await user.click(screen.getByRole("button", { name: "Protocolar" }));
    expect(await screen.findByRole("button", { name: "Protocolar" })).toBeEnabled();
    expect(router.push).toHaveBeenCalledWith("/fazenda/p1");
  });

  it("troca as peças quando o programa é o Avança", async () => {
    const user = userEvent.setup();
    mockBackend({ "POST v1/fazenda/processos": { data: { id: "p1", numero: "1" } } });
    renderApp(<Protocolar programa="avanca" />);
    expect(screen.getByRole("checkbox", { name: "Documento único de regularidade" })).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Identificação" })).not.toBeInTheDocument();
    await user.selectOptions(screen.getByRole("combobox", { name: "Entrada" }), "procurador");
    expect(screen.getByRole("checkbox", { name: "Procuração" })).toBeInTheDocument();
  });
});
