"use client";

import {
  Activity,
  BarChart3,
  Bell,
  Bot,
  Building2,
  Calendar,
  ChevronDown,
  Key,
  KeyRound,
  LayoutDashboard,
  Network,
  Palette,
  Repeat,
  Share2,
  ShieldCheck,
  Sliders,
  Users,
  type LucideProps,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState, type ComponentType } from "react";

import { useBranding } from "@/components/branding/BrandingContext";
import { ModuleIcon } from "@/components/layout/ModuleIcon";
import { CONFIG_TABS, MODULE_CHILDREN, isNavHrefActive, visibleTabs } from "@/lib/nav/sectionTabs";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { ModuleStatus } from "@/lib/plataforma/types";

export interface NavItem {
  href: string;
  label: string;
  icon?: ComponentType<LucideProps> | string;
  children?: NavItem[];
}

export interface NavGroups {
  assistencia: NavItem[];
  ativos: NavItem[];
  modules: NavItem[];
  admin: NavItem[];
  configuracao: NavItem[];
}

// Módulos que não entram no menu de "Módulos": o núcleo tem entradas
// próprias em Administração; Egress, Canais e Integrações vivem em Configuração;
// a Busca vive no cabeçalho (#global-search).
const HIDDEN_FROM_MENU = new Set([
  "iam",
  "audit",
  "egress",
  "search",
  "integracoes",
  "canais",
]);

// Módulos agrupados sob a seção dedicada de Assistência (estilo ITIL/GLPI).
const ASSISTENCIA_KEYS = new Set(["atendimento", "problemas", "mudancas", "projetos"]);
const ASSISTENCIA_ORDER = ["atendimento", "problemas", "mudancas", "projetos"];

// Módulos agrupados sob a seção dedicada de Ativos (CMDB estilo GLPI).
const ATIVOS_KEYS = new Set(["ativos", "produtos"]);
const ATIVOS_ORDER = ["ativos", "produtos"];

// Módulos cuja tela principal é de gestão (a face pública fica no site
// institucional): só aparecem no menu para quem tem a permissão.
const MENU_PERMISSION: Record<string, string> = {
  catalog: "catalog:manage",
  contact: "contact:read",
  cofre: "cofre:read",
};

function childrenOf(
  key: string,
  href: string,
  can: (p: string) => boolean,
  canGlobal: (p: string) => boolean,
  enabled: (m: string) => boolean,
): NavItem[] {
  const tabs = key === "config" ? CONFIG_TABS : (MODULE_CHILDREN[key] ?? []);
  return visibleTabs(tabs, can, enabled, canGlobal)
    .filter((t) => t.href !== href)
    .map((t) => ({ href: t.href, label: t.label }));
}

/** Monta o menu a partir do estado dos módulos no Kernel e das permissões
 * efetivas — organizado no padrão GLPI e ITIL v5 com 5 seções bem definidas:
 * Assistência, Ativos, Módulos, Administração e Configuração. */
