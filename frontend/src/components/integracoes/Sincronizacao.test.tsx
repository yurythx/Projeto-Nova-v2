import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { Sincronizacao } from "./Sincronizacao";

describe("Sincronizar até a virada", () => {
  it("desligada, liga com um clique", async () => {
    const onMudar = vi.fn();
    render(<Sincronizacao ligada={false} ocupado={false} onMudar={onMudar} />);
    expect(screen.getByText("Desligada")).toBeInTheDocument();
    expect(screen.queryByText(/Última extração/)).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Ligar a sincronização" }));
    expect(onMudar).toHaveBeenCalledWith(true);
  });

  it("ligada, desliga só depois de confirmar", async () => {
    const onMudar = vi.fn();
    render(
      <Sincronizacao ligada extraidoEm="2026-10-28T12:00:00Z" ocupado={false} onMudar={onMudar} />,
    );
    expect(screen.getByText("Ligada")).toBeInTheDocument();
    expect(screen.getByText(/Última extração/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Desligar a sincronização" }));
    expect(onMudar).not.toHaveBeenCalled();
    const dialogo = await screen.findByRole("dialog");
    await userEvent.click(within(dialogo).getByRole("button", { name: "Desligar" }));
    expect(onMudar).toHaveBeenCalledWith(false);
  });
});
