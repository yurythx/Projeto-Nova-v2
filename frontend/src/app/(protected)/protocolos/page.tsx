"use client";

import Link from "next/link";

import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { buttonClass } from "@/components/ui/Button";
import { useApiQuery } from "@/lib/api/swr";

interface Pedido {
  id: string;
  numero: string;
  nome: string;
  servico_nome: string;
  status: string;
  prioridade: string;
  created_at: string;
}

export default function ProtocolosPage() {
  const lista = useApiQuery<Pedido[]>("v1/protocolos/pedidos");

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Protocolos"
        title="Mesa"
        description="Pedidos da carta de serviços que a sua unidade pode tratar."
        actions={
          <div className="flex flex-wrap gap-2">
            <Link href="/protocolos/balcao" className={buttonClass("secondary", "sm")}>
              Balcão
            </Link>
            <Link href="/protocolos/formularios" className={buttonClass("secondary", "sm")}>
              Formulários
            </Link>
            <Link href="/protocolos/servicos" className={buttonClass("secondary", "sm")}>
              Serviços
            </Link>
            <Link href="/protocolos/responsaveis" className={buttonClass("secondary", "sm")}>
              Responsáveis
            </Link>
            <Link href="/carta" className={buttonClass("secondary", "sm")}>
              Carta pública
            </Link>
          </div>
        }
      />
      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={(lista.data ?? []).length === 0}
        emptyTitle="Nenhum pedido na mesa"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {(lista.data ?? []).map((pedido) => (
            <li key={pedido.id}>
              <Link href={`/protocolos/${pedido.id}`} className="flex items-center justify-between gap-3 p-4">
                <div>
                  <p className="font-medium">
                    {pedido.numero} · {pedido.nome}
                  </p>
                  <p className="text-sm text-muted">{pedido.servico_nome}</p>
                </div>
                <span className="flex items-center gap-2 text-xs text-muted">
                  <Badge>{pedido.status}</Badge>
                  <Badge>{pedido.prioridade}</Badge>
                  {fmtDateTime(pedido.created_at)}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      </DataState>
    </div>
  );
}