export function buildNav(
  modules: ModuleStatus[],
  can: (p: string) => boolean,
  canGlobal: (p: string) => boolean = can,
  enabled: (m: string) => boolean = () => true,
): NavGroups {
  const activeMods = modules
    .filter((m) => m.enabled && !m.core && !HIDDEN_FROM_MENU.has(m.key) && m.route)
    .filter(
      (m) =>
        !MENU_PERMISSION[m.key] ||
        can(MENU_PERMISSION[m.key]!) ||
        (m.key === "cofre" && can("cofre:manage")),
    );

  const assistenciaMods: { key: string; item: NavItem }[] = [];
  const ativosMods: { key: string; item: NavItem }[] = [];
  const mods: NavItem[] = [];

  for (const m of activeMods) {
    const href =
      m.key === "blog"
        ? can("blog:manage")
          ? "/gestao/blog"
          : m.route === "/gestao/blog"
            ? "/blog"
            : m.route
        : m.route;
    const item: NavItem = {
      href,
      label: m.name,
      icon: m.icon,
      children: childrenOf(m.key, href, can, canGlobal, enabled),
    };

    if (ASSISTENCIA_KEYS.has(m.key)) {
      // Regras de permissão para módulos de assistência
      let visivel = false;
      if (m.key === "atendimento") {
        visivel =
          can("atendimento:abrir") || can("atendimento:atender") || can("atendimento:manage");
      } else if (m.key === "problemas") {
        visivel = can("problemas:read") || can("problemas:manage");
      } else if (m.key === "mudancas") {
        visivel = can("mudancas:manage") || can("mudancas:aprovar");
      } else if (m.key === "projetos") {
        visivel = can("projetos:read") || can("projetos:manage");
      }
      if (visivel) {
        assistenciaMods.push({ key: m.key, item });
      }
    } else if (ATIVOS_KEYS.has(m.key)) {
      // Regras de permissão para CMDB / Ativos
      let visivel = false;
      if (m.key === "ativos") {
        visivel = can("ativos:read") || can("ativos:manage");
      } else if (m.key === "produtos") {
        visivel = can("produtos:read") || can("produtos:manage");
      }
      if (visivel) {
        ativosMods.push({ key: m.key, item });
      }
    } else {
      mods.push(item);
    }
  }

  // 1. ASSISTÊNCIA (ITIL v5 / GLPI Helpdesk)
  // O Dashboard e o Calendário de TI são reservados a quem atende ou gerencia serviços.
  // Usuários normais (apenas atendimento:abrir) veem diretamente o portal de Atendimento.
  const assistencia: NavItem[] = [];
  const podeVerDashboard =
    can("atendimento:atender") ||
    can("problemas:read") ||
    can("mudancas:manage") ||
    can("projetos:read") ||
    can("projetos:manage");
  const podeVerCalendario =
    can("atendimento:atender") ||
    can("mudancas:manage") ||
    can("mudancas:aprovar") ||
    can("projetos:read") ||
    can("projetos:manage");

  if (assistenciaMods.length > 0) {
    if (podeVerDashboard) {
      assistencia.push({
        href: "/assistencia",
        label: "Dashboard",
        icon: BarChart3,
      });
    }
    if (podeVerCalendario) {
      assistencia.push({
        href: "/assistencia/calendario",
        label: "Calendário",
        icon: Calendar,
      });
    }
    assistenciaMods.sort(
      (a, b) => ASSISTENCIA_ORDER.indexOf(a.key) - ASSISTENCIA_ORDER.indexOf(b.key),
    );
    assistencia.push(...assistenciaMods.map((m) => m.item));
  }

  // 2. ATIVOS (CMDB / GLPI Assets)
  const ativos: NavItem[] = [];
  if (ativosMods.length > 0) {
    ativosMods.sort((a, b) => ATIVOS_ORDER.indexOf(a.key) - ATIVOS_ORDER.indexOf(b.key));
    ativos.push(...ativosMods.map((m) => m.item));
  }

  // 3. ADMINISTRAÇÃO (Pessoas, Estrutura e Governança do Core)
  const admin: NavItem[] = [];
  if (can("users:read") || can("users:manage")) {
    admin.push({ href: "/configuracao/usuarios", label: "Usuários", icon: Users });
  }
  if (can("iam:manage")) {
    admin.push({
      href: "/configuracao/organizacao",
      label: "Estrutura organizacional",
      icon: Building2,
    });
  }
  if (canGlobal("iam:manage")) {
    admin.push({ href: "/configuracao/perfis", label: "Perfis de acesso", icon: KeyRound });
    admin.push({ href: "/configuracao/mapeamento-ad", label: "Mapeamento AD", icon: Network });
  }
  if (can("audit:read")) {
    admin.push({ href: "/auditoria", label: "Auditoria", icon: ShieldCheck });
  }

  // 4. CONFIGURAÇÃO (Infraestrutura, Conectores Técnicos e Plugins)
  const configuracao: NavItem[] = [];
  if (can("branding:manage")) {
    configuracao.push({ href: "/configuracao", label: "Identidade visual", icon: Palette });
  }
  if (can("modules:manage")) {
    configuracao.push({ href: "/configuracao/modulos", label: "Módulos", icon: Sliders });
  }
  if (can("canais:manage") && enabled("canais")) {
    configuracao.push({ href: "/configuracao/canais", label: "Canais de aviso", icon: Bell });
  }
  if (can("integracoes:manage") && enabled("integracoes")) {
    configuracao.push({ href: "/integracoes", label: "Integrações (GLPI)", icon: Repeat });
  }
  if (can("keycloak:manage")) {
    configuracao.push({ href: "/configuracao/keycloak", label: "Keycloak (IAM)", icon: Key });
  }
  if (can("ia:manage")) {
    configuracao.push({ href: "/configuracao/ia", label: "Inteligência artificial", icon: Bot });
  }
  if (can("egress:manage") && enabled("egress")) {
    configuracao.push({ href: "/configuracao/egress", label: "Egress & Webhooks", icon: Share2 });
  }
  if (can("monitoring:read") || can("monitoring:manage")) {
    configuracao.push({ href: "/monitoramento", label: "Monitoramento", icon: Activity });
  }

  return { assistencia, ativos, modules: mods, admin, configuracao };
}

