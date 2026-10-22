import { describe, expect, it } from "vitest";

import { buildNav } from "./Sidebar";
import { isNavHrefActive } from "@/lib/nav/sectionTabs";
import type { ModuleStatus } from "@/lib/sismarq/types";
import { hasPermission } from "@/lib/sismarq/permissions";

function mod(key: string, extra: Partial<ModuleStatus> = {}): ModuleStatus {
  return {
    key, name: key, description: "", core: false, default_enabled: true, depends_on: [], permissions: [],
    public: false, icon: "box", route: `/${key}`, enabled: true, configured: true, dependents: [], blocked_by: [], ...extra,
  };
}

describe("buildNav", () => {
  const modules = [
    mod("iam", { core: true }), mod("audit", { core: true }), mod("blog"), mod("wiki", { enabled: false }),
    mod("egress"), mod("search"), mod("tramite"),
  ];

  it("lista só plugins ativos, fora do núcleo/ocultos", () => {
    const nav = buildNav(modules, () => false);
    expect(nav.modules.map((m) => m.href)).toEqual(["/blog", "/tramite"]);
    expect(nav.admin).toEqual([]);
  });

  it("Trâmite não tem submenu", () => {
    const nav = buildNav(modules, () => false);
    const tramite = nav.modules.find((m) => m.href === "/tramite");
    expect(tramite?.children).toEqual([]);
  });

  it("administração depende das permissões efetivas (com curinga)", () => {
    const nav = buildNav(modules, (p) => hasPermission(["audit:*"], p));
    expect(nav.admin.map((a) => a.href)).toEqual(["/auditoria"]);
    const all = buildNav(modules, (p) => hasPermission(["*"], p));
    expect(all.admin.map((a) => a.href)).toEqual(["/auditoria", "/monitoramento", "/configuracao"]);
    expect(all.admin.find((a) => a.href === "/configuracao")?.children?.some((c) => c.href === "/configuracao/modulos")).toBe(true);
  });

  it("módulos de gestão (catálogo, contato) exigem a permissão para entrar no menu", () => {
    const mods = [mod("catalog", { route: "/gestao/servicos" }), mod("contact", { route: "/gestao/contato" })];
    expect(buildNav(mods, () => false).modules).toEqual([]);
    const nav = buildNav(mods, (p) => hasPermission(["contact:read"], p));
    expect(nav.modules.map((m) => m.href)).toEqual(["/gestao/contato"]);
  });
});

describe("isNavHrefActive", () => {
  const config = [
    { href: "/configuracao" },
    { href: "/configuracao/modulos" },
  ];
  it("a subpágina mais específica ganha do índice", () => {
    expect(isNavHrefActive("/configuracao/modulos", "/configuracao", config)).toBe(false);
    expect(isNavHrefActive("/configuracao/modulos", "/configuracao/modulos", config)).toBe(true);
    expect(isNavHrefActive("/configuracao/ia", "/configuracao", config)).toBe(true);
  });
});
