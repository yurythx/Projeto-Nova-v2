import { describe, expect, it } from "vitest";

import { centavosDeReais, reais } from "./moeda";

function texto(valor: string) {
  return valor.replace(/\u00a0|\u202f/g, " ");
}

describe("reais", () => {
  it("converte centavos para real", () => {
    expect(texto(reais(120000))).toBe("R$ 1.200,00");
    expect(texto(reais(10))).toBe("R$ 0,10");
    expect(texto(reais(0))).toBe("R$ 0,00");
  });

  it("lê reais digitados em centavos", () => {
    expect(centavosDeReais("1.200,00")).toBe(120000);
    expect(centavosDeReais("100")).toBe(10000);
    expect(centavosDeReais("0,10")).toBe(10);
    expect(centavosDeReais("abc")).toBe(0);
  });
});
