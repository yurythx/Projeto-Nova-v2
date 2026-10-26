import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock("@/lib/api/client", () => ({ apiClient: { get, post } }));

import { DeflexaoConhecimento } from "@/components/atendimento/DeflexaoConhecimento";

const artigo = {
  module: "wiki",
  type: "page",
  id: "p1",
  title: "Como redefinir a senha de rede",
  snippet: "Passos publicados na wiki.",
  url: "/wiki/senha",
  score: 1,
};

describe("DeflexaoConhecimento", () => {
  it("não consulta a wiki para um termo curto", () => {
    get.mockClear();
    const { container } = render(<DeflexaoConhecimento termo="oi" />);
    expect(container.firstChild).toBeNull();
    expect(get).not.toHaveBeenCalled();
  });

  it("mostra o artigo que a busca devolve e deixa de abrir o chamado", async () => {
    get.mockReset();
    post.mockReset();
    post.mockResolvedValue({});
    get.mockResolvedValue({ data: { query: "senha", results: [artigo], modules: ["wiki"], degraded: [], took_ms: 1 } });
    const user = userEvent.setup();
    const onResolvido = vi.fn();

    render(<DeflexaoConhecimento termo="não lembro minha senha" onResolvido={onResolvido} />);

    expect(await screen.findByTestId("deflexao-conhecimento")).toBeInTheDocument();
    await vi.waitFor(() =>
      expect(post).toHaveBeenCalledWith("v1/atendimento/deflexao", { artigos: ["p1"], resolveu: false }),
    );
    expect(screen.getByRole("link", { name: artigo.title })).toHaveAttribute("href", "/wiki/senha");
    expect(get).toHaveBeenCalledWith(expect.stringContaining("module=wiki"));

    await user.click(screen.getByRole("button", { name: "Não preciso mais abrir" }));
    expect(onResolvido).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith("v1/atendimento/deflexao", { artigos: ["p1"], resolveu: true });
    expect(screen.getByRole("status")).toHaveTextContent("não foi enviada");
  });

  it("não mostra nada quando a busca falha", async () => {
    get.mockReset();
    get.mockRejectedValue(new Error("sem wiki"));
    const { container } = render(<DeflexaoConhecimento termo="vpn corporativa" />);
    await vi.waitFor(() => expect(get).toHaveBeenCalled());
    expect(container.querySelector("[data-testid='deflexao-conhecimento']")).toBeNull();
  });
});
