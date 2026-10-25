"use client";

import { type FormEvent } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { deCampoData } from "@/lib/servicos/servicos";

interface Congelamento {
  id: string;
  unidade_id?: string;
  inicio: string;
  fim: string;
  motivo: string;
}

/** Janelas em que mudança nova não é agendada. */
export function Congelamentos() {
  const lista = useApiQuery<Congelamento[]>("v1/mudancas/congelamentos");
  const { run, pending } = useAction();

  async function criar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const inicio = deCampoData(String(fd.get("inicio") ?? ""));
    const fim = deCampoData(String(fd.get("fim") ?? ""));
    if (!inicio || !fim) return;
    const ok = await run(
      () =>
        apiClient.post("v1/mudancas/congelamentos", {
          unidade_id: String(fd.get("unidade_id") || "") || null,
          inicio,
          fim,
          motivo: String(fd.get("motivo") ?? "").trim(),
        }),
      "Congelamento registrado",
    );
    if (ok) {
      form.reset();
      void lista.mutate();
    }
  }

  async function excluir(id: string) {
    const ok = await run(
      () => apiClient.delete(`v1/mudancas/congelamentos/${id}`),
      "Congelamento removido",
    );
    if (ok) void lista.mutate();
  }

  const itens = lista.data ?? [];
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Congelamento</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 pb-5">
        <p className="text-sm text-muted">
          Nesse período, uma mudança nova não entra em avaliação. A que já estava agendada continua.
          Sem unidade, o congelamento vale para toda a instituição.
        </p>
        <form onSubmit={criar} className="grid gap-3 sm:grid-cols-2">
          <Input id="congelamento-inicio" name="inicio" type="datetime-local" label="Início *" required />
          <Input id="congelamento-fim" name="fim" type="datetime-local" label="Fim *" required />
          <div className="sm:col-span-2">
            <Input id="congelamento-motivo" name="motivo" label="Motivo" maxLength={4000} />
          </div>
          <UnidadeDonaSelect id="congelamento-unidade" permission="mudancas:manage" label="Unidade" />
          <div className="flex items-end">
            <Button type="submit" loading={pending}>
              Registrar congelamento
            </Button>
          </div>
        </form>
        {lista.error ? (
          <p className="text-sm text-danger" role="alert">
            Não foi possível carregar os congelamentos.
          </p>
        ) : null}
        <ul className="flex flex-col gap-2">
          {itens.map((item) => (
            <li key={item.id} className="flex flex-wrap items-center justify-between gap-2 text-sm">
              <span>
                {new Date(item.inicio).toLocaleString("pt-BR")} até {new Date(item.fim).toLocaleString("pt-BR")}
                {item.motivo ? ` · ${item.motivo}` : ""}
                {item.unidade_id ? "" : " · institucional"}
              </span>
              <Button type="button" variant="secondary" onClick={() => void excluir(item.id)}>
                Remover congelamento
              </Button>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}
