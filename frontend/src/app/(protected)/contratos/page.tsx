"use client";

import { type FormEvent, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Fornecedor {
  id: string;
  nome: string;
  email: string;
  telefone: string;
}

interface Contrato {
  id: string;
  numero: string;
  titulo: string;
  fornecedor_nome: string;
  inicio: string;
  fim: string;
  valor_centavos: number;
}

interface Licenca {
  id: string;
  nome: string;
  fornecedor_nome: string;
  quantidade: number;
  fim?: string;
}

function dia(valor: string) {
  return valor ? `${valor}T00:00:00Z` : "";
}

/** Fornecedores, contratos e licenças. A licença não pede chave secreta. */
export default function ContratosPage() {
  const { can } = usePlataforma();
  const gere = can("contratos:manage");
  const fornecedores = useApiQuery<Fornecedor[]>(gere ? "v1/contratos/fornecedores" : null);
  const contratos = useApiQuery<Contrato[]>(gere ? "v1/contratos/contratos" : null);
  const licencas = useApiQuery<Licenca[]>(gere ? "v1/contratos/licencas" : null);
  const { run, pending } = useAction();
  const [aberto, setAberto] = useState<"fornecedor" | "contrato" | "licenca" | null>(null);

  if (!gere) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Gestão de serviços" title="Contratos" />
        <EmptyState
          title="Seu perfil não gere contratos"
          description="Peça o acesso à gestão da sua unidade."
        />
      </div>
    );
  }

  const opcoes = (fornecedores.data ?? []).map((f) => ({ value: f.id, label: f.nome }));

  async function salvarFornecedor(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(
      () =>
        apiClient.post("v1/contratos/fornecedores", {
          nome: String(fd.get("nome")).trim(),
          email: String(fd.get("email") ?? "").trim(),
          telefone: String(fd.get("telefone") ?? "").trim(),
          unidade_id: String(fd.get("unidade_id") || "") || null,
        }),
      "Fornecedor cadastrado",
    );
    if (ok) {
      form.reset();
      setAberto(null);
      void fornecedores.mutate();
    }
  }

  async function salvarContrato(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(
      () =>
        apiClient.post("v1/contratos/contratos", {
          numero: String(fd.get("numero")).trim(),
          titulo: String(fd.get("titulo")).trim(),
          fornecedor_id: String(fd.get("fornecedor_id")),
          inicio: dia(String(fd.get("inicio"))),
          fim: dia(String(fd.get("fim"))),
          valor_centavos: Math.round(Number(String(fd.get("valor") || "0").replace(",", ".")) * 100),
          unidade_id: String(fd.get("unidade_id") || "") || null,
        }),
      "Contrato cadastrado",
    );
    if (ok) {
      form.reset();
      setAberto(null);
      void contratos.mutate();
    }
  }

  async function salvarLicenca(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const fim = String(fd.get("fim") ?? "");
    const ok = await run(
      () =>
        apiClient.post("v1/contratos/licencas", {
          nome: String(fd.get("nome")).trim(),
          fornecedor_id: String(fd.get("fornecedor_id")),
          quantidade: Number(fd.get("quantidade")),
          fim: fim ? dia(fim) : null,
          unidade_id: String(fd.get("unidade_id") || "") || null,
        }),
      "Licença cadastrada",
    );
    if (ok) {
      form.reset();
      setAberto(null);
      void licencas.mutate();
    }
  }

  async function excluir(caminho: string, recarregar: () => void) {
    const ok = await run(() => apiClient.delete(caminho), "Cadastro removido");
    if (ok) recarregar();
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Gestão de serviços"
        title="Contratos e licenças"
        description="Fornecedor, vigência e quantidade de assentos. A chave da licença não é gravada aqui. O sino avisa 30 dias, 7 dias, no dia e depois do vencimento."
      />

      <Card>
        <CardHeader>
          <CardTitle as="h2">Fornecedores</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 pb-5">
          <Button type="button" variant="secondary" onClick={() => setAberto(aberto === "fornecedor" ? null : "fornecedor")}>
            Novo fornecedor
          </Button>
          {aberto === "fornecedor" ? (
            <form onSubmit={salvarFornecedor} className="grid gap-3 sm:grid-cols-2">
              <Input id="fornecedor-nome" name="nome" label="Nome *" required maxLength={160} />
              <Input id="fornecedor-email" name="email" type="email" label="E-mail" maxLength={160} />
              <Input id="fornecedor-telefone" name="telefone" label="Telefone" maxLength={40} />
              <UnidadeDonaSelect id="fornecedor-unidade" permission="contratos:manage" label="Unidade" />
              <div>
                <Button type="submit" loading={pending}>
                  Cadastrar fornecedor
                </Button>
              </div>
            </form>
          ) : null}
          <DataState
            loading={fornecedores.isLoading}
            error={fornecedores.error}
            onRetry={() => void fornecedores.mutate()}
            empty={(fornecedores.data ?? []).length === 0}
            emptyTitle="Nenhum fornecedor"
          >
            <ul className="divide-y divide-surface-border text-sm">
              {(fornecedores.data ?? []).map((f) => (
                <li key={f.id} className="flex items-center justify-between gap-2 py-2">
                  <span>
                    {f.nome}
                    {f.email ? ` · ${f.email}` : ""}
                  </span>
                  <Button type="button" variant="secondary" onClick={() => void excluir(`v1/contratos/fornecedores/${f.id}`, () => void fornecedores.mutate())}>
                    Remover {f.nome}
                  </Button>
                </li>
              ))}
            </ul>
          </DataState>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle as="h2">Contratos</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 pb-5">
          <Button type="button" variant="secondary" onClick={() => setAberto(aberto === "contrato" ? null : "contrato")} disabled={opcoes.length === 0}>
            Novo contrato
          </Button>
          {aberto === "contrato" ? (
            <form onSubmit={salvarContrato} className="grid gap-3 sm:grid-cols-2">
              <Input id="contrato-numero" name="numero" label="Número *" required maxLength={80} />
              <Input id="contrato-titulo" name="titulo" label="Título *" required maxLength={160} />
              <Select id="contrato-fornecedor" name="fornecedor_id" label="Fornecedor *" required options={opcoes} />
              <Input id="contrato-valor" name="valor" label="Valor em reais" inputMode="decimal" />
              <Input id="contrato-inicio" name="inicio" type="date" label="Início *" required />
              <Input id="contrato-fim" name="fim" type="date" label="Fim *" required />
              <UnidadeDonaSelect id="contrato-unidade" permission="contratos:manage" label="Unidade" />
              <div>
                <Button type="submit" loading={pending}>
                  Cadastrar contrato
                </Button>
              </div>
            </form>
          ) : null}
          <DataState
            loading={contratos.isLoading}
            error={contratos.error}
            onRetry={() => void contratos.mutate()}
            empty={(contratos.data ?? []).length === 0}
            emptyTitle="Nenhum contrato"
          >
            <ul className="divide-y divide-surface-border text-sm">
              {(contratos.data ?? []).map((c) => (
                <li key={c.id} className="flex items-center justify-between gap-2 py-2">
                  <span>
                    {c.numero} · {c.titulo} · {c.fornecedor_nome} · até {c.fim.slice(0, 10)}
                  </span>
                  <Button type="button" variant="secondary" onClick={() => void excluir(`v1/contratos/contratos/${c.id}`, () => void contratos.mutate())}>
                    Remover contrato {c.numero}
                  </Button>
                </li>
              ))}
            </ul>
          </DataState>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle as="h2">Licenças</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 pb-5">
          <Button type="button" variant="secondary" onClick={() => setAberto(aberto === "licenca" ? null : "licenca")} disabled={opcoes.length === 0}>
            Nova licença
          </Button>
          {aberto === "licenca" ? (
            <form onSubmit={salvarLicenca} className="grid gap-3 sm:grid-cols-2">
              <Input id="licenca-nome" name="nome" label="Nome *" required maxLength={160} />
              <Select id="licenca-fornecedor" name="fornecedor_id" label="Fornecedor *" required options={opcoes} />
              <Input id="licenca-quantidade" name="quantidade" type="number" min={1} label="Assentos *" required defaultValue={1} />
              <Input id="licenca-fim" name="fim" type="date" label="Vencimento" />
              <UnidadeDonaSelect id="licenca-unidade" permission="contratos:manage" label="Unidade" />
              <div>
                <Button type="submit" loading={pending}>
                  Cadastrar licença
                </Button>
              </div>
            </form>
          ) : null}
          <DataState
            loading={licencas.isLoading}
            error={licencas.error}
            onRetry={() => void licencas.mutate()}
            empty={(licencas.data ?? []).length === 0}
            emptyTitle="Nenhuma licença"
          >
            <ul className="divide-y divide-surface-border text-sm">
              {(licencas.data ?? []).map((l) => (
                <li key={l.id} className="flex items-center justify-between gap-2 py-2">
                  <span>
                    {l.nome} · {l.quantidade} assentos · {l.fornecedor_nome}
                    {l.fim ? ` · até ${l.fim.slice(0, 10)}` : ""}
                  </span>
                  <Button type="button" variant="secondary" onClick={() => void excluir(`v1/contratos/licencas/${l.id}`, () => void licencas.mutate())}>
                    Remover {l.nome}
                  </Button>
                </li>
              ))}
            </ul>
          </DataState>
        </CardContent>
      </Card>
    </div>
  );
}
