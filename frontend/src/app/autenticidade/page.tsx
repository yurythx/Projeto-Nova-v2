import type { Metadata } from "next";

import { PublicShell } from "@/components/layout/PublicShell";

import { Consulta } from "./Consulta";

export const metadata: Metadata = {
  title: "Autenticidade",
  description: "Confere o código do protocolo da Fazenda. O QR é gerado no navegador.",
};

export default function AutenticidadePage() {
  return (
    <PublicShell>
      <Consulta />
    </PublicShell>
  );
}
