import { describe, expect, it } from "vitest";

import { andamento, nomePrograma, nomeSituacao, tomSituacao } from "./rotulos";

describe("rótulos do protocolo", () => {
  it("nomeia programa e situação e guarda o código desconhecido", () => {
    expect(nomePrograma("regulariza")).toBe("Regulariza Rondonópolis");
    expect(nomePrograma("avanca")).toBe("Avança Rondonópolis");
    expect(nomePrograma("outro")).toBe("outro");
    expect(nomeSituacao("em_exigencia")).toBe("Em exigência");
    expect(nomeSituacao("aguardando_pagamento")).toBe("Aguardando pagamento");
    expect(nomeSituacao("nova")).toBe("nova");
    expect(tomSituacao("decidido")).toBe("success");
    expect(tomSituacao("em_exigencia")).toBe("warning");
    expect(tomSituacao("arquivado")).toBe("neutral");
    expect(tomSituacao("nova")).toBe("neutral");
  });

  it("monta o andamento em ordem e omite data vazia", () => {
    expect(
      andamento({
        recibo_em: "2026-06-01T15:00:00Z",
        exigencia_aberta: "2026-06-10T15:00:00Z",
        despacho_em: "0001-01-01T00:00:00Z",
      }),
    ).toEqual([
      { rotulo: "Protocolo registrado", quando: "01/06/2026, 11:00" },
      { rotulo: "Exigência aberta", quando: "10/06/2026, 11:00" },
    ]);
  });
});
