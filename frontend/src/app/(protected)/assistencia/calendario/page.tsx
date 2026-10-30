"use client";

import Link from "next/link";
import {
  AlertTriangle,
  ArrowLeft,
  ChevronLeft,
  ChevronRight,
  ExternalLink,
  Filter,
} from "lucide-react";
import { useMemo, useState } from "react";

import { PageHeader } from "@/components/plataforma/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Dialog } from "@/components/ui/Dialog";
import { useApiPage, useApiQuery } from "@/lib/api/swr";
import type { Solicitacao } from "@/lib/atendimento/atendimento";
import { ESTADO_PROJETO, PRIORIDADE_PROJETO, type Projeto } from "@/lib/projetos/projetos";
import { ESTADO_MUDANCA, RISCO, TIPO_MUDANCA, type Mudanca } from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import { useEntityFilter } from "@/lib/plataforma/useEntityFilter";

export type TipoEvento = "mudanca" | "projeto" | "sla";

export interface EventoCalendario {
  id: string;
  tipo: TipoEvento;
  titulo: string;
  subtitulo: string;
  data: Date;
  dataFim?: Date;
  estado: string;
  estadoRotulo: string;
  prioridadeOuRisco?: string;
  responsavel?: string;
  href: string;
  descricao?: string;
  conflito?: boolean;
  motivoConflito?: string;
  detalhesExtras?: Record<string, string | number | undefined>;
}

const DIAS_SEMANA = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];

