import type { Metadata } from "next";
import Link from "next/link";
import { ArrowRight, HelpCircle, MessageSquare } from "lucide-react";

import { PublicShell } from "@/components/layout/PublicShell";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { APP_URL } from "@/lib/env";
import { FAQ } from "@/lib/plataforma/conteudo";

const description = "Perguntas frequentes sobre o Projeto-nova-v2, o microkernel e os plug-ins que nascem desligados.";

export const metadata: Metadata = {
  title: "Perguntas Frequentes (FAQ)",
  description,
  alternates: { canonical: `${APP_URL}/faq` },
  openGraph: {
    title: "Perguntas Frequentes (FAQ)",
    description,
    type: "website",
    url: `${APP_URL}/faq`,
  },
};

export default function FAQPage() {
  return (
    <PublicShell>
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-12 px-6 py-12">
        <div className="flex flex-col gap-3">
          <p className="dateline">Central de Ajuda & Dúvidas</p>
          <h1 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Perguntas Frequentes (FAQ)
          </h1>
          <p className="max-w-2xl text-base text-muted">{description}</p>
        </div>

        <div className="flex flex-col gap-10">
          {FAQ.map((sec) => (
            <section key={sec.categoria} className="flex flex-col gap-4">
              <h2 className="text-xl font-bold text-foreground border-b border-surface-border pb-2">
                {sec.categoria}
              </h2>
              <div className="grid gap-4">
                {sec.itens.map((item, idx) => (
                  <Card key={idx} className="border-surface-border">
                    <CardHeader className="flex flex-row items-start gap-3 pb-2">
                      <HelpCircle size={20} className="mt-0.5 text-accent shrink-0" aria-hidden="true" />
                      <CardTitle as="h3" className="text-base font-semibold text-foreground">
                        {item.q}
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="flex flex-col gap-2 pl-4 sm:pl-11 text-sm leading-relaxed text-muted">
                      <p>{item.a}</p>
                      <p className="text-xs font-medium text-foreground/70">Base: {item.fonte}</p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </section>
          ))}
        </div>

        {/* CTA */}
        <div className="rounded-xl border border-surface-border bg-surface-2 p-6 sm:p-8 text-center sm:text-left sm:flex sm:items-center sm:justify-between">
          <div className="flex flex-col gap-1">
            <h3 className="text-lg font-bold text-foreground">Ainda tem dúvidas?</h3>
            <p className="text-sm text-muted">Entre na plataforma ou leia como o núcleo e os plug-ins se organizam.</p>
          </div>
          <div className="mt-4 sm:mt-0 flex flex-wrap shrink-0 items-center justify-center sm:justify-start gap-3">
            <Link
              href="/login"
              className="inline-flex items-center gap-2 rounded-lg bg-accent px-5 py-2.5 text-sm font-semibold text-accent-foreground shadow-xs hover:bg-accent/90"
            >
              <MessageSquare size={16} aria-hidden="true" /> Entrar
            </Link>
            <Link href="/sobre" className="inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline">
              Sobre o núcleo <ArrowRight size={14} aria-hidden="true" />
            </Link>
          </div>
        </div>
      </div>
    </PublicShell>
  );
}
