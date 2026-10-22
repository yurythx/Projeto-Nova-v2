"use client";

import { Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";

import { deleteCookie, THEME_COOKIE, THEME_COOKIE_LEGACY } from "@/lib/prefs/cookies";
import { usePrefersDark } from "@/lib/theme/usePrefersDark";

const COOKIE_NAME = THEME_COOKIE;

// Alterna entre claro/escuro persistindo a escolha num cookie (lido
// server-side em app/layout.tsx e PublicShell) e atualiza instantaneamente
// o atributo [data-theme] e a classe .dark do Tailwind em document.documentElement.
export function ThemeToggle({ initialTheme }: { initialTheme?: "light" | "dark" }) {
  const prefersDark = usePrefersDark();
  const [explicit, setExplicit] = useState<"light" | "dark" | null>(() => {
    if (initialTheme) return initialTheme;
    if (typeof document !== "undefined") {
      const fromAttr = document.documentElement.getAttribute("data-theme");
      if (fromAttr === "dark" || fromAttr === "light") return fromAttr;
      const match =
        document.cookie.match(new RegExp(`(?:^|; )${COOKIE_NAME}=([^;]*)`)) ??
        document.cookie.match(new RegExp(`(?:^|; )${THEME_COOKIE_LEGACY}=([^;]*)`));
      if (match && (match[1] === "dark" || match[1] === "light")) return match[1] as "light" | "dark";
    }
    return null;
  });

  const theme = explicit ?? (prefersDark ? "dark" : "light");

  // Garante sincronização imediata no cliente com as classes Tailwind e o DOM
  useEffect(() => {
    document.documentElement.setAttribute("data-theme", theme);
    document.documentElement.classList.toggle("dark", theme === "dark");
  }, [theme]);

  function toggle() {
    const next = theme === "dark" ? "light" : "dark";
    setExplicit(next);
    document.documentElement.setAttribute("data-theme", next);
    document.documentElement.classList.toggle("dark", next === "dark");
    // 1 ano, path=/ — persistência lida em Server Components e no próximo reload
    document.cookie = `${COOKIE_NAME}=${next}; path=/; max-age=31536000; samesite=lax`;
    deleteCookie(THEME_COOKIE_LEGACY);
    try {
      localStorage.setItem(COOKIE_NAME, next);
    } catch {
      // storage pode estar indisponível
    }
  }

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={theme === "dark" ? "Ativar tema claro" : "Ativar tema escuro"}
      title={theme === "dark" ? "Ativar tema claro" : "Ativar tema escuro"}
      className="inline-flex h-10 w-10 items-center justify-center rounded-lg text-muted transition-colors hover:bg-black/5 hover:text-foreground dark:hover:bg-white/5"
    >
      {theme === "dark" ? <Sun size={18} aria-hidden="true" /> : <Moon size={18} aria-hidden="true" />}
    </button>
  );
}
