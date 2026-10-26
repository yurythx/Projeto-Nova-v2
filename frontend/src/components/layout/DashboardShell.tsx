"use client";

import { Suspense, useState, useSyncExternalStore, type ReactNode } from "react";
import Link from "next/link";
import { Globe } from "lucide-react";

import { AuthFlashToast } from "@/components/layout/AuthFlashToast";
import { ConnectionStateProvider } from "@/components/layout/ConnectionStateContext";
import { GovFooter } from "@/components/layout/GovFooter";
import { GovHeader } from "@/components/layout/GovHeader";
import { LGPDConsentModal } from "@/components/layout/LGPDConsentModal";
import { ModuleGate } from "@/components/layout/ModuleGate";
import { Sidebar } from "@/components/layout/Sidebar";
import { UserMenu } from "@/components/layout/UserMenu";
import { NotificationBell } from "@/components/notifications/NotificationBell";
import { NotificationCenter } from "@/components/notifications/NotificationCenter";
import { NotificationHistoryProvider } from "@/components/notifications/NotificationHistoryProvider";
import { ToastProvider } from "@/components/notifications/ToastProvider";
import { RealtimeProvider, useRealtimeState } from "@/components/realtime/RealtimeProvider";
import { ThemeToggle } from "@/components/ui/ThemeToggle";
import {
  getSidebarCollapsedSnapshot,
  setSidebarCollapsed,
  subscribeSidebarCollapsed,
} from "@/lib/layout/sidebarCollapsedStore";
import { PlataformaProvider } from "@/lib/plataforma/PlataformaProvider";
import { CONNECTION_LABEL, CONNECTION_TONE } from "@/lib/websocket/connectionCopy";

const MD_BREAKPOINT_QUERY = "(min-width: 768px)";

function ConnectionIndicator() {
  const state = useRealtimeState();
  return (
    <span className="hidden items-center gap-2 text-xs text-muted lg:flex" aria-live="polite">
      <span
        className={`h-2 w-2 rounded-full ${CONNECTION_TONE[state].dotClass}`}
        aria-hidden="true"
      />
      {CONNECTION_LABEL[state]}
    </span>
  );
}

function ShellBody({
  userLabel,
  initialTheme,
  initialCollapsed,
  children,
}: {
  userLabel: string;
  initialTheme?: "light" | "dark";
  initialCollapsed: boolean;
  children: ReactNode;
}) {
  const state = useRealtimeState();
  const collapsed = useSyncExternalStore(
    subscribeSidebarCollapsed,
    getSidebarCollapsedSnapshot,
    () => initialCollapsed,
  );
  const [mobileOpen, setMobileOpen] = useState(false);

  function toggle() {
    if (window.matchMedia(MD_BREAKPOINT_QUERY).matches) setSidebarCollapsed(!collapsed);
    else setMobileOpen((v) => !v);
  }

  return (
    <>
      <Suspense fallback={null}>
        <AuthFlashToast />
      </Suspense>
      <GovHeader
        homeHref="/dashboard"
        searchAction="/busca"
        onToggleMenu={toggle}
        menuExpanded={mobileOpen || !collapsed}
        actions={
          <>
            <Link
              href="/"
              className="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 text-xs font-semibold text-muted transition-colors hover:bg-surface-hover hover:text-foreground"
              title="Acessar portal público inicial"
            >
              <Globe size={14} aria-hidden="true" />
              <span className="hidden sm:inline">Site Público</span>
            </Link>
            <ConnectionIndicator />
            <ThemeToggle initialTheme={initialTheme} />
            <NotificationBell />
            <UserMenu userLabel={userLabel} />
          </>
        }
      />
      <Sidebar
        collapsed={collapsed}
        mobileOpen={mobileOpen}
        onCloseMobile={() => setMobileOpen(false)}
      />
      <div
        className={`flex min-h-dvh flex-col pt-[var(--topbar-h)] transition-[padding] duration-[var(--shell-motion)] ${
          collapsed ? "md:pl-[var(--sidebar-w-collapsed)]" : "md:pl-[var(--sidebar-w)]"
        }`}
      >
        <main
          id="main-content"
          tabIndex={-1}
          className="min-w-0 flex-1 px-4 pb-10 pt-6 outline-none sm:px-8"
        >
          <ConnectionStateProvider value={state}>
            <ModuleGate>{children}</ModuleGate>
          </ConnectionStateProvider>
        </main>
        <GovFooter />
      </div>
      <NotificationCenter />
      <LGPDConsentModal />
    </>
  );
}

/** Shell da área autenticada: sessão Projeto-nova-v2 (/me + módulos), conexão de
 * tempo real, cabeçalho/rodapé DSGov e menu montado pelo Kernel. */
export function DashboardShell({
  userLabel,
  initialTheme,
  initialCollapsed = false,
  children,
}: {
  userLabel: string;
  initialTheme?: "light" | "dark";
  initialCollapsed?: boolean;
  children: ReactNode;
}) {
  return (
    <PlataformaProvider>
      <RealtimeProvider>
        <ToastProvider>
          <NotificationHistoryProvider>
            <ShellBody
              userLabel={userLabel}
              initialTheme={initialTheme}
              initialCollapsed={initialCollapsed}
            >
              {children}
            </ShellBody>
          </NotificationHistoryProvider>
        </ToastProvider>
      </RealtimeProvider>
    </PlataformaProvider>
  );
}
