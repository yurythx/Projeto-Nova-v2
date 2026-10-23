"use client";

import Link from "next/link";
import { type DragEvent, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import {
  ESTADO_PROJETO,
  PRIORIDADE_PROJETO,
  type EstadoProjeto,
  type Projeto,
} from "@/lib/projetos/projetos";

type ColunaProjeto = EstadoProjeto;

const COLUNAS: { id: ColunaProjeto; titulo: string }[] = [
  { id: "planejado", titulo: "Planejados" },
  { id: "em_andamento", titulo: "Em andamento" },
  { id: "suspenso", titulo: "Suspensos" },
  { id: "concluido", titulo: "Concluídos" },
  { id: "cancelado", titulo: "Cancelados" },
];

const TRANSOES: Record<EstadoProjeto, EstadoProjeto[]> = {
  planejado: ["em_andamento", "cancelado"],
  em_andamento: ["suspenso", "concluido", "cancelado"],
  suspenso: ["em_andamento", "cancelado"],
  concluido: [],
  cancelado: [],
};

function aceita(p: Projeto, coluna: ColunaProjeto): boolean {
  if (p.estado === coluna) return false;
  return TRANSOES[p.estado]?.includes(coluna) ?? false;
}

export function ProjetosKanban({
  itens,
  onMover,
  pending,
}: {
  itens: Projeto[];
  onMover: (p: Projeto, para: EstadoProjeto) => void;
  pending: boolean;
}) {
  const [arrastado, setArrastado] = useState<Projeto | null>(null);
  const [sobre, setSobre] = useState<ColunaProjeto | null>(null);

  function soltar(e: DragEvent, coluna: ColunaProjeto) {
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
      aria-label="Quadro de projetos"
      data-testid="projetos-kanban"
    >
      {COLUNAS.map((coluna) => {
        const cartoes = itens.filter((p) => p.estado === coluna.id);
        const alvo = arrastado !== null && aceita(arrastado, coluna.id);
        return (
          <section
            key={coluna.id}
            role="listitem"
            aria-labelledby={`coluna-proj-${coluna.id}`}
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
              id={`coluna-proj-${coluna.id}`}
              className="flex items-center justify-between px-1 text-sm font-semibold"
            >
              <span>{coluna.titulo}</span>
              <span className="text-xs font-normal text-muted">{cartoes.length}</span>
            </h2>

            <ul className="flex flex-col gap-2">
              {cartoes.map((p) => {
                const moves = TRANSOES[p.estado] ?? [];
                const prioridadeMeta = PRIORIDADE_PROJETO[p.prioridade];

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
                      <div className="flex items-center gap-1">
                        {prioridadeMeta && (
                          <Badge tone={prioridadeMeta.tone}>
                            {prioridadeMeta.label}
                          </Badge>
                        )}
                        <Badge tone={ESTADO_PROJETO[p.estado].tone}>
                          {ESTADO_PROJETO[p.estado].label}
                        </Badge>
                      </div>
                    </div>

                    <Link
                      href={`/projetos/${p.id}`}
                      className="text-sm font-medium text-primary hover:underline line-clamp-2"
                    >
                      {p.titulo}
                    </Link>

                    {p.descricao && (
                      <p className="text-xs text-muted line-clamp-2">
                        {p.descricao}
                      </p>
                    )}

                    <div className="flex items-center justify-between gap-2 text-[11px] text-muted pt-1 border-t border-surface-border/60">
                      <span>{p.gerente_nome ? p.gerente_nome : "Sem gerente"}</span>
                      {Number(p.orcamento) > 0 && (
                        <span className="font-mono">
                          R$ {Number(p.orcamento).toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
                        </span>
                      )}
                    </div>

                    {moves.length > 0 && (
                      <label className="mt-1 flex flex-col gap-1 text-xs">
                        <span className="sr-only">Mudar estado de {p.titulo}</span>
                        <select
                          aria-label={`Mudar estado de ${p.titulo}`}
                          value=""
                          disabled={pending}
                          onChange={(e) => {
                            if (e.target.value) onMover(p, e.target.value as EstadoProjeto);
                          }}
                          className="h-8 rounded-md border border-surface-border bg-surface px-2 text-xs text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                        >
                          <option value="">Mover para…</option>
                          {moves.map((m) => (
                            <option key={m} value={m}>
                              {ESTADO_PROJETO[m].label}
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
