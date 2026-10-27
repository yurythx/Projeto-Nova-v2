import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { PublicShell } from "@/components/layout/PublicShell";
import { PaginaDeStatus } from "@/components/ocorrencias/PaginaDeStatus";
import { PublicApiError, publicGet } from "@/lib/api/publicServer";
import { APP_URL } from "@/lib/env";
import type { Situacao } from "@/lib/ocorrencias/ocorrencias";

// A mesma janela do cache da API: num incidente esta é a página mais
// acessada, e a notícia não pode ficar velha.
export const revalidate = 30;

const description =
  "Situação dos serviços digitais: o que está fora do ar, desde quando e o que já se sabe.";

export const metadata: Metadata = {
  title: "Situação dos serviços",
  description,
  alternates: { canonical: `${APP_URL}/status` },
};

export default async function StatusPage() {
  let situacao: Situacao;
  try {
    situacao = (await publicGet<Situacao>("status", revalidate)).data;
  } catch (err) {
    // Módulo de ocorrências desligado: a página não existe.
    if (err instanceof PublicApiError && err.status === 404) notFound();
    throw err;
  }

  return (
    <PublicShell>
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-12">
        <div className="flex flex-col gap-3">
          <p className="dateline">Situação dos serviços</p>
          <h1 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Situação dos serviços
          </h1>
          <p className="text-base text-muted">{description}</p>
        </div>
        <PaginaDeStatus situacao={situacao} />
        <p className="text-xs text-muted">
          Horários de Mato Grosso. Recarregue a página para ver a última notícia (o servidor a
          renova a cada 30 segundos).
        </p>
      </div>
    </PublicShell>
  );
}
