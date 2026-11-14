import { screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { mockBackend, renderApp } from "@/test/backend";

import { Retorno } from "./Retorno";
import { CHAVE_GOVBR } from "./sessao";

afterEach(() => {
  sessionStorage.clear();
});

describe("Retorno do gov.br", () => {
  it("recusa retorno sem o pedido deste navegador", async () => {
    sessionStorage.setItem(CHAVE_GOVBR, JSON.stringify({ state: "st", verifier: "ver" }));
    renderApp(<Retorno code="c" state="outro" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("não confere");
    expect(sessionStorage.getItem(CHAVE_GOVBR)).toBeNull();
  });

  it("mostra nome e e-mail e omite o CPF", async () => {
    sessionStorage.setItem(CHAVE_GOVBR, JSON.stringify({ state: "st", verifier: "ver" }));
    const backend = mockBackend({
      "POST v1/fazenda/govbr/sessao": {
        data: { nome: "Ana", email: "a@x", sub: "ana", cpf: "00000000000" },
      },
    });
    renderApp(<Retorno code="c" state="st" />);
    expect(await screen.findByText("Ana")).toBeInTheDocument();
    expect(screen.getByText("a@x")).toBeInTheDocument();
    expect(screen.queryByText("00000000000")).not.toBeInTheDocument();
    expect(sessionStorage.getItem(CHAVE_GOVBR)).toBeNull();
    expect(backend.to("POST v1/fazenda/govbr/sessao")[0]?.body).toEqual({
      code: "c",
      verifier: "ver",
    });
  });
});
