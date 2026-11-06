"use client";

import { useRouter } from "next/navigation";
import type { ReactNode } from "react";

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
  ESTADO_PROJETO,
  PRIORIDADE_PROJETO,
  dataCurta,
  type EstadoProjeto,
  type Projeto,
} from "@/lib/projetos/projetos";

export const COLUNAS_PROJETO: ColunaFaixa[] = colunasEmOrdem([
  { id: "planejado", titulo: "Planejados", tom: "info" },
  { id: "em_andamento", titulo: "Em andamento", tom: "warning" },
  { id: "suspenso", titulo: "Suspensos", tom: "danger" },
  { id: "concluido", titulo: "Concluídos", tom: "success" },
  { id: "cancelado", titulo: "Cancelados", tom: "neutral" },
]);

const TRANSOES: Record<EstadoProjeto, EstadoProjeto[]> = {
  planejado: ["em_andamento", "cancelado"],
  em_andamento: ["suspenso", "concluido", "cancelado"],
  suspenso: ["em_andamento", "cancelado"],
  concluido: [],
  cancelado: [],
};

function aceita(colunas: ColunaFaixa[], p: Projeto, coluna: string): boolean {
  if (colunas.find((item) => item.id === coluna)?.ativa === false) return false;
  if (p.estado === coluna) return false;
  return TRANSOES[p.estado]?.includes(coluna as EstadoProjeto) ?? false;
}

export function ProjetosKanban({
  itens,
  onMover,
  pending,
  extra,
  aoCriar,
  aoRenomear,
  aoAbrir,
  colunas = COLUNAS_PROJETO,
  controles,
}: {
  itens: Projeto[];
  onMover: (p: Projeto, para: EstadoProjeto) => void;
  pending: boolean;
  extra?: (colunaId: string) => ReactNode;
  aoCriar?: (colunaId: string, nome: string) => void;
  aoRenomear?: (p: Projeto, nome: string) => void;
  aoAbrir?: (p: Projeto) => void;
  colunas?: ColunaFaixa[];
  controles?: ControlesColuna;
}) {
  const router = useRouter();
  function abrir(p: Projeto) {
    if (aoAbrir) aoAbrir(p);
    else router.push(`/projetos/${p.id}`);
  }
  return (
    <Quadro
      rotulo="Quadro de projetos"
      memoria="nova-faixas:projetos-quadro"
      aparencia="nova-faixas:projetos"
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
      testeId="projetos-kanban"
      arrastavel={(p) => (TRANSOES[p.estado] ?? []).length > 0}
      podeSoltar={(p, coluna) => aceita(colunas, p, coluna)}
      {...controles}
      aoSoltar={(p, coluna) => onMover(p, coluna as EstadoProjeto)}
      render={(p, vista) => {
        const moves = TRANSOES[p.estado] ?? [];
        const prioridadeMeta = PRIORIDADE_PROJETO[p.prioridade];
        return (
          <>
            <div className="flex items-center justify-between gap-2 text-xs">
              <span className="font-semibold tabular-nums text-foreground">#{p.numero}</span>
              {prioridadeMeta && <Badge tone={prioridadeMeta.tone}>{prioridadeMeta.label}</Badge>}
            </div>
            <NomeClicavel
              nome={vista.titulo || p.titulo}
              href={`/projetos/${p.id}`}
              onSalvar={vista.salvarNome}
              className="line-clamp-2 text-left text-sm font-semibold text-foreground hover:underline"
            />
            <div className="flex items-center justify-between gap-2 text-[11px] text-muted">
              <span className="truncate">{p.gerente_nome || "Sem responsável"}</span>
              <span className="shrink-0">{dataCurta(p.data_fim_planejada) || "Sem prazo"}</span>
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
                      {seloDoEstado(colunas, m, ESTADO_PROJETO[m as EstadoProjeto]).label}
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
