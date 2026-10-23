"use client";

import {
  AlertTriangle,
  Calendar,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Clock,
  Flag,
  Layers,
  Milestone,
  Plus,
  User,
} from "lucide-react";
import { useMemo, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import {
  ESTADO_TAREFA,
  type EstadoTarefa,
  type Projeto,
  type Tarefa,
  type VinculoProjeto,
} from "@/lib/projetos/projetos";

interface ProjetoGanttChartProps {
  projeto: Projeto;
  tarefas: Tarefa[];
  vinculos?: VinculoProjeto[];
  onAdicionarTarefa?: () => void;
  podeEditar?: boolean;
}

export function ProjetoGanttChart({
  projeto,
  tarefas,
  vinculos = [],
  onAdicionarTarefa,
  podeEditar,
}: ProjetoGanttChartProps) {
  // Filtro por estado da tarefa
  const [filtroEstado, setFiltroEstado] = useState<string>("todos");
  // Janela em dias (30, 60 ou 90 dias)
  const [janelaDias, setJanelaDias] = useState<number>(60);

  // Data base do cronograma
  const [dataBase, setDataBase] = useState(() => {
    if (projeto.data_inicio_planejada) {
      const d = new Date(projeto.data_inicio_planejada);
      d.setDate(d.getDate() - 2); // margem de 2 dias antes
      return d;
    }
    const d = new Date();
    d.setDate(d.getDate() - 7); // 7 dias antes de hoje
    return d;
  });

  const dataFim = useMemo(() => {
    const d = new Date(dataBase);
    d.setDate(d.getDate() + janelaDias);
    return d;
  }, [dataBase, janelaDias]);

  // Controles de navegação temporal
  const navegarSemanas = (semanas: number) => {
    const d = new Date(dataBase);
    d.setDate(d.getDate() + semanas * 7);
    setDataBase(d);
  };

  const irParaHoje = () => {
    const d = new Date();
    d.setDate(d.getDate() - 7);
    setDataBase(d);
  };

  // Filtragem de tarefas
  const tarefasFiltradas = useMemo(() => {
    if (filtroEstado === "todos") return tarefas;
    return tarefas.filter((t) => t.estado === filtroEstado);
  }, [tarefas, filtroEstado]);

  // Estatísticas do projeto
  const totalTarefas = tarefas.length;
  const concluidas = tarefas.filter((t) => t.estado === "concluida").length;
  const emAndamento = tarefas.filter((t) => t.estado === "em_andamento").length;
  const bloqueadas = tarefas.filter((t) => t.estado === "bloqueada").length;
  const progressoGeral =
    totalTarefas > 0 ? Math.round((concluidas / totalTarefas) * 100) : 0;

  // Dias e semanas do cabeçalho
  const colunasCabecalho = useMemo(() => {
    const colunas: { label: string; sub: string; offsetPct: number; widthPct: number; isFimDeSemana?: boolean }[] = [];
    const inicioMs = dataBase.getTime();
    const fimMs = dataFim.getTime();
    const totalMs = fimMs - inicioMs;

    // Gerar agrupamento por semanas (7 dias)
    let cur = new Date(dataBase);
    let semanaIdx = 1;

    while (cur < dataFim) {
      const semStart = new Date(cur);
      const semEnd = new Date(cur);
      semEnd.setDate(semEnd.getDate() + 7);

      const startMs = Math.max(semStart.getTime(), inicioMs);
      const endMs = Math.min(semEnd.getTime(), fimMs);

      if (endMs > startMs) {
        const offsetPct = ((startMs - inicioMs) / totalMs) * 100;
        const widthPct = ((endMs - startMs) / totalMs) * 100;
        const label = `Semana ${semanaIdx}`;
        const sub = `${semStart.toLocaleDateString("pt-BR", { day: "2-digit", month: "short" })}`;
        colunas.push({ label, sub, offsetPct, widthPct });
      }

      cur.setDate(cur.getDate() + 7);
      semanaIdx++;
    }
    return colunas;
  }, [dataBase, dataFim]);

  // Posição percentual da linha "Hoje"
  const hojePct = useMemo(() => {
    const agora = new Date().getTime();
    const inicioMs = dataBase.getTime();
    const fimMs = dataFim.getTime();
    if (agora < inicioMs || agora > fimMs) return null;
    return ((agora - inicioMs) / (fimMs - inicioMs)) * 100;
  }, [dataBase, dataFim]);

  // Calcula coordenadas da barra de cada tarefa no Gantt
  function calcularBarra(t: Tarefa, idx: number) {
    const inicioMs = dataBase.getTime();
    const fimMs = dataFim.getTime();
    const totalMs = fimMs - inicioMs;

    // Data de início da tarefa:
    // Baseada em projeto.data_inicio_planejada + offset proporcional por ordem ou created_at
    const baseProj = projeto.data_inicio_planejada
      ? new Date(projeto.data_inicio_planejada)
      : new Date(t.created_at || Date.now());

    const tInicio = new Date(baseProj);
    // Cada ordem acrescenta um espaçamento natural no cronograma
    tInicio.setDate(tInicio.getDate() + Math.max(0, (t.ordem - 1) * 3));

    // Data de fim da tarefa:
    const tFim = new Date(tInicio);
    if (t.data_limite) {
      const limite = new Date(t.data_limite);
      if (limite > tInicio) {
        tFim.setTime(limite.getTime());
      } else {
        tFim.setDate(tInicio.getDate() + 5);
      }
    } else {
      // Duração estimada: concluídas 5 dias, em andamento 7 dias, a fazer 5 dias
      const diasEstimados = t.estado === "concluida" ? 5 : t.estado === "em_andamento" ? 8 : 6;
      tFim.setDate(tInicio.getDate() + diasEstimados);
    }

    const tStartMs = Math.max(tInicio.getTime(), inicioMs);
    const tEndMs = Math.min(tFim.getTime(), fimMs);

    const left = Math.max(0, Math.min(100, ((tInicio.getTime() - inicioMs) / totalMs) * 100));
    const right = Math.max(0, Math.min(100, ((tFim.getTime() - inicioMs) / totalMs) * 100));
    const width = Math.max(2.5, right - left);

    // Tarefa atrasada: não concluída e prazo final no passado em relação a Hoje
    const atrasada = t.estado !== "concluida" && tFim.getTime() < Date.now();

    return {
      tInicio,
      tFim,
      left,
      width,
      atrasada,
    };
  }

  // Estilo visual da barra do Gantt por estado
  const estiloBarra: Record<EstadoTarefa, { bg: string; border: string; text: string; fillPct: number }> = {
    concluida: {
      bg: "bg-success/20",
      border: "border-success",
      text: "text-success font-semibold",
      fillPct: 100,
    },
    em_andamento: {
      bg: "bg-primary/20",
      border: "border-primary",
      text: "text-primary font-semibold",
      fillPct: 60,
    },
    bloqueada: {
      bg: "bg-danger/20",
      border: "border-danger",
      text: "text-danger font-semibold",
      fillPct: 30,
    },
    a_fazer: {
      bg: "bg-surface-border/40",
      border: "border-surface-border",
      text: "text-muted font-normal",
      fillPct: 0,
    },
  };

  return (
    <div
      className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-4 shadow-sm"
      data-testid="projeto-gantt-chart"
    >
      {/* Cabeçalho do Gantt e Resumo Operacional */}
      <div className="flex flex-col gap-3 pb-3 border-b border-surface-border sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2.5">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Milestone className="h-5 w-5" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-bold text-foreground">
                Cronograma & Diagrama de Gantt
              </h3>
              <Badge tone="info">{totalTarefas} atividades</Badge>
            </div>
            <p className="text-xs text-muted">
              Visualização temporal, marcos de entrega e interdependências.
            </p>
          </div>
        </div>

        {/* Progresso Geral */}
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="text-right">
              <span className="text-[11px] text-muted block">Progresso Geral</span>
              <span className="text-xs font-bold text-foreground">{progressoGeral}% entregue</span>
            </div>
            <div className="h-2 w-24 rounded-full bg-surface-border overflow-hidden">
              <div
                className="h-full bg-success transition-all duration-300"
                style={{ width: `${progressoGeral}%` }}
              />
            </div>
          </div>

          {podeEditar && onAdicionarTarefa && (
            <Button
              size="sm"
              variant="secondary"
              onClick={onAdicionarTarefa}
              className="inline-flex items-center gap-1.5"
            >
              <Plus className="h-3.5 w-3.5" />
              Nova tarefa
            </Button>
          )}
        </div>
      </div>

      {/* Controles de Navegação e Filtros */}
      <div className="flex flex-wrap items-center justify-between gap-3 text-xs">
        <div className="flex items-center gap-1.5">
          <span className="text-muted font-medium mr-1">Filtrar:</span>
          {(["todos", "em_andamento", "a_fazer", "concluida", "bloqueada"] as const).map((est) => (
            <button
              key={est}
              type="button"
              onClick={() => setFiltroEstado(est)}
              className={`rounded-md px-2 py-1 text-xs font-medium transition-colors ${
                filtroEstado === est
                  ? "bg-primary text-primary-foreground"
                  : "bg-surface-border/40 text-muted hover:text-foreground"
              }`}
            >
              {est === "todos"
                ? `Todas (${totalTarefas})`
                : `${ESTADO_TAREFA[est as EstadoTarefa]?.label} (${
                    tarefas.filter((t) => t.estado === est).length
                  })`}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-2">
          {/* Zoom temporal */}
          <div className="flex items-center rounded-lg border border-surface-border p-0.5 bg-black/[0.02] dark:bg-white/[0.02]">
            {[30, 60, 90].map((dias) => (
              <button
                key={dias}
                type="button"
                onClick={() => setJanelaDias(dias)}
                className={`rounded px-2 py-0.5 text-[11px] font-medium transition-all ${
                  janelaDias === dias
                    ? "bg-surface shadow-xs text-foreground font-semibold"
                    : "text-muted hover:text-foreground"
                }`}
              >
                {dias}d
              </button>
            ))}
          </div>

          {/* Navegação de semanas */}
          <div className="flex items-center gap-1">
            <Button
              size="sm"
              variant="secondary"
              onClick={() => navegarSemanas(-2)}
              aria-label="Duas semanas antes"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
            </Button>
            <Button size="sm" variant="secondary" onClick={irParaHoje} className="text-xs px-2.5">
              Hoje
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => navegarSemanas(2)}
              aria-label="Duas semanas depois"
            >
              <ChevronRight className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </div>

      {/* Grade Visual do Gráfico de Gantt */}
      <div className="overflow-x-auto rounded-lg border border-surface-border bg-surface">
        <div className="min-w-[840px]">
          {/* Cabeçalho de Escala Temporal */}
          <div className="grid grid-cols-12 border-b border-surface-border bg-black/[0.02] text-xs font-semibold text-muted dark:bg-white/[0.02]">
            <div className="col-span-4 p-3 border-r border-surface-border flex items-center justify-between">
              <span>Atividade & Responsável</span>
              <span className="text-[11px] font-normal text-muted">Estado</span>
            </div>
            <div className="col-span-8 relative h-12 flex">
              {colunasCabecalho.map((col, i) => (
                <div
                  key={i}
                  className="absolute top-0 bottom-0 border-l border-surface-border px-2 flex flex-col justify-center"
                  style={{
                    left: `${col.offsetPct}%`,
                    width: `${col.widthPct}%`,
                  }}
                >
                  <span className="text-[11px] font-medium text-foreground">{col.label}</span>
                  <span className="text-[10px] text-muted">{col.sub}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Linhas de Atividades */}
          {tarefasFiltradas.length === 0 ? (
            <div className="p-8 text-center text-xs text-muted">
              Nenhuma atividade encontrada para os critérios selecionados.
            </div>
          ) : (
            <div className="relative divide-y divide-surface-border/60">
              {/* Marcador vertical 'Hoje' */}
              {hojePct !== null && (
                <div
                  className="absolute top-0 bottom-0 z-20 w-0.5 bg-danger pointer-events-none"
                  style={{ left: `calc(33.333% + (66.666% * ${hojePct / 100}))` }}
                  title="Data Atual (Hoje)"
                >
                  <span className="absolute -top-2.5 -translate-x-1/2 rounded bg-danger px-1.5 py-0.2 text-[9px] font-bold text-white uppercase shadow-xs">
                    Hoje
                  </span>
                </div>
              )}

              {tarefasFiltradas.map((t, idx) => {
                const barra = calcularBarra(t, idx);
                const estilo = estiloBarra[t.estado] || estiloBarra.a_fazer;
                const meta = ESTADO_TAREFA[t.estado];

                return (
                  <div
                    key={t.id}
                    className="grid grid-cols-12 items-center hover:bg-black/[0.015] dark:hover:bg-white/[0.015] transition-colors group"
                    data-testid={`gantt-row-${t.id}`}
                  >
                    {/* Coluna Esquerda: Dados da Tarefa */}
                    <div className="col-span-4 p-2.5 border-r border-surface-border flex items-center justify-between gap-2 overflow-hidden">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5">
                          <span className="font-mono text-[11px] text-muted font-bold shrink-0">
                            #{t.ordem || idx + 1}
                          </span>
                          <span
                            className="text-xs font-medium text-foreground truncate block"
                            title={t.titulo}
                          >
                            {t.titulo}
                          </span>
                        </div>
                        {t.responsavel_nome && (
                          <div className="flex items-center gap-1 text-[11px] text-muted truncate mt-0.5">
                            <User className="h-3 w-3 shrink-0" />
                            <span>{t.responsavel_nome}</span>
                          </div>
                        )}
                      </div>
                      <div className="shrink-0 flex items-center gap-1">
                        {barra.atrasada && (
                          <span
                            className="flex items-center text-[10px] text-danger font-semibold bg-danger/10 px-1 py-0.5 rounded"
                            title="Atividade fora do prazo previsto"
                          >
                            <AlertTriangle className="h-3 w-3 mr-0.5" />
                            Atrasada
                          </span>
                        )}
                        <Badge tone={meta?.tone ?? "neutral"} className="text-[10px] py-0 px-1.5">
                          {meta?.label ?? t.estado}
                        </Badge>
                      </div>
                    </div>

                    {/* Coluna Direita: Linha do Tempo e Barra do Gantt */}
                    <div className="col-span-8 relative h-14 flex items-center px-1">
                      {/* Grid de fundo sutil para cada semana */}
                      <div className="absolute inset-0 flex pointer-events-none">
                        {colunasCabecalho.map((col, i) => (
                          <div
                            key={i}
                            className="absolute top-0 bottom-0 border-l border-surface-border/40"
                            style={{ left: `${col.offsetPct}%`, width: `${col.widthPct}%` }}
                          />
                        ))}
                      </div>

                      {/* Barra de Atividade */}
                      <div
                        className={`absolute h-7 rounded-md border shadow-xs flex items-center px-2 transition-all group-hover:ring-1 group-hover:ring-primary/40 ${estilo.bg} ${estilo.border}`}
                        style={{
                          left: `${barra.left}%`,
                          width: `${barra.width}%`,
                          minWidth: "32px",
                        }}
                        title={`${t.titulo} (${meta?.label}): ${barra.tInicio.toLocaleDateString("pt-BR")} até ${barra.tFim.toLocaleDateString("pt-BR")}`}
                      >
                        {/* Preenchimento de progresso interno */}
                        <div
                          className="absolute inset-y-0 left-0 rounded-l-md opacity-25 bg-current"
                          style={{ width: `${estilo.fillPct}%` }}
                        />

                        {/* Rótulo da Barra */}
                        <div className="relative z-10 flex items-center gap-1.5 min-w-0 overflow-hidden">
                          {t.estado === "concluida" && (
                            <CheckCircle2 className="h-3.5 w-3.5 text-success shrink-0" />
                          )}
                          {t.estado === "em_andamento" && (
                            <Clock className="h-3.5 w-3.5 text-primary shrink-0 animate-pulse" />
                          )}
                          <span className={`text-[11px] truncate ${estilo.text}`}>
                            {t.titulo}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {/* Marcos e Conexões com Mudanças (CAB) Vinculadas */}
          {vinculos.length > 0 && (
            <div className="border-t border-surface-border bg-black/[0.01] p-3 text-xs text-muted dark:bg-white/[0.01]">
              <div className="flex items-center gap-2 font-semibold text-foreground mb-1">
                <Flag className="h-3.5 w-3.5 text-info" />
                <span>Marcos Vinculados da Assistência</span>
              </div>
              <div className="flex flex-wrap items-center gap-2 mt-1">
                {vinculos.map((v) => (
                  <span
                    key={v.id}
                    className="inline-flex items-center gap-1 text-[11px] rounded bg-surface border border-surface-border px-2 py-0.5 text-foreground"
                  >
                    <span className="h-1.5 w-1.5 rounded-full bg-info" />
                    Marco: {v.rotulo || v.alvo_id}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
