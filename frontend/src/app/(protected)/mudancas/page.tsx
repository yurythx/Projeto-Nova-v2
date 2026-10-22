"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { MudancaForm, type MudancaCorpo } from "@/components/servicos/MudancaForm";
import { DataState } from "@/components/sismarq/DataState";
import { PageHeader } from "@/components/sismarq/PageHeader";
import { fmtDateTime, useAction } from "@/components/sismarq/useAction";
import { Badge } from "@/components/ui/Badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import { ESTADO_MUDANCA, RISCO, type EstadoMudanca, type Mudanca } from "@/lib/servicos/servicos";
import { useSismarq } from "@/lib/sismarq/SismarqProvider";

/** Mudanças: o que vai mudar no que está em operação, quando, com que
 * risco e como desfazer — e quem aprovou. */
export default function MudancasPage() {
  const { can } = useSismarq();
  const router = useRouter();
  const gere = can("mudancas:manage");
  const ve = gere || can("mudancas:aprovar");
  const [estado, setEstado] = useState("");
  const mudancas = useApiQuery<Mudanca[]>(ve ? withQuery("v1/mudancas", { estado }) : null);
  const { run, pending } = useAction();

  if (!ve) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Gestão de serviços" title="Mudanças" />
        <EmptyState
          title="Seu perfil não gere nem aprova mudanças"
          description="Peça o acesso à gestão da sua unidade."
        />
      </div>
    );
  }

  async function criar(corpo: MudancaCorpo) {
    const criada = await run(
      () => apiClient.post<Mudanca>("v1/mudancas", corpo),
      "Mudança registrada",
    );
    if (criada) router.push(`/mudancas/${criada.data.id}`);
  }

  const lista = mudancas.data ?? [];
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Gestão de serviços"
        title="Mudanças"
        description="A mudança padrão é pré-aprovada. A normal e a emergencial passam pela aprovação de outra pessoa, com a janela e o plano de reversão definidos."
      />
      {gere && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Nova mudança</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <MudancaForm
              rotulo="Registrar rascunho"
              pending={pending}
              onSalvar={(c) => void criar(c)}
            />
          </CardContent>
        </Card>
      )}
      <div className="max-w-xs">
        <Select
          id="mudancas-estado"
          label="Estado"
          value={estado}
          onChange={(e) => setEstado(e.target.value)}
          options={[
            { value: "", label: "Todos" },
            ...(Object.keys(ESTADO_MUDANCA) as EstadoMudanca[]).map((e) => ({
              value: e,
              label: ESTADO_MUDANCA[e].label,
            })),
          ]}
        />
      </div>
      <DataState
        loading={mudancas.isLoading}
        error={mudancas.error}
        onRetry={() => void mudancas.mutate()}
        empty={lista.length === 0}
        emptyTitle="Nenhuma mudança neste recorte"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {lista.map((m) => (
            <li
              key={m.id}
              className="flex flex-col gap-1 p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <Link
                  href={`/mudancas/${m.id}`}
                  className="font-medium text-primary hover:underline"
                >
                  #{m.numero} · {m.titulo}
                </Link>
                <p className="text-sm text-muted">
                  {m.inicio_planejado ? `Janela ${fmtDateTime(m.inicio_planejado)}` : "Sem janela"}{" "}
                  · pedida por {m.solicitante_nome}
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Badge tone={RISCO[m.risco].tone}>{RISCO[m.risco].label}</Badge>
                <Badge tone={ESTADO_MUDANCA[m.estado].tone}>{ESTADO_MUDANCA[m.estado].label}</Badge>
              </div>
            </li>
          ))}
        </ul>
      </DataState>
    </div>
  );
}
