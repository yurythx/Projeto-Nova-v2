"use client";

import Link from "next/link";
import { useSession } from "next-auth/react";
import { Suspense, useEffect, useState, type ReactNode } from "react";

import { AuthFlashToast } from "@/components/layout/AuthFlashToast";
import { GovFooter } from "@/components/layout/GovFooter";
import { GovHeader } from "@/components/layout/GovHeader";
import { LGPDConsentModal } from "@/components/layout/LGPDConsentModal";
import { ToastProvider } from "@/components/notifications/ToastProvider";
import { Button } from "@/components/ui/Button";
import { ThemeToggle } from "@/components/ui/ThemeToggle";
import { publicFetch } from "@/lib/api/publicClient";
import { DoorOpen, X } from "lucide-react";
import type { PublicModule } from "@/lib/sismarq/types";

const PUBLIC_LINKS: { href: string; label: string; module?: string }[] = [
  { href: "/servicos", label: "Serviços", module: "catalog" },
  { href: "/eventos", label: "Agenda", module: "calendar" },
  { href: "/blog", label: "Blog & Novidades", module: "blog" },
  { href: "/faq", label: "FAQ" },
  { href: "/sobre", label: "Sobre" },
  { href: "/contato", label: "Contato", module: "contact" },
];

/** Shell do site institucional (visitante anônimo). Os links de plugins
 * desativados no Kernel somem do menu.
 *
 * initialEnabled vem do servidor (PublicShell) já no 1º HTML. Antes o
 * estado só chegava por fetch no navegador e, enquanto isso, TODOS os
 * links eram mostrados — a cada F5 os plugins desativados piscavam no
 * menu. Sem estado conhecido, links de plugin ficam escondidos; o fetch
 * aqui só atualiza um estado do servidor em cache (até 15s). */
export function PublicShellClient({
  children,
  initialEnabled = null,
  initialTheme,
}: {
  children: ReactNode;
  initialEnabled?: Record<string, boolean> | null;
  initialTheme?: "light" | "dark";
}) {
  const { status } = useSession();
  const [enabled, setEnabled] = useState<Record<string, boolean> | null>(initialEnabled);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    publicFetch<PublicModule[]>("v1/system/public-modules")
      .then((mods) => setEnabled(Object.fromEntries(mods.map((m) => [m.key, m.enabled]))))
      .catch(() => setEnabled((prev) => prev ?? {}));
  }, []);

  useEffect(() => {
    if (!mobileMenuOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") setMobileMenuOpen(false);
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [mobileMenuOpen]);

  const links = PUBLIC_LINKS.filter((l) => !l.module || Boolean(enabled?.[l.module]));

  return (
    <ToastProvider>
      <div className="flex min-h-screen flex-col pt-[var(--topbar-h)]">
        <Suspense fallback={null}>
          <AuthFlashToast />
        </Suspense>
        <GovHeader
          homeHref="/"
          onToggleMenu={() => setMobileMenuOpen((v) => !v)}
          menuExpanded={mobileMenuOpen}
          nav={
            <nav
              id="main-menu"
              tabIndex={-1}
              aria-label="Menu principal"
              className="ml-4 hidden outline-none lg:block"
            >
              <ul className="flex items-center gap-5 text-sm">
                {links.map((l) => (
                  <li key={l.href}>
                    <Link
                      href={l.href}
                      className="text-muted transition-colors hover:text-foreground"
                    >
                      {l.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          }
          actions={
            <>
              <ThemeToggle initialTheme={initialTheme} />
              <Link
                href={status === "authenticated" ? "/dashboard" : "/login"}
                title={status === "authenticated" ? "Acessar o painel" : "Entrar no sistema"}
              >
                <Button
                  size="sm"
                  variant="secondary"
                  className="flex h-10 w-10 items-center justify-center p-0 rounded-lg border-surface-border text-foreground hover:bg-surface-2"
                  aria-label={status === "authenticated" ? "Acessar o painel" : "Entrar"}
                >
                  <DoorOpen size={18} aria-hidden="true" />
                </Button>
              </Link>
            </>
          }
        />

        {/* Backdrop e gaveta móvel de navegação */}
        {mobileMenuOpen && (
          <div
            className="fixed inset-0 top-[var(--topbar-h)] z-40 bg-black/50 backdrop-blur-xs transition-opacity lg:hidden"
            onClick={() => setMobileMenuOpen(false)}
            aria-hidden="true"
          />
        )}
        <div
          id="mobile-drawer"
          aria-label="Menu móvel"
          className={`fixed inset-y-0 left-0 top-[var(--topbar-h)] z-50 flex w-72 max-w-[85vw] flex-col justify-between border-r border-surface-border bg-surface p-5 shadow-2xl transition-transform duration-200 ease-out lg:hidden ${
            mobileMenuOpen ? "translate-x-0" : "-translate-x-full pointer-events-none"
          }`}
        >
          <div className="flex flex-col gap-5">
            <div className="flex items-center justify-between border-b border-surface-border pb-3">
              <span className="text-xs font-bold uppercase tracking-wider text-muted">
                Navegação
              </span>
              <button
                type="button"
                onClick={() => setMobileMenuOpen(false)}
                className="flex h-8 w-8 items-center justify-center rounded-md text-muted hover:bg-surface-hover hover:text-foreground"
                aria-label="Fechar menu"
              >
                <X size={18} aria-hidden="true" />
              </button>
            </div>

            <nav aria-label="Links móveis">
              <ul className="flex flex-col gap-1">
                {links.map((l) => (
                  <li key={l.href}>
                    <Link
                      href={l.href}
                      onClick={() => setMobileMenuOpen(false)}
                      className="flex min-h-[44px] items-center rounded-lg px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-surface-hover hover:text-primary active:bg-surface-2"
                    >
                      {l.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          </div>

          <div className="flex flex-col gap-2.5 border-t border-surface-border pt-4">
            <Link
              href={status === "authenticated" ? "/dashboard" : "/login"}
              onClick={() => setMobileMenuOpen(false)}
              className="flex min-h-[44px] w-full items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-xs transition-colors hover:bg-primary/90"
            >
              <DoorOpen size={18} aria-hidden="true" />
              <span>{status === "authenticated" ? "Acessar o Painel" : "Entrar no Sistema"}</span>
            </Link>
            <Link
              href="/contato"
              onClick={() => setMobileMenuOpen(false)}
              className="flex min-h-[44px] w-full items-center justify-center rounded-lg border border-surface-border bg-surface-2 px-4 py-2 text-xs font-medium text-muted transition-colors hover:text-foreground"
            >
              Central de Atendimento
            </Link>
          </div>
        </div>

        <main id="main-content" tabIndex={-1} className="flex-1 outline-none">
          {children}
        </main>
        <GovFooter />
        <LGPDConsentModal />
      </div>
    </ToastProvider>
  );
}