export default function AssistenciaCalendarioPage() {
  const { can, enabled } = usePlataforma();

  // Mês em visualização
  const [mesAtual, setMesAtual] = useState(() => {
    const d = new Date();
    return new Date(d.getFullYear(), d.getMonth(), 1);
  });

  // Filtros de tipo
  const [tiposAtivos, setTiposAtivos] = useState<Record<TipoEvento, boolean>>({
    mudanca: true,
    projeto: true,
    sla: true,
  });

  const [apenasConflitos, setApenasConflitos] = useState(false);
  const [modoVisao, setModoVisao] = useState<"grade" | "lista">("grade");
  const [eventoSelecionado, setEventoSelecionado] = useState<EventoCalendario | null>(null);
  const { filtrarPorEntidade } = useEntityFilter();

  // Consultas de dados
  const atende = can("atendimento:atender");
  const fila = useApiPage<Solicitacao>(
    enabled("atendimento") && atende ? "v1/atendimento/fila?page_size=100" : null,
  );

  const gereMudancas = can("mudancas:manage");
  const veMudancas = gereMudancas || can("mudancas:aprovar");
  const mudancas = useApiQuery<Mudanca[]>(enabled("mudancas") && veMudancas ? "v1/mudancas" : null);

  const gereProjetos = can("projetos:manage");
  const leProjetos = gereProjetos || can("projetos:read");
  const projetos = useApiQuery<Projeto[]>(enabled("projetos") && leProjetos ? "v1/projetos" : null);

  // Mapeamento e unificação dos eventos
  const todosEventos = useMemo<EventoCalendario[]>(() => {
    const eventos: EventoCalendario[] = [];

    // 1. Mudanças do CAB
    for (const m of filtrarPorEntidade(mudancas.data ?? [])) {
      const dataStr = m.inicio_planejado || m.created_at;
      if (!dataStr) continue;
      const d = new Date(dataStr);
      if (isNaN(d.getTime())) continue;

      const estadoInfo = ESTADO_MUDANCA[m.estado];
      eventos.push({
        id: `mud-${m.id}`,
        tipo: "mudanca",
        titulo: `M#${m.numero} ${m.titulo}`,
        subtitulo: `Janela CAB • ${TIPO_MUDANCA[m.tipo] ?? m.tipo}`,
        data: d,
        dataFim: m.fim_planejado ? new Date(m.fim_planejado) : undefined,
        estado: m.estado,
        estadoRotulo: estadoInfo?.label ?? m.estado,
        prioridadeOuRisco: m.risco,
        responsavel: m.solicitante_nome,
        href: "/mudancas",
        descricao: m.descricao || m.justificativa,
        detalhesExtras: {
          "Tipo de Mudança": TIPO_MUDANCA[m.tipo] ?? m.tipo,
          "Grau de Risco": RISCO[m.risco]?.label ?? m.risco,
          "Plano de Reversão": m.plano_reversao || "Não especificado",
          Solicitante: m.solicitante_nome,
        },
      });
    }

    // 2. Marcos de Projetos
    for (const p of filtrarPorEntidade(projetos.data ?? [])) {
      const dataStr = p.data_fim_planejada || p.data_inicio_planejada || p.created_at;
      if (!dataStr) continue;
      const d = new Date(dataStr);
      if (isNaN(d.getTime())) continue;

      const estadoInfo = ESTADO_PROJETO[p.estado];
      eventos.push({
        id: `proj-${p.id}`,
        tipo: "projeto",
        titulo: `PRJ#${p.numero} ${p.titulo}`,
        subtitulo: `Marco de Projeto • ${p.gerente_nome || "Sem gerente"}`,
        data: d,
        dataFim: p.data_fim_planejada ? new Date(p.data_fim_planejada) : undefined,
        estado: p.estado,
        estadoRotulo: estadoInfo?.label ?? p.estado,
        prioridadeOuRisco: p.prioridade,
        responsavel: p.gerente_nome,
        href: `/projetos/${p.id}`,
        descricao: p.descricao || p.objetivo,
        detalhesExtras: {
          Prioridade: PRIORIDADE_PROJETO[p.prioridade]?.label ?? p.prioridade,
          Gerente: p.gerente_nome || "Não informado",
          Orçamento: p.orcamento
            ? `R$ ${Number(p.orcamento).toLocaleString("pt-BR", { minimumFractionDigits: 2 })}`
            : "Não definido",
        },
      });
    }

    // 3. Prazos de SLA de Atendimento
    for (const s of filtrarPorEntidade(fila.data?.items ?? [])) {
      const venceEm = s.prazo?.solucao.vence_em || s.prazo?.resposta.vence_em || s.created_at;
      if (!venceEm) continue;
      const d = new Date(venceEm);
      if (isNaN(d.getTime())) continue;

      eventos.push({
        id: `sla-${s.id}`,
        tipo: "sla",
        titulo: `SLA #${s.numero} ${s.titulo}`,
        subtitulo: `Prazo de Atendimento • ${s.oferta_nome}`,
        data: d,
        estado: s.estado,
        estadoRotulo: s.estado.replace(/_/g, " "),
        prioridadeOuRisco: s.prioridade,
        responsavel: s.responsavel_nome || s.solicitante_nome,
        href: `/atendimento/solicitacoes/${s.id}`,
        descricao: s.descricao,
        detalhesExtras: {
          Serviço: s.oferta_nome,
          Prioridade: s.prioridade || "Média",
          Solicitante: s.solicitante_nome,
          Responsável: s.responsavel_nome || "Fila geral",
        },
      });
    }

    // 4. Detecção de Conflitos e Sobreposições Operacionais
    const mapaDatas = new Map<string, EventoCalendario[]>();
    for (const ev of eventos) {
      const k = ev.data.toISOString().slice(0, 10);
      mapaDatas.set(k, [...(mapaDatas.get(k) ?? []), ev]);
    }

    for (const [, lista] of mapaDatas) {
      const temMudancaCritica = lista.some(
        (e) =>
          e.tipo === "mudanca" &&
          (e.prioridadeOuRisco === "alto" || e.prioridadeOuRisco === "medio"),
      );
      if (temMudancaCritica && lista.length > 1) {
        for (const e of lista) {
          e.conflito = true;
          e.motivoConflito =
            "Sobreposição operacional: Mudança do CAB agendada simultaneamente com outros marcos ou prazos de serviço neste mesmo dia.";
        }
      }
    }

    return eventos;
  }, [mudancas.data, projetos.data, fila.data?.items, filtrarPorEntidade]);

  // Filtragem de eventos conforme toggles
  const eventosFiltrados = useMemo(() => {
    return todosEventos.filter((ev) => {
      if (!tiposAtivos[ev.tipo]) return false;
      if (apenasConflitos && !ev.conflito) return false;
      return true;
    });
  }, [todosEventos, tiposAtivos, apenasConflitos]);

  // Contagem de eventos no mês atual
  const mesAtualStr = mesAtual.toISOString().slice(0, 7);
  const eventosNoMes = useMemo(() => {
    return eventosFiltrados.filter((e) => e.data.toISOString().slice(0, 7) === mesAtualStr);
  }, [eventosFiltrados, mesAtualStr]);

  const totalConflitosNoMes = useMemo(() => {
    const datasConflito = new Set<string>();
    for (const e of eventosNoMes) {
      if (e.conflito) datasConflito.add(e.data.toISOString().slice(0, 10));
    }
    return datasConflito.size;
  }, [eventosNoMes]);

  // Construção dos dias da matriz do mês (Grade 7xN)
  const matrizDias = useMemo(() => {
    const ano = mesAtual.getFullYear();
    const mes = mesAtual.getMonth();

    const primeiroDiaMes = new Date(ano, mes, 1);
    const ultimoDiaMes = new Date(ano, mes + 1, 0);

    const inicioGrade = new Date(primeiroDiaMes);
    inicioGrade.setDate(inicioGrade.getDate() - inicioGrade.getDay());

    const fimGrade = new Date(ultimoDiaMes);
    fimGrade.setDate(fimGrade.getDate() + (6 - fimGrade.getDay()));

    const dias: { data: Date; doMesAtual: boolean; chave: string }[] = [];
    const cur = new Date(inicioGrade);

    while (cur <= fimGrade) {
      dias.push({
        data: new Date(cur),
        doMesAtual: cur.getMonth() === mes,
        chave: cur.toISOString().slice(0, 10),
      });
      cur.setDate(cur.getDate() + 1);
    }

    return dias;
  }, [mesAtual]);

  // Agrupamento de eventos filtrados por chave de data "YYYY-MM-DD"
  const eventosPorDia = useMemo(() => {
    const map = new Map<string, EventoCalendario[]>();
    for (const ev of eventosFiltrados) {
      const k = ev.data.toISOString().slice(0, 10);
      map.set(k, [...(map.get(k) ?? []), ev]);
    }
    return map;
  }, [eventosFiltrados]);

  // Navegação de mês
  function irMesAnterior() {
    setMesAtual((prev) => new Date(prev.getFullYear(), prev.getMonth() - 1, 1));
  }

  function irProximoMes() {
    setMesAtual((prev) => new Date(prev.getFullYear(), prev.getMonth() + 1, 1));
  }

  function irHoje() {
    const d = new Date();
    setMesAtual(new Date(d.getFullYear(), d.getMonth(), 1));
  }

  const rotuloMesAno = mesAtual.toLocaleDateString("pt-BR", {
    month: "long",
    year: "numeric",
  });
  const hojeChave = new Date().toISOString().slice(0, 10);

  function toggleTipo(tipo: TipoEvento) {
    setTiposAtivos((prev) => ({ ...prev, [tipo]: !prev[tipo] }));
  }

  return (
    <div className="flex flex-col gap-6" data-testid="assistencia-calendario">
      <PageHeader
        eyebrow="Assistência"
        title="Calendário Operacional de Serviços"
        description="Visão cronológica unificada de janelas de mudanças do CAB, entregas de projetos e prazos de SLA da equipe de atendimento."
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Link href="/assistencia" className={buttonClass("secondary", "sm")}>
              <ArrowLeft className="h-4 w-4" />
              Voltar ao Dashboard
            </Link>
            <div className="flex items-center rounded-lg border border-surface-border bg-surface p-0.5">
              <button
                type="button"
                onClick={() => setModoVisao("grade")}
                aria-label="Ver como grade mensal"
                className={`rounded px-2.5 py-1 text-xs font-medium transition-colors ${
                  modoVisao === "grade"
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted hover:text-foreground"
                }`}
              >
                Grade
              </button>
              <button
                type="button"
                onClick={() => setModoVisao("lista")}
                aria-label="Ver como lista"
                className={`rounded px-2.5 py-1 text-xs font-medium transition-colors ${
                  modoVisao === "lista"
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted hover:text-foreground"
                }`}
              >
                Lista
              </button>
            </div>
          </div>
        }
      />

      {/* Alerta de Conflito Operacional se houver sobreposições no mês */}
      {totalConflitosNoMes > 0 && (
        <div
          role="alert"
          className="flex items-start gap-3 rounded-xl border border-amber-500/30 bg-amber-500/10 p-4 text-sm text-foreground shadow-sm"
        >
          <AlertTriangle className="h-5 w-5 shrink-0 text-amber-500 mt-0.5" />
          <div className="flex-1">
            <p className="font-semibold text-foreground">
              Alerta de Sobreposição Operacional Detectado
            </p>
            <p className="mt-0.5 text-xs text-muted">
              Existem <strong>{totalConflitosNoMes} dia(s)</strong> neste mês com janelas críticas
              de Mudança do CAB agendadas simultaneamente a outros eventos e prazos. Verifique os
              dias destacados para evitar colisões na infraestrutura de TI.
            </p>
          </div>
          <Button
            size="sm"
            variant={apenasConflitos ? "primary" : "secondary"}
            onClick={() => setApenasConflitos((v) => !v)}
            className="text-xs shrink-0"
          >
            {apenasConflitos ? "Ver todos os eventos" : "Filtrar apenas conflitos"}
          </Button>
        </div>
      )}

      {/* Controles de Navegação e Filtros */}
      <div className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface p-4 shadow-sm sm:flex-row sm:items-center sm:justify-between">
        {/* Navegação de Mês */}
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="secondary"
            onClick={irMesAnterior}
            aria-label="Mês anterior"
            className="h-8 w-8 p-0"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <span className="text-base font-semibold capitalize text-foreground min-w-[170px] text-center">
            {rotuloMesAno}
          </span>
          <Button
            size="sm"
            variant="secondary"
            onClick={irProximoMes}
            aria-label="Próximo mês"
            className="h-8 w-8 p-0"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
          <Button size="sm" variant="secondary" onClick={irHoje} className="text-xs ml-1">
            Hoje
          </Button>
        </div>

        {/* Filtros por Módulo/Tipo */}
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <span className="text-muted flex items-center gap-1 mr-1">
            <Filter className="h-3.5 w-3.5" /> Filtros:
          </span>
          <button
            type="button"
            onClick={() => toggleTipo("mudanca")}
            className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 font-medium transition-colors border ${
              tiposAtivos.mudanca
                ? "bg-purple-500/15 text-purple-700 dark:text-purple-300 border-purple-500/30"
                : "bg-surface text-muted border-surface-border opacity-50"
            }`}
          >
            <span className="h-2 w-2 rounded-full bg-purple-500" />
            Mudanças (CAB)
          </button>

          <button
            type="button"
            onClick={() => toggleTipo("projeto")}
            className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 font-medium transition-colors border ${
              tiposAtivos.projeto
                ? "bg-blue-500/15 text-blue-700 dark:text-blue-300 border-blue-500/30"
                : "bg-surface text-muted border-surface-border opacity-50"
            }`}
          >
            <span className="h-2 w-2 rounded-full bg-blue-500" />
            Marcos de Projetos
          </button>

          <button
            type="button"
            onClick={() => toggleTipo("sla")}
            className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 font-medium transition-colors border ${
              tiposAtivos.sla
                ? "bg-amber-500/15 text-amber-700 dark:text-amber-300 border-amber-500/30"
                : "bg-surface text-muted border-surface-border opacity-50"
            }`}
          >
            <span className="h-2 w-2 rounded-full bg-amber-500" />
            Prazos de SLA
          </button>
        </div>
      </div>

      {/* Visualização: Grade Mensal */}
      {modoVisao === "grade" && (
        <div className="overflow-x-auto rounded-xl border border-surface-border bg-surface shadow-sm">
          <div className="min-w-[800px]">
            {/* Cabeçalho dos Dias da Semana */}
            <div className="grid grid-cols-7 border-b border-surface-border bg-black/[0.02] dark:bg-white/[0.02]">
              {DIAS_SEMANA.map((dia) => (
                <div
                  key={dia}
                  className="py-2.5 text-center text-xs font-semibold uppercase tracking-wider text-muted"
                >
                  {dia}
                </div>
              ))}
            </div>

            {/* Matriz dos Dias */}
            <div className="grid grid-cols-7 divide-x divide-y divide-surface-border">
              {matrizDias.map(({ data, doMesAtual, chave }) => {
                const eventosDoDia = eventosPorDia.get(chave) ?? [];
                const eHoje = chave === hojeChave;
                const temConflitoNoDia = eventosDoDia.some((e) => e.conflito);

                return (
                  <div
                    key={chave}
                    data-date={chave}
                    className={`min-h-[120px] p-2 flex flex-col justify-between transition-colors ${
                      !doMesAtual
                        ? "bg-black/[0.01] dark:bg-white/[0.01] opacity-40"
                        : temConflitoNoDia
                          ? "bg-amber-500/[0.04]"
                          : "bg-surface"
                    } ${eHoje ? "ring-2 ring-inset ring-primary/40" : ""}`}
                  >
                    {/* Número do Dia & Badges */}
                    <div className="flex items-center justify-between mb-1.5">
                      <span
                        className={`inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold ${
                          eHoje
                            ? "bg-primary text-primary-foreground font-bold shadow-xs"
                            : doMesAtual
                              ? "text-foreground"
                              : "text-muted"
                        }`}
                      >
                        {data.getDate()}
                      </span>

                      {temConflitoNoDia && (
                        <span
                          title="Conflito de intervenções neste dia"
                          className="flex items-center gap-0.5 text-[10px] font-semibold text-amber-600 dark:text-amber-400"
                        >
                          <AlertTriangle className="h-3 w-3" />
                        </span>
                      )}
                    </div>

                    {/* Lista de Eventos do Dia */}
                    <div className="flex flex-col gap-1 overflow-y-auto max-h-[140px]">
                      {eventosDoDia.slice(0, 3).map((ev) => {
                        let corEstilo = "bg-primary/10 text-primary border-primary/20";
                        if (ev.tipo === "mudanca") {
                          corEstilo =
                            ev.prioridadeOuRisco === "alto"
                              ? "bg-rose-500/15 text-rose-700 dark:text-rose-300 border-rose-500/30"
                              : "bg-purple-500/15 text-purple-700 dark:text-purple-300 border-purple-500/30";
                        } else if (ev.tipo === "projeto") {
                          corEstilo =
                            "bg-blue-500/15 text-blue-700 dark:text-blue-300 border-blue-500/30";
                        } else if (ev.tipo === "sla") {
                          corEstilo =
                            "bg-amber-500/15 text-amber-700 dark:text-amber-300 border-amber-500/30";
                        }

                        return (
                          <button
                            key={ev.id}
                            type="button"
                            onClick={() => setEventoSelecionado(ev)}
                            className={`w-full text-left truncate rounded px-1.5 py-0.5 text-[11px] font-medium border transition-transform hover:scale-[1.01] active:scale-[0.99] ${corEstilo}`}
                            title={`${ev.titulo} — ${ev.subtitulo}`}
                          >
                            <span className="truncate">{ev.titulo}</span>
                          </button>
                        );
                      })}

                      {eventosDoDia.length > 3 && (
                        <button
                          type="button"
                          onClick={() => setEventoSelecionado(eventosDoDia[3]!)}
                          className="text-[10px] text-muted hover:text-foreground font-medium text-center py-0.5"
                        >
                          +{eventosDoDia.length - 3} mais
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      )}

      {/* Visualização: Lista de Eventos / Prazos */}
      {modoVisao === "lista" && (
        <Card>
          <CardHeader>
            <CardTitle as="h2" className="text-base font-semibold">
              Eventos e Prazos do Mês ({rotuloMesAno})
            </CardTitle>
          </CardHeader>
          <CardContent>
            {eventosNoMes.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted">
                Nenhum evento registrado para os filtros selecionados neste mês.
              </p>
            ) : (
              <div className="divide-y divide-surface-border">
                {eventosNoMes
                  .sort((a, b) => a.data.getTime() - b.data.getTime())
                  .map((ev) => (
                    <div
                      key={ev.id}
                      className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 py-3 hover:bg-black/[0.01] dark:hover:bg-white/[0.01] transition-colors"
                    >
                      <div className="flex items-start gap-3">
                        <div className="flex flex-col items-center justify-center rounded-lg border border-surface-border bg-black/[0.02] dark:bg-white/[0.02] px-2.5 py-1 text-center shrink-0">
                          <span className="text-[10px] font-semibold uppercase text-muted">
                            {ev.data.toLocaleDateString("pt-BR", { month: "short" })}
                          </span>
                          <span className="text-base font-bold text-foreground">
                            {ev.data.getDate()}
                          </span>
                        </div>

                        <div>
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-semibold text-sm text-foreground">
                              {ev.titulo}
                            </span>
                            <Badge
                              tone={
                                ev.tipo === "mudanca"
                                  ? "info"
                                  : ev.tipo === "projeto"
                                    ? "success"
                                    : "warning"
                              }
                            >
                              {ev.tipo === "mudanca"
                                ? "Mudança CAB"
                                : ev.tipo === "projeto"
                                  ? "Projeto"
                                  : "Prazo SLA"}
                            </Badge>
                            {ev.conflito && (
                              <Badge tone="danger" className="inline-flex items-center gap-1">
                                <AlertTriangle className="h-3 w-3" />
                                Conflito
                              </Badge>
                            )}
                          </div>
                          <p className="text-xs text-muted mt-0.5">{ev.subtitulo}</p>
                          {ev.responsavel && (
                            <p className="text-[11px] text-muted mt-0.5">
                              Responsável: {ev.responsavel}
                            </p>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-2 shrink-0 self-end sm:self-center">
                        <Button
                          size="sm"
                          variant="secondary"
                          onClick={() => setEventoSelecionado(ev)}
                          className="text-xs"
                        >
                          Ver Detalhes
                        </Button>
                        <Link href={ev.href} className={buttonClass("ghost", "sm", "text-xs")}>
                          Abrir <ExternalLink className="h-3.5 w-3.5 ml-1" />
                        </Link>
                      </div>
                    </div>
                  ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Modal de Detalhes do Evento Selecionado */}
      <Dialog
        open={!!eventoSelecionado}
        onClose={() => setEventoSelecionado(null)}
        title={eventoSelecionado?.titulo ?? "Detalhes do Evento"}
        description={eventoSelecionado?.subtitulo}
      >
        {eventoSelecionado && (
          <div className="flex flex-col gap-4 mt-2">
            {eventoSelecionado.conflito && (
              <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-foreground flex items-start gap-2">
                <AlertTriangle className="h-4 w-4 text-amber-500 shrink-0 mt-0.5" />
                <div>
                  <p className="font-semibold">Aviso de Sobreposição</p>
                  <p className="mt-0.5 text-muted">{eventoSelecionado.motivoConflito}</p>
                </div>
              </div>
            )}

            <div className="grid grid-cols-2 gap-3 text-xs">
              <div className="rounded-lg border border-surface-border p-2.5">
                <span className="text-muted block">Data / Início</span>
                <span className="font-semibold text-foreground text-sm">
                  {eventoSelecionado.data.toLocaleString("pt-BR", {
                    dateStyle: "short",
                    timeStyle: "short",
                  })}
                </span>
              </div>
              <div className="rounded-lg border border-surface-border p-2.5">
                <span className="text-muted block">Situação / Estado</span>
                <span className="font-semibold text-foreground text-sm capitalize">
                  {eventoSelecionado.estadoRotulo}
                </span>
              </div>
            </div>

            {eventoSelecionado.descricao && (
              <div className="rounded-lg border border-surface-border p-3 text-xs">
                <span className="text-muted font-medium block mb-1">Descrição / Contexto</span>
                <p className="text-foreground leading-relaxed whitespace-pre-wrap">
                  {eventoSelecionado.descricao}
                </p>
              </div>
            )}

            {eventoSelecionado.detalhesExtras && (
              <div className="grid grid-cols-2 gap-2 text-xs">
                {Object.entries(eventoSelecionado.detalhesExtras).map(([rotulo, valor]) => (
                  <div
                    key={rotulo}
                    className="p-2 rounded border border-surface-border bg-black/[0.01] dark:bg-white/[0.01]"
                  >
                    <span className="text-muted block text-[11px]">{rotulo}</span>
                    <span className="font-medium text-foreground">{valor}</span>
                  </div>
                ))}
              </div>
            )}

            <div className="flex justify-end gap-2 pt-2 border-t border-surface-border">
              <Button variant="secondary" onClick={() => setEventoSelecionado(null)}>
                Fechar
              </Button>
              <Link href={eventoSelecionado.href} className={buttonClass("primary", "md")}>
                Abrir no Módulo <ExternalLink className="h-3.5 w-3.5 ml-1.5" />
              </Link>
            </div>
          </div>
        )}
      </Dialog>
    </div>
  );
}
