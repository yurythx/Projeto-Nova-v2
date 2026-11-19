import type { Metadata } from "next";

import { PublicShell } from "@/components/layout/PublicShell";

import { CartaPublica } from "./CartaPublica";

export const metadata: Metadata = {
  title: "Carta de serviços",
  description: "Abra um pedido e consulte o andamento pelo número, sem CPF.",
};

export default function CartaPage() {
  return (
    <PublicShell>
      <CartaPublica />
    </PublicShell>
  );
}
