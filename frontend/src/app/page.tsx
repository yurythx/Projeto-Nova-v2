import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight, Boxes, Landmark, ShieldCheck } from "lucide-react";

import { PublicShell } from "@/components/layout/PublicShell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { getServerBranding } from "@/lib/branding/server";
import { APP_URL } from "@/lib/env";
import { PILARES, PRODUTO } from "@/lib/sismarq/conteudo";

export const dynamic = "force-dynamic";

const description = PRODUTO.finalidade;

export const metadata: Metadata = {
  title: "Início",
  description,
  alternates: { canonical: `${APP_URL}` },
  openGraph: { title: "Início — Projeto-nova-v2", description, type: "website", url: `${APP_URL}` },
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
                <Boxes size={18} aria-hidden="true" /> Como o núcleo funciona
              </Link>
            </div>
          </div>
        </section>

        <section className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-6" aria-labelledby="pilares">
          <div className="flex flex-col gap-2">
            <p className="dateline">Microkernel</p>
            <h2 id="pilares" className="text-3xl font-bold tracking-tight text-foreground">
              O que a plataforma entrega
            </h2>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            {PILARES.map((pilar) => (
              <Card key={pilar.nome}>
                <CardHeader>
                  <CardTitle>{pilar.nome}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm leading-relaxed text-muted">{pilar.descricao}</CardContent>
              </Card>
            ))}
          </div>
          <Link href="/faq" className="inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline">
            Perguntas frequentes <ArrowRight size={14} aria-hidden="true" />
          </Link>
        </section>
      </div>
    </PublicShell>
  );
}
