"use client";

import Link from "next/link";
import { ArrowRight, Lock } from "lucide-react";

import { ModuleIcon } from "@/components/layout/ModuleIcon";
import { Badge } from "@/components/ui/Badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { Skeleton } from "@/components/ui/Skeleton";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

/** Módulos ativos no Kernel neste instante (GET /system/modules) — a
 * grade acompanha ativação/desativação em runtime sem recarregar. */
export function ModulesGrid() {
  const { modules, loading } = usePlataforma();

  if (loading && modules.length === 0) {
    return (
      <div role="status" aria-live="polite" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <span className="sr-only">Carregando módulos…</span>
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} className="h-36 w-full" />
        ))}
      </div>
    );
  }

  const active = modules.filter((m) => m.enabled);
  const inactive = modules.filter((m) => !m.enabled);

  return (
    <div className="flex flex-col gap-4">
      <ul className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {active.map((m) => (
          <li key={m.key}>
            <Card className="flex h-full flex-col transition-shadow hover:shadow-md">
              <CardHeader className="flex flex-col gap-2">
                <div className="flex items-start justify-between gap-2">
                  <span className="inline-flex h-10 w-10 items-center justify-center rounded-xl border border-blue-200 bg-blue-50 text-blue-700 shadow-xs dark:border-blue-800/60 dark:bg-blue-950/60 dark:text-blue-300">
                    <ModuleIcon name={m.icon} size={20} aria-hidden="true" />
                  </span>
                  {m.core ? <Badge tone="info">Núcleo</Badge> : <Badge tone="success">Ativo</Badge>}
                </div>
                <CardTitle className="text-base">{m.name}</CardTitle>
                <CardDescription className="text-xs leading-relaxed">{m.description}</CardDescription>
              </CardHeader>
              <CardContent className="mt-auto pb-4">
                {m.route && (
                  <Link href={m.route} className="inline-flex items-center gap-1 text-sm font-semibold text-primary hover:underline">
                    Acessar <span className="sr-only">{m.name}</span> <ArrowRight size={14} aria-hidden="true" />
                  </Link>
                )}
              </CardContent>
            </Card>
          </li>
        ))}
      </ul>
      {inactive.length > 0 && (
        <p className="flex flex-wrap items-center gap-2 text-xs text-muted">
          <Lock size={12} aria-hidden="true" /> Desativados: {inactive.map((m) => m.name).join(", ")}
        </p>
      )}
    </div>
  );
}
