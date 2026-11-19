import type { Metadata } from "next";

import { PublicShell } from "@/components/layout/PublicShell";

import { AbrirPedido } from "./AbrirPedido";

export const metadata: Metadata = {
  title: "Abrir pedido",
  description: "Formulário público da carta de serviços.",
};

export default async function AbrirPedidoPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return (
    <PublicShell>
      <AbrirPedido id={id} />
    </PublicShell>
  );
}
