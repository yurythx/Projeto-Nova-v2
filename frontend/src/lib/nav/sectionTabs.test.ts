import { describe, expect, it } from "vitest";

import { hasPermission } from "@/lib/plataforma/permissions";

import { CONFIG_TABS, MODULE_CHILDREN, isNavHrefActive, visibleTabs } from "./sectionTabs";

describe("abas de Configurações", () => {
  it("cada aba só aparece com a permissão correspondente", () => {
    const tabs = visibleTabs(
      CONFIG_TABS,
      (p) => hasPermission(["users:read", "iam:*"], p),
      () => true,
    );
    expect(tabs.map((t) => t.href)).toEqual([
      "/configuracao/organizacao",
      "/configuracao/perfis",
      "/configuracao/mapeamento-ad",
      "/configuracao/usuarios",
    ]);
  });

  it("Egress exige a permissão E o plug-in ativo", () => {
    const can = (p: string) => hasPermission(["*"], p);
    expect(visibleTabs(CONFIG_TABS, can, () => true).some((t) => t.href.endsWith("egress"))).toBe(
      true,
    );
    expect(
      visibleTabs(CONFIG_TABS, can, (m) => m !== "egress").some((t) => t.href.endsWith("egress")),
    ).toBe(false);
  });

  it("administração delegada (ADR 013): perfis e mapeamentos só com concessão global", () => {
    const can = (p: string) => hasPermission(["users:read", "iam:*"], p);
    const tabs = visibleTabs(
      CONFIG_TABS,
      can,
      () => true,
      () => false,
    );
    expect(tabs.map((t) => t.href)).toEqual([
      "/configuracao/organizacao",
      "/configuracao/usuarios",
    ]);
  });

  it("sem permissões: nenhuma aba", () => {
    expect(
      visibleTabs(
        CONFIG_TABS,
        () => false,
        () => true,
      ),
    ).toEqual([]);
  });
});

describe("submenus dos módulos", () => {
  it("Trâmite não tem submenu", () => {
    expect(MODULE_CHILDREN.tramite).toBeUndefined();
  });

  it("Atendimento: fila para quem atende, cadastro para quem gere", () => {
    const tabs = (perms: string[]) =>
      visibleTabs(
        MODULE_CHILDREN.atendimento!,
        (p) => hasPermission(perms, p),
        () => true,
      ).map((t) => t.href);
    expect(tabs(["atendimento:abrir"])).toEqual(["/atendimento/aprovacoes"]);
    expect(tabs(["atendimento:atender"])).toEqual([
      "/atendimento/aprovacoes",
      "/atendimento/fila",
      "/atendimento/campo",
      "/atendimento/relatorio",
    ]);
    expect(tabs(["atendimento:*"])).toEqual([
      "/atendimento/aprovacoes",
      "/atendimento/fila",
      "/atendimento/campo",
      "/atendimento/relatorio",
      "/atendimento/cadastro",
      "/atendimento/prazos",
      "/atendimento/regras",
    ]);
  });

  it("isNavHrefActive: prefixo mais longo", () => {
    const irmaos = [{ href: "/configuracao" }, { href: "/configuracao/modulos" }];
    expect(isNavHrefActive("/configuracao/modulos", "/configuracao", irmaos)).toBe(false);
    expect(isNavHrefActive("/configuracao/modulos", "/configuracao/modulos", irmaos)).toBe(true);
  });
});
