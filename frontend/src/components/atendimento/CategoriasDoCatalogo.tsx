"use client";

import { type FormEvent, useState } from "react";

import { IconeDoCatalogo, OPCOES_DE_ICONE } from "@/components/atendimento/IconeDoCatalogo";
import { ConfirmButton } from "@/components/plataforma/ConfirmButton";
import { DataState } from "@/components/plataforma/DataState";
import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import type { Categoria } from "@/lib/atendimento/atendimento";
import type { useApiQuery } from "@/lib/api/swr";

/**
 * Categorias do catálogo em blocos (ADR 078). São da instituição: só quem
 * gere o atendimento nela toda as cria e altera. Apagar não apaga as
 * ofertas, que vão para "Outros".
 */
export function CategoriasDoCatalogo({
  categorias,
}: {
  categorias: ReturnType<typeof useApiQuery<Categoria[]>>;
}) {
  const { run, pending } = useAction();
  const [editando, setEditando] = useState<Categoria | null>(null);
  const lista = categorias.data ?? [];

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const body = {
      nome: String(fd.get("nome") ?? "").trim(),
      descricao: String(fd.get("descricao") ?? "").trim(),
      icone: String(fd.get("icone") ?? ""),
      ordem: Number(fd.get("ordem") || 0),
      ativa: fd.get("ativa") === "on",
    };
    const atual = editando;
    const ok = await run(
      () =>
        atual
          ? apiClient.put(`v1/atendimento/categorias/${atual.id}`, body)
          : apiClient.post("v1/atendimento/categorias", body),
      atual ? "Categoria atualizada" : "Categoria criada",
    );
    if (ok) {
      setEditando(null);
      if (!atual) form.reset();
      void categorias.mutate();
    }
  }

  async function apagar(c: Categoria) {
    if (
      await run(() => apiClient.delete(`v1/atendimento/categorias/${c.id}`), "Categoria apagada")
    ) {
      if (editando?.id === c.id) setEditando(null);
      void categorias.mutate();
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Categorias do catálogo</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4 pb-5">
        <p className="text-sm text-muted">
          Os blocos que quem pede vê na central, em cada porta (pedido ou problema). A oferta sem
          categoria aparece em “Outros”.
        </p>
        <form
          key={editando?.id ?? "nova-categoria"}
          onSubmit={salvar}
          aria-label={editando ? "Editar categoria" : "Nova categoria"}
          className="grid gap-3 sm:grid-cols-2"
        >
          <Input
            id="categoria-nome"
            name="nome"
            label="Nome *"
            required
            maxLength={80}
            defaultValue={editando?.nome}
            autoComplete="off"
          />
          <Input
            id="categoria-descricao"
            name="descricao"
            label="Descrição"
            maxLength={300}
            defaultValue={editando?.descricao}
            autoComplete="off"
          />
          <Select
            id="categoria-icone"
            name="icone"
            label="Ícone"
            defaultValue={editando?.icone ?? ""}
            options={OPCOES_DE_ICONE}
          />
          <Input
            id="categoria-ordem"
            name="ordem"
            type="number"
            label="Ordem"
            min={0}
            max={999}
            defaultValue={editando?.ordem ?? 0}
          />
          <label className="flex items-center gap-2 text-sm">
            <input
              id="categoria-ativa"
              name="ativa"
              type="checkbox"
              defaultChecked={editando ? editando.ativa : true}
              className="h-4 w-4 accent-primary"
            />
            Aparece na central
          </label>
          <div className="flex flex-wrap gap-2 sm:col-span-2">
            <Button type="submit" loading={pending}>
              {editando ? "Salvar categoria" : "Criar categoria"}
            </Button>
            {editando && (
              <Button type="button" variant="secondary" onClick={() => setEditando(null)}>
                Cancelar
              </Button>
            )}
          </div>
        </form>
        <DataState
          loading={categorias.isLoading}
          error={categorias.error}
          onRetry={() => void categorias.mutate()}
          empty={lista.length === 0}
          emptyTitle="Nenhuma categoria ainda"
          emptyDescription="Sem categorias, todas as ofertas aparecem em “Outros”."
          rows={2}
        >
          <ul className="flex flex-col divide-y divide-surface-border rounded-lg border border-surface-border">
            {lista.map((c) => (
              <li key={c.id} className="flex flex-wrap items-center gap-3 p-3">
                <IconeDoCatalogo nome={c.icone} size={20} className="text-primary" />
                <div className="min-w-0 flex-1">
                  <p className="font-medium">
                    {c.nome} {!c.ativa && <Badge tone="neutral">Fora da central</Badge>}
                  </p>
                  {c.descricao ? <p className="text-sm text-muted">{c.descricao}</p> : null}
                </div>
                <span className="text-xs text-muted">ordem {c.ordem}</span>
                <Button type="button" variant="secondary" size="sm" onClick={() => setEditando(c)}>
                  Editar
                </Button>
                <ConfirmButton
                  title={`Apagar a categoria ${c.nome}?`}
                  description="As ofertas dela não são apagadas: passam a aparecer em “Outros”."
                  confirmLabel="Apagar"
                  onConfirm={() => apagar(c)}
                >
                  Apagar
                </ConfirmButton>
              </li>
            ))}
          </ul>
        </DataState>
      </CardContent>
    </Card>
  );
}
