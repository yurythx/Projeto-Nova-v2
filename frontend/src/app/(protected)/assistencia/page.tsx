"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Award, Calendar, FileSpreadsheet, Printer } from "lucide-react";

import { PageHeader } from "@/components/plataforma/PageHeader";
import { AssistenciaGraficos } from "@/components/servicos/AssistenciaGraficos";
import { BarraDeFiltros, filtrosIniciais, PainelDaAssistencia, type FiltrosAssistencia } from "@/components/servicos/PainelDaAssistencia";
import { RelatorioExecutivoModal } from "@/components/servicos/RelatorioExecutivoModal";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import type { Governanca, PontoIndicador, Solicitacao } from "@/lib/atendimento/atendimento";
import type { Projeto } from "@/lib/projetos/projetos";
import { estiloGraficoDe, type EstiloGrafico } from "@/lib/servicos/painelAssistencia";
import type { Mudanca, Problema } from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

const ESTILO_MEMORIA = "nova-assistencia:estilo";

export default function AssistenciaDashboardPage() {
  const { can, enabled } = usePlataforma();
  const [modalRelatorio, setModalRelatorio] = useState(false);
  const [aba, setAba] = useState<"painel" | "graficos">("painel");
  const [editar, setEditar] = useState(false);
  const [filtros, setFiltros] = useState<FiltrosAssistencia>(filtrosIniciais);
  const [estilo, setEstilo] = useState<EstiloGrafico>("barras");

  useEffect(() => {
    try {
      const lido = estiloGraficoDe(localStorage.getItem(ESTILO_MEMORIA));
      setEstilo(lido);
    } catch {
      // Sem armazenamento: fica em barras.
    }
  }, []);

  function escolherEstilo(proximo: EstiloGrafico) {
    setEstilo(proximo);
    try {
      localStorage.setItem(ESTILO_MEMORIA, proximo);
    } catch {
      // A escolha vale só nesta visita.
    }
  }

  const atende = can("atendimento:atender");
  const atendimentoLigado = enabled("atendimento");
  const veFila = atendimentoLigado && atende;
  const fila = useApiPage<Solicitacao>(
    veFila
      ? withQuery("v1/atendimento/fila", {
          page_size: 100,
          tipo: filtros.tipo,
          situacao: filtros.situacao,
          estado: filtros.estadoChamado,
        })
      : null,
  );
  const governanca = useApiQuery<Governanca>(veFila ? "v1/atendimento/governanca" : null);
  const serie = useApiQuery<PontoIndicador[]>(
    veFila ? withQuery("v1/atendimento/governanca/serie", { dias: filtros.dias }) : null,
  );

  const gereProblemas = can("problemas:manage");
  const leProblemas = gereProblemas || can("problemas:read");
  const problemas = useApiQuery<Problema[]>(enabled("problemas") && leProblemas ? "v1/problemas" : null);

  const gereMudancas = can("mudancas:manage");
  const veMudancas = gereMudancas || can("mudancas:aprovar");
  const mudancas = useApiQuery<Mudanca[]>(enabled("mudancas") && veMudancas ? "v1/mudancas" : null);

  const gereProjetos = can("projetos:manage");
  const leProjetos = gereProjetos || can("projetos:read");
  const projetos = useApiQuery<Projeto[]>(enabled("projetos") && leProjetos ? "v1/projetos" : null);

  const solicitacoes = fila.data?.items ?? [];
  const totalFila = fila.data?.meta?.total_items ?? solicitacoes.length;
  const listaProblemas = (problemas.data ?? []).filter((item) => !filtros.estadoProblema || item.estado === filtros.estadoProblema);
  const listaMudancas = (mudancas.data ?? []).filter((item) => !filtros.estadoMudanca || item.estado === filtros.estadoMudanca);
  const listaProjetos = (projetos.data ?? []).filter((item) => !filtros.estadoProjeto || item.estado === filtros.estadoProjeto);
  const incidentes = solicitacoes.filter((s) => s.tipo === "incidente").length;
  const requisicoes = solicitacoes.filter((s) => s.tipo === "requisicao").length;
  const notaFila = veFila && totalFila > solicitacoes.length ? `A fila tem ${totalFila}. Esta página mostra ${solicitacoes.length}.` : undefined;

  function exportarRelatorioGeral() {
    const cabecalho = ["Modulo", "Metrica", "Valor", "DataExtracao"];
    const hoje = new Date().toLocaleDateString("pt-BR");
    const linhas: (string | number)[][] = [
      ["Atendimento", "Na pagina da fila", solicitacoes.length, hoje],
      ["Atendimento", "Total informado", veFila ? totalFila : "fora do alcance", hoje],
      ["Atendimento", "Incidentes", incidentes, hoje],
      ["Atendimento", "Requisicoes", requisicoes, hoje],
      ["Problemas", "No recorte", listaProblemas.length, hoje],
      ["Mudancas", "No recorte", listaMudancas.length, hoje],
      ["Projetos", "No recorte", listaProjetos.length, hoje],
    ];
    if (governanca.data) {
      linhas.push(
        ["Atendimento", "Resolucao (h)", governanca.data.mttr_horas ?? "sem amostra", hoje],
        ["Atendimento", "Prazo (%)", governanca.data.sla_percentual ?? "sem amostra", hoje],
        ["Atendimento", "Primeiro contato (%)", governanca.data.fcr_percentual ?? "sem amostra", hoje],
      );
    }
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
        description="Os filtros valem para o painel e para os gráficos. A ordem dos cartões fica neste navegador."
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button size="sm" variant="secondary" type="button" onClick={() => setEditar((atual) => !atual)}>
              {editar ? "Concluir organização" : "Organizar painel"}
            </Button>
            <Button size="sm" variant="secondary" type="button" onClick={() => setModalRelatorio(true)} className="inline-flex items-center gap-1.5">
              <Award className="h-4 w-4 text-primary" />
              Relatório Executivo
            </Button>
            <Link href="/assistencia/calendario" className={buttonClass("secondary", "sm")}>
              <Calendar className="h-4 w-4" />
              Calendário
            </Link>
            <Button size="sm" variant="secondary" type="button" onClick={() => window.print()} className="inline-flex items-center gap-1.5">
              <Printer className="h-4 w-4" />
              Imprimir Painel
            </Button>
            <Button size="sm" variant="secondary" type="button" onClick={exportarRelatorioGeral} className="inline-flex items-center gap-1.5">
              <FileSpreadsheet className="h-4 w-4" />
              Exportar CSV
            </Button>
            <Badge tone="info">Central de Assistência</Badge>
          </div>
        }
      />

      <BarraDeFiltros filtros={filtros} onChange={setFiltros} />

      <div role="tablist" aria-label="Dashboard da assistência" className="flex gap-2 border-b border-surface-border">
        <button
          type="button"
          role="tab"
          id="aba-painel"
          aria-selected={aba === "painel"}
          aria-controls="painel-assistencia"
          className={`border-b-2 px-4 py-2 text-sm font-semibold ${aba === "painel" ? "border-primary text-primary" : "border-transparent text-muted"}`}
          onClick={() => setAba("painel")}
        >
          Painel
        </button>
        <button
          type="button"
          role="tab"
          id="aba-graficos"
          aria-selected={aba === "graficos"}
          aria-controls="graficos-assistencia"
          className={`border-b-2 px-4 py-2 text-sm font-semibold ${aba === "graficos" ? "border-primary text-primary" : "border-transparent text-muted"}`}
          onClick={() => setAba("graficos")}
        >
          Gráficos
        </button>
      </div>

      {aba === "painel" ? (
        <div role="tabpanel" id="painel-assistencia" aria-labelledby="aba-painel">
          <PainelDaAssistencia
            editar={editar}
            solicitacoes={solicitacoes}
            totalFila={totalFila}
            veFila={veFila}
            governanca={governanca.data}
            serie={serie.data}
            serieErro={serie.error}
            problemas={listaProblemas}
            mudancas={listaMudancas}
            projetos={listaProjetos}
            modulos={{
              atendimento: atendimentoLigado,
              problemas: enabled("problemas"),
              mudancas: enabled("mudancas"),
              projetos: enabled("projetos"),
            }}
          />
        </div>
      ) : (
        <div role="tabpanel" id="graficos-assistencia" aria-labelledby="aba-graficos">
          <AssistenciaGraficos
            estilo={estilo}
            onEstilo={escolherEstilo}
            incidentes={incidentes}
            requisicoes={requisicoes}
            solicitacoes={solicitacoes}
            problemas={listaProblemas}
            mudancas={listaMudancas}
            projetos={listaProjetos}
            serie={serie.data}
            notaFila={notaFila}
          />
        </div>
      )}

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
