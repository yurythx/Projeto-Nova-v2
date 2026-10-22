import { ShieldCheck } from "lucide-react";
import { connection } from "next/server";
import { cookies } from "next/headers";
import Link from "next/link";
import { Suspense } from "react";

import { LoginCard } from "@/components/auth/LoginCard";
import { getServerBranding } from "@/lib/branding/server";
import { AccessibilityBar } from "@/components/layout/AccessibilityBar";
import { Logo } from "@/components/ui/Logo";
import { Seal } from "@/components/ui/Seal";
import { ThemeToggle } from "@/components/ui/ThemeToggle";
import { pickCookie, THEME_COOKIE, THEME_COOKIE_LEGACY } from "@/lib/prefs/cookies";

// O painel de marca descreve exatamente o que a plataforma faz para um
// fiscal de contrato — sem linguagem de vendedor ("enterprise", "a
// tecnologia mais avançada do mercado"), que a versão anterior tinha
// herdado de um template.
const capabilities = [
  "Núcleo de identidade (IAM) e trilha de auditoria, sempre ligados.",
  "Plug-ins que nascem desligados e entram quando a instalação liga.",
  "Cada módulo fala com os outros só pelos contratos do núcleo.",
  "Acessibilidade e-MAG e proteção de dados pessoais (LGPD).",
];

export default async function LoginPage() {
  await connection();
  const brand = await getServerBranding();

  const cookieStore = await cookies();
  const themeCookie = pickCookie(cookieStore, THEME_COOKIE, THEME_COOKIE_LEGACY);
  const initialTheme = themeCookie === "dark" || themeCookie === "light" ? themeCookie : undefined;

  return (
    <div className="flex min-h-screen flex-col pt-10">
      <header className="fixed inset-x-0 top-0 z-50">
        <AccessibilityBar showSearchShortcut={false} />
      </header>

      <main id="main-content" tabIndex={-1} className="flex min-h-0 flex-1 outline-none">
      <div
        className="relative hidden overflow-hidden bg-brand-panel text-white lg:flex lg:w-[44%] lg:flex-col lg:justify-between lg:p-12"
      >
        <Seal
          size={520}
          decorative
          className="pointer-events-none absolute -bottom-40 -right-40 text-white/[0.06]"
        />

        <div className="relative flex items-center gap-2.5 text-lg font-semibold tracking-wide">
          <Logo size={32} />
          {brand.appName || "Projeto-nova-v2"}
        </div>

        <div className="relative flex flex-col gap-6">
          <p className="dateline text-accent font-semibold tracking-wider text-xs uppercase">{brand.orgName || "Projeto-nova-v2"}</p>
          <h1 className="max-w-md text-3xl font-extrabold leading-tight text-white sm:text-4xl">
            Acesso ao microkernel
          </h1>
          <p className="max-w-sm text-sm text-white/75 leading-relaxed">
            {brand.appDescription || "Entre com sua conta corporativa unificada (SSO Keycloak) ou credencial administrativa local."}
          </p>
          <ul className="mt-2 flex flex-col gap-3.5 text-sm text-white/90">
            {capabilities.map((item) => (
              <li key={item} className="flex items-start gap-3 leading-snug">
                <span aria-hidden="true" className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-accent shadow-xs" />
                <span>{item}</span>
              </li>
            ))}
          </ul>
        </div>

        <p className="relative flex items-center gap-2 text-xs text-white/60">
          <ShieldCheck size={15} className="text-accent" aria-hidden="true" />
          Entre com sua conta corporativa (SSO) ou credencial local.
        </p>
      </div>

      {/* Painel do formulário */}
      <div className="relative flex flex-1 flex-col items-center justify-center px-4 py-12 sm:p-8">
        <div className="absolute right-4 top-4">
          <ThemeToggle initialTheme={initialTheme} />
        </div>

        <Link href="/" className="mb-6 sm:mb-8 flex items-center gap-2 text-lg font-semibold lg:hidden">
          <Logo size={30} />
          {brand.appName}
        </Link>

        <Suspense fallback={null}>
          <LoginCard />
        </Suspense>

        <Link href="/" className="mt-8 text-xs text-muted hover:text-foreground">
          ← Voltar para a página inicial
        </Link>
      </div>
      </main>

      <footer id="gov-footer" tabIndex={-1} className="sr-only">
        Rodapé da página de login
      </footer>
    </div>
  );
}
