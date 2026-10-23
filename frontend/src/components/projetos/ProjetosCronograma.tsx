"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { Calendar, ChevronLeft, ChevronRight, Clock } from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import {
  ESTADO_PROJETO,
  PRIORIDADE_PROJETO,
  type EstadoProjeto,
  type Projeto,
} from "@/lib/projetos/projetos";

interface Props {
  projetos: Projeto[];
}

export function ProjetosCronograma({ projetos }: Props) {
  // Data base de visualização (início do período visível)
  const [dataBase, setDataBase] = useState(() => {
    const d = new Date();
    d.setDate(1); // primeiro dia do mês atual
    return d;
  });

  // Janela visível de 90 dias a partir da dataBase
  const diasVisiveis = 90;
  const dataFim = useMemo(() => {
    const d = new Date(dataBase);
    d.setDate(d.getDate() + diasVisiveis);
    return d;
  }, [dataBase, diasVisiveis]);

  // Navegação de mês
  const mesAnterior = () => {
    const d = new Date(dataBase);
    d.setMonth(d.getMonth() - 1);
    setDataBase(d);
  };

  const proximoMes = () => {
    const d = new Date(dataBase);
    d.setMonth(d.getMonth() + 1);
    setDataBase(d);
  };

  const hojeMes = () => {
    const d = new Date();
    d.setDate(1);
    setDataBase(d);
  };

  // Gerar marcos de meses no cabeçalho
  const mesesCabecalho = useMemo(() => {
    const meses: { nome: string; offsetPct: number; widthPct: number }[] = [];
    const inicioMs = dataBase.getTime();
    const fimMs = dataFim.getTime();
    const totalMs = fimMs - inicioMs;

    const cur = new Date(dataBase);
    while (cur < dataFim) {
      const inicioMes = new Date(cur.getFullYear(), cur.getMonth(), 1);
      const fimMes = new Date(cur.getFullYear(), cur.getMonth() + 1, 0, 23, 59, 59);

      const mesStart = Math.max(inicioMes.getTime(), inicioMs);
      const mesEnd = Math.min(fimMes.getTime(), fimMs);

      if (mesEnd > mesStart) {
        const offsetPct = ((mesStart - inicioMs) / totalMs) * 100;
        const widthPct = ((mesEnd - mesStart) / totalMs) * 100;
        const nome = inicioMes.toLocaleDateString("pt-BR", {
          month: "short",
          year: "numeric",
        });
        meses.push({ nome, offsetPct, widthPct });
      }

      cur.setMonth(cur.getMonth() + 1);
      cur.setDate(1);
    }
    return meses;
  }, [dataBase, dataFim]);

  // Linha do dia de "Hoje"
  const hojePct = useMemo(() => {
    const agora = new Date().getTime();
    const inicioMs = dataBase.getTime();
    const fimMs = dataFim.getTime();
    if (agora < inicioMs || agora > fimMs) return null;
    return ((agora - inicioMs) / (fimMs - inicioMs)) * 100;
  }, [dataBase, dataFim]);

  const coresBarra: Record<EstadoProjeto, string> = {
    planejado: "bg-accent/80 border-accent text-accent-foreground",
    em_andamento: "bg-primary/80 border-primary text-primary-foreground",
    suspenso: "bg-warning/80 border-warning text-warning-foreground",
    concluido: "bg-success/80 border-success text-success-foreground",
    cancelado: "bg-muted border-border text-muted-foreground",
  };

  return (
    <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-4 shadow-sm" data-testid="projetos-cronograma">
      {/* Controles do Cronograma */}
      <div className="flex flex-wrap items-center justify-between gap-3 pb-3 border-b border-surface-border">
        <div className="flex items-center gap-2">
          <Calendar className="h-4 w-4 text-primary" />
          <span className="text-sm font-semibold text-foreground">
            Cronograma & Linha do Tempo (90 dias)
          </span>
        </div>

        <div className="flex items-center gap-2">
          <Button size="sm" variant="secondary" onClick={mesAnterior} aria-label="Mês anterior">
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <Button size="sm" variant="secondary" onClick={hojeMes}>
            Hoje
          </Button>
          <Button size="sm" variant="secondary" onClick={proximoMes} aria-label="Próximo mês">
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Tabela Gantt */}
      <div className="overflow-x-auto">
        <div className="min-w-[760px]">
          {/* Cabeçalho de Meses */}
          <div className="grid grid-cols-12 gap-2 pb-2 text-xs text-muted border-b border-surface-border">
            <div className="col-span-4 font-semibold text-foreground">
              Projeto / Responsável
            </div>
            <div className="col-span-8 relative h-6 bg-black/[0.02] dark:bg-white/[0.02] rounded">
              {mesesCabecalho.map((m, idx) => (
                <div
                  key={idx}
                  className="absolute top-0 bottom-0 flex items-center justify-center border-l border-surface-border text-[11px] font-medium uppercase text-muted"
                  style={{
                    left: `${m.offsetPct}%`,
                    width: `${m.widthPct}%`,
                  }}
                >
                  {m.nome}
                </div>
              ))}
            </div>
          </div>

          {/* Linhas de Projetos */}
          <div className="relative divide-y divide-surface-border/60">
            {/* Linha vertical Hoje */}
            {hojePct !== null && (
              <div
                className="absolute top-0 bottom-0 z-10 w-0.5 bg-danger/80 pointer-events-none"
                style={{ left: `calc(33.333% + (66.666% * ${hojePct / 100}))` }}
                title="Hoje"
              >
                <span className="absolute -top-4 -translate-x-1/2 rounded bg-danger px-1 text-[9px] font-bold text-white uppercase">
                  Hoje
                </span>
              </div>
            )}

            {projetos.length === 0 ? (
              <div className="py-8 text-center text-xs text-muted">
                Nenhum projeto registrado no período.
              </div>
            ) : (
              projetos.map((p) => {
                const inicioMs = dataBase.getTime();
                const fimMs = dataFim.getTime();
                const totalMs = fimMs - inicioMs;

                // Definir início e fim do projeto (ou estimativa de 30 dias se sem datas)
                const dInicio = p.data_inicio_real
                  ? new Date(p.data_inicio_real)
                  : p.data_inicio_planejada
                    ? new Date(p.data_inicio_planejada)
                    : new Date(p.created_at);

                const dFim = p.data_fim_real
                  ? new Date(p.data_fim_real)
                  : p.data_fim_planejada
                    ? new Date(p.data_fim_planejada)
                    : new Date(dInicio.getTime() + 30 * 24 * 60 * 60 * 1000);

                const projStart = Math.max(dInicio.getTime(), inicioMs);
                const projEnd = Math.min(dFim.getTime(), fimMs);

                let barraLeftPct = 0;
                let barraWidthPct = 0;
                const visivel = projEnd > inicioMs && projStart < fimMs;

                if (visivel) {
                  barraLeftPct = Math.max(0, ((projStart - inicioMs) / totalMs) * 100);
                  barraWidthPct = Math.min(100 - barraLeftPct, Math.max(4, ((projEnd - projStart) / totalMs) * 100));
                }

                const prioridadeMeta = PRIORIDADE_PROJETO[p.prioridade];
                const estadoMeta = ESTADO_PROJETO[p.estado];

                return (
                  <div key={p.id} className="grid grid-cols-12 gap-2 py-3 items-center hover:bg-black/[0.01] dark:hover:bg-white/[0.01]">
                    <div className="col-span-4 pr-2">
                      <div className="flex items-center gap-1.5">
                        <span className="font-mono text-xs font-semibold text-primary">
                          #{p.numero}
                        </span>
                        <Link
                          href={`/projetos/${p.id}`}
                          className="text-xs font-medium text-foreground hover:underline truncate"
                          title={p.titulo}
                        >
                          {p.titulo}
                        </Link>
                      </div>
                      <div className="flex items-center gap-2 mt-0.5 text-[11px] text-muted">
                        <span>{p.gerente_nome || "Sem gerente"}</span>
                        {prioridadeMeta && (
                          <span>• {prioridadeMeta.label}</span>
                        )}
                      </div>
                    </div>

                    <div className="col-span-8 relative h-7 flex items-center bg-black/[0.02] dark:bg-white/[0.02] rounded overflow-hidden">
                      {visivel ? (
                        <div
                          className={`absolute h-5 rounded border px-2 flex items-center text-[10px] font-semibold truncate transition-all shadow-xs ${coresBarra[p.estado] || "bg-primary"}`}
                          style={{
                            left: `${barraLeftPct}%`,
                            width: `${barraWidthPct}%`,
                          }}
                          title={`${p.titulo} (${estadoMeta?.label}) - De ${dInicio.toLocaleDateString("pt-BR")} até ${dFim.toLocaleDateString("pt-BR")}`}
                        >
                          <span className="truncate">{p.titulo}</span>
                        </div>
                      ) : (
                        <span className="text-[10px] text-muted/60 pl-2">
                          Fora do período ({dInicio.toLocaleDateString("pt-BR")})
                        </span>
                      )}
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>
      </div>

      {/* Legenda de Estados */}
      <div className="flex flex-wrap items-center gap-4 pt-2 text-xs text-muted border-t border-surface-border">
        <span className="font-medium text-foreground">Legenda:</span>
        <div className="flex items-center gap-1.5">
          <div className="h-3 w-3 rounded-full bg-accent" />
          <span>Planejado</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="h-3 w-3 rounded-full bg-primary" />
          <span>Em andamento</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="h-3 w-3 rounded-full bg-warning" />
          <span>Suspenso</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="h-3 w-3 rounded-full bg-success" />
          <span>Concluído</span>
        </div>
      </div>
    </div>
  );
}
