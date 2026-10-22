import { Boxes } from "lucide-react";
import type { Metadata } from "next";
import { connection } from "next/server";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { PublicShell } from "@/components/layout/PublicShell";
import { getServerBranding } from "@/lib/branding/server";
import { PILARES, PRODUTO } from "@/lib/sismarq/conteudo";

const description = PRODUTO.finalidade;

export const metadata: Metadata = {
  title: "Sobre",
  description,
  openGraph: { title: "Sobre o Projeto-nova-v2", description, type: "website" },
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
        <div className="grid gap-4">
          {PILARES.map((pilar) => (
            <Card key={pilar.nome}>
              <CardHeader className="flex flex-row items-center gap-2">
                <Boxes size={18} aria-hidden="true" className="text-primary" />
                <CardTitle>{pilar.nome}</CardTitle>
              </CardHeader>
              <CardContent className="text-sm leading-relaxed text-muted">{pilar.descricao}</CardContent>
            </Card>
          ))}
        </div>
      </div>
    </PublicShell>
  );
}
