import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { LoadTestTab } from "./LoadTestTab";

/** O relógio só anda dentro do fetch falso (`passo` ms por requisição):
 * com 5 workers, cada latência medida é 5 × passo. */
let agora = 0;
function respostas(passo: number, ...status: (number | "erro")[]) {
  let i = 0;
  vi.spyOn(performance, "now").mockImplementation(() => agora);
  const fn = vi.fn(async () => {
    agora += passo;
    const s = status[Math.min(i++, status.length - 1)];
    if (s === "erro") throw new Error("rede");
    return new Response(null, { status: s });
  });
  vi.stubGlobal("fetch", fn);
  return fn;
}

/** Roda até o fim e abre o diagnóstico do teste mais recente. */
async function rodar(diagnostico: RegExp) {
  fireEvent.click(screen.getByRole("button", { name: /Iniciar Teste de Carga/ }));
  await waitFor(() => expect(screen.getByRole("button", { name: /Iniciar Teste de Carga/ })).toBeInTheDocument());
  fireEvent.click(screen.getAllByRole("button", { name: "Expandir detalhes" })[0]!);
  expect(screen.getAllByText(diagnostico).length).toBeGreaterThan(0);
  vi.restoreAllMocks();
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("LoadTestTab", () => {
  it("roda o benchmark, classifica os status e registra o histórico com diagnóstico", async () => {
    render(<LoadTestTab />);
    expect(screen.getByText("Nenhum teste de carga executado ainda nesta sessão.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "5x" }));
    fireEvent.click(screen.getByRole("button", { name: "50" }));

    const get = respostas(1, 200);
    await rodar(/Desempenho Excepcional/);
    expect(get).toHaveBeenCalledTimes(50);
    expect(get.mock.calls[0]).toBeDefined();

    respostas(1, 429, 404, 200);
    await rodar(/Defesa Ativa Confirmada/);
    respostas(1, 503, "erro");
    await rodar(/Alerta de Instabilidade/);

    fireEvent.change(screen.getByLabelText(/Endpoint \/ Cenário Alvo/), { target: { value: "contact_ratelimit" } });
    const post = respostas(20, 201);
    await rodar(/Desempenho Saudável/);
    expect((post.mock.calls[0] as unknown[])[1]).toMatchObject({ method: "POST" });
    respostas(100, 200);
    await rodar(/Tempo de Resposta Moderado/);

    expect(screen.getAllByRole("button", { name: /detalhes/ })).toHaveLength(5);
    fireEvent.click(screen.getByRole("button", { name: "Recolher detalhes" }));
    expect(screen.queryByRole("button", { name: "Recolher detalhes" })).not.toBeInTheDocument();

    // Copiar o resumo e exportar o histórico.
    const writeText = vi.fn();
    vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText } });
    vi.useFakeTimers();
    fireEvent.click(screen.getByRole("button", { name: /Copiar Resumo/ }));
    expect(writeText).toHaveBeenCalledWith(expect.stringContaining("RELATÓRIO DE BENCHMARK"));
    expect(screen.getByRole("button", { name: /Copiado!/ })).toBeInTheDocument();
    act(() => vi.advanceTimersByTime(2000));
    expect(screen.getByRole("button", { name: /Copiar Resumo/ })).toBeInTheDocument();
    vi.useRealTimers();
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    fireEvent.click(screen.getByRole("button", { name: /Exportar JSON/ }));
    expect(click).toHaveBeenCalled();
  });

  it("interrompe o teste em andamento e aborta as requisições", async () => {
    const fetch = vi.fn(
      (_url: string, init?: RequestInit) =>
        new Promise<Response>((_ok, falha) => {
          init?.signal?.addEventListener("abort", () => falha(Object.assign(new Error("abortado"), { name: "AbortError" })));
        }),
    );
    vi.stubGlobal("fetch", fetch);
    const { unmount } = render(<LoadTestTab />);
    fireEvent.click(screen.getByRole("button", { name: /Iniciar Teste de Carga/ }));
    expect(await screen.findByRole("button", { name: /Interromper Teste/ })).toBeInTheDocument();
    expect(screen.getByLabelText(/Endpoint \/ Cenário Alvo/)).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: /Interromper Teste/ }));
    expect(await screen.findByRole("button", { name: /Iniciar Teste de Carga/ })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Expandir detalhes" })).not.toBeInTheDocument();

    // Desmontar no meio de outro teste também aborta.
    fireEvent.click(screen.getByRole("button", { name: /Iniciar Teste de Carga/ }));
    await screen.findByRole("button", { name: /Interromper Teste/ });
    unmount();
  });
});
