"use client";

import Link from "next/link";
import { type FormEvent, useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { Conexao, Modo } from "@/lib/integracoes/types";

const MODOS = [
  { value: "conector", label: "Conector na rede da origem" },
  { value: "direto", label: "Direto: o servidor alcança a origem" },
];

function texto(fd: FormData, nome: string) {
  return String(fd.get(nome) ?? "").trim();
}

export default function IntegracoesPage() {
  const list = useApiQuery<Conexao[]>("v1/integracoes/conexoes");
  const { run, pending } = useAction();
  const [modo, setModo] = useState<Modo>("conector");

  async function criar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(
      () =>
        apiClient.post("v1/integracoes/conexoes", {
          tipo: "glpi",
          nome: texto(fd, "nome"),
          modo,
          url: texto(fd, "url"),
          perfil: texto(fd, "perfil"),
        }),
      "Conexão criada",
    );
    if (ok) {
      form.reset();
      void list.mutate();
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Migração"
        title="Integrações"
        description="Traga o histórico de outra ferramenta (GLPI): extrair, revisar a tabela de-para e carregar. Quando o servidor não alcança a origem, um conector instalado na rede dela faz a leitura."
      />

      <Card>
        <CardHeader>
          <CardTitle as="h2">Nova conexão com o GLPI</CardTitle>
        </CardHeader>
        <CardContent className="pb-5">
          <form onSubmit={criar} className="mt-3 grid gap-3">
            <Input
              id="conexao-nome"
              name="nome"
              label="Nome *"
              required
              maxLength={160}
              autoComplete="off"
            />
            <Select
              id="conexao-modo"
              name="modo"
              label="Como ler a origem"
              value={modo}
              onChange={(e) => setModo(e.target.value as Modo)}
              options={MODOS}
            />
            {modo === "direto" && (
              <Input
                id="conexao-url"
                name="url"
                label="Endereço da API do GLPI *"
                required
                placeholder="https://glpi.exemplo/apirest.php"
                maxLength={500}
                autoComplete="off"
              />
            )}
            <Input
              id="conexao-perfil"
              name="perfil"
              label="Perfil no GLPI (vazio: Super-Admin)"
              maxLength={80}
              autoComplete="off"
            />
            <div>
              <Button type="submit" loading={pending}>
                Criar conexão
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <DataState
        loading={list.isLoading}
        error={list.error}
        onRetry={() => void list.mutate()}
        empty={(list.data ?? []).length === 0}
        emptyTitle="Nenhuma conexão"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {(list.data ?? []).map((c) => (
            <li key={c.id} className="flex flex-col gap-1 p-4">
              <Link
                href={`/integracoes/${c.id}`}
                className="font-medium text-primary hover:underline"
              >
                {c.nome}
              </Link>
              <p className="text-sm text-muted">
                {c.tipo.toUpperCase()} · {c.modo === "conector" ? "pelo conector" : "direto"}
                {c.extraido_em
                  ? ` · extraído em ${new Date(c.extraido_em).toLocaleString("pt-BR")}`
                  : " · ainda não extraído"}
              </p>
            </li>
          ))}
        </ul>
      </DataState>
    </div>
  );
}
