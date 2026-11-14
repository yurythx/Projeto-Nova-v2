import { describe, expect, it } from "vitest";

import { destinoGovBR, lerDesafio } from "./sessao";

describe("destino do gov.br", () => {
  it("aceita os dois endereços oficiais e recusa o resto", () => {
    expect(destinoGovBR("https://sso.acesso.gov.br/authorize")).toBe(
      "https://sso.acesso.gov.br/authorize",
    );
    expect(destinoGovBR("https://sso.staging.acesso.gov.br/authorize")).toContain("staging");
    expect(destinoGovBR("http://sso.acesso.gov.br/authorize")).toBeNull();
    expect(destinoGovBR("https://evil.example/authorize")).toBeNull();
    expect(destinoGovBR("não é url")).toBeNull();
  });

  it("ignora verificador ilegível", () => {
    sessionStorage.setItem("fazenda.govbr", "{");
    expect(lerDesafio()).toBeNull();
    sessionStorage.clear();
  });
});
