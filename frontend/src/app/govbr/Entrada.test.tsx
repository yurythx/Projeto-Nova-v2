import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { fail, mockBackend, renderApp } from "@/test/backend";

import { Entrada } from "./Entrada";
import { CHAVE_GOVBR } from "./sessao";

afterEach(() => {
  sessionStorage.clear();
  vi.unstubAllGlobals();
});

describe("Entrada do gov.br", () => {
  it("mostra a falta da credencial", async () => {
    const user = userEvent.setup();
    mockBackend({
      "GET v1/fazenda/govbr": fail(
        503,
        "credencial do Login Único ainda não configurada",
        "DEPENDENCY_UNAVAILABLE",
      ),
    });
    renderApp(<Entrada />);
    await user.click(screen.getByRole("button", { name: "Entrar com gov.br" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "credencial do Login Único ainda não configurada",
    );
  });

  it("guarda o verificador e segue só para o Login Único", async () => {
    const user = userEvent.setup();
    const assign = vi.fn();
    vi.stubGlobal("location", { assign });
    mockBackend({
      "GET v1/fazenda/govbr": {
        data: {
          state: "st",
          verifier: "ver",
          url: "https://sso.staging.acesso.gov.br/authorize?state=st",
        },
      },
    });
    renderApp(<Entrada />);
    await user.click(screen.getByRole("button", { name: "Entrar com gov.br" }));
    expect(assign).toHaveBeenCalledWith("https://sso.staging.acesso.gov.br/authorize?state=st");
    expect(sessionStorage.getItem(CHAVE_GOVBR)).toContain("ver");
  });

  it("recusa endereço fora do Login Único", async () => {
    const user = userEvent.setup();
    const assign = vi.fn();
    vi.stubGlobal("location", { assign });
    mockBackend({
      "GET v1/fazenda/govbr": {
        data: { state: "st", verifier: "ver", url: "https://evil.example/authorize" },
      },
    });
    renderApp(<Entrada />);
    await user.click(screen.getByRole("button", { name: "Entrar com gov.br" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("não é o do Login Único");
    expect(assign).not.toHaveBeenCalled();
    expect(sessionStorage.getItem(CHAVE_GOVBR)).toBeNull();
  });
});
