import { Boxes } from "lucide-react";
import type { Metadata } from "next";
import { connection } from "next/server";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { PublicShell } from "@/components/layout/PublicShell";
import { getServerBranding } from "@/lib/branding/server";
import { FUNDAMENTOS, PILARES, PRATICAS, PRODUTO, REGRAS_PUBLICAS } from "@/lib/plataforma/conteudo";

const description = PRODUTO.finalidade;

export const metadata: Metadata = {
  title: "Sobre",
  description,
  openGraph: { title: `Sobre o ${PRODUTO.nome}`, description, type: "website" },
};

export default async function SobrePage() {
  await connection();
  const brand = await getServerBranding().catch(() => null);

  return (
    <PublicShell>
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-10 px-6 py-12">
        <div className="flex flex-col gap-3">
          <p className="dateline">{brand?.orgName || PRODUTO.nome}</p>
          <h1 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            {brand?.appName || PRODUTO.nome}
          </h1>
          <p className="max-w-3xl text-base leading-relaxed text-muted">{description}</p>
        </div>
        <section className="flex flex-col gap-4" aria-labelledby="praticas-sobre">
          <h2 id="praticas-sobre" className="text-xl font-bold text-foreground">Como o serviço funciona</h2>
          <div className="grid gap-4">
            {PRATICAS.map((pratica) => (
              <Card key={pratica.nome}>
                <CardHeader className="flex flex-row items-center gap-2">
                  <Boxes size={18} aria-hidden="true" className="text-primary" />
                  <CardTitle>{pratica.nome}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm leading-relaxed text-muted">{pratica.descricao}</CardContent>
              </Card>
            ))}
          </div>
        </section>
        <section className="flex flex-col gap-4" aria-labelledby="fundamentos-sobre">
          <h2 id="fundamentos-sobre" className="text-xl font-bold text-foreground">Objetivo e fundamentos</h2>
          <div className="grid gap-4 md:grid-cols-2">
            {FUNDAMENTOS.map((item) => (
              <Card key={item.nome}>
                <CardHeader>
                  <CardTitle>{item.nome}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm leading-relaxed text-muted">{item.descricao}</CardContent>
              </Card>
            ))}
          </div>
        </section>
        <section className="flex flex-col gap-4" aria-labelledby="regras-sobre">
          <h2 id="regras-sobre" className="text-xl font-bold text-foreground">Regras e políticas</h2>
          <div className="grid gap-4 md:grid-cols-2">
            {REGRAS_PUBLICAS.map((regra) => (
              <Card key={regra.nome}>
                <CardHeader>
                  <CardTitle>{regra.nome}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm leading-relaxed text-muted">{regra.descricao}</CardContent>
              </Card>
            ))}
          </div>
        </section>
        <section className="flex flex-col gap-4" aria-labelledby="nucleo-sobre">
          <h2 id="nucleo-sobre" className="text-xl font-bold text-foreground">Como a plataforma se organiza</h2>
          <div className="grid gap-4">
            {PILARES.map((pilar) => (
              <Card key={pilar.nome}>
                <CardHeader>
                  <CardTitle>{pilar.nome}</CardTitle>
                </CardHeader>
                <CardContent className="text-sm leading-relaxed text-muted">{pilar.descricao}</CardContent>
              </Card>
            ))}
          </div>
        </section>
      </div>
    </PublicShell>
  );
}
