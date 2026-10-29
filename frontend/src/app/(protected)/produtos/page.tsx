"use client";

import Link from "next/link";
import { type FormEvent, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

const ESTAGIOS = [
  { value: "descoberta", label: "Descoberta" },
  { value: "desenho", label: "Desenho" },
  { value: "aquisicao", label: "Aquisição" },
  { value: "construcao", label: "Construção" },
  { value: "transicao", label: "Transição" },
  { value: "operacao", label: "Operação" },
  { value: "entrega", label: "Entrega" },
  { value: "suporte", label: "Suporte" },
];

interface Produto {
  id: string;
  nome: string;
  descricao: string;
  estagio: string;
  utilidade: string;
  garantia: string;
  unidade_id?: string;
}

function rotuloEstagio(estagio: string) {
  return ESTAGIOS.find((item) => item.value === estagio)?.label ?? estagio;
}

function texto(fd: FormData, nome: string) {
  return String(fd.get(nome) ?? "").trim();
}

export default function ProdutosPage() {
  const { can } = usePlataforma();
  const list = useApiQuery<Produto[]>("v1/produtos");
  const { run, pending } = useAction();
  const [edit, setEdit] = useState<Produto | null>(null);
  const gerencia = can("produtos:manage");

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const editando = edit;
    const body = {
      nome: texto(fd, "nome"),
      descricao: texto(fd, "descricao"),
      utilidade: texto(fd, "utilidade"),
      garantia: texto(fd, "garantia"),
      unidade_id: texto(fd, "unidade_id") || null,
      ...(editando ? {} : { estagio: texto(fd, "estagio") }),
    };
    const ok = await run(
      () =>
        editando
          ? apiClient.put(`v1/produtos/${editando.id}`, body)
          : apiClient.post("v1/produtos", body),
      editando ? "Produto atualizado" : "Produto cadastrado",
    );
    if (ok) {
      setEdit(null);
      if (!editando) form.reset();
      void list.mutate();
    }
  }

  async function aplicarEstagio(e: FormEvent<HTMLFormElement>, id: string) {
    e.preventDefault();
    const estagio = texto(new FormData(e.currentTarget), "estagio");
    const ok = await run(
      () => apiClient.post(`v1/produtos/${id}/estagio`, { estagio }),
      "Estágio atualizado",
    );
    if (ok) void list.mutate();
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Ciclo de vida"
        title="Produtos digitais"
        description="O que a organização mantém: da descoberta do serviço até o suporte. O estágio só muda pela ação própria, para o histórico não se misturar com uma correção de texto."
      />

      {gerencia && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">{edit ? "Editar produto" : "Novo produto"}</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form key={edit?.id ?? "novo"} onSubmit={salvar} className="mt-3 grid gap-3">
              <Input
                id="produto-nome"
                name="nome"
                label="Nome *"
                required
                maxLength={160}
                defaultValue={edit?.nome}
                autoComplete="off"
              />
              <Textarea
                id="produto-descricao"
                name="descricao"
                label="Descrição"
                maxLength={4000}
                defaultValue={edit?.descricao}
                rows={3}
              />
              {!edit && (
                <Select
                  id="produto-estagio"
                  name="estagio"
                  label="Estágio inicial *"
                  required
                  defaultValue="descoberta"
                  options={ESTAGIOS}
                />
              )}
              <Input
                id="produto-utilidade"
                name="utilidade"
                label="Utilidade"
                maxLength={4000}
                defaultValue={edit?.utilidade}
                autoComplete="off"
              />
              <Input
                id="produto-garantia"
                name="garantia"
                label="Garantia"
                maxLength={4000}
                defaultValue={edit?.garantia}
                autoComplete="off"
              />
              <UnidadeDonaSelect
                id="produto-unidade"
                permission="produtos:manage"
                label="Unidade dona"
                defaultValue={edit?.unidade_id}
              />
              <div className="flex flex-wrap gap-2">
                <Button type="submit" loading={pending}>
                  {edit ? "Salvar alterações" : "Cadastrar produto"}
                </Button>
                {edit && (
                  <Button type="button" variant="secondary" onClick={() => setEdit(null)}>
                    Cancelar edição
                  </Button>
                )}
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <DataState
        loading={list.isLoading}
        error={list.error}
        onRetry={() => void list.mutate()}
        empty={(list.data ?? []).length === 0}
        emptyTitle="Nenhum produto cadastrado"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {(list.data ?? []).map((produto) => (
            <li key={produto.id} className="flex flex-col gap-3 p-4">
              <div>
                <p className="font-medium">{produto.nome}</p>
                <p className="text-sm text-muted">{rotuloEstagio(produto.estagio)}</p>
                <p className="text-xs text-muted">
                  {produto.unidade_id ? "Da unidade" : "Institucional"}
                </p>
                {produto.descricao ? <p className="mt-1 text-sm">{produto.descricao}</p> : null}
                {can("ativos:manage") && (
                  <Link
                    href={`/ativos?produto=${produto.id}`}
                    className="mt-1 mr-4 inline-block text-sm text-primary hover:underline"
                  >
                    Ver ativos de {produto.nome}
                  </Link>
                )}
                {(can("projetos:read") || can("projetos:manage")) && (
                  <Link
                    href={`/projetos?vista=lista&produto=${produto.id}`}
                    className="mt-1 inline-block text-sm text-primary hover:underline"
                  >
                    Ver projetos de {produto.nome}
                  </Link>
                )}
              </div>
              {gerencia && (
                <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
                  <Button type="button" variant="secondary" onClick={() => setEdit(produto)}>
                    Editar {produto.nome}
                  </Button>
                  <form
                    onSubmit={(e) => void aplicarEstagio(e, produto.id)}
                    className="flex flex-wrap items-end gap-2"
                  >
                    <Select
                      id={`estagio-${produto.id}`}
                      name="estagio"
                      label={`Estágio de ${produto.nome}`}
                      defaultValue={
                        ESTAGIOS.some((item) => item.value === produto.estagio)
                          ? produto.estagio
                          : "descoberta"
                      }
                      options={ESTAGIOS}
                    />
                    <Button type="submit">Aplicar estágio de {produto.nome}</Button>
                  </form>
                </div>
              )}
            </li>
          ))}
        </ul>
      </DataState>
    </div>
  );
}
