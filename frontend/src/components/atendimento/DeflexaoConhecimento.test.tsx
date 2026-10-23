import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { DeflexaoConhecimento } from "@/components/atendimento/DeflexaoConhecimento";

describe("DeflexaoConhecimento (Self-service / Deflexão de chamados)", () => {
  it("não renderiza nada para termos curtos", () => {
    const { container } = render(<DeflexaoConhecimento termo="oi" />);
    expect(container.firstChild).toBeNull();
  });

  it("sugere guias de autoatendimento para termos como 'senha'", async () => {
    render(<DeflexaoConhecimento termo="não lembro minha senha" />);

    expect(screen.getByTestId("deflexao-conhecimento")).toBeInTheDocument();
    expect(screen.getByText("Encontramos soluções rápidas para o seu caso")).toBeInTheDocument();
    expect(
      screen.getByText("Como redefinir sua senha de rede ou desbloquear sua conta"),
    ).toBeInTheDocument();
  });

  it("abre modal com passo a passo e permite marcar como resolvido", async () => {
    const user = userEvent.setup();
    const onResolvido = vi.fn();

    render(
      <DeflexaoConhecimento
        termo="falha na vpn corporativa"
        onResolvido={onResolvido}
      />,
    );

    expect(
      screen.getByText("Guia de conexão e solução de falhas na VPN corporativa"),
    ).toBeInTheDocument();

    // Clica no card da sugestão para abrir o modal
    await user.click(screen.getByText("Guia de conexão e solução de falhas na VPN corporativa"));

    expect(screen.getByText("Procedimento Recomendado:")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Sim, resolveu!/i })).toBeInTheDocument();

    // Clica em 'Sim, resolveu!'
    await user.click(screen.getByRole("button", { name: /Sim, resolveu!/i }));

    expect(onResolvido).toHaveBeenCalledTimes(1);
    expect(await screen.findByText("Excelente! Problema resolvido com sucesso.")).toBeInTheDocument();
  });
});
