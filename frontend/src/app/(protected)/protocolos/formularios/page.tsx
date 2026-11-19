"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

const TIPOS = [
  "text",
  "number",
  "email",
  "phone",
  "date",
  "time",
  "password",
  "url",
  "cpf",
  "cnpj",
  "cep",
  "textarea",
  "select",
  "checkbox",
  "checkbox_group",
  "radio",
  "boolean",
  "file",
  "map",
  "map_area",
  "ambulante",
  "page",
].map((tipo) => ({ value: tipo, label: tipo }));

interface Campo {
  id: string;
  tipo: string;
  rotulo: string;
  obrigatorio: boolean;
  opcoes?: string[];
  revela?: string[];
}

interface Formulario {
  id: string;
  titulo: string;
  descricao: string;
  ativo: boolean;
  campos: Campo[];
}

export default function FormulariosPage() {
  const { can } = usePlataforma();
  const lista = useApiQuery<Formulario[]>("v1/protocolos/formularios");
  const { run, pending } = useAction();
  const [campos, setCampos] = useState<Campo[]>([
    { id: "nome_social", tipo: "text", rotulo: "Nome", obrigatorio: false },
  ]);

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const dados = new FormData(e.currentTarget);
    const ok = await run(
      () =>
        apiClient.post("v1/protocolos/formularios", {
          unidade_id: String(dados.get("unidade_id") ?? ""),
          titulo: String(dados.get("titulo") ?? "").trim(),
          descricao: String(dados.get("descricao") ?? "").trim(),
          instrucao: String(dados.get("instrucao") ?? "").trim(),
          campos,
          ativo: true,
        }),
      "Formulário salvo",
    );
    if (ok) {
      e.currentTarget.reset();
      void lista.mutate();
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Protocolos"
        title="Formulários"
        description="O desenho que o cidadão preenche. O sim/não pode revelar outros campos. O arquivo sobe ao armazenamento."
        actions={
          <Link href="/protocolos" className={buttonClass("secondary", "sm")}>
            Mesa
          </Link>
        }
      />
      {can("protocolos:gerir") && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Novo formulário</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form onSubmit={salvar} className="flex flex-col gap-3">
              <UnidadeDonaSelect id="unidade-form" permission="protocolos:gerir" exigir />
              <Input id="titulo" name="titulo" label="Título *" required maxLength={200} />
              <Textarea id="descricao" name="descricao" label="Descrição" maxLength={4000} />
              <Textarea
                id="instrucao"
                name="instrucao"
                label="Instrução ao cidadão"
                maxLength={4000}
              />
              <ul className="flex flex-col gap-3">
                {campos.map((campo, indice) => (
                  <li
                    key={indice}
                    className="grid gap-3 rounded-lg border border-surface-border p-3 sm:grid-cols-4"
                  >
                    <Input
                      id={`campo-id-${indice}`}
                      label="Identificador *"
                      value={campo.id}
                      maxLength={40}
                      onChange={(e) => atualizar(setCampos, indice, { id: e.target.value })}
                    />
                    <Input
                      id={`campo-rotulo-${indice}`}
                      label="Rótulo *"
                      value={campo.rotulo}
                      maxLength={200}
                      onChange={(e) => atualizar(setCampos, indice, { rotulo: e.target.value })}
                    />
                    <Select
                      id={`campo-tipo-${indice}`}
                      label="Tipo"
                      value={campo.tipo}
                      options={TIPOS}
                      onChange={(e) => atualizar(setCampos, indice, { tipo: e.target.value })}
                    />
                    <Input
                      id={`campo-opcoes-${indice}`}
                      label="Opções (vírgula)"
                      defaultValue={(campo.opcoes ?? []).join(", ")}
                      onChange={(e) =>
                        atualizar(setCampos, indice, {
                          opcoes: e.target.value
                            .split(",")
                            .map((item) => item.trim())
                            .filter(Boolean),
                        })
                      }
                    />
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        className="h-4 w-4 accent-primary"
                        checked={campo.obrigatorio}
                        onChange={(e) =>
                          atualizar(setCampos, indice, { obrigatorio: e.target.checked })
                        }
                      />
                      Obrigatório
                    </label>
                    {campo.tipo === "boolean" && (
                      <Input
                        id={`campo-revela-${indice}`}
                        label="Revela (identificadores)"
                        defaultValue={(campo.revela ?? []).join(", ")}
                        onChange={(e) =>
                          atualizar(setCampos, indice, {
                            revela: e.target.value
                              .split(",")
                              .map((item) => item.trim())
                              .filter(Boolean),
                          })
                        }
                      />
                    )}
                    <Button
                      type="button"
                      variant="ghost"
                      onClick={() => setCampos((atual) => atual.filter((_, i) => i !== indice))}
                    >
                      Remover
                    </Button>
                  </li>
                ))}
              </ul>
              <div className="flex flex-wrap gap-2">
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() =>
                    setCampos((atual) => [
                      ...atual,
                      { id: "", tipo: "text", rotulo: "", obrigatorio: false },
                    ])
                  }
                >
                  Campo
                </Button>
                <Button type="submit" loading={pending}>
                  Salvar
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={(lista.data ?? []).length === 0}
        emptyTitle="Nenhum formulário"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {(lista.data ?? []).map((form) => (
            <li key={form.id} className="flex items-center justify-between gap-3 p-4">
              <div>
                <p className="font-medium">{form.titulo}</p>
                <p className="text-sm text-muted">{(form.campos ?? []).length} campos</p>
              </div>
              <Badge>{form.ativo ? "ativo" : "inativo"}</Badge>
            </li>
          ))}
        </ul>
      </DataState>
    </div>
  );
}

function atualizar(
  setCampos: (fn: (atual: Campo[]) => Campo[]) => void,
  indice: number,
  patch: Partial<Campo>,
) {
  setCampos((atual) => atual.map((campo, i) => (i === indice ? { ...campo, ...patch } : campo)));
}
