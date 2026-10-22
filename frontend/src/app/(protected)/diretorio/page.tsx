"use client";

import { useRouter } from "next/navigation";
import { Building, Building2, Mail, MapPin, MessagesSquare, Phone, Search } from "lucide-react";
import { useMemo, useState } from "react";

import { ScopePicker } from "@/components/iam/ScopePicker";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { Pagination } from "@/components/plataforma/Pagination";
import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { ChatRoom, Person, ScopeRef, Sector } from "@/lib/plataforma/types";

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase())
    .join("");
}

function People() {
  const { enabled, me } = usePlataforma();
  const router = useRouter();
  const { run } = useAction();
  const [q, setQ] = useState("");
  const [query, setQuery] = useState("");
  const [scope, setScope] = useState<ScopeRef>({});
  const [page, setPage] = useState(1);
  const list = useApiPage<Person>(
    withQuery("v1/directory/people", {
      q: query,
      unidade_id: scope.unidade_id,
      departamento_id: scope.departamento_id,
      page,
      page_size: 24,
    }),
  );

  async function chat(p: Person) {
    const res = await run(() =>
      apiClient.post<ChatRoom>("v1/mercurio/direct", { user_id: p.user_id }),
    );
    if (res) router.push(`/mercurio?sala=${res.data.id}`);
  }

  return (
    <div className="flex flex-col gap-4">
      <form
        role="search"
        className="flex flex-col gap-3"
        onSubmit={(e) => {
          e.preventDefault();
          setPage(1);
          setQuery(q.trim());
        }}
      >
        <div className="flex items-end gap-2">
          <div className="flex-1">
            <Input
              id="dir-q"
              label="Nome, cargo ou e-mail"
              value={q}
              onChange={(e) => setQ(e.target.value)}
            />
          </div>
          <Button type="submit" variant="secondary">
            <Search size={14} aria-hidden="true" className="mr-1" /> Buscar
          </Button>
        </div>
        <ScopePicker
          idPrefix="dir-filter"
          value={scope}
          onChange={(v) => {
            setPage(1);
            setScope(v);
          }}
        />
      </form>
      <DataState
        loading={list.isLoading}
        error={list.error}
        onRetry={() => void list.mutate()}
        empty={(list.data?.items ?? []).length === 0}
        emptyTitle="Ninguém encontrado"
      >
        <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {(list.data?.items ?? []).map((p) => (
            <li key={p.user_id}>
              <Card className="flex h-full gap-3 p-4">
                <span
                  aria-hidden="true"
                  className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-semibold text-primary"
                >
                  {initials(p.name || p.username)}
                </span>
                <div className="flex min-w-0 flex-1 flex-col gap-1 text-sm">
                  <p className="font-semibold">{p.name || p.username}</p>
                  {p.job_title && <p className="text-xs text-muted">{p.job_title}</p>}
                  {(p.departamento || p.unidade) && (
                    <p className="flex items-center gap-1 text-xs text-muted">
                      <Building size={12} aria-hidden="true" />{" "}
                      {[p.departamento, p.unidade].filter(Boolean).join(" · ")}
                    </p>
                  )}
                  {p.email && (
                    <a
                      href={`mailto:${p.email}`}
                      className="flex items-center gap-1 truncate text-xs text-primary hover:underline"
                    >
                      <Mail size={12} aria-hidden="true" /> {p.email}
                    </a>
                  )}
                  {(p.phone || p.extension) && (
                    <p className="flex items-center gap-1 text-xs">
                      <Phone size={12} aria-hidden="true" /> {p.phone}{" "}
                      {p.extension && `ramal ${p.extension}`}
                    </p>
                  )}
                  <div className="mt-auto flex items-center justify-between pt-1">
                    {p.from_ad ? <Badge tone="info">AD</Badge> : <span />}
                    {enabled("mercurio") && p.user_id !== me?.id && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => void chat(p)}
                        aria-label={`Conversar com ${p.name || p.username}`}
                      >
                        <MessagesSquare size={14} aria-hidden="true" />
                      </Button>
                    )}
                  </div>
                </div>
              </Card>
            </li>
          ))}
        </ul>
        <Pagination meta={list.data?.meta} onPage={setPage} />
      </DataState>
    </div>
  );
}

