"use client";

import { AlertTriangle } from "lucide-react";
import Link from "next/link";
import { type DragEvent, useState } from "react";

import { PrazoBadge } from "@/components/atendimento/PrazoBadge";
import { Badge } from "@/components/ui/Badge";
import {
  relogioEmFoco,
  rotuloEstado,
  rotuloTipo,
  type Estado,
  type Solicitacao,
} from "@/lib/atendimento/atendimento";

type Coluna =
  "aberto" | "em_atendimento" | "aguardando_servidor" | "aguardando_terceiro" | "concluida";

const COLUNAS: { id: Coluna; titulo: string }[] = [
  { id: "aberto", titulo: "Abertas" },
  { id: "em_atendimento", titulo: "Em atendimento" },
  { id: "aguardando_servidor", titulo: "Aguardando o servidor" },
  { id: "aguardando_terceiro", titulo: "Aguardando terceiro" },
  { id: "concluida", titulo: "Concluídas" },
];

/** A coluna de conclusão junta os dois fins: resolvido (incidente) e
 * atendido (requisição). */
function colunaDe(estado: Estado): Coluna | null {
  if (estado === "resolvido" || estado === "atendido") return "concluida";
  return COLUNAS.some((c) => c.id === estado) ? (estado as Coluna) : null;
}

function destino(s: Solicitacao, coluna: Coluna): Estado {
  if (coluna !== "concluida") return coluna;
  return s.tipo === "incidente" ? "resolvido" : "atendido";
}

/** A coluna aceita o cartão só se a máquina deixa (as passagens vêm da API). */
function aceita(s: Solicitacao, coluna: Coluna): boolean {
  return colunaDe(s.estado) !== coluna && (s.proximos ?? []).includes(destino(s, coluna));
}

/** A mesma fila da lista, por estado. Mover o cartão chama a mesma
 * passagem da tela de detalhe; se o domínio recusar, o cartão fica onde
 * estava. Arrastar é atalho de mouse: cada cartão tem o seletor "Mover
 * para", que serve ao teclado, ao toque e ao leitor de tela. */
