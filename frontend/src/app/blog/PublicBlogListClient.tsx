"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { ArrowRight, Calendar, Clock, Newspaper, Rss, Search, X } from "lucide-react";

import { BlogCoverImage } from "@/components/blog/BlogCoverImage";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import type { Post } from "@/lib/plataforma/types";

// Filtros pelo tipo da publicação.
const TIPO_DO_FILTRO = { Notícias: "noticia", Comunicados: "comunicado" } as const;
type Filtro = "Todos" | keyof typeof TIPO_DO_FILTRO;
const FILTROS: Filtro[] = ["Todos", "Notícias", "Comunicados"];

interface PublicBlogListClientProps {
  initialPosts: Post[];
}

export function PublicBlogListClient({ initialPosts }: PublicBlogListClientProps) {
  const [search, setSearch] = useState("");
  const [selectedTag, setSelectedTag] = useState<Filtro>("Todos");

  const filteredPosts = useMemo(() => {
    const q = search.trim().toLowerCase();
    return initialPosts.filter((post) => {
      if (selectedTag !== "Todos" && post.kind !== TIPO_DO_FILTRO[selectedTag]) return false;

      if (!q) return true;

      const title = post.title.toLowerCase();
      const summary = (post.summary || "").toLowerCase();
      const body = (post.body || "").toLowerCase();
      return title.includes(q) || summary.includes(q) || body.includes(q);
    });
  }, [initialPosts, selectedTag, search]);

  const formatDate = (dateStr?: string | null) => {
    if (!dateStr) return "";
    try {
      return new Intl.DateTimeFormat("pt-BR", {
        day: "2-digit",
        month: "short",
        year: "numeric",
      }).format(new Date(dateStr));
    } catch {
      return dateStr;
    }
  };

  const estimateReadTime = (body = "", summary = "") => {
    if (body) {
      const words = body.split(/\s+/).length;
      return `${Math.max(1, Math.ceil(words / 200))} min de leitura`;
    }
    // Fallback proporcional se apenas o resumo estiver carregado
    const words = (summary || "").split(/\s+/).length * 8;
    return `${Math.max(3, Math.ceil(words / 200))} min de leitura`;
  };

  const isBrowsingAll = !search && selectedTag === "Todos";
  const heroPost = isBrowsingAll ? filteredPosts[0] : null;
  const gridPosts = isBrowsingAll ? filteredPosts.slice(1) : filteredPosts;

  return (
    <div className="flex flex-col gap-10">
      {/* Busca, Filtros e Link RSS */}
      <div className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="relative w-full max-w-md">
            <label htmlFor="blog-search" className="sr-only">
              Buscar artigos
            </label>
            <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-muted">
              <Search size={18} aria-hidden="true" />
            </div>
            <input
              id="blog-search"
              type="search"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Buscar por assunto ou termo..."
              className="w-full min-h-[44px] rounded-lg border border-surface-border bg-surface-1 py-2.5 pl-10 pr-9 text-sm text-foreground placeholder:text-muted focus:border-accent focus:outline-hidden focus:ring-1 focus:ring-accent"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearch("")}
                className="absolute inset-y-0 right-0 flex min-h-[44px] min-w-[44px] items-center justify-center pr-3 text-muted hover:text-foreground"
                aria-label="Limpar busca"
              >
                <X size={16} aria-hidden="true" />
              </button>
            )}
          </div>

          <a
            href="/blog/rss.xml"
            target="_blank"
            rel="noopener noreferrer"
            title="Assinar Feed RSS"
            className="inline-flex min-h-[40px] items-center gap-1.5 rounded-lg border border-surface-border bg-surface px-3 py-2 text-xs font-medium text-foreground transition-colors hover:bg-surface-2 hover:text-accent"
          >
            <Rss size={14} className="text-amber-500" aria-hidden="true" />
            <span>Feed RSS</span>
          </a>
        </div>

        <div
          role="tablist"
          aria-label="Filtros do blog"
          className="flex items-center gap-2 overflow-x-auto pb-2 -mx-2 px-2 scrollbar-none sm:mx-0 sm:px-0 sm:flex-wrap sm:overflow-visible"
        >
          {FILTROS.map((tag) => {
            const isSelected = selectedTag === tag;
            return (
              <button
                key={tag}
                type="button"
                role="tab"
                aria-selected={isSelected}
                onClick={() => setSelectedTag(tag)}
                className={`inline-flex shrink-0 items-center gap-1.5 rounded-full px-3.5 py-2 text-xs font-medium transition-all ${
                  isSelected
                    ? "bg-accent font-semibold text-accent-foreground shadow-xs"
                    : "bg-surface-2 text-muted hover:bg-surface-3 hover:text-foreground"
                }`}
              >
                {tag}
              </button>
            );
          })}
        </div>
      </div>

      {/* Hero Featured Post (quando não filtrado) */}
      {heroPost && (
        <section aria-labelledby="hero-title" className="group">
          <Link
            href={`/blog/${heroPost.slug}`}
            className="grid gap-6 overflow-hidden rounded-2xl border border-surface-border bg-surface p-6 shadow-xs transition-all duration-300 hover:border-accent/50 hover:shadow-lg md:grid-cols-2 md:items-center"
          >
            <div className="overflow-hidden rounded-xl">
              <BlogCoverImage
                coverUrl={heroPost.cover_url}
                title={heroPost.title}
                slug={heroPost.slug}
                aspect="hero"
              />
            </div>

            <div className="flex flex-col gap-3">
              <div className="flex items-center gap-2 text-xs text-muted">
                <span className="rounded bg-accent/15 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-accent">
                  Destaque Editorial
                </span>
                <span>•</span>
                <span className="inline-flex items-center gap-1">
                  <Calendar size={12} aria-hidden="true" />
                  {formatDate(heroPost.published_at || heroPost.updated_at)}
                </span>
                <span>•</span>
                <span className="inline-flex items-center gap-1">
                  <Clock size={12} aria-hidden="true" />
                  {estimateReadTime(heroPost.body, heroPost.summary)}
                </span>
              </div>

              <h2
                id="hero-title"
                className="text-2xl font-extrabold tracking-tight text-foreground transition-colors group-hover:text-accent sm:text-3xl"
              >
                {heroPost.title}
              </h2>

              <p className="line-clamp-3 text-sm leading-relaxed text-muted sm:text-base">
                {heroPost.summary}
              </p>

              <div className="mt-2 inline-flex items-center gap-2 text-sm font-semibold text-accent">
                <span>Continuar lendo o artigo</span>
                <ArrowRight
                  size={16}
                  className="transition-transform group-hover:translate-x-1"
                  aria-hidden="true"
                />
              </div>
            </div>
          </Link>
        </section>
      )}

      {/* Posts Grid */}
      {filteredPosts.length === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-surface-border py-16 text-center text-muted">
          <Newspaper size={32} aria-hidden="true" />
          <p className="max-w-md text-sm">
            Nenhuma publicação encontrada para o critério selecionado.
          </p>
          {(search || selectedTag !== "Todos") && (
            <button
              type="button"
              onClick={() => {
                setSearch("");
                setSelectedTag("Todos");
              }}
              className="mt-2 text-xs font-semibold text-accent hover:underline"
            >
              Limpar filtros
            </button>
          )}
        </div>
      ) : (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {gridPosts.map((post) => (
            <Link key={post.id} href={`/blog/${post.slug}`} className="group block h-full">
              <Card className="flex h-full flex-col overflow-hidden border-surface-border transition-all duration-200 hover:-translate-y-0.5 hover:border-accent/40 hover:shadow-md">
                <div className="overflow-hidden">
                  <BlogCoverImage
                    coverUrl={post.cover_url}
                    title={post.title}
                    slug={post.slug}
                    aspect="card"
                  />
                </div>

                <CardHeader className="flex flex-col gap-2.5 p-5">
                  <div className="flex items-center justify-between text-xs text-muted">
                    <span className="inline-flex items-center gap-1">
                      <Calendar size={13} aria-hidden="true" />
                      {formatDate(post.published_at || post.updated_at)}
                    </span>
                    <span className="inline-flex items-center gap-1">
                      <Clock size={13} aria-hidden="true" />
                      {estimateReadTime(post.body, post.summary)}
                    </span>
                  </div>
                  <CardTitle
                    as="h3"
                    className="line-clamp-2 text-base font-bold leading-snug text-foreground group-hover:text-accent"
                  >
                    {post.title}
                  </CardTitle>
                </CardHeader>
                <CardContent className="flex flex-1 flex-col justify-between gap-4 px-5 pb-5 pt-0">
                  <CardDescription className="line-clamp-3 text-sm leading-relaxed text-muted">
                    {post.summary}
                  </CardDescription>
                  <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-accent transition-transform group-hover:translate-x-1">
                    Ler artigo completo <ArrowRight size={13} aria-hidden="true" />
                  </span>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
