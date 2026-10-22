"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/sismarq/DataState";
import { PageHeader } from "@/components/sismarq/PageHeader";
import { useAction } from "@/components/sismarq/useAction";
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
  ESTADO_ATIVO,
  TIPO_ATIVO,
  type Ativo,
  type EstadoAtivo,
  type TipoAtivo,
} from "@/lib/servicos/servicos";
import { useSismarq } from "@/lib/sismarq/SismarqProvider";

interface Produto {
  id: string;
  nome: string;
}

export default function AtivosPage() {
  return (
    <Suspense fallback={null}>
      <Ativos />
    </Suspense>
  );
}

/** Ativos (itens de configuração): o que sustenta cada produto digital.
 * Com ?produto=, mostra só os ativos daquele produto. */
function Ativos() {
  const { can } = useSismarq();
  const gere = can("ativos:manage");
  const produto = useSearchParams().get("produto") ?? undefined;
  const ativos = useApiQuery<Ativo[]>(
    gere ? withQuery("v1/ativos", { produto_id: produto }) : null,
  );
  const produtos = useApiQuery<Produto[]>(gere && can("produtos:manage") ? "v1/produtos" : null);
  const [editando, setEditando] = useState<Ativo | null>(null);
  const { run, pending } = useAction();

  if (!gere) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Gestão de serviços" title="Ativos" />
        <EmptyState
          title="Seu perfil não gere ativos"
          description="Peça o acesso à gestão da sua unidade."
        />
      </div>
    );
  }

  const nomeDoProduto = new Map((produtos.data ?? []).map((p) => [p.id, p.nome]));

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const alvo = editando;
    const body = {
      nome: String(fd.get("nome")).trim(),
      tipo: String(fd.get("tipo")),
      estado: String(fd.get("estado")),
      identificador: String(fd.get("identificador")).trim(),
      descricao: String(fd.get("descricao")).trim(),
      produto_id: String(fd.get("produto_id")) || null,
      unidade_id: String(fd.get("unidade_id")) || null,
    };
    const ok = await run(
      () =>
        alvo ? apiClient.put(`v1/ativos/${alvo.id}`, body) : apiClient.post("v1/ativos", body),
      alvo ? "Ativo atualizado" : "Ativo cadastrado",
    );
    if (ok) {
      setEditando(null);
      if (!alvo) form.reset();
      void ativos.mutate();
    }
  }

  const lista = ativos.data ?? [];
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Gestão de serviços"
        title="Ativos"
        description="Equipamentos, sistemas, licenças, links e serviços contratados que sustentam os produtos digitais."
      />
      {produto && (
        <p role="status" className="text-sm">
          Mostrando os ativos de <strong>{nomeDoProduto.get(produto) ?? "um produto"}</strong>.{" "}
          <Link href="/ativos" className="text-primary hover:underline">
            Ver todos
          </Link>
        </p>
      )}

      <Card>
        <CardHeader>
          <CardTitle as="h2">{editando ? "Editar ativo" : "Novo ativo"}</CardTitle>
        </CardHeader>
        <CardContent className="pb-5">
          <form
            key={editando?.id ?? "novo"}
            onSubmit={salvar}
            className="mt-3 grid gap-3 sm:grid-cols-2"
          >
            <Input
              id="ativo-nome"
              name="nome"
              label="Nome *"
              required
              maxLength={160}
              defaultValue={editando?.nome}
              autoComplete="off"
            />
            <Input
              id="ativo-identificador"
              name="identificador"
              label="Patrimônio ou número de série"
              maxLength={80}
              defaultValue={editando?.identificador}
              autoComplete="off"
            />
            <Select
              id="ativo-tipo"
              name="tipo"
              label="Tipo"
              defaultValue={editando?.tipo ?? "hardware"}
              options={(Object.keys(TIPO_ATIVO) as TipoAtivo[]).map((t) => ({
                value: t,
                label: TIPO_ATIVO[t],
              }))}
            />
            <Select
              id="ativo-estado"
              name="estado"
              label="Estado"
              defaultValue={editando?.estado ?? "em_uso"}
              options={(Object.keys(ESTADO_ATIVO) as EstadoAtivo[]).map((e) => ({
                value: e,
                label: ESTADO_ATIVO[e].label,
              }))}
            />
            <Select
              id="ativo-produto"
              name="produto_id"
              label="Produto que sustenta"
              defaultValue={editando?.produto_id ?? produto ?? ""}
              options={[
                { value: "", label: "Nenhum" },
                ...(produtos.data ?? []).map((p) => ({ value: p.id, label: p.nome })),
              ]}
            />
            <UnidadeDonaSelect
              id="ativo-unidade"
              permission="ativos:manage"
              label="Unidade"
              defaultValue={editando?.unidade_id}
            />
            <div className="sm:col-span-2">
              <Textarea
                id="ativo-descricao"
                name="descricao"
                label="Descrição"
                maxLength={4000}
                rows={2}
                defaultValue={editando?.descricao}
              />
            </div>
            <div className="flex flex-wrap gap-2 sm:col-span-2">
              <Button type="submit" loading={pending}>
                {editando ? "Salvar ativo" : "Cadastrar ativo"}
              </Button>
              {editando && (
                <Button type="button" variant="secondary" onClick={() => setEditando(null)}>
                  Cancelar edição
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      <DataState
        loading={ativos.isLoading}
        error={ativos.error}
        onRetry={() => void ativos.mutate()}
        empty={lista.length === 0}
        emptyTitle="Nenhum ativo cadastrado"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {lista.map((a) => (
            <li
              key={a.id}
              className="flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <p className="font-medium">{a.nome}</p>
                <p className="text-sm text-muted">
                  {TIPO_ATIVO[a.tipo]}
                  {a.identificador ? ` · ${a.identificador}` : ""}
                  {a.produto_id
                    ? ` · ${nomeDoProduto.get(a.produto_id) ?? "produto vinculado"}`
                    : ""}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Badge tone={ESTADO_ATIVO[a.estado].tone}>{ESTADO_ATIVO[a.estado].label}</Badge>
                <Button type="button" variant="secondary" onClick={() => setEditando(a)}>
                  Editar {a.nome}
                </Button>
              </div>
            </li>
          ))}
        </ul>
      </DataState>
    </div>
  );
}
