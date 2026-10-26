import { describe, expect, it } from "vitest";

import { dispor, estiloGraficoDe, mostrar, mover, ocultar } from "./painelAssistencia";

const ids = ["fila", "erros", "janelas"];

describe("disposição do painel", () => {
  it("começa na ordem pedida quando não há nada gravado", () => {
    expect(dispor(ids, null)).toEqual({ ordem: ids, ocultos: [] });
  });

  it("mantém a ordem gravada e acrescenta cartão novo no fim", () => {
    expect(dispor(ids, { ordem: ["janelas", "fila", "sumido"], ocultos: ["erros", "sumido"] })).toEqual({
      ordem: ["janelas", "fila", "erros"],
      ocultos: ["erros"],
    });
  });

  it("sobe e desce só entre os cartões visíveis", () => {
    const base = { ordem: ["fila", "erros", "janelas"], ocultos: ["erros"] };
    expect(mover(base, "janelas", -1).ordem).toEqual(["janelas", "erros", "fila"]);
    expect(mover(base, "fila", -1)).toBe(base);
  });

  it("oculta e devolve o cartão", () => {
    const base = dispor(ids, null);
    const oculto = ocultar(base, "fila");
    expect(oculto.ocultos).toEqual(["fila"]);
    expect(mostrar(oculto, "fila").ocultos).toEqual([]);
    expect(ocultar(oculto, "fila")).toBe(oculto);
  });

  it("estilo desconhecido volta para barras", () => {
    expect(estiloGraficoDe("rosca")).toBe("rosca");
    expect(estiloGraficoDe("pizza")).toBe("barras");
    expect(estiloGraficoDe(null)).toBe("barras");
  });
});
