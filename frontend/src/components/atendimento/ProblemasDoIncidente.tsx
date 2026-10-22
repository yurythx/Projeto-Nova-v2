"use client";

import Link from "next/link";

import { Badge } from "@/components/ui/Badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { useApiQuery } from "@/lib/api/swr";
import { ESTADO_PROBLEMA, type Problema } from "@/lib/servicos/servicos";
import { useSismarq } from "@/lib/sismarq/SismarqProvider";

/** O problema por trás do incidente e, quando já é erro conhecido, a
 * solução de contorno que a equipe aplica enquanto a causa não sai. */
export function ProblemasDoIncidente({ solicitacaoId }: { solicitacaoId: string }) {
  const { can } = useSismarq();
  const le = can("problemas:read") || can("problemas:manage");
  const problemas = useApiQuery<Problema[]>(
    le ? `v1/problemas?solicitacao_id=${solicitacaoId}` : null,
  );
  const lista = problemas.data ?? [];
  if (lista.length === 0) return null;
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Problema relacionado</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 pb-5">
        {lista.map((p) => (
          <div key={p.id} className="flex flex-col gap-1 text-sm">
            <div className="flex flex-wrap items-center gap-2">
              <Link
                href={`/problemas/${p.id}`}
                className="font-medium text-primary hover:underline"
              >
                Problema #{p.numero} · {p.titulo}
              </Link>
              <Badge tone={ESTADO_PROBLEMA[p.estado].tone}>{ESTADO_PROBLEMA[p.estado].label}</Badge>
            </div>
            {p.contorno && (
              <p className="rounded-lg border border-warning/40 bg-warning/10 p-2">
                <strong>Contorno:</strong> {p.contorno}
              </p>
            )}
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
