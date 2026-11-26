"use client";

import {
  AlarmClock,
  CircleCheck,
  FileStack,
  Hourglass,
  MessageSquareWarning,
  type LucideIcon,
} from "lucide-react";

import { rotuloPrioridade, rotuloStatus } from "./painel";
import { contarPor, type PedidoResumo } from "./quadro";

const ABERTOS = new Set([
  "novo",
  "em_andamento",
  "pendente",
  "em_analise",
  "em_complementacao",
  "pending_approval",
]);

type Tom = "neutral" | "success" | "warning" | "danger";

const TOM_TILE: Record<Tom, { borda: string; icone: string }> = {
  neutral: { borda: "border-surface-border", icone: "bg-black/5 text-foreground dark:bg-white/10" },
  success: { borda: "border-success/30", icone: "bg-success/10 text-success" },
  warning: { borda: "border-warning/40", icone: "bg-warning/10 text-warning" },
  danger: { borda: "border-danger/40", icone: "bg-danger/10 text-danger" },
};

export interface FiltroIndicador {
  prazo: string;
  status: string;
}

/**
 * Faixa de números da mesa. Cada cartão filtra o quadro e a lista; clicar de
 * novo no cartão ativo limpa o filtro.
 */
export function FaixaIndicadores({
  pedidos,
  filtro,
  onFiltrar,
}: {
  pedidos: readonly PedidoResumo[];
  filtro: FiltroIndicador;
  onFiltrar: (proximo: FiltroIndicador) => void;
}) {
  const abertos = pedidos.filter((p) => ABERTOS.has(p.status)).length;
  const prazo = (situacao: string) =>
    pedidos.filter((p) => (p.situacao_prazo || "no_prazo") === situacao).length;
  const complementacao = pedidos.filter((p) => p.status === "em_complementacao").length;

  const cartoes: {
    chave: string;
    titulo: string;
    valor: number;
    detalhe: string;
    icone: LucideIcon;
    tom: Tom;
    ativo: boolean;
    alvo: FiltroIndicador;
  }[] = [
    {
      chave: "abertos",
      titulo: "Em aberto",
      valor: abertos,
      detalhe: `de ${pedidos.length} pedidos`,
      icone: FileStack,
      tom: "neutral",
      ativo: filtro.prazo === "todos" && filtro.status === "",
      alvo: { prazo: "todos", status: "" },
    },
    {
      chave: "no_prazo",
      titulo: "No prazo",
      valor: prazo("no_prazo"),
      detalhe: "dentro do prazo",
      icone: CircleCheck,
      tom: "success",
      ativo: filtro.prazo === "no_prazo",
      alvo: { prazo: "no_prazo", status: "" },
    },
    {
      chave: "atencao",
      titulo: "Atenção",
      valor: prazo("atencao"),
      detalhe: "passaram de 3/4 do prazo",
      icone: Hourglass,
      tom: "warning",
      ativo: filtro.prazo === "atencao",
      alvo: { prazo: "atencao", status: "" },
    },
    {
      chave: "atrasado",
      titulo: "Atrasados",
      valor: prazo("atrasado"),
      detalhe: "prazo vencido",
      icone: AlarmClock,
      tom: "danger",
      ativo: filtro.prazo === "atrasado",
      alvo: { prazo: "atrasado", status: "" },
    },
    {
      chave: "complementacao",
      titulo: "Aguardando cidadão",
      valor: complementacao,
      detalhe: "em complementação",
      icone: MessageSquareWarning,
      tom: "warning",
      ativo: filtro.status === "em_complementacao",
      alvo: { prazo: "todos", status: "em_complementacao" },
    },
  ];

  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-5">
      {cartoes.map((c) => {
        const tom = TOM_TILE[c.tom];
        const Icone = c.icone;
        return (
          <button
            key={c.chave}
            type="button"
            aria-pressed={c.ativo}
            onClick={() =>
              onFiltrar(c.ativo && c.chave !== "abertos" ? { prazo: "todos", status: "" } : c.alvo)
            }
            className={`flex items-start gap-3 rounded-xl border bg-surface p-4 text-left shadow-sm transition hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-primary ${
              c.ativo ? "border-primary ring-1 ring-primary" : tom.borda
            }`}
          >
            <span
              className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${tom.icone}`}
            >
              <Icone className="h-4 w-4" aria-hidden />
            </span>
            <span className="min-w-0">
              <span className="block text-xs font-medium text-muted">{c.titulo}</span>
              <span className="block text-2xl font-semibold tabular-nums text-foreground">
                {c.valor}
              </span>
              <span className="block truncate text-[11px] text-muted">{c.detalhe}</span>
            </span>
          </button>
        );
      })}
    </div>
  );
}

/** Barras horizontais de uma série: o título nomeia a medida, sem legenda. */
function Barras({
  titulo,
  linhas,
  total,
}: {
  titulo: string;
  linhas: { nome: string; total: number }[];
  total: number;
}) {
  const maior = Math.max(1, ...linhas.map((l) => l.total));
  return (
    <section className="rounded-xl border border-surface-border bg-surface p-5 shadow-sm">
      <h3 className="mb-4 text-sm font-semibold text-foreground">{titulo}</h3>
      {linhas.length === 0 ? (
        <p className="text-sm text-muted">Sem pedidos.</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {linhas.map((linha) => {
            const pct = total > 0 ? Math.round((linha.total / total) * 100) : 0;
            return (
              <li
                key={linha.nome}
                className="group"
                title={`${linha.nome}: ${linha.total} (${pct}%)`}
              >
                <div className="mb-1 flex items-baseline justify-between gap-3 text-xs">
                  <span className="truncate text-foreground">{linha.nome}</span>
                  <span className="shrink-0 tabular-nums text-muted">
                    <span className="font-semibold text-foreground">{linha.total}</span> · {pct}%
                  </span>
                </div>
                <div className="h-2 rounded bg-black/5 dark:bg-white/10" aria-hidden>
                  <div
                    className="h-2 rounded bg-primary transition-[filter] group-hover:brightness-110"
                    style={{ width: `${Math.max(2, Math.round((linha.total / maior) * 100))}%` }}
                  />
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

/** Aba de indicadores: como os pedidos se distribuem. */
export function IndicadoresDaMesa({ pedidos }: { pedidos: readonly PedidoResumo[] }) {
  const total = pedidos.length;
  const porMes = new Map<string, { nome: string; total: number }>();
  for (const p of pedidos) {
    const data = new Date(p.created_at);
    if (Number.isNaN(data.getTime())) continue;
    const chave = data.toISOString().slice(0, 7);
    const atual = porMes.get(chave) ?? {
      nome: data.toLocaleDateString("pt-BR", {
        month: "long",
        year: "numeric",
        timeZone: "America/Cuiaba",
      }),
      total: 0,
    };
    atual.total += 1;
    porMes.set(chave, atual);
  }
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Barras
        titulo="Pedidos por status"
        total={total}
        linhas={contarPor(pedidos, (p) => rotuloStatus(p.status))}
      />
      <Barras
        titulo="Pedidos por serviço"
        total={total}
        linhas={contarPor(pedidos, (p) => p.servico_nome)}
      />
      <Barras
        titulo="Pedidos por prioridade"
        total={total}
        linhas={contarPor(pedidos, (p) => rotuloPrioridade(p.prioridade))}
      />
      <Barras
        titulo="Pedidos abertos por mês"
        total={total}
        linhas={[...porMes.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([, v]) => v)}
      />
    </div>
  );
}
