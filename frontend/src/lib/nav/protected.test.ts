import { describe, expect, it } from "vitest";

import { isProtectedPath, PROTECTED_PREFIXES } from "./protected";

describe("rotas protegidas", () => {
  it("a caixa de notificações exige sessão", () => {
    expect(PROTECTED_PREFIXES).toContain("/notificacoes");
    expect(PROTECTED_PREFIXES).toContain("/produtos");
    expect(PROTECTED_PREFIXES).toContain("/atendimento");
    expect(isProtectedPath("/produtos")).toBe(true);
    expect(isProtectedPath("/atendimento/ofertas")).toBe(true);
    expect(isProtectedPath("/notificacoes")).toBe(true);
    expect(isProtectedPath("/tramite/p1")).toBe(true);
  });

  it("o site público não pede login", () => {
    expect(isProtectedPath("/")).toBe(false);
    expect(isProtectedPath("/sobre")).toBe(false);
    expect(isProtectedPath("/faq")).toBe(false);
    expect(isProtectedPath("/login")).toBe(false);
  });
});
