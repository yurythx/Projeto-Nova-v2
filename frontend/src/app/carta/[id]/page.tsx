import type { Metadata } from "next";

import { PublicShell } from "@/components/layout/PublicShell";

import { AbrirPedido } from "./AbrirPedido";

export const metadata: Metadata = {
  title: "Abrir pedido",
  description: "Formulário público da carta de serviços.",
};

export default async function AbrirPedidoPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ pedir?: string }>;
}) {
  const { id } = await params;
  const busca = await searchParams;
  return (
    <PublicShell>
      <AbrirPedido id={id} pedirInicial={busca.pedir === "1"} />
    </PublicShell>
  );
}
