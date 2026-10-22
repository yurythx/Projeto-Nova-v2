"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { ArrowRight, Search, X } from "lucide-react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ServiceIcon, getCategoryIconStyles } from "@/lib/catalog/icons";
import type { CatalogService } from "@/lib/sismarq/types";

interface ServicosCatalogClientProps {
  initialServices: CatalogService[];
}

export function ServicosCatalogClient({ initialServices }: ServicosCatalogClientProps) {
  const [search, setSearch] = useState("");
  const [selectedCategory, setSelectedCategory] = useState<string>("Todas");

  // Extract unique categories and counts
  const categories = useMemo(() => {
    const counts = new Map<string, number>();
    for (const svc of initialServices) {
      const cat = svc.category || "Geral";
      counts.set(cat, (counts.get(cat) || 0) + 1);
    }

    const list = Array.from(counts.entries()).map(([name, count]) => ({
      name,
      count,
    }));

    // Sort alphabetically
    list.sort((a, b) => a.name.localeCompare(b.name, "pt-BR"));

    return [{ name: "Todas", count: initialServices.length }, ...list];
  }, [initialServices]);

  // Filter services by search and category
  const filteredServices = useMemo(() => {
    const q = search.trim().toLowerCase();
    return initialServices.filter((svc) => {
      const matchesCategory =
        selectedCategory === "Todas" || (svc.category || "Geral") === selectedCategory;

      if (!matchesCategory) return false;

      if (!q) return true;

      const title = svc.title.toLowerCase();
      const summary = (svc.summary || "").toLowerCase();
      const cat = (svc.category || "").toLowerCase();
      const slug = svc.slug.toLowerCase();

      return (
        title.includes(q) ||
        summary.includes(q) ||
        cat.includes(q) ||
        slug.includes(q)
      );
    });
  }, [initialServices, selectedCategory, search]);

  return (
    <div className="flex flex-col gap-8">
      {/* Search and Filters */}
      <div className="flex flex-col gap-4">
        {/* Search Input */}
        <div className="relative max-w-md">
          <label htmlFor="service-search" className="sr-only">
            Buscar serviços
          </label>
          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-muted">
            <Search size={18} aria-hidden="true" />
          </div>
          <input
            id="service-search"
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Buscar por nome, categoria ou palavra-chave..."
            className="w-full min-h-[44px] rounded-lg border border-surface-border bg-surface-1 py-2.5 pl-10 pr-9 text-sm text-foreground placeholder:text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
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

        {/* Category Pills - Scroll horizontal no mobile, wrap no desktop */}
        <div
          role="tablist"
          aria-label="Categorias de serviços"
          className="flex items-center gap-2 overflow-x-auto pb-2 -mx-2 px-2 scrollbar-none sm:mx-0 sm:px-0 sm:flex-wrap sm:overflow-visible"
        >
          {categories.map((cat) => {
            const isSelected = selectedCategory === cat.name;
            return (
              <button
                key={cat.name}
                type="button"
                role="tab"
                aria-selected={isSelected}
                onClick={() => setSelectedCategory(cat.name)}
                className={`inline-flex shrink-0 items-center gap-1.5 rounded-full px-3.5 py-2 text-xs font-medium transition-all ${
                  isSelected
                    ? "bg-primary text-primary-foreground shadow-sm font-semibold"
                    : "border border-surface-border bg-surface-2 text-muted hover:bg-surface-3 hover:text-foreground"
                }`}
              >
                <span>{cat.name}</span>
                <span
                  className={`rounded-full px-1.5 py-0.2 text-[10px] ${
                    isSelected
                      ? "bg-white/20 text-primary-foreground"
                      : "bg-surface-3 text-muted"
                  }`}
                >
                  {cat.count}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Services Grid or Empty State */}
      {filteredServices.length === 0 ? (
        <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed border-surface-border py-16 text-center text-muted">
          <p className="max-w-md text-sm">
            {initialServices.length === 0
              ? "Nenhum serviço publicado ainda"
              : "Nenhum serviço encontrado para o filtro aplicado."}
          </p>
          {(search || selectedCategory !== "Todas") && (
            <button
              type="button"
              onClick={() => {
                setSearch("");
                setSelectedCategory("Todas");
              }}
              className="mt-2 text-xs font-semibold text-primary hover:underline"
            >
              Limpar filtros de busca
            </button>
          )}
        </div>
      ) : (
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {filteredServices.map((svc) => {
            const iconStyle = getCategoryIconStyles(svc.category);
            return (
              <Link key={svc.id} href={`/servicos/${svc.slug}`} className="group block h-full">
                <Card className="flex h-full flex-col border-surface-border transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md">
                  <CardHeader className="flex flex-col gap-3">
                    <div className="flex items-center justify-between gap-2">
                      <span className={`inline-flex h-11 w-11 items-center justify-center rounded-xl shadow-xs transition-transform group-hover:scale-105 ${iconStyle.badge}`}>
                        <ServiceIcon name={svc.icon} size={22} aria-hidden="true" />
                      </span>
                      {svc.category && (
                        <span className={`rounded-md border px-2.5 py-0.5 text-[11px] font-semibold ${iconStyle.tag}`}>
                          {svc.category}
                        </span>
                      )}
                    </div>
                    <CardTitle as="h2" className="text-lg font-bold text-foreground group-hover:text-primary">
                      {svc.title}
                    </CardTitle>
                  </CardHeader>
                <CardContent className="flex flex-1 flex-col justify-between gap-4">
                  <CardDescription className="line-clamp-3 text-sm leading-relaxed text-muted">
                    {svc.summary}
                  </CardDescription>
                  <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-accent transition-transform group-hover:translate-x-1">
                    Ver o serviço <ArrowRight size={13} aria-hidden="true" />
                  </span>
                </CardContent>
              </Card>
            </Link>
          );
        })}
        </div>
      )}
    </div>
  );
}
