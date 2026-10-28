import { describe, expect, it } from "vitest";

import { rotaDaCentral, soSolicitante } from "./central";
import type { Me } from "./types";

const me = (permissions: string[], roles: string[] = []) => ({ permissions, roles }) as Me;

describe("central do solicitante", () => {
  it("só quem abre atendimento e nada mais entra na central", () => {
    expect(soSolicitante(undefined)).toBe(false);
    expect(soSolicitante(me([]))).toBe(false);
    expect(soSolicitante(me(["atendimento:abrir"]))).toBe(true);
    expect(soSolicitante(me(["atendimento:abrir", "atendimento:atender"]))).toBe(false);
    expect(soSolicitante(me(["atendimento:abrir"], ["admin"]))).toBe(false);
    expect(soSolicitante({ roles: [] } as unknown as Me)).toBe(false);
  });

  it("a central oferece o atendimento de quem pede, a wiki, o perfil e o sino", () => {
    for (const r of [
      "/atendimento",
      "/atendimento/solicitacoes/s1",
      "/atendimento/aprovacoes",
      "/wiki",
      "/wiki/como-trocar-a-senha",
      "/perfil",
      "/notificacoes",
    ])
      expect(rotaDaCentral(r)).toBe(true);
    for (const r of [
      "/dashboard",
      "/atendimento/fila",
      "/atendimento/cadastro",
      "/configuracao",
      "/wikis",
    ])
      expect(rotaDaCentral(r)).toBe(false);
  });
});
