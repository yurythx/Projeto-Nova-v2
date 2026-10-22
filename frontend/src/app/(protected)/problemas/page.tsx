"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { type FormEvent, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import {
  ESTADO_PROBLEMA,
  type EstadoProblema,
  type Problema,
  type Recorrencia,
} from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

/** Problemas e a base de erros conhecidos. Quem gere registra e
 * investiga; quem atende a fila consulta o contorno. */
export default function ProblemasPage() {
  const { can } = usePlataforma();
  const router = useRouter();
  const gere = can("problemas:manage");
  const le = gere || can("problemas:read");
  const [estado, setEstado] = useState("");
  const [busca, setBusca] = useState("");
  const problemas = useApiQuery<Problema[]>(
    le ? withQuery("v1/problemas", { estado, q: busca }) : null,
  );
  const sugestoes = useApiQuery<Recorrencia[]>(gere ? "v1/problemas/sugestoes" : null);
  const { run, pending } = useAction();

  if (!le) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Gestão de serviços" title="Problemas" />
        <EmptyState
          title="Seu perfil não consulta problemas"
          description="Peça o acesso à gestão da sua unidade."
        />
      </div>
    );
  }

  async function criar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const criado = await run(
      () =>
        apiClient.post<Problema>("v1/problemas", {
          titulo: String(fd.get("titulo")).trim(),
          descricao: String(fd.get("descricao")).trim(),
          unidade_id: String(fd.get("unidade_id")) || null,
        }),
      "Problema registrado",
    );
    if (criado) router.push(`/problemas/${criado.data.id}`);
  }

  /** Abre o problema da recorrência e já liga os incidentes dela. */
  async function abrirDaSugestao(r: Recorrencia) {
    const criado = await run(async () => {
      const { data } = await apiClient.post<Problema>("v1/problemas", {
        titulo: `Incidentes repetidos: ${r.oferta_nome}`.slice(0, 160),
        unidade_id: r.unidade_id ?? null,
      });
      for (const s of r.solicitacoes)
        await apiClient.put(`v1/problemas/${data.id}/incidentes/${s}`);
      return data;
    }, "Problema aberto com os incidentes");
    if (criado) router.push(`/problemas/${criado.id}`);
  }

  const lista = problemas.data ?? [];
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Gestão de serviços"
        title="Problemas"
        description="A causa por trás dos incidentes que se repetem. Com a solução de contorno registrada, o problema vira erro conhecido e a equipe de atendimento consulta aqui."
      />

      {gere && (sugestoes.data ?? []).length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Incidentes que se repetem</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <ul className="flex flex-col gap-2">
              {(sugestoes.data ?? []).map((r) => (
                <li
                  key={r.oferta_id}
                  className="flex flex-wrap items-center justify-between gap-2 text-sm"
                >
                  <span>
                    <strong>{r.oferta_nome}</strong>: {r.total} incidentes em 30 dias, o último em{" "}
                    {fmtDateTime(r.ultimo_em)}
                  </span>
                  <Button
                    type="button"
                    size="sm"
                    variant="secondary"
                    disabled={pending}
                    onClick={() => void abrirDaSugestao(r)}
                  >
                    Abrir problema de {r.oferta_nome}
                  </Button>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {gere && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Novo problema</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form onSubmit={criar} className="mt-3 grid gap-3">
              <Input
                id="problema-titulo"
                name="titulo"
                label="Título *"
                required
                maxLength={160}
                autoComplete="off"
              />
              <Textarea
                id="problema-descricao"
                name="descricao"
                label="O que se observa"
                maxLength={4000}
                rows={3}
              />
              <UnidadeDonaSelect
                id="problema-unidade"
                permission="problemas:manage"
                label="Unidade"
              />
              <div>
                <Button type="submit" loading={pending}>
                  Registrar problema
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        <Select
          id="problemas-estado"
          label="Estado"
          value={estado}
          onChange={(e) => setEstado(e.target.value)}
          options={[
            { value: "", label: "Todos" },
            ...(Object.keys(ESTADO_PROBLEMA) as EstadoProblema[]).map((e) => ({
              value: e,
              label: ESTADO_PROBLEMA[e].label,
            })),
          ]}
        />
        <Input
          id="problemas-busca"
          label="Buscar no título ou no contorno"
          value={busca}
          onChange={(e) => setBusca(e.target.value)}
        />
      </div>

      <DataState
        loading={problemas.isLoading}
        error={problemas.error}
        onRetry={() => void problemas.mutate()}
        empty={lista.length === 0}
        emptyTitle="Nenhum problema neste recorte"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {lista.map((p) => (
            <li key={p.id} className="flex flex-col gap-1 p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <Link
                  href={`/problemas/${p.id}`}
                  className="font-medium text-primary hover:underline"
                >
                  #{p.numero} · {p.titulo}
                </Link>
                <Badge tone={ESTADO_PROBLEMA[p.estado].tone}>
                  {ESTADO_PROBLEMA[p.estado].label}
                </Badge>
              </div>
              {p.contorno && <p className="text-sm text-muted">Contorno: {p.contorno}</p>}
            </li>
          ))}
        </ul>
      </DataState>
    </div>
  );
}
