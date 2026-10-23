"use client";

import Link from "next/link";
import { type DragEvent, useState } from "react";
import { Calendar, CheckCircle2, Clock, GitCommit, User } from "lucide-react";

import { fmtDateTime } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import {
  ESTADO_MUDANCA,
  RISCO,
  TIPO_MUDANCA,
  type EstadoMudanca,
  type Mudanca,
} from "@/lib/servicos/servicos";

type ColunaMudanca =
  | "rascunho"
  | "em_avaliacao"
  | "aprovada"
  | "em_execucao"
  | "concluida"
  | "fechada";

const COLUNAS: { id: ColunaMudanca; titulo: string }[] = [
  { id: "rascunho", titulo: "Rascunho" },
  { id: "em_avaliacao", titulo: "Em avaliação" },
  { id: "aprovada", titulo: "Aprovadas" },
  { id: "em_execucao", titulo: "Em execução" },
  { id: "concluida", titulo: "Concluídas" },
  { id: "fechada", titulo: "Rejeitada / Falhou" },
];

function colunaDe(estado: EstadoMudanca): ColunaMudanca {
  if (estado === "rejeitada" || estado === "falhou" || estado === "cancelada") return "fechada";
  return estado as ColunaMudanca;
}

function destino(m: Mudanca, coluna: ColunaMudanca): EstadoMudanca | null {
  if (coluna !== "fechada") return coluna as EstadoMudanca;
  const valid = (m.proximos ?? []).find((p) => p === "cancelada" || p === "rejeitada" || p === "falhou");
  return valid ?? null;
}

function aceita(m: Mudanca, coluna: ColunaMudanca): boolean {
  if (colunaDe(m.estado) === coluna) return false;
  const d = destino(m, coluna);
  return d !== null && (m.proximos ?? []).includes(d);
}

export function MudancasKanban({
  itens,
  onMover,
  pending,
}: {
  itens: Mudanca[];
  onMover: (m: Mudanca, para: EstadoMudanca) => void;
  pending: boolean;
}) {
  const [arrastado, setArrastado] = useState<Mudanca | null>(null);
  const [sobre, setSobre] = useState<ColunaMudanca | null>(null);

  function soltar(e: DragEvent, coluna: ColunaMudanca) {
    e.preventDefault();
    setSobre(null);
    if (arrastado && aceita(arrastado, coluna)) {
      const d = destino(arrastado, coluna);
      if (d) onMover(arrastado, d);
    }
    setArrastado(null);
  }

  return (
    <div
      className="grid gap-3 overflow-x-auto pb-2 md:grid-cols-6"
      role="list"
      aria-label="Quadro de mudanças"
    >
      {COLUNAS.map((coluna) => {
        const cartoes = itens.filter((m) => colunaDe(m.estado) === coluna.id);
        const alvo = arrastado !== null && aceita(arrastado, coluna.id);
        return (
          <section
            key={coluna.id}
            role="listitem"
            aria-labelledby={`coluna-mud-${coluna.id}`}
            data-coluna={coluna.id}
            onDragOver={(e) => {
              if (!alvo) return;
              e.preventDefault();
              setSobre(coluna.id);
            }}
            onDragLeave={() => setSobre((atual) => (atual === coluna.id ? null : atual))}
            onDrop={(e) => soltar(e, coluna.id)}
            className={`flex min-h-48 min-w-56 flex-col gap-2 rounded-xl border p-2 transition-colors ${
              sobre === coluna.id
                ? "border-primary bg-primary/5"
                : alvo
                  ? "border-dashed border-primary/60 bg-surface"
                  : "border-surface-border bg-black/[0.02] dark:bg-white/[0.03]"
            }`}
          >
            <h2
              id={`coluna-mud-${coluna.id}`}
              className="flex items-center justify-between px-1 text-sm font-semibold"
            >
              <span>{coluna.titulo}</span>
              <span className="text-xs font-normal text-muted">{cartoes.length}</span>
            </h2>

            <ul className="flex flex-col gap-2">
              {cartoes.map((m) => {
                const moves = m.proximos ?? [];
                return (
                  <li
                    key={m.id}
                    draggable={!pending && moves.length > 0}
                    onDragStart={(e) => {
                      e.dataTransfer.effectAllowed = "move";
                      setArrastado(m);
                    }}
                    onDragEnd={() => {
                      setArrastado(null);
                      setSobre(null);
                    }}
                    className="flex flex-col gap-2 rounded-lg border border-surface-border bg-surface p-3 shadow-sm"
                  >
                    <div className="flex items-center justify-between gap-1.5 text-xs">
                      <span className="font-semibold tabular-nums text-foreground">#{m.numero}</span>
                      <div className="flex items-center gap-1">
                        <Badge tone={RISCO[m.risco].tone}>{RISCO[m.risco].label}</Badge>
                        <Badge tone={ESTADO_MUDANCA[m.estado].tone}>
                          {ESTADO_MUDANCA[m.estado].label}
                        </Badge>
                      </div>
                    </div>

                    <Link
                      href={`/mudancas/${m.id}`}
                      className="text-sm font-medium text-primary hover:underline line-clamp-2"
                    >
                      {m.titulo}
                    </Link>

                    <div className="flex flex-col gap-1 text-[11px] text-muted">
                      <span className="truncate">
                        <strong>Tipo:</strong> {TIPO_MUDANCA[m.tipo]}
                      </span>
                      {m.solicitante_nome && (
                        <span className="truncate">
                          <strong>Pedida por:</strong> {m.solicitante_nome}
                        </span>
                      )}
                      {m.inicio_planejado && (
                        <span className="truncate">
                          <strong>Janela:</strong> {fmtDateTime(m.inicio_planejado)}
                        </span>
                      )}
                    </div>

                    {moves.length > 0 && (
                      <label className="mt-1 flex flex-col gap-1 text-xs">
                        <span className="sr-only">Mover a mudança #{m.numero} para</span>
                        <select
                          value=""
                          disabled={pending}
                          onChange={(e) => {
                            if (e.target.value) onMover(m, e.target.value as EstadoMudanca);
                          }}
                          className="h-8 rounded-md border border-surface-border bg-surface px-2 text-xs text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                        >
                          <option value="">Mover para…</option>
                          {moves.map((s) => (
                            <option key={s} value={s}>
                              {ESTADO_MUDANCA[s].acao}
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
