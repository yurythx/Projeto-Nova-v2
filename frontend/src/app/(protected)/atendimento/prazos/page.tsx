"use client";

import { type FormEvent, useState } from "react";

import { PoliticaForm, sugestao, type PoliticaCorpo } from "@/components/atendimento/PoliticaForm";
import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { ConfirmButton } from "@/components/sismarq/ConfirmButton";
import { DataState } from "@/components/sismarq/DataState";
import { PageHeader } from "@/components/sismarq/PageHeader";
import { fmtDate, useAction } from "@/components/sismarq/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import {
  TIPO_FERIADO,
  type Feriado,
  type Politica,
  type TipoFeriado,
} from "@/lib/atendimento/atendimento";
import { useSismarq } from "@/lib/sismarq/SismarqProvider";

/** Prazos da unidade: quando se trabalha, como impacto e urgência viram
 * prioridade, as metas de cada prioridade e os feriados. A subunidade sem
 * política própria usa a de cima; a institucional vale para todas. */
export default function AtendimentoPrazosPage() {
  const { can, canGlobal } = useSismarq();
  const gere = can("atendimento:manage");
  const global = canGlobal("atendimento:manage");
  // Gestão global começa pela institucional; a de unidade escolhe a sua.
  const [escolha, setEscolha] = useState<string | null>(null);
  const unidade = escolha ?? (global ? "" : null);
  const filtro = unidade === null ? null : { unidade_id: unidade || undefined };
  const politica = useApiQuery<Politica | null>(
    filtro ? withQuery("v1/atendimento/politica", filtro) : null,
  );
  const feriados = useApiQuery<Feriado[]>(
    filtro ? withQuery("v1/atendimento/feriados", filtro) : null,
  );
  const { run, pending } = useAction();

  if (!gere) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Atendimento" title="Prazos" />
        <EmptyState
          title="Seu perfil não gere o atendimento"
          description="A política de prazo é cadastrada pela gestão do atendimento da unidade."
        />
      </div>
    );
  }

  async function salvar(corpo: PoliticaCorpo) {
    const ok = await run(
      () => apiClient.put("v1/atendimento/politica", { ...corpo, unidade_id: unidade || null }),
      "Política salva",
    );
    if (ok) void politica.mutate();
  }

  async function criarFeriado(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(
      () =>
        apiClient.post("v1/atendimento/feriados", {
          unidade_id: unidade || null,
          data: String(fd.get("data")),
          nome: String(fd.get("nome")).trim(),
          tipo: String(fd.get("tipo")),
        }),
      "Feriado cadastrado",
    );
    if (ok) {
      form.reset();
      void feriados.mutate();
    }
  }

  async function excluirFeriado(f: Feriado) {
    const ok = await run(
      () => apiClient.delete(`v1/atendimento/feriados/${f.id}`),
      "Feriado excluído",
    );
    if (ok) void feriados.mutate();
  }

  const p = politica.data;
  const propria = p && !p.herdada ? p : null;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Prazos"
        description="O prazo conta só o tempo útil: o expediente da unidade, sem feriados e sem as esperas pelo servidor ou por terceiros. Mudar a política vale para as próximas classificações."
      />

      <div className="max-w-md">
        <UnidadeDonaSelect
          id="prazos-unidade"
          permission="atendimento:manage"
          label="Unidade"
          defaultValue={escolha}
          onChange={setEscolha}
        />
      </div>

      {filtro === null ? (
        <EmptyState
          title="Escolha a unidade"
          description="A política e os feriados são de cada unidade."
        />
      ) : (
        <>
          <Card>
            <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
              <CardTitle as="h2">Política de prazo</CardTitle>
              {p?.herdada && <Badge tone="info">Herdada</Badge>}
              {!p && !politica.isLoading && !politica.error && (
                <Badge tone="warning">Sem política</Badge>
              )}
            </CardHeader>
            <CardContent className="flex flex-col gap-4 pb-5">
              <DataState
                loading={politica.isLoading}
                error={politica.error}
                onRetry={() => void politica.mutate()}
                empty={false}
              >
                {p?.herdada && (
                  <p role="status" className="text-sm text-muted">
                    Esta unidade usa a política de uma unidade acima dela. Salvar cria a política
                    própria, a partir desta.
                  </p>
                )}
                {!p && (
                  <p role="status" className="text-sm text-muted">
                    Sem política, as solicitações desta unidade ficam sem prazo. Os valores abaixo
                    são uma sugestão e só valem depois de salvos.
                  </p>
                )}
                <PoliticaForm
                  key={`${unidade}-${p?.id ?? "nova"}-${p?.herdada ?? false}`}
                  inicial={p ?? sugestao()}
                  pending={pending}
                  onSalvar={(corpo) => void salvar(corpo)}
                />
              </DataState>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle as="h2">Feriados e pontos facultativos</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4 pb-5">
              <p className="text-sm text-muted">
                Valem para esta unidade e as abaixo dela.
                {propria ? "" : " Os da unidade acima também valem aqui."}
              </p>
              <form
                onSubmit={criarFeriado}
                className="grid gap-3 sm:grid-cols-[10rem_1fr_12rem_auto] sm:items-end"
              >
                <Input id="feriado-data" name="data" type="date" label="Data *" required />
                <Input
                  id="feriado-nome"
                  name="nome"
                  label="Nome *"
                  required
                  maxLength={160}
                  autoComplete="off"
                />
                <Select
                  id="feriado-tipo"
                  name="tipo"
                  label="Tipo"
                  defaultValue="municipal"
                  options={(Object.keys(TIPO_FERIADO) as TipoFeriado[]).map((t) => ({
                    value: t,
                    label: TIPO_FERIADO[t],
                  }))}
                />
                <Button type="submit" loading={pending}>
                  Adicionar
                </Button>
              </form>
              <DataState
                loading={feriados.isLoading}
                error={feriados.error}
                onRetry={() => void feriados.mutate()}
                empty={(feriados.data ?? []).length === 0}
                emptyTitle="Nenhum feriado cadastrado nesta unidade"
              >
                <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border">
                  {(feriados.data ?? []).map((f) => (
                    <li
                      key={f.id}
                      className="flex flex-wrap items-center justify-between gap-2 p-3 text-sm"
                    >
                      <span>
                        <span className="font-medium tabular-nums">
                          {fmtDate(`${f.data}T12:00:00`)}
                        </span>{" "}
                        · {f.nome} <Badge>{TIPO_FERIADO[f.tipo]}</Badge>
                      </span>
                      <ConfirmButton
                        title={`Excluir ${f.nome}?`}
                        description="O dia volta a contar como tempo útil nas próximas medições."
                        confirmLabel="Excluir"
                        onConfirm={() => excluirFeriado(f)}
                      >
                        Excluir {f.nome}
                      </ConfirmButton>
                    </li>
                  ))}
                </ul>
              </DataState>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
