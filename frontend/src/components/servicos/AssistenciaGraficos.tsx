"use client";

import { useMemo } from "react";
import { AlertTriangle, CheckCircle, Clock, ShieldAlert, TrendingUp } from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import type { Mudanca } from "@/lib/servicos/servicos";
import type { Projeto } from "@/lib/projetos/projetos";

interface Props {
  incidentes: number;
  requisicoes: number;
  mudancas: Mudanca[];
  projetos: Projeto[];
}

export function AssistenciaGraficos({
  incidentes,
  requisicoes,
  mudancas,
  projetos,
}: Props) {
  const totalDemandas = incidentes + requisicoes;
  const pctIncidentes = totalDemandas > 0 ? Math.round((incidentes / totalDemandas) * 100) : 0;
  const pctRequisicoes = totalDemandas > 0 ? Math.round((requisicoes / totalDemandas) * 100) : 0;

  // Mudanças por risco
  const mudBaixo = mudancas.filter((m) => m.risco === "baixo").length;
  const mudMedio = mudancas.filter((m) => m.risco === "medio").length;
  const mudAlto = mudancas.filter((m) => m.risco === "alto").length;
  const totalMudancas = mudancas.length;

  const pctBaixo = totalMudancas > 0 ? (mudBaixo / totalMudancas) * 100 : 0;
  const pctMedio = totalMudancas > 0 ? (mudMedio / totalMudancas) * 100 : 0;
  const pctAlto = totalMudancas > 0 ? (mudAlto / totalMudancas) * 100 : 0;

  // Circunferência do Donut (raio 40 => perímetro ~ 251.3)
  const perimetro = 251.3;
  const dashBaixo = (pctBaixo / 100) * perimetro;
  const dashMedio = (pctMedio / 100) * perimetro;
  const dashAlto = (pctAlto / 100) * perimetro;

  // Projetos por estado
  const projConcluidos = projetos.filter((p) => p.estado === "concluido").length;
  const projEmAndamento = projetos.filter((p) => p.estado === "em_andamento").length;
  const projPlanejados = projetos.filter((p) => p.estado === "planejado").length;
  const totalProjetos = projetos.length;
  const taxaConclusaoProjetos =
    totalProjetos > 0 ? Math.round((projConcluidos / totalProjetos) * 100) : 0;

  return (
    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3" data-testid="assistencia-graficos">
      {/* Gráfico 1: Proporção de Incidentes vs Requisições */}
      <Card className="flex flex-col justify-between">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle as="h3" className="text-sm font-semibold">
              Demandas de Atendimento
            </CardTitle>
            <Badge tone={incidentes > requisicoes ? "warning" : "info"}>
              {totalDemandas} chamados
            </Badge>
          </div>
          <p className="text-xs text-muted">
            Distribuição entre incidentes corretivos e requisições de serviço.
          </p>
        </CardHeader>
        <CardContent className="flex flex-col gap-4 pt-2">
          {/* Barra proporcional horizontal */}
          <div className="space-y-1.5">
            <div className="flex justify-between text-xs font-medium">
              <span className="text-warning flex items-center gap-1">
                <AlertTriangle className="h-3.5 w-3.5" />
                Incidentes ({pctIncidentes}%)
              </span>
              <span className="text-primary flex items-center gap-1">
                Requisições ({pctRequisicoes}%)
              </span>
            </div>
            <div className="h-4 w-full rounded-full bg-surface-border overflow-hidden flex">
              <div
                className="h-full bg-warning transition-all"
                style={{ width: `${pctIncidentes}%` }}
                title={`Incidentes: ${incidentes}`}
              />
              <div
                className="h-full bg-primary transition-all"
                style={{ width: `${pctRequisicoes}%` }}
                title={`Requisições: ${requisicoes}`}
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2 text-center text-xs">
            <div className="rounded-lg border border-surface-border p-2 bg-black/[0.01] dark:bg-white/[0.01]">
              <span className="block text-lg font-bold text-warning">{incidentes}</span>
              <span className="text-muted">Incidentes</span>
            </div>
            <div className="rounded-lg border border-surface-border p-2 bg-black/[0.01] dark:bg-white/[0.01]">
              <span className="block text-lg font-bold text-primary">{requisicoes}</span>
              <span className="text-muted">Requisições</span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Gráfico 2: Donut de Riscos de Mudança */}
      <Card className="flex flex-col justify-between">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle as="h3" className="text-sm font-semibold">
              Matriz de Risco das Mudanças
            </CardTitle>
            <Badge tone="neutral">{totalMudancas} mudanças</Badge>
          </div>
          <p className="text-xs text-muted">
            Perfil de criticidade e risco das janelas submetidas ao CAB.
          </p>
        </CardHeader>
        <CardContent className="flex items-center justify-between gap-4 pt-2">
          {/* SVG Donut */}
          <div className="relative flex items-center justify-center shrink-0">
            <svg width="100" height="100" viewBox="0 0 100 100" className="-rotate-90">
              {/* Círculo base */}
              <circle
                cx="50"
                cy="50"
                r="40"
                fill="transparent"
                stroke="currentColor"
                strokeWidth="14"
                className="text-surface-border"
              />
              {/* Baixo */}
              <circle
                cx="50"
                cy="50"
                r="40"
                fill="transparent"
                stroke="#10b981"
                strokeWidth="14"
                strokeDasharray={`${dashBaixo} ${perimetro - dashBaixo}`}
                strokeDashoffset="0"
                className="transition-all"
              />
              {/* Médio */}
              <circle
                cx="50"
                cy="50"
                r="40"
                fill="transparent"
                stroke="#f59e0b"
                strokeWidth="14"
                strokeDasharray={`${dashMedio} ${perimetro - dashMedio}`}
                strokeDashoffset={`${-dashBaixo}`}
                className="transition-all"
              />
              {/* Alto */}
              <circle
                cx="50"
                cy="50"
                r="40"
                fill="transparent"
                stroke="#ef4444"
                strokeWidth="14"
                strokeDasharray={`${dashAlto} ${perimetro - dashAlto}`}
                strokeDashoffset={`${-(dashBaixo + dashMedio)}`}
                className="transition-all"
              />
            </svg>
            <div className="absolute text-center">
              <span className="block text-sm font-bold text-foreground">
                {totalMudancas}
              </span>
              <span className="text-[9px] uppercase tracking-wider text-muted">
                Total
              </span>
            </div>
          </div>

          {/* Legenda de Riscos */}
          <div className="flex flex-col gap-1.5 text-xs flex-1">
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-1.5 text-muted">
                <span className="h-2.5 w-2.5 rounded-full bg-emerald-500" />
                Baixo risco
              </span>
              <span className="font-semibold text-foreground">{mudBaixo}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-1.5 text-muted">
                <span className="h-2.5 w-2.5 rounded-full bg-amber-500" />
                Médio risco
              </span>
              <span className="font-semibold text-foreground">{mudMedio}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="flex items-center gap-1.5 text-muted">
                <span className="h-2.5 w-2.5 rounded-full bg-rose-500" />
                Alto risco
              </span>
              <span className="font-semibold text-foreground">{mudAlto}</span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Gráfico 3: Progresso e Conclusão de Projetos */}
      <Card className="flex flex-col justify-between">
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle as="h3" className="text-sm font-semibold">
              Progresso Estratégico
            </CardTitle>
            <Badge tone="success">{taxaConclusaoProjetos}% concluído</Badge>
          </div>
          <p className="text-xs text-muted">
            Taxa de avanço das iniciativas e projetos da organização.
          </p>
        </CardHeader>
        <CardContent className="flex flex-col gap-4 pt-2">
          <div className="space-y-1.5">
            <div className="flex justify-between text-xs font-medium text-muted">
              <span>Execução de Projetos</span>
              <span className="font-semibold text-foreground">
                {projConcluidos} de {totalProjetos} entregues
              </span>
            </div>
            <div className="h-3 w-full rounded-full bg-surface-border overflow-hidden">
              <div
                className="h-full bg-success transition-all duration-500"
                style={{ width: `${taxaConclusaoProjetos}%` }}
              />
            </div>
          </div>

          <div className="grid grid-cols-3 gap-1.5 text-center text-xs">
            <div className="rounded-lg border border-surface-border p-2 bg-black/[0.01] dark:bg-white/[0.01]">
              <span className="block font-bold text-accent">{projPlanejados}</span>
              <span className="text-[10px] text-muted">Planejados</span>
            </div>
            <div className="rounded-lg border border-surface-border p-2 bg-black/[0.01] dark:bg-white/[0.01]">
              <span className="block font-bold text-primary">{projEmAndamento}</span>
              <span className="text-[10px] text-muted">Em curso</span>
            </div>
            <div className="rounded-lg border border-surface-border p-2 bg-black/[0.01] dark:bg-white/[0.01]">
              <span className="block font-bold text-success">{projConcluidos}</span>
              <span className="text-[10px] text-muted">Entregues</span>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
