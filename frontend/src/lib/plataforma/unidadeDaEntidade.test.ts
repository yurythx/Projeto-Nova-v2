import { describe, expect, it } from "vitest";

import type { OrgTree } from "@/lib/plataforma/types";

import { unidadeDaEntidadeAtiva } from "./unidadeDaEntidade";

const tree = [
  {
    id: "pref",
    nome: "Prefeitura",
    unidades: [{ id: "u-pref" }],
  },
  {
    id: "sms",
    nome: "SMS",
    unidades: [{ id: "u-sms" }, { id: "u-visa" }],
  },
] as OrgTree[];

describe("unidadeDaEntidadeAtiva", () => {
  it("usa a única unidade da instituição selecionada", () => {
    expect(unidadeDaEntidadeAtiva(tree, "pref")).toBe("u-pref");
  });

  it("não escolhe sozinha quando a instituição tem várias unidades", () => {
    expect(unidadeDaEntidadeAtiva(tree, "sms")).toBe("");
  });

  it("aceita a própria unidade quando ela é a seleção", () => {
    expect(unidadeDaEntidadeAtiva(tree, "u-visa")).toBe("u-visa");
  });

  it("não sugere unidade na visão de todas as instituições", () => {
    expect(unidadeDaEntidadeAtiva(tree, "")).toBe("");
    expect(unidadeDaEntidadeAtiva(undefined, "pref")).toBe("");
  });
});
