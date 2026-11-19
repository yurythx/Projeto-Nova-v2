"use client";

import Link from "next/link";
import type { FormEvent } from "react";
import { useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Responsavel {
  id: string;
  nome: string;
  cpf: string;
  ativo?: boolean;
}

export default function ResponsaveisPage() {
  const { can } = usePlataforma();
  const { run, pending } = useAction();
  const [ultimo, setUltimo] = useState<Responsavel | null>(null);
  const lista = useApiQuery<Responsavel[]>(
    can("protocolos:gerir") ? "v1/protocolos/responsaveis" : null,
  );

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const dados = new FormData(e.currentTarget);
    const criado = await run(
      () =>
        apiClient.post<Responsavel>("v1/protocolos/responsaveis", {
          unidade_id: String(dados.get("unidade_id") ?? ""),
          nome: String(dados.get("nome") ?? "").trim(),
          cpf: String(dados.get("cpf") ?? "").trim(),
        }),
      "Responsável técnico cadastrado",
    );
    if (criado) {
      setUltimo(criado.data);
      e.currentTarget.reset();
      void lista.mutate();
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Protocolos"
        title="Responsáveis técnicos"
        description="Serviço marcado como exclusivo só abre para um CPF ativo neste cadastro. O mesmo CPF não entra duas vezes."
        actions={
          <Link href="/protocolos" className={buttonClass("secondary", "sm")}>
            Mesa
          </Link>
        }
      />
      {can("protocolos:gerir") && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Novo responsável</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form onSubmit={(e) => void salvar(e)} className="grid gap-3 sm:grid-cols-2">
              <UnidadeDonaSelect id="unidade-responsavel" permission="protocolos:gerir" exigir />
              <Input id="nome" name="nome" label="Nome *" required minLength={2} maxLength={200} />
              <Input
                id="cpf"
                name="cpf"
                label="CPF *"
                required
                inputMode="numeric"
                maxLength={14}
              />
              <div className="sm:col-span-2">
                <Button type="submit" loading={pending}>
                  Cadastrar
                </Button>
              </div>
            </form>
            {ultimo && (
              <p className="mt-4 text-sm">
                {ultimo.nome} ficou ativo ({ultimo.cpf}). O serviço exclusivo confere o CPF na
                abertura do pedido.
              </p>
            )}
          </CardContent>
        </Card>
      )}
      {can("protocolos:gerir") && (
        <DataState
          loading={lista.isLoading}
          error={lista.error}
          onRetry={() => void lista.mutate()}
          empty={(lista.data ?? []).length === 0}
          emptyTitle="Nenhum responsável técnico"
        >
          <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
            {(lista.data ?? []).map((pessoa) => (
              <li key={pessoa.id} className="flex items-center justify-between gap-3 p-4">
                <div>
                  <p className="font-medium">{pessoa.nome}</p>
                  <p className="text-sm text-muted">{pessoa.cpf}</p>
                </div>
                <p className="text-sm text-muted">{pessoa.ativo === false ? "Inativo" : "Ativo"}</p>
              </li>
            ))}
          </ul>
        </DataState>
      )}
    </div>
  );
}
