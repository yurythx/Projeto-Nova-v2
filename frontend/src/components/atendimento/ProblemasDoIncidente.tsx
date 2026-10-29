"use client";

import Link from "next/link";

import { Badge } from "@/components/ui/Badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { useApiQuery } from "@/lib/api/swr";
import { ESTADO_PROJETO, type Projeto } from "@/lib/projetos/projetos";
import {
  ESTADO_MUDANCA,
  ESTADO_PROBLEMA,
  type Mudanca,
  type Problema,
} from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

/** O problema por trás do incidente e, quando já é erro conhecido, a
 * solução de contorno que a equipe aplica enquanto a causa não sai. */
export function ProblemasDoIncidente({ solicitacaoId }: { solicitacaoId: string }) {
  const { can } = usePlataforma();
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

/** Mudanças de engenharia vinculadas à solicitação na cadeia de valor ITIL v5. */
export function MudancasDaSolicitacao({ solicitacaoId }: { solicitacaoId: string }) {
  const { can, enabled } = usePlataforma();
  const ve = enabled("mudancas") && (can("mudancas:manage") || can("mudancas:aprovar"));
  const mudancas = useApiQuery<Mudanca[]>(
    ve ? `v1/mudancas?alvo_tipo=incidente&alvo_id=${solicitacaoId}` : null,
  );
  const lista = mudancas.data ?? [];
  if (lista.length === 0) return null;
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Mudanças relacionadas</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 pb-5">
        {lista.map((m) => (
          <div key={m.id} className="flex flex-wrap items-center gap-2 text-sm">
            <Link href={`/mudancas/${m.id}`} className="font-medium text-primary hover:underline">
              Mudança #{m.numero} · {m.titulo}
            </Link>
            <Badge tone={ESTADO_MUDANCA[m.estado].tone}>{ESTADO_MUDANCA[m.estado].label}</Badge>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

/** Projetos vinculados à solicitação na cadeia de valor ITIL v5. */
export function ProjetosDaSolicitacao({ solicitacaoId }: { solicitacaoId: string }) {
  const { can, enabled } = usePlataforma();
  const le = enabled("projetos") && (can("projetos:read") || can("projetos:manage"));
  const projetos = useApiQuery<Projeto[]>(
    le ? `v1/projetos?alvo_tipo=solicitacao&alvo_id=${solicitacaoId}` : null,
  );
  const lista = projetos.data ?? [];
  if (lista.length === 0) return null;
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Projetos relacionados</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 pb-5">
        {lista.map((proj) => (
          <div key={proj.id} className="flex flex-wrap items-center gap-2 text-sm">
            <Link
              href={`/projetos/${proj.id}`}
              className="font-medium text-primary hover:underline"
            >
              Projeto #{proj.numero} · {proj.titulo}
            </Link>
            <Badge tone={ESTADO_PROJETO[proj.estado]?.tone ?? "neutral"}>
              {ESTADO_PROJETO[proj.estado]?.label ?? proj.estado}
            </Badge>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