export function Sidebar({
  collapsed,
  mobileOpen,
  onCloseMobile,
}: {
  collapsed: boolean;
  mobileOpen: boolean;
  onCloseMobile: () => void;
}) {
  const pathname = usePathname();
  const { branding } = useBranding();
  const { modules, can, canGlobal, enabled } = usePlataforma();
  const nav = buildNav(modules, can, canGlobal, enabled);
  const [aberto, setAberto] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (!mobileOpen) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onCloseMobile();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [mobileOpen, onCloseMobile]);

  const grupoAberto = (item: NavItem) => {
    if (collapsed) return false;
    if (aberto[item.href] !== undefined) return aberto[item.href];
    return isNavHrefActive(pathname, item.href);
  };

  const renderLink = (item: NavItem, { nested = false }: { nested?: boolean } = {}) => {
    const siblings = nested ? [] : (item.children ?? []);
    const active = isNavHrefActive(pathname, item.href, siblings);
    return (
      <Link
        href={item.href}
        onClick={onCloseMobile}
        aria-current={active ? "page" : undefined}
        title={collapsed && !nested ? item.label : undefined}
        className={`flex min-w-0 flex-1 items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
          nested ? "py-1.5 pl-10 text-[13px] font-normal" : ""
        } ${
          active ? "bg-primary/10 text-primary" : "text-foreground hover:bg-surface-hover"
        } ${collapsed && !nested ? "md:justify-center" : ""}`}
      >
        {!nested && item.icon ? (
          typeof item.icon === "string" ? (
            <ModuleIcon name={item.icon} size={18} aria-hidden="true" className="shrink-0" />
          ) : (
            <item.icon size={18} aria-hidden="true" className="shrink-0" />
          )
        ) : null}
        <span className={collapsed && !nested ? "md:sr-only" : "truncate"}>{item.label}</span>
      </Link>
    );
  };

  const renderGroup = (item: NavItem) => {
    const kids = item.children ?? [];
    const open = kids.length > 0 && grupoAberto(item);
    const subId = `submenu-${item.href.replace(/\W+/g, "-")}`;
    return (
      <li key={item.href}>
        <div className="flex items-center gap-0.5">
          {renderLink(item)}
          {kids.length > 0 && (
            <button
              type="button"
              className={`shrink-0 rounded-md p-2 text-muted hover:bg-surface-hover hover:text-foreground ${collapsed ? "md:hidden" : ""}`}
              aria-expanded={open}
              aria-controls={subId}
              aria-label={open ? `Recolher ${item.label}` : `Expandir ${item.label}`}
              onClick={() => setAberto((a) => ({ ...a, [item.href]: !open }))}
            >
              <ChevronDown
                size={16}
                aria-hidden="true"
                className={`transition-transform ${open ? "rotate-180" : ""}`}
              />
            </button>
          )}
        </div>
        {kids.length > 0 && (
          <ul
            id={subId}
            hidden={!open}
            className="mt-0.5 flex flex-col gap-0.5 border-l border-surface-border ml-5"
          >
            {kids.map((c) => (
              <li key={c.href}>{renderLink(c, { nested: true })}</li>
            ))}
          </ul>
        )}
      </li>
    );
  };

  return (
    <>
      {mobileOpen && (
        <div
          className="fixed inset-x-0 bottom-0 top-[var(--topbar-h)] z-40 bg-black/40 md:hidden"
          onClick={onCloseMobile}
          aria-hidden="true"
        />
      )}
      <nav
        id="main-menu"
        tabIndex={-1}
        aria-label="Menu principal"
        className={`fixed bottom-0 left-0 top-[var(--topbar-h)] z-40 flex w-72 flex-col overflow-y-auto border-r border-surface-border bg-surface outline-none transition-[transform,width] duration-[var(--shell-motion)] md:translate-x-0 ${
          collapsed ? "md:w-[var(--sidebar-w-collapsed)]" : "md:w-[var(--sidebar-w)]"
        } ${mobileOpen ? "translate-x-0" : "-translate-x-full"}`}
      >
        <ul className="flex flex-col gap-0.5 px-2 pt-3">
          {renderGroup({ href: "/dashboard", label: "Visão geral", icon: LayoutDashboard })}
        </ul>

        {nav.assistencia.length > 0 && (
          <div className="mt-3">
            <p
              className={`px-4 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted ${collapsed ? "md:sr-only" : ""}`}
            >
              Assistência
            </p>
            <ul className="flex flex-col gap-0.5 px-2">{nav.assistencia.map(renderGroup)}</ul>
          </div>
        )}

        {nav.ativos.length > 0 && (
          <div className="mt-3">
            <p
              className={`px-4 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted ${collapsed ? "md:sr-only" : ""}`}
            >
              Ativos
            </p>
            <ul className="flex flex-col gap-0.5 px-2">{nav.ativos.map(renderGroup)}</ul>
          </div>
        )}

        {nav.modules.length > 0 && (
          <div className="mt-3">
            <p
              className={`px-4 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted ${collapsed ? "md:sr-only" : ""}`}
            >
              Módulos
            </p>
            <ul className="flex flex-col gap-0.5 px-2">{nav.modules.map(renderGroup)}</ul>
          </div>
        )}

        {nav.admin.length > 0 && (
          <div className="mt-3">
            <p
              className={`px-4 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted ${collapsed ? "md:sr-only" : ""}`}
            >
              Administração
            </p>
            <ul className="flex flex-col gap-0.5 px-2">{nav.admin.map(renderGroup)}</ul>
          </div>
        )}

        {nav.configuracao.length > 0 && (
          <div className="mt-3">
            <p
              className={`px-4 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted ${collapsed ? "md:sr-only" : ""}`}
            >
              Configuração
            </p>
            <ul className="flex flex-col gap-0.5 px-2 pb-3">
              {nav.configuracao.map(renderGroup)}
            </ul>
          </div>
        )}

        <p
          className={`mt-auto border-t border-surface-border px-4 py-3 text-[11px] text-muted ${collapsed ? "md:sr-only" : ""}`}
        >
          {branding.appName}
        </p>
      </nav>
    </>
  );
}
