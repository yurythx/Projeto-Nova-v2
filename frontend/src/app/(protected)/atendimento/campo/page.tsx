"use client";

import Link from "next/link";

import { DataState } from "@/components/sismarq/DataState";
import { PageHeader } from "@/components/sismarq/PageHeader";
import { fmtDateTime } from "@/components/sismarq/useAction";
import { Badge } from "@/components/ui/Badge";
import { EmptyState } from "@/components/ui/EmptyState";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from "@/components/ui/Table";
import { useApiQuery } from "@/lib/api/swr";
import { distancia, linkMapa, type VisitaEmCampo } from "@/lib/atendimento/atendimento";
import { useSismarq } from "@/lib/sismarq/SismarqProvider";

/** Supervisão: quem está em visita agora, nas unidades que a pessoa
 * atende. A posição aparece só enquanto a visita está aberta; o mapa
 * abre fora do sistema, sem embutir tiles de terceiros. */
export default function AtendimentoCampoPage() {
  const { can } = useSismarq();
  const atende = can("atendimento:atender");
  const visitas = useApiQuery<VisitaEmCampo[]>(atende ? "v1/atendimento/visitas/em-campo" : null, {
    refreshInterval: 60_000,
  });

  if (!atende) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Atendimento" title="Em campo" />
        <EmptyState
          title="Seu perfil não atende a fila"
          description="A supervisão mostra as visitas das unidades em que você atende."
        />
      </div>
    );
  }

  const lista = visitas.data ?? [];
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Em campo"
        description="Visitas em andamento agora. A posição é a do check-in e some da supervisão quando a visita termina."
      />
      <DataState
        loading={visitas.isLoading}
        error={visitas.error}
        onRetry={() => void visitas.mutate()}
        empty={lista.length === 0}
        emptyTitle="Ninguém em campo agora"
      >
        <Table caption="Visitas em andamento" className="min-w-[760px]">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Técnico</TableHeaderCell>
              <TableHeaderCell>Solicitação</TableHeaderCell>
              <TableHeaderCell>Local</TableHeaderCell>
              <TableHeaderCell>Chegada</TableHeaderCell>
              <TableHeaderCell>Destino</TableHeaderCell>
              <TableHeaderCell>
                <span className="sr-only">Mapa</span>
              </TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {lista.map((v) => (
              <TableRow key={v.id}>
                <TableCell className="font-medium">{v.tecnico_nome}</TableCell>
                <TableCell>
                  <Link
                    href={`/atendimento/solicitacoes/${v.solicitacao_id}`}
                    className="text-primary hover:underline"
                  >
                    #{v.numero} · {v.titulo}
                  </Link>
                </TableCell>
                <TableCell>{v.local || <span className="text-muted">—</span>}</TableCell>
                <TableCell className="whitespace-nowrap">{fmtDateTime(v.inicio_em)}</TableCell>
                <TableCell>
                  {v.distancia_m === undefined ? (
                    <span className="text-muted">Sem destino</span>
                  ) : (
                    <span className="inline-flex flex-wrap items-center gap-1">
                      {distancia(v.distancia_m)}
                      {v.fora_da_area && <Badge tone="warning">Fora da área</Badge>}
                    </span>
                  )}
                </TableCell>
                <TableCell>
                  {v.inicio && (
                    <a
                      href={linkMapa(v.inicio)}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline"
                    >
                      Ver no mapa
                      <span className="sr-only">
                        {" "}
                        a posição de {v.tecnico_nome} (abre em nova aba)
                      </span>
                    </a>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </DataState>
    </div>
  );
}
