"use client";

import Link from "next/link";
import { Pencil, Pin, Plus } from "lucide-react";
import { useState } from "react";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { Pagination } from "@/components/plataforma/Pagination";
import { fmtDate } from "@/components/plataforma/useAction";
import { PublicoBadge } from "@/components/iam/PublicoAlvo";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { useApiPage, withQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { Post } from "@/lib/plataforma/types";

const STATUS_LABEL: Record<Post["status"], string> = { draft: "Rascunho", published: "Publicado", archived: "Arquivado" };

export default function BlogPage() {
  const { can } = usePlataforma();
  const manage = can("blog:manage");
  const [q, setQ] = useState("");
  const [query, setQuery] = useState("");
  const [kind, setKind] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const list = useApiPage<Post>(withQuery("v1/blog/admin/posts", { q: query, kind, status: manage ? status : "", page, page_size: 12 }));

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Comunicação"
        title="Blog & comunicados"
        description="Comunicados e notícias do Projeto Nova."
        actions={
          manage && (
            <Link href="/gestao/blog/novo" className={buttonClass("primary", "md")}>
              <Plus size={16} aria-hidden="true" className="mr-1" /> Nova publicação
            </Link>
          )
        }
      />

      <form
        role="search"
        className="flex flex-wrap items-end gap-3"
        onSubmit={(e) => {
          e.preventDefault();
          setPage(1);
          setQuery(q.trim());
        }}
      >
        <Input id="blog-q" label="Buscar" value={q} onChange={(e) => setQ(e.target.value)} />
        <Select
          id="blog-kind"
          label="Tipo"
          value={kind}
          onChange={(e) => {
            setPage(1);
            setKind(e.target.value);
          }}
          options={[
            { value: "", label: "Todos" },
            { value: "noticia", label: "Notícias" },
            { value: "comunicado", label: "Comunicados" },
          ]}
        />
        {manage && (
          <Select
            id="blog-status"
            label="Situação"
            value={status}
            onChange={(e) => {
              setPage(1);
              setStatus(e.target.value);
            }}
            options={[
              { value: "", label: "Todas" },
              { value: "draft", label: "Rascunhos" },
              { value: "published", label: "Publicadas" },
              { value: "archived", label: "Arquivadas" },
            ]}
          />
        )}
        <Button type="submit" variant="secondary">
          Filtrar
        </Button>
      </form>

      <DataState loading={list.isLoading} error={list.error} onRetry={() => void list.mutate()} empty={(list.data?.items ?? []).length === 0} emptyTitle="Nenhuma publicação">
        <ul className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {(list.data?.items ?? []).map((p) => {
            const cardHref = manage ? `/gestao/blog/${p.slug}` : `/blog/${p.slug}`;
            return (
              <li key={p.id}>
                <Link href={cardHref} className="block h-full rounded-xl focus:outline-none focus-visible:ring-2 focus-visible:ring-primary">
                  <Card className="flex h-full flex-col overflow-hidden transition-all hover:-translate-y-0.5 hover:border-accent/40 hover:shadow-md">
                    {p.cover_url && (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={p.cover_url} alt="" className="h-40 w-full object-cover" />
                    )}
                    <div className="flex flex-1 flex-col gap-2 p-4">
                      <div className="flex flex-wrap items-center gap-1 text-xs text-muted">
                        {p.pinned && <Pin size={12} aria-label="Fixado" />}
                        <Badge tone={p.kind === "comunicado" ? "info" : "neutral"}>{p.kind === "comunicado" ? "Comunicado" : "Notícia"}</Badge>
                        {p.status !== "published" && <Badge tone="warning">{STATUS_LABEL[p.status]}</Badge>}
                        <PublicoBadge publico={p.publico} />
                        <span>{fmtDate(p.published_at ?? p.updated_at)}</span>
                      </div>
                      <h2 className="font-semibold leading-snug">{p.title}</h2>
                      {p.summary && <p className="line-clamp-3 text-sm text-muted">{p.summary}</p>}
                      <div className="mt-auto flex items-center justify-between border-t border-surface-border/60 pt-2.5 text-xs">
                        {manage ? (
                          <span className="inline-flex items-center gap-1 font-semibold text-accent">
                            <Pencil size={12} aria-hidden="true" /> Gerenciar / Editar
                          </span>
                        ) : (
                          <span className="text-muted">por {p.author_name || "Projeto Nova"}</span>
                        )}
                        {manage && p.author_name && (
                          <span className="text-muted text-[11px]">por {p.author_name}</span>
                        )}
                      </div>
                    </div>
                  </Card>
                </Link>
              </li>
            );
          })}
        </ul>
        <Pagination meta={list.data?.meta} onPage={setPage} />
      </DataState>
    </div>
  );
}
