import type { Metadata } from "next";

import { PublicShell } from "@/components/layout/PublicShell";

import { Retorno } from "../Retorno";

export const metadata: Metadata = {
  title: "Retorno do gov.br",
  description: "Conclusão do Login Único do contribuinte. O CPF não aparece.",
};

export default async function RetornoPage({
  searchParams,
}: {
  searchParams: Promise<{ code?: string; state?: string }>;
}) {
  const params = await searchParams;
  return (
    <PublicShell>
      <Retorno code={params.code ?? ""} state={params.state ?? ""} />
    </PublicShell>
  );
}
