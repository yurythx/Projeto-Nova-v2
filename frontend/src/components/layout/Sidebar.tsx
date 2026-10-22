"use client";

import { Activity, ChevronDown, LayoutDashboard, Settings, ShieldCheck, type LucideProps } from "lucide-react";
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

// Módulos que não entram no menu de "Módulos": o núcleo tem entradas
// próprias em Administração; Egress é configuração; a Busca vive no
// cabeçalho (#global-search).
const HIDDEN_FROM_MENU = new Set(["iam", "audit", "egress", "search"]);

// Módulos cuja tela principal é de gestão (a face pública fica no site
// institucional): só aparecem no menu para quem tem a permissão.
const MENU_PERMISSION: Record<string, string> = {
  catalog: "catalog:manage",
  contact: "contact:read",
};

const ADMIN_PERMISSIONS = [
  "modules:manage", "iam:manage", "users:read", "branding:manage", "keycloak:manage", "ia:manage",
  "egress:manage", "catalog:manage", "contact:read", "mercurio:manage", "calendar:manage", "blog:manage",
];

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
 * efetivas — um plugin desativado some do menu sem recarregar a página. */
export function buildNav(
  modules: ModuleStatus[],
  can: (p: string) => boolean,
  canGlobal: (p: string) => boolean = can,
  enabled: (m: string) => boolean = () => true,
): { modules: NavItem[]; admin: NavItem[] } {
  const mods = modules
    .filter((m) => m.enabled && !m.core && !HIDDEN_FROM_MENU.has(m.key) && m.route)
    .filter((m) => !MENU_PERMISSION[m.key] || can(MENU_PERMISSION[m.key]!))
    .map((m) => {
      const href = m.key === "blog" ? (can("blog:manage") ? "/gestao/blog" : (m.route === "/gestao/blog" ? "/blog" : m.route)) : m.route;
      return {
        href,
        label: m.name,
        icon: m.icon,
        children: childrenOf(m.key, href, can, canGlobal, enabled),
      };
    });
  const admin: NavItem[] = [];
  if (can("audit:read")) admin.push({ href: "/auditoria", label: "Auditoria", icon: ShieldCheck });
  if (can("monitoring:read")) admin.push({ href: "/monitoramento", label: "Monitoramento", icon: Activity });
  if (ADMIN_PERMISSIONS.some(can)) {
    admin.push({
      href: "/configuracao",
      label: "Configurações",
      icon: Settings,
      children: childrenOf("config", "/configuracao", can, canGlobal, enabled),
    });
  }
  return { modules: mods, admin };
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
              <ChevronDown size={16} aria-hidden="true" className={`transition-transform ${open ? "rotate-180" : ""}`} />
            </button>
          )}
        </div>
        {kids.length > 0 && (
          <ul id={subId} hidden={!open} className="mt-0.5 flex flex-col gap-0.5 border-l border-surface-border ml-5">
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
        <div className="fixed inset-x-0 bottom-0 top-[var(--topbar-h)] z-40 bg-black/40 md:hidden" onClick={onCloseMobile} aria-hidden="true" />
      )}
      <nav
        id="main-menu"
        tabIndex={-1}
        aria-label="Menu principal"
        className={`fixed bottom-0 left-0 top-[var(--topbar-h)] z-40 flex w-72 flex-col overflow-y-auto border-r border-surface-border bg-surface outline-none transition-[transform,width] duration-[var(--shell-motion)] md:translate-x-0 ${
          collapsed ? "md:w-[var(--sidebar-w-collapsed)]" : "md:w-[var(--sidebar-w)]"
        } ${mobileOpen ? "translate-x-0" : "-translate-x-full"}`}
      >
        <ul className="flex flex-col gap-0.5 px-2 pt-3">{renderGroup({ href: "/dashboard", label: "Visão geral", icon: LayoutDashboard })}</ul>

        {nav.modules.length > 0 && (
          <div className="mt-3">
            <p className={`px-4 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted ${collapsed ? "md:sr-only" : ""}`}>Módulos</p>
            <ul className="flex flex-col gap-0.5 px-2">{nav.modules.map(renderGroup)}</ul>
          </div>
        )}

        {nav.admin.length > 0 && (
          <div className="mt-3">
            <p className={`px-4 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted ${collapsed ? "md:sr-only" : ""}`}>Administração</p>
            <ul className="flex flex-col gap-0.5 px-2 pb-3">{nav.admin.map(renderGroup)}</ul>
          </div>
        )}

        <p className={`mt-auto border-t border-surface-border px-4 py-3 text-[11px] text-muted ${collapsed ? "md:sr-only" : ""}`}>
          {branding.appName}
        </p>
      </nav>
    </>
  );
}
