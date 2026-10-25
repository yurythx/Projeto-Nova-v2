"use client";

import Link from "next/link";
import { useState } from "react";
import {
  AlertTriangle,
  ArrowRight,
  Award,
  Calendar,
  CheckCircle2,
  Clock,
  Coins,
  FileSpreadsheet,
  FolderKanban,
  GitCommit,
  Headphones,
  Printer,
  ShieldAlert,
  Sparkles,
} from "lucide-react";

import { PageHeader } from "@/components/plataforma/PageHeader";
import { AssistenciaGraficos } from "@/components/servicos/AssistenciaGraficos";
import { RelatorioExecutivoModal } from "@/components/servicos/RelatorioExecutivoModal";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { useApiPage, useApiQuery } from "@/lib/api/swr";
import type { Governanca, PontoIndicador, Solicitacao } from "@/lib/atendimento/atendimento";
import type { Projeto } from "@/lib/projetos/projetos";
import type { Mudanca, Problema } from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

function textoMedida(valor: number | null, sufixo: string) {
  if (valor == null) return "sem amostra";
  return `${valor.toLocaleString("pt-BR")}${sufixo}`;
}

function SerieIndicadores({ serie, erro }: { serie?: PontoIndicador[]; erro?: unknown }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Histórico dos indicadores</CardTitle>
      </CardHeader>
      <CardContent className="pb-5">
        <p className="mb-3 text-sm text-muted">
          O dia segue o fuso da unidade. O dia de hoje é atualizado. Os anteriores ficam como foram medidos.
        </p>
        {erro ? (
          <p className="text-sm text-danger" role="alert">
            Não foi possível carregar o histórico.
          </p>
        ) : null}
        {(serie ?? []).length === 0 && !erro ? (
          <p className="text-sm">Ainda não há dia medido neste escopo.</p>
        ) : null}
        {(serie ?? []).length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="text-muted">
                  <th className="py-1 pr-3 font-medium">Dia</th>
                  <th className="py-1 pr-3 font-medium">Resolução</th>
                  <th className="py-1 pr-3 font-medium">Entre falhas</th>
                  <th className="py-1 pr-3 font-medium">Prazo</th>
                  <th className="py-1 pr-3 font-medium">Primeiro contato</th>
                  <th className="py-1 font-medium">Satisfação</th>
                </tr>
              </thead>
              <tbody>
                {serie!.map((p) => (
                  <tr key={p.dia} className="border-t border-surface-border">
                    <td className="py-1 pr-3">{p.dia.slice(0, 10)}{p.truncado ? " · amostra parcial" : ""}</td>
                    <td className="py-1 pr-3">{textoMedida(p.mttr_horas, " h")}</td>
                    <td className="py-1 pr-3">{textoMedida(p.mtbf_horas, " h")}</td>
                    <td className="py-1 pr-3">{textoMedida(p.sla_percentual, "%")}</td>
                    <td className="py-1 pr-3">{textoMedida(p.fcr_percentual, "%")}</td>
                    <td className="py-1">{textoMedida(p.csat_media, "")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

export default function AssistenciaDashboardPage() {
  const { can, enabled } = usePlataforma();
  const [modalRelatorio, setModalRelatorio] = useState(false);

  const atende = can("atendimento:atender");
  const fila = useApiPage<Solicitacao>(
    enabled("atendimento") && atende ? "v1/atendimento/fila?page_size=100" : null,
  );
  const governanca = useApiQuery<Governanca>(
    enabled("atendimento") && atende ? "v1/atendimento/governanca" : null,
  );
  const serie = useApiQuery<PontoIndicador[]>(
    enabled("atendimento") && atende ? "v1/atendimento/governanca/serie?dias=90" : null,
  );

  const gereProblemas = can("problemas:manage");
  const leProblemas = gereProblemas || can("problemas:read");
  const problemas = useApiQuery<Problema[]>(
    enabled("problemas") && leProblemas ? "v1/problemas" : null,
  );

  const gereMudancas = can("mudancas:manage");
  const veMudancas = gereMudancas || can("mudancas:aprovar");
  const mudancas = useApiQuery<Mudanca[]>(
    enabled("mudancas") && veMudancas ? "v1/mudancas" : null,
  );

  const gereProjetos = can("projetos:manage");
  const leProjetos = gereProjetos || can("projetos:read");
  const projetos = useApiQuery<Projeto[]>(
    enabled("projetos") && leProjetos ? "v1/projetos" : null,
  );

  // Métricas de Atendimento
  const solicitacoes = fila.data?.items ?? [];
  const abertas = solicitacoes.filter((s) => s.estado === "aberto").length;
  const emAtendimento = solicitacoes.filter((s) => s.estado === "em_atendimento").length;
  const aguardando = solicitacoes.filter(
    (s) => s.estado === "aguardando_servidor" || s.estado === "aguardando_terceiro",
  ).length;
  const concluidas = solicitacoes.filter(
    (s) => s.estado === "resolvido" || s.estado === "atendido",
  ).length;
  const incidentes = solicitacoes.filter((s) => s.tipo === "incidente").length;
  const requisicoes = solicitacoes.filter((s) => s.tipo === "requisicao").length;

  // Métricas de Problemas
  const listaProblemas = problemas.data ?? [];
  const probEmAnalise = listaProblemas.filter((p) => p.estado === "em_analise").length;
  const errosConhecidos = listaProblemas.filter((p) => p.estado === "erro_conhecido").length;
  const probResolvidos = listaProblemas.filter(
    (p) => p.estado === "resolvido" || p.estado === "encerrado",
  ).length;

  // Métricas de Mudanças
  const listaMudancas = mudancas.data ?? [];
  const mudAvaliacao = listaMudancas.filter((m) => m.estado === "em_avaliacao").length;
  const mudAprovadas = listaMudancas.filter((m) => m.estado === "aprovada").length;
  const mudExecucao = listaMudancas.filter((m) => m.estado === "em_execucao").length;
  const mudConcluidas = listaMudancas.filter((m) => m.estado === "concluida").length;
  const mudFalhas = listaMudancas.filter((m) => m.estado === "falhou").length;

  // Métricas de Projetos
  const listaProjetos = projetos.data ?? [];
  const projPlanejados = listaProjetos.filter((p) => p.estado === "planejado").length;
  const projEmAndamento = listaProjetos.filter((p) => p.estado === "em_andamento").length;
  const projSuspensos = listaProjetos.filter((p) => p.estado === "suspenso").length;
  const projConcluidos = listaProjetos.filter((p) => p.estado === "concluido").length;
  const totalOrcamento = listaProjetos.reduce(
    (acc, p) => acc + (Number(p.orcamento) || 0),
    0,
  );

  function exportarRelatorioGeral() {
    const cabecalho = ["Modulo", "Metrica", "Valor", "DataExtracao"];
    const linhas = [
      ["Atendimento", "Total de Solicitacoes", solicitacoes.length, new Date().toLocaleDateString("pt-BR")],
      ["Atendimento", "Abertas", abertas, new Date().toLocaleDateString("pt-BR")],
      ["Atendimento", "Em Atendimento", emAtendimento, new Date().toLocaleDateString("pt-BR")],
      ["Atendimento", "Incidentes", incidentes, new Date().toLocaleDateString("pt-BR")],
      ["Atendimento", "Requisicoes", requisicoes, new Date().toLocaleDateString("pt-BR")],
      ["Problemas", "Total Registrados", listaProblemas.length, new Date().toLocaleDateString("pt-BR")],
      ["Problemas", "Em Investigacao", probEmAnalise, new Date().toLocaleDateString("pt-BR")],
      ["Problemas", "Erros Conhecidos (Contorno)", errosConhecidos, new Date().toLocaleDateString("pt-BR")],
      ["Problemas", "Resolvidos", probResolvidos, new Date().toLocaleDateString("pt-BR")],
      ["Mudancas", "Total Registradas", listaMudancas.length, new Date().toLocaleDateString("pt-BR")],
      ["Mudancas", "Em Avaliacao (CAB)", mudAvaliacao, new Date().toLocaleDateString("pt-BR")],
      ["Mudancas", "Aprovadas para Janela", mudAprovadas, new Date().toLocaleDateString("pt-BR")],
      ["Mudancas", "Em Execucao", mudExecucao, new Date().toLocaleDateString("pt-BR")],
      ["Mudancas", "Concluidas com Sucesso", mudConcluidas, new Date().toLocaleDateString("pt-BR")],
      ["Projetos", "Total Cadastrados", listaProjetos.length, new Date().toLocaleDateString("pt-BR")],
      ["Projetos", "Em Andamento", projEmAndamento, new Date().toLocaleDateString("pt-BR")],
      ["Projetos", "Planejados", projPlanejados, new Date().toLocaleDateString("pt-BR")],
      ["Projetos", "Concluidos", projConcluidos, new Date().toLocaleDateString("pt-BR")],
      ["Projetos", "Orcamento Alocado (R$)", totalOrcamento, new Date().toLocaleDateString("pt-BR")],
    ];

    const conteudo = "\uFEFF" + [cabecalho.join(";"), ...linhas.map((l) => l.join(";"))].join("\n");
    const blob = new Blob([conteudo], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute("download", `relatorio-assistencia-${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Assistência"
        title="Dashboard de Serviços"
        description="Painel unificado da central de serviços: solicitações de atendimento, incidentes, problemas conhecidos, gestão de mudanças e projetos estratégicos."
        actions={
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="secondary"
              type="button"
              onClick={() => setModalRelatorio(true)}
              className="inline-flex items-center gap-1.5"
            >
              <Award className="h-4 w-4 text-primary" />
              Relatório Executivo
            </Button>
            <Link
              href="/assistencia/calendario"
              className={buttonClass("secondary", "sm")}
            >
              <Calendar className="h-4 w-4" />
              Calendário
            </Link>
            <Button
              size="sm"
              variant="secondary"
              type="button"
              onClick={() => window.print()}
              className="inline-flex items-center gap-1.5"
            >
              <Printer className="h-4 w-4" />
              Imprimir Painel
            </Button>
            <Button
              size="sm"
              variant="secondary"
              type="button"
              onClick={exportarRelatorioGeral}
              className="inline-flex items-center gap-1.5"
            >
              <FileSpreadsheet className="h-4 w-4" />
              Exportar CSV
            </Button>
            <Badge tone="info">Central de Assistência</Badge>
          </div>
        }
      />

      {governanca.data?.csat && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Satisfação do atendimento</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 pb-5">
            <p className="text-sm">
              Média <strong>{governanca.data.csat.media.toLocaleString("pt-BR")}</strong> de 5, em{" "}
              {governanca.data.csat.total} {governanca.data.csat.total === 1 ? "avaliação" : "avaliações"}.
            </p>
            <ul className="flex flex-col gap-2">
              {governanca.data.csat.recentes.map((a) => (
                <li key={a.id} className="text-sm">
                  <span className="font-medium">{a.nota}/5</span>
                  <span className="text-muted"> · solicitação #{a.numero}</span>
                  {a.comentario ? <span className="text-muted"> — {a.comentario}</span> : null}
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {enabled("atendimento") && atende ? <SerieIndicadores serie={serie.data} erro={serie.error} /> : null}

      {/* Destaques Macro / KPIs da Assistência */}
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl border border-surface-border bg-surface p-4 shadow-sm flex items-center justify-between">
          <div>
            <p className="text-xs text-muted">Fila de Atendimento</p>
            <p className="text-2xl font-bold text-foreground mt-0.5">
              {solicitacoes.length}
            </p>
            <span className="text-[11px] text-muted">
              {abertas} abertas • {emAtendimento} em curso
            </span>
          </div>
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Headphones className="h-5 w-5" />
          </div>
        </div>

        <div className="rounded-xl border border-surface-border bg-surface p-4 shadow-sm flex items-center justify-between">
          <div>
            <p className="text-xs text-muted">Erros Conhecidos</p>
            <p className="text-2xl font-bold text-success mt-0.5">
              {errosConhecidos}
            </p>
            <span className="text-[11px] text-muted">
              {probEmAnalise} problemas em análise
            </span>
          </div>
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-warning/10 text-warning">
            <AlertTriangle className="h-5 w-5" />
          </div>
        </div>

        <div className="rounded-xl border border-surface-border bg-surface p-4 shadow-sm flex items-center justify-between">
          <div>
            <p className="text-xs text-muted">Janelas de Mudança</p>
            <p className="text-2xl font-bold text-info mt-0.5">
              {mudAprovadas + mudExecucao}
            </p>
            <span className="text-[11px] text-muted">
              {mudAvaliacao} sob aprovação do CAB
            </span>
          </div>
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info/10 text-info">
            <GitCommit className="h-5 w-5" />
          </div>
        </div>

        <div className="rounded-xl border border-surface-border bg-surface p-4 shadow-sm flex items-center justify-between">
          <div>
            <p className="text-xs text-muted">Projetos Ativos</p>
            <p className="text-2xl font-bold text-primary mt-0.5">
              {projEmAndamento + projPlanejados}
            </p>
            <span className="text-[11px] text-muted">
              {projConcluidos} projetos concluídos
            </span>
          </div>
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <FolderKanban className="h-5 w-5" />
          </div>
        </div>
      </div>

      {/* Gráficos Visuais e Tendências */}
      <AssistenciaGraficos
        incidentes={incidentes}
        requisicoes={requisicoes}
        mudancas={listaMudancas}
        projetos={listaProjetos}
      />

      {/* Grid de Seções da Assistência */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* Card Atendimento */}
        {enabled("atendimento") && (
          <Card className="flex flex-col justify-between">
            <div>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <div className="flex items-center gap-2">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Headphones className="h-5 w-5" />
                  </div>
                  <CardTitle as="h2">Atendimento</CardTitle>
                </div>
                <Badge tone={abertas > 0 ? "warning" : "success"}>
                  {solicitacoes.length} ativas
                </Badge>
              </CardHeader>
              <CardContent className="flex flex-col gap-4 pt-2">
                <div className="grid grid-cols-2 gap-2 text-center">
                  <div className="rounded-lg border border-surface-border bg-black/[0.02] p-2 dark:bg-white/[0.02]">
                    <span className="block text-xl font-bold text-foreground">{abertas}</span>
                    <span className="text-xs text-muted">Abertas</span>
                  </div>
                  <div className="rounded-lg border border-surface-border bg-black/[0.02] p-2 dark:bg-white/[0.02]">
                    <span className="block text-xl font-bold text-primary">{emAtendimento}</span>
                    <span className="text-xs text-muted">Em atendimento</span>
                  </div>
                </div>

                <div className="flex flex-col gap-1.5 text-xs text-muted">
                  <div className="flex justify-between">
                    <span>Aguardando resposta</span>
                    <span className="font-semibold text-foreground">{aguardando}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Incidentes vs Requisições</span>
                    <span className="font-semibold text-foreground">
                      {incidentes} / {requisicoes}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span>Concluídas recentemente</span>
                    <span className="font-semibold text-success">{concluidas}</span>
                  </div>
                </div>
              </CardContent>
            </div>
            <div className="border-t border-surface-border px-5 py-3">
              <Link
                href="/atendimento/fila"
                className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
              >
                Abrir fila e quadro de atendimento
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </div>
          </Card>
        )}

        {/* Card Problemas */}
        {enabled("problemas") && (
          <Card className="flex flex-col justify-between">
            <div>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <div className="flex items-center gap-2">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-warning/10 text-warning">
                    <AlertTriangle className="h-5 w-5" />
                  </div>
                  <CardTitle as="h2">Problemas</CardTitle>
                </div>
                <Badge tone="neutral">{listaProblemas.length} itens</Badge>
              </CardHeader>
              <CardContent className="flex flex-col gap-4 pt-2">
                <div className="grid grid-cols-2 gap-2 text-center">
                  <div className="rounded-lg border border-surface-border bg-black/[0.02] p-2 dark:bg-white/[0.02]">
                    <span className="block text-xl font-bold text-warning">{probEmAnalise}</span>
                    <span className="text-xs text-muted">Em investigação</span>
                  </div>
                  <div className="rounded-lg border border-surface-border bg-black/[0.02] p-2 dark:bg-white/[0.02]">
                    <span className="block text-xl font-bold text-success">{errosConhecidos}</span>
                    <span className="text-xs text-muted">Erros conhecidos</span>
                  </div>
                </div>

                <div className="flex flex-col gap-1.5 text-xs text-muted">
                  <div className="flex justify-between">
                    <span>Com solução de contorno</span>
                    <span className="font-semibold text-foreground">{errosConhecidos}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Problemas resolvidos</span>
                    <span className="font-semibold text-success">{probResolvidos}</span>
                  </div>
                </div>
              </CardContent>
            </div>
            <div className="border-t border-surface-border px-5 py-3">
              <Link
                href="/problemas"
                className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
              >
                Gerenciar problemas e contornos
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </div>
          </Card>
        )}

        {/* Card Mudanças */}
        {enabled("mudancas") && (
          <Card className="flex flex-col justify-between">
            <div>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <div className="flex items-center gap-2">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-info/10 text-info">
                    <GitCommit className="h-5 w-5" />
                  </div>
                  <CardTitle as="h2">Mudanças</CardTitle>
                </div>
                <Badge tone="neutral">{listaMudancas.length} registradas</Badge>
              </CardHeader>
              <CardContent className="flex flex-col gap-4 pt-2">
                <div className="grid grid-cols-2 gap-2 text-center">
                  <div className="rounded-lg border border-surface-border bg-black/[0.02] p-2 dark:bg-white/[0.02]">
                    <span className="block text-xl font-bold text-info">{mudAvaliacao}</span>
                    <span className="text-xs text-muted">Em avaliação</span>
                  </div>
                  <div className="rounded-lg border border-surface-border bg-black/[0.02] p-2 dark:bg-white/[0.02]">
                    <span className="block text-xl font-bold text-primary">{mudExecucao}</span>
                    <span className="text-xs text-muted">Em execução</span>
                  </div>
                </div>

                <div className="flex flex-col gap-1.5 text-xs text-muted">
                  <div className="flex justify-between">
                    <span>Aprovadas para janela</span>
                    <span className="font-semibold text-foreground">{mudAprovadas}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Concluídas com sucesso</span>
                    <span className="font-semibold text-success">{mudConcluidas}</span>
                  </div>
                  {mudFalhas > 0 && (
                    <div className="flex justify-between text-danger">
                      <span>Mudanças com falha</span>
                      <span className="font-semibold">{mudFalhas}</span>
                    </div>
                  )}
                </div>
              </CardContent>
            </div>
            <div className="border-t border-surface-border px-5 py-3">
              <Link
                href="/mudancas"
                className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
              >
                Planejar e acompanhar mudanças
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </div>
          </Card>
        )}

        {/* Card Projetos */}
        {enabled("projetos") && (
          <Card className="flex flex-col justify-between">
            <div>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <div className="flex items-center gap-2">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <FolderKanban className="h-5 w-5" />
                  </div>
                  <CardTitle as="h2">Projetos</CardTitle>
                </div>
                <Badge tone={projEmAndamento > 0 ? "warning" : "neutral"}>
                  {listaProjetos.length} cadastrados
                </Badge>
              </CardHeader>
              <CardContent className="flex flex-col gap-4 pt-2">
                <div className="grid grid-cols-2 gap-2 text-center">
                  <div className="rounded-lg border border-surface-border bg-black/[0.02] p-2 dark:bg-white/[0.02]">
                    <span className="block text-xl font-bold text-warning">{projEmAndamento}</span>
                    <span className="text-xs text-muted">Em andamento</span>
                  </div>
                  <div className="rounded-lg border border-surface-border bg-black/[0.02] p-2 dark:bg-white/[0.02]">
                    <span className="block text-xl font-bold text-info">{projPlanejados}</span>
                    <span className="text-xs text-muted">Planejados</span>
                  </div>
                </div>

                <div className="flex flex-col gap-1.5 text-xs text-muted">
                  <div className="flex justify-between">
                    <span>Projetos concluídos</span>
                    <span className="font-semibold text-success">{projConcluidos}</span>
                  </div>
                  {projSuspensos > 0 && (
                    <div className="flex justify-between text-warning">
                      <span>Projetos suspensos</span>
                      <span className="font-semibold">{projSuspensos}</span>
                    </div>
                  )}
                  {totalOrcamento > 0 && (
                    <div className="flex justify-between">
                      <span>Orçamento alocado</span>
                      <span className="font-semibold font-mono text-foreground">
                        R$ {totalOrcamento.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
                      </span>
                    </div>
                  )}
                </div>
              </CardContent>
            </div>
            <div className="border-t border-surface-border px-5 py-3">
              <Link
                href="/projetos"
                className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
              >
                Gerenciar projetos e tarefas
                <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </div>
          </Card>
        )}
      </div>

      <RelatorioExecutivoModal
        open={modalRelatorio}
        onClose={() => setModalRelatorio(false)}
        solicitacoes={solicitacoes}
        problemas={listaProblemas}
        mudancas={listaMudancas}
        projetos={listaProjetos}
        governanca={governanca.data}
      />
    </div>
  );
}
