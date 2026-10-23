"use client";

import Link from "next/link";
import { type DragEvent, useState } from "react";
import { AlertCircle, CheckCircle, HelpCircle, Wrench } from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import {
  ESTADO_PROBLEMA,
  type EstadoProblema,
  type Problema,
} from "@/lib/servicos/servicos";

type ColunaProblema =
  | "registrado"
  | "em_analise"
  | "erro_conhecido"
  | "resolvido"
  | "encerrado";

const COLUNAS: { id: ColunaProblema; titulo: string }[] = [
  { id: "registrado", titulo: "Registrados" },
  { id: "em_analise", titulo: "Em análise" },
  { id: "erro_conhecido", titulo: "Erros conhecidos" },
  { id: "resolvido", titulo: "Resolvidos" },
  { id: "encerrado", titulo: "Encerrados" },
];

function aceita(p: Problema, coluna: ColunaProblema): boolean {
  if (p.estado === coluna) return false;
  return (p.proximos ?? []).includes(coluna);
}

export function ProblemasKanban({
  itens,
  onMover,
  pending,
}: {
  itens: Problema[];
  onMover: (p: Problema, para: EstadoProblema) => void;
  pending: boolean;
}) {
  const [arrastado, setArrastado] = useState<Problema | null>(null);
  const [sobre, setSobre] = useState<ColunaProblema | null>(null);

  function soltar(e: DragEvent, coluna: ColunaProblema) {
    e.preventDefault();
    setSobre(null);
    if (arrastado && aceita(arrastado, coluna)) {
      onMover(arrastado, coluna);
    }
    setArrastado(null);
  }

  return (
    <div
      className="grid gap-3 overflow-x-auto pb-2 md:grid-cols-5"
      role="list"
      aria-label="Quadro de problemas"
    >
      {COLUNAS.map((coluna) => {
        const cartoes = itens.filter((p) => p.estado === coluna.id);
        const alvo = arrastado !== null && aceita(arrastado, coluna.id);
        return (
          <section
            key={coluna.id}
            role="listitem"
            aria-labelledby={`coluna-prob-${coluna.id}`}
            data-coluna={coluna.id}
            onDragOver={(e) => {
              if (!alvo) return;
              e.preventDefault();
              setSobre(coluna.id);
            }}
            onDragLeave={() => setSobre((atual) => (atual === coluna.id ? null : atual))}
            onDrop={(e) => soltar(e, coluna.id)}
            className={`flex min-h-48 min-w-60 flex-col gap-2 rounded-xl border p-2 transition-colors ${
              sobre === coluna.id
                ? "border-primary bg-primary/5"
                : alvo
                  ? "border-dashed border-primary/60 bg-surface"
                  : "border-surface-border bg-black/[0.02] dark:bg-white/[0.03]"
            }`}
          >
            <h2
              id={`coluna-prob-${coluna.id}`}
              className="flex items-center justify-between px-1 text-sm font-semibold"
            >
              <span className="flex items-center gap-1.5">
                {coluna.titulo}
              </span>
              <span className="text-xs font-normal text-muted">{cartoes.length}</span>
            </h2>

            <ul className="flex flex-col gap-2">
              {cartoes.map((p) => {
                const moves = p.proximos ?? [];
                return (
                  <li
                    key={p.id}
                    draggable={!pending && moves.length > 0}
                    onDragStart={(e) => {
                      e.dataTransfer.effectAllowed = "move";
                      setArrastado(p);
                    }}
                    onDragEnd={() => {
                      setArrastado(null);
                      setSobre(null);
                    }}
                    className="flex flex-col gap-2 rounded-lg border border-surface-border bg-surface p-3 shadow-sm"
                  >
                    <div className="flex items-center justify-between gap-2 text-xs text-muted">
                      <span className="font-semibold tabular-nums text-foreground">#{p.numero}</span>
                      <Badge tone={ESTADO_PROBLEMA[p.estado].tone}>
                        {ESTADO_PROBLEMA[p.estado].label}
                      </Badge>
                    </div>

                    <Link
                      href={`/problemas/${p.id}`}
                      className="text-sm font-medium text-primary hover:underline line-clamp-2"
                    >
                      {p.titulo}
                    </Link>

                    {p.contorno && (
                      <p className="rounded bg-warning/10 p-1.5 text-xs text-muted line-clamp-2">
                        <strong className="text-foreground">Contorno:</strong> {p.contorno}
                      </p>
                    )}

                    {p.causa_raiz && (
                      <p className="rounded bg-success/10 p-1.5 text-xs text-muted line-clamp-2">
                        <strong className="text-foreground">Causa raiz:</strong> {p.causa_raiz}
                      </p>
                    )}

                    {(p.incidentes ?? []).length > 0 && (
                      <p className="text-[11px] text-muted">
                        {p.incidentes.length} {p.incidentes.length === 1 ? "incidente ligado" : "incidentes ligados"}
                      </p>
                    )}

                    {moves.length > 0 && (
                      <label className="mt-1 flex flex-col gap-1 text-xs">
                        <span className="sr-only">Mover o problema #{p.numero} para</span>
                        <select
                          value=""
                          disabled={pending}
                          onChange={(e) => {
                            if (e.target.value) onMover(p, e.target.value as EstadoProblema);
                          }}
                          className="h-8 rounded-md border border-surface-border bg-surface px-2 text-xs text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                        >
                          <option value="">Mover para…</option>
                          {moves.map((m) => (
                            <option key={m} value={m}>
                              {ESTADO_PROBLEMA[m].label}
                            </option>
                          ))}
                        </select>
                      </label>
                    )}
                  </li>
                );
              })}
            </ul>
          </section>
        );
      })}
    </div>
  );
}
