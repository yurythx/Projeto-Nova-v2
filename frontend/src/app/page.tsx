import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight, Boxes, Landmark, ShieldCheck } from "lucide-react";

import { PublicShell } from "@/components/layout/PublicShell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { getServerBranding } from "@/lib/branding/server";
import { APP_URL } from "@/lib/env";
import { FUNDAMENTOS, PRATICAS, PRODUTO, REGRAS_PUBLICAS } from "@/lib/plataforma/conteudo";

export const dynamic = "force-dynamic";

const description = PRODUTO.finalidade;

export const metadata: Metadata = {
  title: "Início",
  description,
  alternates: { canonical: `${APP_URL}` },
  openGraph: { title: `Início — ${PRODUTO.nome}`, description, type: "website", url: `${APP_URL}` },
};

export default async function HomePage() {
  const brand = await getServerBranding().catch(() => null);

  return (
    <PublicShell>
      <div className="flex flex-col gap-16 pb-20">
        <section className="border-b border-surface-border bg-gradient-to-b from-surface-1 to-surface-2/40 px-6 py-20 sm:py-24">
          <div className="mx-auto flex max-w-5xl flex-col items-center text-center">
            <span className="inline-flex items-center gap-2 rounded-full border border-surface-border bg-surface px-4 py-1.5 text-xs font-semibold text-primary shadow-xs">
              <Landmark size={14} aria-hidden="true" /> {brand?.orgName || PRODUTO.nome}
            </span>
            <h1 className="mt-6 text-4xl font-extrabold tracking-tight text-foreground sm:text-5xl">
              {brand?.appName || PRODUTO.nome}
            </h1>
            <p className="mt-6 max-w-3xl text-lg leading-relaxed text-muted">{description}</p>
            <div className="mt-10 flex w-full flex-col items-stretch justify-center gap-3 sm:w-auto sm:flex-row sm:items-center">
              <Link
                href="/login"
                className="inline-flex items-center justify-center gap-2 rounded-xl bg-accent px-6 py-3.5 text-base font-semibold text-accent-foreground shadow-md hover:bg-accent/90"
              >
                <ShieldCheck size={18} aria-hidden="true" /> Entrar
              </Link>
              <Link
                href="/sobre"
                className="inline-flex items-center justify-center gap-2 rounded-xl border border-surface-border bg-surface px-6 py-3.5 text-base font-semibold text-foreground shadow-xs hover:bg-surface-2"
              >
                <Boxes size={18} aria-hidden="true" /> Como funciona
              </Link>
            </div>
          </div>
        </section>

        <section
          className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6"
          aria-labelledby="praticas"
        >
          <div className="flex flex-col gap-2">
            <p className="dateline">ITIL (Version 5)</p>
            <h2 id="praticas" className="text-3xl font-bold tracking-tight text-foreground">
              Como o serviço funciona
            </h2>
            <p className="max-w-3xl text-sm leading-relaxed text-muted">
              Quatro práticas carregam a operação. O incidente e a requisição atendem o agora. O
              problema trata a causa. A mudança altera com controle. O projeto conduz a iniciativa.
            </p>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            {PRATICAS.map((pratica) => (
              <Card key={pratica.nome}>
                <CardHeader>
                  <CardTitle>{pratica.nome}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm leading-relaxed text-muted">
                  {pratica.descricao}
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        <section
          className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6"
          aria-labelledby="fundamentos"
        >
          <div className="flex flex-col gap-2">
            <p className="dateline">Fundamentos</p>
            <h2 id="fundamentos" className="text-3xl font-bold tracking-tight text-foreground">
              Objetivo, princípios e ciclo
            </h2>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            {FUNDAMENTOS.map((item) => (
              <Card key={item.nome}>
                <CardHeader>
                  <CardTitle>{item.nome}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm leading-relaxed text-muted">
                  {item.descricao}
                </CardContent>
              </Card>
            ))}
          </div>
        </section>

        <section
          className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6"
          aria-labelledby="regras"
        >
          <div className="flex flex-col gap-2">
            <p className="dateline">Política de uso</p>
            <h2 id="regras" className="text-3xl font-bold tracking-tight text-foreground">
              Regras que a operação segue
            </h2>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            {REGRAS_PUBLICAS.map((regra) => (
              <Card key={regra.nome}>
                <CardHeader>
                  <CardTitle>{regra.nome}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm leading-relaxed text-muted">
                  {regra.descricao}
                </CardContent>
              </Card>
            ))}
          </div>
          <Link
            href="/faq"
            className="inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline"
          >
            Perguntas frequentes <ArrowRight size={14} aria-hidden="true" />
          </Link>
        </section>
      </div>
    </PublicShell>
  );
}
