"use client";

import { useRouter } from "next/navigation";
import type { ReactNode } from "react";

import { NomeClicavel, Quadro, colunasEmOrdem, seloDoEstado, type ColunaFaixa, type ControlesColuna } from "@/components/plataforma/Faixas";
import { Badge } from "@/components/ui/Badge";
import {
  ESTADO_PROBLEMA,
  type EstadoProblema,
  type Problema,
} from "@/lib/servicos/servicos";

export const COLUNAS_PROBLEMA: ColunaFaixa[] = colunasEmOrdem([
  { id: "registrado", titulo: "Registrados", tom: "info" },
  { id: "em_analise", titulo: "Em análise", tom: "info" },
  { id: "erro_conhecido", titulo: "Erros conhecidos", tom: "warning" },
  { id: "resolvido", titulo: "Resolvidos", tom: "success" },
  { id: "encerrado", titulo: "Encerrados", tom: "neutral" },
]);

function aceita(colunas: ColunaFaixa[], p: Problema, coluna: string): boolean {
  if (colunas.find((item) => item.id === coluna)?.ativa === false) return false;
  if (p.estado === coluna) return false;
  return (p.proximos ?? []).includes(coluna as EstadoProblema);
}

export function ProblemasKanban({
  itens,
  onMover,
  pending,
  extra,
  aoCriar,
  aoRenomear,
  aoAbrir,
  colunas = COLUNAS_PROBLEMA,
  controles,
}: {
  itens: Problema[];
  onMover: (p: Problema, para: EstadoProblema) => void;
  pending: boolean;
  extra?: (colunaId: string) => ReactNode;
  aoCriar?: (colunaId: string, nome: string) => void;
  aoRenomear?: (p: Problema, nome: string) => void;
  aoAbrir?: (p: Problema) => void;
  colunas?: ColunaFaixa[];
  controles?: ControlesColuna;
}) {
  const router = useRouter();
  function abrir(p: Problema) {
    if (aoAbrir) aoAbrir(p);
    else router.push(`/problemas/${p.id}`);
  }
  return (
    <Quadro
      rotulo="Quadro de problemas"
      memoria="nova-faixas:problemas-quadro"
      aparencia="nova-faixas:problemas"
      colunas={colunas}
      itens={itens}
      colunaDe={(p) => p.estado}
      idDe={(p) => p.id}
      nomeDe={(p) => p.titulo}
      pending={pending}
      extra={extra}
      aoCriar={aoCriar}
      aoRenomear={aoRenomear}
      aoAbrir={abrir}
      arrastavel={(p) => (p.proximos ?? []).length > 0}
      podeSoltar={(p, coluna) => aceita(colunas, p, coluna)}
      {...controles}
      aoSoltar={(p, coluna) => onMover(p, coluna as EstadoProblema)}
      render={(p, vista) => {
        const moves = p.proximos ?? [];
        return (
          <>
            <div className="flex items-center justify-between gap-2 text-xs text-muted">
              <span className="font-semibold tabular-nums text-foreground">#{p.numero}</span>
              <Badge tone={seloDoEstado(colunas, p.estado, ESTADO_PROBLEMA[p.estado as EstadoProblema]).tone}>
                {seloDoEstado(colunas, p.estado, ESTADO_PROBLEMA[p.estado as EstadoProblema]).label}
              </Badge>
            </div>
            <NomeClicavel
              nome={vista.titulo || p.titulo}
              href={`/problemas/${p.id}`}
              onSalvar={vista.salvarNome}
            />
            {p.contorno && (
              <p className="line-clamp-2 rounded bg-warning/10 p-1.5 text-xs text-muted">
                <strong className="text-foreground">Contorno:</strong> {p.contorno}
              </p>
            )}
            {p.causa_raiz && (
              <p className="line-clamp-2 rounded bg-success/10 p-1.5 text-xs text-muted">
                <strong className="text-foreground">Causa raiz:</strong> {p.causa_raiz}
              </p>
            )}
            {(p.incidentes ?? []).length > 0 && (
              <p className="text-[11px] text-muted">
                {p.incidentes.length} {p.incidentes.length === 1 ? "incidente ligado" : "incidentes ligados"}
              </p>
            )}
            <button type="button" className="text-left text-xs font-medium text-primary hover:underline" onClick={() => abrir(p)}>
              Editar
            </button>
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
                      {seloDoEstado(colunas, m, ESTADO_PROBLEMA[m as EstadoProblema]).label}
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