function Sectors() {
  const sectors = useApiQuery<Sector[]>("v1/directory/sectors");
  const [q, setQ] = useState("");
  const [entidadeFilter, setEntidadeFilter] = useState("");
  const [kindFilter, setKindFilter] = useState<"todos" | "unidade" | "departamento">("todos");

  const rawList = useMemo(() => sectors.data ?? [], [sectors.data]);

  const entidades = useMemo(() => {
    const set = new Set<string>();
    rawList.forEach((s) => {
      if (s.entidade) set.add(s.entidade);
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b));
  }, [rawList]);

  const unidadesCount = useMemo(
    () => rawList.filter((s) => s.kind === "unidade").length,
    [rawList],
  );
  const departamentosCount = useMemo(
    () => rawList.filter((s) => s.kind === "departamento").length,
    [rawList],
  );

  const filtered = useMemo(() => {
    const term = q.trim().toLowerCase();
    return rawList.filter((s) => {
      if (entidadeFilter && s.entidade !== entidadeFilter) return false;
      if (kindFilter !== "todos" && s.kind !== kindFilter) return false;
      if (!term) return true;
      const haystack =
        `${s.nome} ${s.sigla} ${s.entidade} ${s.unidade || ""} ${s.endereco || ""} ${s.email || ""} ${s.telefone || ""}`.toLowerCase();
      return haystack.includes(term);
    });
  }, [rawList, q, entidadeFilter, kindFilter]);

  return (
    <DataState
      loading={sectors.isLoading}
      error={sectors.error}
      empty={rawList.length === 0}
      emptyTitle="Nenhum setor cadastrado"
    >
      <div className="flex flex-col gap-5">
        <div className="rounded-xl border border-surface-border bg-surface-elevated/50 p-4">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <p className="flex items-center gap-2 text-sm font-semibold text-foreground">
                <Building2 size={16} className="text-primary" aria-hidden="true" />
                Estrutura Organizacional & Lotações
              </p>
              <p className="text-xs text-muted max-w-2xl">
                O diretório organiza as lotações em 3 níveis: <strong>Entidades</strong>,{" "}
                <strong>Unidades</strong> e <strong>Departamentos</strong>. O módulo nasce
                desligado.
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2 pt-2 sm:pt-0">
              <Badge tone="neutral">{unidadesCount} Unidades</Badge>
              <Badge tone="info">{departamentosCount} Departamentos</Badge>
            </div>
          </div>
        </div>

        <div className="grid gap-3 sm:grid-cols-12 items-end">
          <div className="sm:col-span-5">
            <Input
              id="sector-search"
              label="Buscar por nome, sigla ou telefone"
              placeholder="Ex: UBS, Escola, RH, Ramal..."
              value={q}
              onChange={(e) => setQ(e.target.value)}
            />
          </div>
          <div className="sm:col-span-4">
            <Select
              id="sector-entidade"
              label="Entidade (Secretaria)"
              placeholder="Todas as secretarias / órgãos"
              value={entidadeFilter}
              onChange={(e) => setEntidadeFilter(e.target.value)}
              options={entidades.map((e) => ({ value: e, label: e }))}
            />
          </div>
          <div className="sm:col-span-3">
            <Select
              id="sector-kind"
              label="Nível hierárquico"
              value={kindFilter}
              onChange={(e) =>
                setKindFilter(e.target.value as "todos" | "unidade" | "departamento")
              }
              options={[
                { value: "todos", label: "Todos os níveis" },
                { value: "unidade", label: "Apenas Unidades" },
                { value: "departamento", label: "Apenas Departamentos" },
              ]}
            />
          </div>
        </div>

        <div className="flex items-center justify-between text-xs text-muted">
          <span>
            Exibindo <strong>{filtered.length}</strong> de {rawList.length} registros
          </span>
          {(q || entidadeFilter || kindFilter !== "todos") && (
            <button
              type="button"
              onClick={() => {
                setQ("");
                setEntidadeFilter("");
                setKindFilter("todos");
              }}
              className="text-primary hover:underline"
            >
              Limpar filtros
            </button>
          )}
        </div>

        {filtered.length === 0 ? (
          <div className="rounded-xl border border-dashed border-surface-border p-8 text-center">
            <p className="text-sm font-medium text-foreground">Nenhum resultado encontrado</p>
            <p className="mt-1 text-xs text-muted">
              Tente ajustar a busca por texto ou selecionar outra entidade.
            </p>
          </div>
        ) : (
          <ul className="grid gap-3 md:grid-cols-2">
            {filtered.map((s) => (
              <li
                key={s.id}
                className={`rounded-xl border border-surface-border bg-surface p-4 text-sm transition-all hover:border-primary/40 ${
                  s.kind === "departamento" ? "ml-3 border-l-4 border-l-primary/60" : ""
                }`}
              >
                <div className="flex items-start justify-between gap-2">
                  <p className="font-semibold text-foreground">
                    {s.nome} {s.sigla && <span className="text-muted">({s.sigla})</span>}
                  </p>
                  <Badge tone={s.kind === "departamento" ? "info" : "neutral"} className="shrink-0">
                    {s.kind === "departamento" ? "Departamento" : "Unidade"}
                  </Badge>
                </div>
                <p className="text-xs text-muted mt-1">
                  {s.kind === "departamento"
                    ? `Departamento · ${s.unidade}`
                    : `Unidade · ${s.entidade}`}
                </p>
                {s.kind === "departamento" && s.entidade && (
                  <p className="text-xs text-muted/80">Órgão: {s.entidade}</p>
                )}
                <div className="mt-3 flex flex-col gap-1 text-xs">
                  {s.email && (
                    <a
                      href={`mailto:${s.email}`}
                      className="flex items-center gap-1.5 text-primary hover:underline"
                    >
                      <Mail size={13} aria-hidden="true" className="shrink-0" />
                      {s.email}
                    </a>
                  )}
                  {s.telefone && (
                    <p className="flex items-center gap-1.5 text-foreground">
                      <Phone size={13} aria-hidden="true" className="shrink-0 text-muted" />
                      {s.telefone}
                    </p>
                  )}
                  {s.endereco && (
                    <p className="flex items-start gap-1.5 text-muted">
                      <MapPin size={13} aria-hidden="true" className="shrink-0 mt-0.5" />
                      {s.endereco}
                    </p>
                  )}
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </DataState>
  );
}

export default function DiretorioPage() {
  const [tab, setTab] = useState<"pessoas" | "setores">("pessoas");
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Diretório"
        title="Pessoas e setores"
        description="Servidores e setores da Superintendência e das unidades, a partir do Active Directory."
      />
      <SectionTabsInline
        label="Visões do diretório"
        value={tab}
        onChange={setTab}
        tabs={[
          { value: "pessoas", label: "Pessoas" },
          { value: "setores", label: "Setores" },
        ]}
      />
      {tab === "pessoas" ? <People /> : <Sectors />}
    </div>
  );
}