export function Kanban({
  itens,
  onMover,
  pending,
}: {
  itens: Solicitacao[];
  onMover: (s: Solicitacao, para: Estado) => void;
  pending: boolean;
}) {
  const [arrastado, setArrastado] = useState<Solicitacao | null>(null);
  const [sobre, setSobre] = useState<Coluna | null>(null);

  function soltar(e: DragEvent, coluna: Coluna) {
    e.preventDefault();
    setSobre(null);
    if (arrastado && aceita(arrastado, coluna)) onMover(arrastado, destino(arrastado, coluna));
    setArrastado(null);
  }

  return (
    <div
      className="grid gap-3 overflow-x-auto pb-2 md:grid-cols-5"
      role="list"
      aria-label="Quadro da fila"
    >
      {COLUNAS.map((coluna) => {
        const cartoes = itens.filter((s) => colunaDe(s.estado) === coluna.id);
        const alvo = arrastado !== null && aceita(arrastado, coluna.id);
        return (
          <section
            key={coluna.id}
            role="listitem"
            aria-labelledby={`coluna-${coluna.id}`}
            data-coluna={coluna.id}
            onDragOver={(e) => {
              if (!alvo) return;
              e.preventDefault();
              setSobre(coluna.id);
            }}
            onDragLeave={() => setSobre((atual) => (atual === coluna.id ? null : atual))}
            onDrop={(e) => soltar(e, coluna.id)}
            className={`flex min-h-40 min-w-56 flex-col gap-2 rounded-xl border p-2 transition-colors ${
              sobre === coluna.id
                ? "border-primary bg-primary/5"
                : alvo
                  ? "border-dashed border-primary/60 bg-surface"
                  : "border-surface-border bg-black/[0.02] dark:bg-white/[0.03]"
            }`}
          >
            <h2
              id={`coluna-${coluna.id}`}
              className="flex items-center justify-between px-1 text-sm font-semibold"
            >
              {coluna.titulo}
              <span className="text-xs font-normal text-muted">{cartoes.length}</span>
            </h2>
            <ul className="flex flex-col gap-2">
              {cartoes.map((s) => {
                const moves = (s.proximos ?? []).filter((p) => p !== "cancelado");
                const foco = s.prazo ? relogioEmFoco(s.prazo) : null;
                const isVencido = foco?.relogio.situacao === "vencido" || foco?.relogio.situacao === "descumprido";
                const isCritico = foco?.relogio.situacao === "critico" || foco?.relogio.situacao === "atencao";
                const percentualSLA = Math.min(100, Math.round(foco?.relogio.percentual ?? 0));

                return (
                  <li
                    key={s.id}
                    draggable={!pending && moves.length > 0}
                    onDragStart={(e) => {
                      e.dataTransfer.effectAllowed = "move";
                      setArrastado(s);
                    }}
                    onDragEnd={() => {
                      setArrastado(null);
                      setSobre(null);
                    }}
                    className={`flex flex-col gap-1.5 rounded-lg border bg-surface p-3 shadow-sm transition-all ${
                      isVencido
                        ? "border-danger/60 border-l-4 border-l-danger bg-danger/[0.02]"
                        : isCritico
                        ? "border-warning/60 border-l-4 border-l-warning bg-warning/[0.02]"
                        : "border-surface-border"
                    }`}
                  >
                    <div className="flex items-center justify-between gap-2 text-xs text-muted">
                      <span className="tabular-nums">#{s.numero}</span>
                      <Badge>{rotuloTipo(s.tipo)}</Badge>
                    </div>
                    <Link
                      href={`/atendimento/solicitacoes/${s.id}`}
                      className="text-sm font-medium text-primary hover:underline"
                    >
                      {s.titulo}
                    </Link>
                    <p className="text-xs text-muted">
                      {s.oferta_nome} · {s.responsavel_nome || "sem responsável"}
                    </p>
                    <p>
                      <PrazoBadge prazo={s.prazo} />
                    </p>
                    {foco && foco.relogio.estado === "correndo" && (
                      <div className="mt-0.5 flex flex-col gap-1" title={`Consumo de SLA: ${foco.relogio.percentual}%`}>
                        <div className="flex items-center justify-between text-[11px] text-muted">
                          <span className="flex items-center gap-1 font-medium">
                            {isVencido && <AlertTriangle className="h-3 w-3 text-danger animate-pulse" />}
                            {isCritico && !isVencido && <AlertTriangle className="h-3 w-3 text-warning" />}
                            {foco.nome}
                          </span>
                          <span
                            className={`font-mono text-xs ${
                              isVencido ? "font-bold text-danger" : isCritico ? "font-semibold text-warning" : "text-muted"
                            }`}
                          >
                            {foco.relogio.percentual}%
                          </span>
                        </div>
                        <div className="h-1.5 w-full rounded-full bg-surface-border overflow-hidden">
                          <div
                            className={`h-full transition-all ${
                              isVencido ? "bg-danger" : isCritico ? "bg-warning" : "bg-success"
                            }`}
                            style={{ width: `${percentualSLA}%` }}
                          />
                        </div>
                      </div>
                    )}
                    {moves.length > 0 && (
                      <label className="flex flex-col gap-1 text-xs">
                        <span className="sr-only">Mover a solicitação #{s.numero} para</span>
                        <select
                          value=""
                          disabled={pending}
                          onChange={(e) => {
                            if (e.target.value) onMover(s, e.target.value as Estado);
                          }}
                          className="h-8 rounded-md border border-surface-border bg-surface px-2 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                        >
                          <option value="">Mover para…</option>
                          {moves.map((p) => (
                            <option key={p} value={p}>
                              {rotuloEstado(p)}
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
