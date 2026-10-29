import { describe, expect, it } from "vitest";

import { buildNav } from "./Sidebar";
import { isNavHrefActive } from "@/lib/nav/sectionTabs";
import type { ModuleStatus } from "@/lib/plataforma/types";
import { hasPermission } from "@/lib/plataforma/permissions";

function mod(key: string, extra: Partial<ModuleStatus> = {}): ModuleStatus {
  return {
    key,
    name: key,
    description: "",
    core: false,
    default_enabled: true,
    depends_on: [],
    permissions: [],
    public: false,
    icon: "box",
    route: `/${key}`,
    enabled: true,
    configured: true,
    dependents: [],
    blocked_by: [],
    ...extra,
  };
}

describe("buildNav - Padrão GLPI e ITIL v5 com 5 Seções", () => {
  const baseModules = [
    mod("iam", { core: true }),
    mod("audit", { core: true }),
    mod("atendimento", { name: "Atendimento", route: "/atendimento" }),
    mod("problemas", { name: "Problemas", route: "/problemas" }),
    mod("mudancas", { name: "Mudanças", route: "/mudancas" }),
    mod("projetos", { name: "Projetos", route: "/projetos" }),
    mod("ativos", { name: "Ativos", route: "/ativos" }),
    mod("produtos", { name: "Produtos", route: "/produtos" }),
    mod("tramite", { name: "Trâmite", route: "/tramite" }),
    mod("wiki", { name: "Wiki", route: "/wiki" }),
    mod("mercurio", { name: "Mercúrio", route: "/mercurio" }),
    mod("cofre", { name: "Cofre", route: "/cofre" }),
    mod("blog", { name: "Blog", route: "/blog" }),
    mod("egress", { route: "/configuracao/egress" }),
    mod("integracoes", { route: "/integracoes" }),
    mod("canais", { route: "/configuracao/canais" }),
    mod("search"),
  ];

  it("Usuário Normal (Servidor com atendimento:abrir) vê apenas Atendimento na Assistência e Módulos gerais", () => {
    const nav = buildNav(baseModules, (p) => p === "atendimento:abrir");

    // Na Assistência, não vê Dashboard nem Calendário de TI nem Problemas/Mudanças
    expect(nav.assistencia.map((a) => a.href)).toEqual(["/atendimento"]);
    expect(nav.assistencia[0]?.label).toBe("Atendimento");

    // Módulos gerais que não exigem gestão (tramite, wiki, mercurio, blog)
    expect(nav.modules.map((m) => m.href)).toEqual(["/tramite", "/wiki", "/mercurio", "/blog"]);

    // Seções restritas ficam 100% vazias
    expect(nav.ativos).toEqual([]);
    expect(nav.admin).toEqual([]);
    expect(nav.configuracao).toEqual([]);
  });

  it("Usuário Normal com acesso ao Cofre (atendimento:abrir + cofre:read)", () => {
    const nav = buildNav(baseModules, (p) => p === "atendimento:abrir" || p === "cofre:read");

    // Vê Atendimento
    expect(nav.assistencia.map((a) => a.href)).toEqual(["/atendimento"]);

    // Vê o Cofre em Módulos
    expect(nav.modules.map((m) => m.href)).toContain("/cofre");

    // Ativos, Admin e Configuração continuam vazios
    expect(nav.ativos).toEqual([]);
    expect(nav.admin).toEqual([]);
    expect(nav.configuracao).toEqual([]);
  });

  it("Técnico / Atendente de TI vê Assistência completa (Dashboard, Calendário, Fila, Problemas, Mudanças, Projetos) e Ativos", () => {
    const permissoesTecnico = [
      "atendimento:abrir",
      "atendimento:atender",
      "problemas:read",
      "problemas:manage",
      "mudancas:manage",
      "projetos:read",
      "projetos:manage",
      "ativos:manage",
    ];
    const nav = buildNav(baseModules, (p) => hasPermission(permissoesTecnico, p));

    // Assistência completa
    expect(nav.assistencia.map((a) => a.href)).toEqual([
      "/assistencia",
      "/assistencia/calendario",
      "/atendimento",
      "/problemas",
      "/mudancas",
      "/projetos",
    ]);
    expect(nav.assistencia[0]?.label).toBe("Dashboard");
    expect(nav.assistencia[1]?.label).toBe("Calendário");

    // Ativos (CMDB)
    expect(nav.ativos.map((a) => a.href)).toEqual(["/ativos"]);

    // Administração e Configuração continuam ocultos
    expect(nav.admin).toEqual([]);
    expect(nav.configuracao).toEqual([]);
  });

  it("Auditor vê Auditoria e Consulta de Usuários em Administração e Monitoramento em Configuração", () => {
    const permissoesAuditor = ["audit:read", "users:read", "monitoring:read"];
    const nav = buildNav(baseModules, (p) => hasPermission(permissoesAuditor, p));

    expect(nav.assistencia).toEqual([]);
    expect(nav.ativos).toEqual([]);

    expect(nav.admin.map((a) => a.href)).toEqual(["/configuracao/usuarios", "/auditoria"]);
    expect(nav.configuracao.map((c) => c.href)).toEqual(["/monitoramento"]);
  });

  it("Gestor de Identidade vê Usuários, Estrutura, Perfis e Mapeamento AD em Administração", () => {
    const permissoesIAM = ["users:read", "users:manage", "iam:manage"];
    const nav = buildNav(baseModules, (p) => hasPermission(permissoesIAM, p));

    expect(nav.admin.map((a) => a.href)).toEqual([
      "/configuracao/usuarios",
      "/configuracao/organizacao",
      "/configuracao/perfis",
      "/configuracao/mapeamento-ad",
    ]);
    expect(nav.configuracao).toEqual([]);
  });

  it("Administrador Geral (curinga *) vê todas as 5 seções completas", () => {
    const nav = buildNav(baseModules, (p) => hasPermission(["*"], p));

    // 1. Assistência
    expect(nav.assistencia.map((a) => a.href)).toEqual([
      "/assistencia",
      "/assistencia/calendario",
      "/atendimento",
      "/problemas",
      "/mudancas",
      "/projetos",
    ]);

    // 2. Ativos
    expect(nav.ativos.map((a) => a.href)).toEqual(["/ativos", "/produtos"]);

    // 3. Administração
    expect(nav.admin.map((a) => a.href)).toEqual([
      "/configuracao/usuarios",
      "/configuracao/organizacao",
      "/configuracao/perfis",
      "/configuracao/mapeamento-ad",
      "/auditoria",
    ]);

    // 4. Configuração
    expect(nav.configuracao.map((c) => c.href)).toEqual([
      "/configuracao",
      "/configuracao/modulos",
      "/configuracao/canais",
      "/integracoes",
      "/configuracao/keycloak",
      "/configuracao/ia",
      "/configuracao/egress",
      "/monitoramento",
    ]);
  });

  it("Desativação de módulos no Kernel: módulo desativado some do menu em tempo real", () => {
    const modulosComMudancaDesativada = baseModules.map((m) =>
      m.key === "mudancas" ? { ...m, enabled: false } : m,
    );
    const nav = buildNav(modulosComMudancaDesativada, (p) => hasPermission(["*"], p));

    // Mudanças não aparece na Assistência
    expect(nav.assistencia.map((a) => a.href)).not.toContain("/mudancas");
    expect(nav.assistencia.map((a) => a.href)).toEqual([
      "/assistencia",
      "/assistencia/calendario",
      "/atendimento",
      "/problemas",
      "/projetos",
    ]);
  });

  it("Desativação total de uma seção: se ativos e produtos forem desligados, a seção Ativos colapsa (fica vazia)", () => {
    const modulosSemAtivos = baseModules.map((m) =>
      m.key === "ativos" || m.key === "produtos" ? { ...m, enabled: false } : m,
    );
    const nav = buildNav(modulosSemAtivos, (p) => hasPermission(["*"], p));

    expect(nav.ativos).toEqual([]);
  });

  it("Módulos com tela de gestão (catálogo, contato) exigem permissão para entrar no menu", () => {
    const mods = [
      mod("catalog", { route: "/gestao/servicos" }),
      mod("contact", { route: "/gestao/contato" }),
    ];
    expect(buildNav(mods, () => false).modules).toEqual([]);
    const nav = buildNav(mods, (p) => hasPermission(["contact:read"], p));
    expect(nav.modules.map((m) => m.href)).toEqual(["/gestao/contato"]);
  });
});

describe("isNavHrefActive", () => {
  const config = [{ href: "/configuracao" }, { href: "/configuracao/modulos" }];
  it("a subpágina mais específica ganha do índice", () => {
    expect(isNavHrefActive("/configuracao/modulos", "/configuracao", config)).toBe(false);
    expect(isNavHrefActive("/configuracao/modulos", "/configuracao/modulos", config)).toBe(true);
    expect(isNavHrefActive("/configuracao/ia", "/configuracao", config)).toBe(true);
  });
});
