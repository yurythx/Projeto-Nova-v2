import type { ReactNode } from "react";
import { cookies } from "next/headers";

import { PublicShellClient } from "@/components/layout/PublicShellClient";
import { getFeatures } from "@/lib/features/getFeatures";
import { THEME_COOKIE } from "@/lib/prefs/cookies";

/** Shell do site institucional. Server Component: resolve o estado dos
 * módulos no servidor e o tema persistido em cookie para o menu e ícones
 * saírem corretos já no 1º HTML (sem flash). */
export async function PublicShell({
  children,
  initialEnabled: explicitEnabled,
}: {
  children: ReactNode;
  initialEnabled?: Record<string, boolean> | null;
}) {
  let initialTheme: "light" | "dark" | undefined;
  try {
    const cookieStore = await cookies();
    const theme = cookieStore.get(THEME_COOKIE)?.value;
    if (theme === "dark" || theme === "light") {
      initialTheme = theme;
    }
  } catch {
    // Fora de escopo de requisição (ex: testes sem mock de headers)
  }

  const featuresResult =
    explicitEnabled !== undefined ? { features: explicitEnabled ?? {} } : await getFeatures();

  const features = explicitEnabled !== undefined ? explicitEnabled : featuresResult.features;
  const initialEnabled = features && Object.keys(features).length > 0 ? features : null;

  return (
    <PublicShellClient initialEnabled={initialEnabled} initialTheme={initialTheme}>
      {children}
    </PublicShellClient>
  );
}
