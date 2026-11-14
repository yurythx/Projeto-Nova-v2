import type { Metadata } from "next";

import { PublicShell } from "@/components/layout/PublicShell";

import { Entrada } from "./Entrada";

export const metadata: Metadata = {
  title: "Entrar com gov.br",
  description: "Início do Login Único do contribuinte no protocolo da Fazenda.",
};

export default function GovBRPage() {
  return (
    <PublicShell>
      <Entrada />
    </PublicShell>
  );
}
