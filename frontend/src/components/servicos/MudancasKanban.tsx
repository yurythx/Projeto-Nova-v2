"use client";

import { useRouter } from "next/navigation";
import type { ReactNode } from "react";

import { fmtDateTime } from "@/components/plataforma/useAction";
import {
  NomeClicavel,
  Quadro,
  colunasEmOrdem,
  seloDoEstado,
  type ColunaFaixa,
  type ControlesColuna,
} from "@/components/plataforma/Faixas";
import { Badge } from "@/components/ui/Badge";
import {
  ESTADO_MUDANCA,
  RISCO,
  TIPO_MUDANCA,
  type EstadoMudanca,
  type Mudanca,
} from "@/lib/servicos/servicos";

type ColunaMudanca =
  "rascunho" | "em_avaliacao" | "aprovada" | "em_execucao" | "concluida" | "fechada";

export const COLUNAS_MUDANCA: ColunaFaixa[] = colunasEmOrdem([
  { id: "rascunho", titulo: "Rascunho", tom: "neutral" },
  { id: "em_avaliacao", titulo: "Em avaliação", tom: "info" },
  { id: "aprovada", titulo: "Aprovadas", tom: "success" },
  { id: "em_execucao", titulo: "Em execução", tom: "info" },
  { id: "concluida", titulo: "Concluídas", tom: "success" },
  { id: "fechada", titulo: "Rejeitada / Falhou", tom: "danger" },
]);

export function faixaDaMudanca(estado: EstadoMudanca): string {
  if (estado === "rejeitada" || estado === "falhou" || estado === "cancelada") return "fechada";
  return estado;
}

function destino(m: Mudanca, coluna: ColunaMudanca): EstadoMudanca | null {
  if (coluna !== "fechada") return coluna as EstadoMudanca;
  const valid = (m.proximos ?? []).find(
    (p) => p === "cancelada" || p === "rejeitada" || p === "falhou",
  );
  return valid ?? null;
}

function aceita(colunas: ColunaFaixa[], m: Mudanca, coluna: string): boolean {
  if (colunas.find((item) => item.id === coluna)?.ativa === false) return false;
  const id = coluna as ColunaMudanca;
  if (faixaDaMudanca(m.estado) === id) return false;
  const d = destino(m, id);
  return d !== null && (m.proximos ?? []).includes(d);
}

export function MudancasKanban({
  itens,
  onMover,
  pending,
  extra,
  aoCriar,
  aoRenomear,
  aoAbrir,
  colunas = COLUNAS_MUDANCA,
  controles,
}: {
  itens: Mudanca[];
  onMover: (m: Mudanca, para: EstadoMudanca) => void;
  pending: boolean;
  extra?: (colunaId: string) => ReactNode;
  aoCriar?: (colunaId: string, nome: string) => void;
  aoRenomear?: (m: Mudanca, nome: string) => void;
  aoAbrir?: (m: Mudanca) => void;
  colunas?: ColunaFaixa[];
  controles?: ControlesColuna;
}) {
  const router = useRouter();
  function abrir(m: Mudanca) {
    if (aoAbrir) aoAbrir(m);
    else router.push(`/mudancas/${m.id}`);
  }
  return (
    <Quadro
      rotulo="Quadro de mudanças"
      memoria="nova-faixas:mudancas-quadro"
      aparencia="nova-faixas:mudancas"
      colunas={colunas}
      itens={itens}
      colunaDe={(m) => faixaDaMudanca(m.estado)}
      idDe={(m) => m.id}
      nomeDe={(m) => m.titulo}
      pending={pending}
      extra={extra}
      aoCriar={aoCriar}
      aoRenomear={aoRenomear}
      aoAbrir={abrir}
      arrastavel={(m) => (m.proximos ?? []).length > 0}
      podeSoltar={(m, coluna) => aceita(colunas, m, coluna)}
      {...controles}
      aoSoltar={(m, coluna) => {
        const d = destino(m, coluna as ColunaMudanca);
        if (d) onMover(m, d);
      }}
      render={(m, vista) => {
        const moves = m.proximos ?? [];
        return (
          <>
            <div className="flex items-center justify-between gap-1.5 text-xs">
              <span className="font-semibold tabular-nums text-foreground">#{m.numero}</span>
              <div className="flex items-center gap-1">
                <Badge tone={RISCO[m.risco].tone}>{RISCO[m.risco].label}</Badge>
                <Badge
                  tone={
                    seloDoEstado(colunas, m.estado, ESTADO_MUDANCA[m.estado as EstadoMudanca]).tone
                  }
                >
                  {seloDoEstado(colunas, m.estado, ESTADO_MUDANCA[m.estado as EstadoMudanca]).label}
                </Badge>
              </div>
            </div>
            <NomeClicavel
              nome={vista.titulo || m.titulo}
              href={`/mudancas/${m.id}`}
              onSalvar={vista.salvarNome}
            />
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
            <button
              type="button"
              className="text-left text-xs font-medium text-primary hover:underline"
              onClick={() => abrir(m)}
            >
              Editar
            </button>
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
                      {seloDoEstado(colunas, s, ESTADO_MUDANCA[s as EstadoMudanca]).acao}
                    </option>
                  ))}
                </select>
              </label>
            )}
          </>
        );
      }}
    />
  );
}
