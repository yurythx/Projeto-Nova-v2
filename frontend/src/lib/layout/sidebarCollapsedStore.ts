"use client";

import { deleteCookie, readCookie, SIDEBAR_COOKIE, writeCookie } from "@/lib/prefs/cookies";

// "External store" (useSyncExternalStore) do estado "Sidebar recolhida",
// uma preferência puramente deste dispositivo/navegador (DashboardShell.tsx).
//
// Persiste num COOKIE, não em localStorage: o layout do servidor lê o mesmo
// cookie e já renderiza o shell recolhido no 1º paint. O DashboardShell
// passa esse valor como server snapshot do useSyncExternalStore, então o
// SSR e o cliente coincidem — sem a Sidebar "expandir e recolher" a cada
// refresh. Mesmo padrão do tema (ver components/ui/ThemeToggle.tsx).
//
// Não é useState+useEffect porque ler a preferência e então chamar setState
// num efeito é o anti-padrão que react-hooks/set-state-in-effect
// desencoraja; useSyncExternalStore é a primitiva certa para "estado fora
// do React que precisa re-renderizar quando muda".

export const SIDEBAR_COLLAPSED_COOKIE = SIDEBAR_COOKIE;

type Listener = () => void;
const listeners = new Set<Listener>();
let cached: boolean | null = null;

function read(): boolean {
  if (cached !== null) return cached;
  cached = readCookie(SIDEBAR_COLLAPSED_COOKIE) === "true";
  return cached;
}

export function getSidebarCollapsedSnapshot(): boolean {
  return read();
}

// Fallback para contextos sem o cookie (SSR sem preferência, testes). O
// DashboardShell passa o seu próprio server snapshot, derivado do cookie
// lido no layout.
export function getSidebarCollapsedServerSnapshot(): boolean {
  return false;
}

export function setSidebarCollapsed(next: boolean): void {
  cached = next;
  if (next === false) {
    // Estado padrão: não deixa cookie sobrando (o layout já assume
    // "expandida" sem cookie).
    deleteCookie(SIDEBAR_COLLAPSED_COOKIE);
  } else {
    writeCookie(SIDEBAR_COLLAPSED_COOKIE, "true");
  }
  listeners.forEach((listener) => listener());
}

export function subscribeSidebarCollapsed(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}
