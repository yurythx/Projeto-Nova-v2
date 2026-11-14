import type { Metadata } from "next";

import { PublicShell } from "@/components/layout/PublicShell";

import { Simulador } from "./Simulador";

export const metadata: Metadata = {
  title: "Simulador do Regulariza",
  description: "Consulta pública de débitos com remissão, parcelas e ITBI estimado. Não abre processo.",
};

export default function RegularizaPage() {
  return (
    <PublicShell>
      <Simulador />
    </PublicShell>
  );
}
