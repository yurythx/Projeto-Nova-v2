import { describe, expect, it } from "vitest";

import { pecasLegais } from "./pecas";

describe("peças do Anexo II", () => {
  it("lista o Regulariza, a posse sem contrato e a procuração", () => {
    expect(
      pecasLegais({
        programa: "regulariza",
        entrada: "procurador",
        itbi: false,
        moduloC: false,
        temCessao: false,
        temContrato: false,
      }),
    ).toEqual([
      "identificacao",
      "instrumento",
      "comprovante_endereco",
      "prova_posse",
      "procuracao",
    ]);
  });

  it("lista o Avança com ITBI e dispensa a posse quando há contrato", () => {
    expect(
      pecasLegais({
        programa: "avanca",
        entrada: "govbr",
        itbi: true,
        moduloC: false,
        temCessao: false,
        temContrato: true,
      }),
    ).toContain("instrumento");
    expect(
      pecasLegais({
        programa: "regulariza",
        entrada: "balcao",
        itbi: false,
        moduloC: true,
        temCessao: true,
        temContrato: true,
      }),
    ).toEqual([
      "identificacao",
      "instrumento",
      "comprovante_endereco",
      "cessao",
      "prova_data",
      "matricula",
    ]);
  });
});
