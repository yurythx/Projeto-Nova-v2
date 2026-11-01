"use client";

import { useMemo, useState } from "react";
import {
  AlertTriangle,
  CheckCircle2,
  Flame,
  Plus,
  RefreshCw,
  ShieldAlert,
} from "lucide-react";

import { PageHeader } from "@/components/plataforma/PageHeader";
import { DataState } from "@/components/plataforma/DataState";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import { useEntityFilter } from "@/lib/plataforma/useEntityFilter";
import {
  CONTROLES_ISO27001_SUGESTOES,
  type CategoriaRisco,
  type CriarRiscoInput,
  type EstrategiaRisco,
  type HeatmapItem,
  type NivelSeveridade,
  type Risco,
  ROTULOS_CATEGORIA,
  ROTULOS_ESTRATEGIA,
  ROTULOS_STATUS,
  type StatusRisco,
  riscosApi,
} from "@/lib/riscos/riscos";

const PROBABILIDADE_ROTULOS: Record<number, string> = {
  5: "5 - Quase Certo",
  4: "4 - Provável",
  3: "3 - Possível",
  2: "2 - Remoto",
  1: "1 - Raro",
};

const IMPACTO_ROTULOS: Record<number, string> = {
  1: "1 - Insignificante",
  2: "2 - Menor",
  3: "3 - Moderado",
  4: "4 - Maior",
  5: "5 - Catastrófico",
};

function calcularNivel(prob: number, imp: number): { score: number; nivel: NivelSeveridade } {
  const score = prob * imp;
  if (score >= 20) return { score, nivel: "critico" };
  if (score >= 13) return { score, nivel: "alto" };
  if (score >= 7) return { score, nivel: "medio" };
  return { score, nivel: "baixo" };
}

function badgeSeveridade(nivel: NivelSeveridade, score?: number) {
  const label = score ? `${score} • ${nivel.toUpperCase()}` : nivel.toUpperCase();
  switch (nivel) {
    case "critico":
      return <Badge tone="danger" className="font-semibold">{label}</Badge>;
    case "alto":
      return <Badge tone="warning" className="font-semibold">{label}</Badge>;
    case "medio":
      return <Badge tone="warning" className="font-semibold">{label}</Badge>;
    case "baixo":
      return <Badge tone="success" className="font-semibold">{label}</Badge>;
  }
}

function badgeStatus(status: StatusRisco) {
  switch (status) {
    case "identificado":
      return <Badge tone="neutral">{ROTULOS_STATUS[status]}</Badge>;
    case "em_tratamento":
      return <Badge tone="info">{ROTULOS_STATUS[status]}</Badge>;
    case "mitigado":
      return <Badge tone="success">{ROTULOS_STATUS[status]}</Badge>;
    case "monitorado":
      return <Badge tone="warning">{ROTULOS_STATUS[status]}</Badge>;
    case "aceito":
      return <Badge tone="danger">{ROTULOS_STATUS[status]}</Badge>;
  }
}

export default function RiscosPage() {
  const { can } = usePlataforma();
  const podeGerenciar = can("riscos:manage");
  const { activeEntity } = useEntityFilter();

  const [filtroCategoria, setFiltroCategoria] = useState<string>("");
  const [filtroStatus, setFiltroStatus] = useState<string>("");
  const [filtroSeveridade, setFiltroSeveridade] = useState<string>("");
  const [busca, setBusca] = useState<string>("");
  const [celulaFiltro, setCelulaFiltro] = useState<{ prob: number; imp: number } | null>(null);

  // Modal de cadastro/edição
  const [modalAberto, setModalAberto] = useState(false);
  const [riscoEmEdicao, setRiscoEmEdicao] = useState<Risco | null>(null);
  const [salvando, setSalvando] = useState(false);
  const [erroForm, setErroForm] = useState<string | null>(null);

  // Form states
  const [titulo, setTitulo] = useState("");
  const [descricao, setDescricao] = useState("");
  const [categoria, setCategoria] = useState<CategoriaRisco>("seguranca_informacao");
  const [probabilidade, setProbabilidade] = useState<number>(3);
  const [impacto, setImpacto] = useState<number>(3);
  const [estrategia, setEstrategia] = useState<EstrategiaRisco>("mitigar");
  const [planoMitigacao, setPlanoMitigacao] = useState("");
  const [controleISO, setControleISO] = useState<string>(CONTROLES_ISO27001_SUGESTOES[0] ?? "");
  const [status, setStatus] = useState<StatusRisco>("identificado");
  const [responsavelNome, setResponsavelNome] = useState("");

  const { data: riscos, error, isLoading, mutate } = useApiQuery<Risco[]>("v1/riscos");
  const { data: heatmap } = useApiQuery<HeatmapItem[]>("v1/riscos/heatmap");

  const riscosFiltrados = useMemo(() => {
    if (!riscos) return [];
    return riscos.filter((r) => {
      if (filtroCategoria && r.categoria !== filtroCategoria) return false;
      if (filtroStatus && r.status !== filtroStatus) return false;
      if (filtroSeveridade && r.nivel_severidade !== filtroSeveridade) return false;
      if (celulaFiltro && (r.probabilidade !== celulaFiltro.prob || r.impacto !== celulaFiltro.imp)) {
        return false;
      }
      if (busca) {
        const termo = busca.toLowerCase();
        const noTitulo = r.titulo.toLowerCase().includes(termo);
        const noCod = r.codigo.toLowerCase().includes(termo);
        const noControle = r.controle_iso27001?.toLowerCase().includes(termo);
        if (!noTitulo && !noCod && !noControle) return false;
      }
      return true;
    });
  }, [riscos, filtroCategoria, filtroStatus, filtroSeveridade, celulaFiltro, busca]);

  const kpis = useMemo(() => {
    if (!riscos) return { total: 0, criticos: 0, altos: 0, emTratamento: 0, mitigados: 0 };
    return {
      total: riscos.length,
      criticos: riscos.filter((r) => r.nivel_severidade === "critico").length,
      altos: riscos.filter((r) => r.nivel_severidade === "alto").length,
      emTratamento: riscos.filter((r) => r.status === "em_tratamento").length,
      mitigados: riscos.filter((r) => r.status === "mitigado").length,
    };
  }, [riscos]);

  const abrirModalCriacao = () => {
    setRiscoEmEdicao(null);
    setTitulo("");
    setDescricao("");
    setCategoria("seguranca_informacao");
    setProbabilidade(3);
    setImpacto(3);
    setEstrategia("mitigar");
    setPlanoMitigacao("");
    setControleISO(CONTROLES_ISO27001_SUGESTOES[0] ?? "");
    setStatus("identificado");
    setResponsavelNome("");
    setErroForm(null);
    setModalAberto(true);
  };

  const abrirModalEdicao = (r: Risco) => {
    setRiscoEmEdicao(r);
    setTitulo(r.titulo);
    setDescricao(r.descricao);
    setCategoria(r.categoria);
    setProbabilidade(r.probabilidade);
    setImpacto(r.impacto);
    setEstrategia(r.estrategia);
    setPlanoMitigacao(r.plano_mitigacao);
    setControleISO(r.controle_iso27001 || "");
    setStatus(r.status);
    setResponsavelNome(r.responsavel_nome || "");
    setErroForm(null);
    setModalAberto(true);
  };

  const salvarRisco = async (e: React.FormEvent) => {
    e.preventDefault();
    setSalvando(true);
    setErroForm(null);

    try {
      if (riscoEmEdicao) {
        await riscosApi.atualizar(riscoEmEdicao.id, {
          titulo,
          descricao,
          categoria,
          probabilidade,
          impacto,
          estrategia,
          plano_mitigacao: planoMitigacao,
          controle_iso27001: controleISO,
          status,
          responsavel_nome: responsavelNome,
        });
      } else {
        const payload: CriarRiscoInput = {
          titulo,
          descricao,
          categoria,
          probabilidade,
          impacto,
          estrategia,
          plano_mitigacao: planoMitigacao,
          controle_iso27001: controleISO,
          responsavel_nome: responsavelNome,
          unidade_id: activeEntity && activeEntity !== "all" ? activeEntity : undefined,
        };
        await riscosApi.criar(payload);
      }
      await mutate();
      setModalAberto(false);
    } catch (err: any) {
      setErroForm(err.message || "Erro ao salvar risco. Verifique os dados.");
    } finally {
      setSalvando(false);
    }
  };

  const matrixCalc = calcularNivel(probabilidade, impacto);

  // Mapa de contagem do heatmap
  const mapaHeatmap = useMemo(() => {
    const mapa = new Map<string, number>();
    if (heatmap) {
      for (const item of heatmap) {
        mapa.set(`${item.probabilidade}-${item.impacto}`, item.total_riscos);
      }
    }
    return mapa;
  }, [heatmap]);

  const opcoesCategoria = useMemo(
    () =>
      Object.entries(ROTULOS_CATEGORIA).map(([value, label]) => ({
        value,
        label,
      })),
    [],
  );

  const opcoesStatus = useMemo(
    () =>
      Object.entries(ROTULOS_STATUS).map(([value, label]) => ({
        value,
        label,
      })),
    [],
  );

  const opcoesSeveridade = useMemo(
    () => [
      { value: "critico", label: "Crítico (20 a 25)" },
      { value: "alto", label: "Alto (13 a 19)" },
      { value: "medio", label: "Médio (7 a 12)" },
      { value: "baixo", label: "Baixo (1 a 6)" },
    ],
    [],
  );

  const opcoesEstrategia = useMemo(
    () =>
      Object.entries(ROTULOS_ESTRATEGIA).map(([value, label]) => ({
        value,
        label,
      })),
    [],
  );

  const opcoesProbabilidade = useMemo(
    () =>
      [1, 2, 3, 4, 5].map((val) => ({
        value: val.toString(),
        label: PROBABILIDADE_ROTULOS[val] ?? String(val),
      })),
    [],
  );

  const opcoesImpacto = useMemo(
    () =>
      [1, 2, 3, 4, 5].map((val) => ({
        value: val.toString(),
        label: IMPACTO_ROTULOS[val] ?? String(val),
      })),
    [],
  );

  return (
    <div className="space-y-6">
      <PageHeader
        title="Matriz de Riscos (ISO/IEC 27001)"
        description="Gestão de Riscos de TI, Continuidade Operacional, Segurança da Informação e Conformidade LGPD no Padrão GRC."
        actions={
          podeGerenciar && (
            <Button onClick={abrirModalCriacao}>
              <Plus size={16} className="mr-1 inline-block" />
              Novo Risco
            </Button>
          )
        }
      />

      {/* Cards de KPIs */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
        <Card className="border-l-4 border-l-primary">
          <CardHeader className="pb-1">
            <CardTitle className="text-xs uppercase tracking-wider text-muted-foreground">
              Total de Riscos
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{kpis.total}</div>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-red-600 bg-red-50/20 dark:bg-red-950/20">
          <CardHeader className="pb-1">
            <CardTitle className="text-xs uppercase tracking-wider text-red-600 font-semibold flex items-center gap-1">
              <Flame size={14} /> Críticos (Score 20-25)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{kpis.criticos}</div>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-orange-500">
          <CardHeader className="pb-1">
            <CardTitle className="text-xs uppercase tracking-wider text-orange-600 font-semibold flex items-center gap-1">
              <AlertTriangle size={14} /> Altos (Score 13-19)
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-orange-600">{kpis.altos}</div>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-blue-500">
          <CardHeader className="pb-1">
            <CardTitle className="text-xs uppercase tracking-wider text-blue-600 font-semibold flex items-center gap-1">
              <RefreshCw size={14} /> Em Tratamento
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-blue-600">{kpis.emTratamento}</div>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-emerald-600">
          <CardHeader className="pb-1">
            <CardTitle className="text-xs uppercase tracking-wider text-emerald-600 font-semibold flex items-center gap-1">
              <CheckCircle2 size={14} /> Mitigados
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-emerald-600">{kpis.mitigados}</div>
          </CardContent>
        </Card>
      </div>

      {/* Heatmap 5x5 Interativo */}
      <Card>
        <CardHeader className="pb-2">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-2">
            <div>
              <CardTitle className="text-lg flex items-center gap-2">
                <ShieldAlert className="text-primary" size={20} />
                Heatmap de Severidade (Probabilidade × Impacto)
              </CardTitle>
              <p className="text-xs text-muted-foreground mt-0.5">
                Padrão internacional ISO/IEC 27005. Clique em uma célula para filtrar os riscos daquele quadrante.
              </p>
            </div>
            {celulaFiltro && (
              <Button
                variant="secondary"
                size="sm"
                onClick={() => setCelulaFiltro(null)}
                className="text-xs"
              >
                Limpar filtro da célula ({celulaFiltro.prob}×{celulaFiltro.imp})
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <div className="min-w-[650px] p-2">
              <div className="grid grid-cols-[140px_repeat(5,1fr)] gap-1 text-center font-medium text-xs">
                {/* Linha de cabeçalho dos Impactos */}
                <div className="flex items-center justify-center font-bold text-muted-foreground">
                  Probabilidade \ Impacto
                </div>
                {[1, 2, 3, 4, 5].map((imp) => (
                  <div key={imp} className="p-1 bg-muted/60 rounded font-semibold text-muted-foreground">
                    {IMPACTO_ROTULOS[imp]}
                  </div>
                ))}

                {/* Linhas da matriz (Probabilidade de 5 até 1) */}
                {[5, 4, 3, 2, 1].map((prob) => (
                  <div key={prob} className="contents">
                    <div className="flex items-center justify-end pr-2 font-semibold text-xs text-muted-foreground">
                      {PROBABILIDADE_ROTULOS[prob]}
                    </div>
                    {[1, 2, 3, 4, 5].map((imp) => {
                      const { score, nivel } = calcularNivel(prob, imp);
                      const contagem = mapaHeatmap.get(`${prob}-${imp}`) || 0;
                      const selecionado =
                        celulaFiltro?.prob === prob && celulaFiltro?.imp === imp;

                      let bgClass = "bg-emerald-50 dark:bg-emerald-950/40 text-emerald-800 border-emerald-200";
                      if (nivel === "critico") {
                        bgClass = "bg-red-500 text-white border-red-600";
                      } else if (nivel === "alto") {
                        bgClass = "bg-orange-400 text-white border-orange-500";
                      } else if (nivel === "medio") {
                        bgClass = "bg-amber-200 text-amber-950 border-amber-300";
                      }

                      return (
                        <button
                          key={`${prob}-${imp}`}
                          type="button"
                          onClick={() =>
                            setCelulaFiltro(
                              selecionado ? null : { prob, imp }
                            )
                          }
                          className={`h-16 rounded border transition-all flex flex-col items-center justify-center relative cursor-pointer hover:opacity-90 ${bgClass} ${
                            selecionado ? "ring-4 ring-primary scale-105 z-10 shadow-lg" : ""
                          }`}
                        >
                          <span className="text-[10px] opacity-75 font-mono">Score {score}</span>
                          <span className="text-lg font-bold">
                            {contagem > 0 ? contagem : "—"}
                          </span>
                          {contagem > 0 && (
                            <span className="text-[10px] font-semibold tracking-tighter uppercase">
                              {contagem === 1 ? "risco" : "riscos"}
                            </span>
                          )}
                        </button>
                      );
                    })}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Barra de Filtros e Busca */}
      <Card>
        <CardContent className="pt-4">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
            <div>
              <Input
                placeholder="Buscar por código, título ou controle..."
                value={busca}
                onChange={(e) => setBusca(e.target.value)}
              />
            </div>

            <div>
              <Select
                value={filtroCategoria}
                onChange={(e) => setFiltroCategoria(e.target.value)}
                placeholder="Todas as Categorias"
                options={opcoesCategoria}
              />
            </div>

            <div>
              <Select
                value={filtroStatus}
                onChange={(e) => setFiltroStatus(e.target.value)}
                placeholder="Todos os Status"
                options={opcoesStatus}
              />
            </div>

            <div>
              <Select
                value={filtroSeveridade}
                onChange={(e) => setFiltroSeveridade(e.target.value)}
                placeholder="Todas as Severidades"
                options={opcoesSeveridade}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tabela de Riscos */}
      <DataState
        loading={isLoading}
        error={error}
        empty={riscosFiltrados.length === 0}
        emptyTitle="Nenhum risco encontrado"
        emptyDescription="Nenhum risco corresponde aos filtros selecionados ou ainda não há riscos cadastrados para esta unidade."
      >
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-muted/50 border-b text-xs uppercase text-muted-foreground font-semibold">
                <tr>
                  <th className="py-3 px-4">Código</th>
                  <th className="py-3 px-4">Título & Descrição</th>
                  <th className="py-3 px-4">Categoria</th>
                  <th className="py-3 px-4">Severidade</th>
                  <th className="py-3 px-4">Controle ISO 27001</th>
                  <th className="py-3 px-4">Estratégia</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4 text-right">Ações</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {riscosFiltrados.map((r) => (
                  <tr key={r.id} className="hover:bg-muted/30 transition-colors">
                    <td className="py-3 px-4 font-mono font-bold text-primary">
                      {r.codigo}
                    </td>
                    <td className="py-3 px-4 max-w-xs">
                      <div className="font-semibold text-foreground">{r.titulo}</div>
                      <div className="text-xs text-muted-foreground line-clamp-2">
                        {r.descricao}
                      </div>
                    </td>
                    <td className="py-3 px-4">
                      <Badge tone="neutral" className="text-xs">
                        {ROTULOS_CATEGORIA[r.categoria]}
                      </Badge>
                    </td>
                    <td className="py-3 px-4">
                      {badgeSeveridade(r.nivel_severidade, r.severidade)}
                      <div className="text-[11px] text-muted-foreground mt-0.5">
                        P: {r.probabilidade} × I: {r.impacto}
                      </div>
                    </td>
                    <td className="py-3 px-4 max-w-[200px]">
                      {r.controle_iso27001 ? (
                        <div className="text-xs font-mono text-muted-foreground truncate" title={r.controle_iso27001}>
                          {r.controle_iso27001}
                        </div>
                      ) : (
                        <span className="text-xs text-muted-foreground italic">—</span>
                      )}
                    </td>
                    <td className="py-3 px-4">
                      <span className="text-xs font-medium">
                        {ROTULOS_ESTRATEGIA[r.estrategia]}
                      </span>
                    </td>
                    <td className="py-3 px-4">
                      {badgeStatus(r.status)}
                    </td>
                    <td className="py-3 px-4 text-right">
                      {podeGerenciar && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => abrirModalEdicao(r)}
                        >
                          Gerenciar
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      </DataState>

      {/* Modal de Criação / Edição de Risco */}
      <Dialog
        open={modalAberto}
        onClose={() => setModalAberto(false)}
        title={riscoEmEdicao ? `Gerenciar Risco ${riscoEmEdicao.codigo}` : "Cadastrar Novo Risco (ISO 27001)"}
      >
        <form onSubmit={salvarRisco} className="space-y-4">
          {erroForm && (
            <div className="p-3 bg-red-100 text-red-700 text-xs rounded border border-red-200">
              {erroForm}
            </div>
          )}

          <div>
            <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground block mb-1">
              Título do Risco *
            </label>
            <Input
              required
              placeholder="Ex: Falha Crítica no Link de Fibra Óptica do Data Center"
              value={titulo}
              onChange={(e) => setTitulo(e.target.value)}
            />
          </div>

          <div>
            <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground block mb-1">
              Descrição do Cenário de Ameaça e Vulnerabilidade *
            </label>
            <Textarea
              required
              rows={3}
              placeholder="Descreva a causa raiz, os ativos afetados e o impacto operacional esperado..."
              value={descricao}
              onChange={(e) => setDescricao(e.target.value)}
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground block mb-1">
                Categoria do Risco *
              </label>
              <Select
                value={categoria}
                onChange={(e) => setCategoria(e.target.value as CategoriaRisco)}
                options={opcoesCategoria}
              />
            </div>

            {riscoEmEdicao && (
              <div>
                <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground block mb-1">
                  Status Atual *
                </label>
                <Select
                  value={status}
                  onChange={(e) => setStatus(e.target.value as StatusRisco)}
                  options={opcoesStatus}
                />
              </div>
            )}
          </div>

          {/* Matriz 5x5: Probabilidade e Impacto */}
          <div className="p-3 bg-muted/40 rounded-lg border space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-bold uppercase tracking-wider">
                Avaliação de Severidade (ISO 27005)
              </span>
              <div className="flex items-center gap-2">
                <span className="text-xs text-muted-foreground">Resultado:</span>
                {badgeSeveridade(matrixCalc.nivel, matrixCalc.score)}
              </div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div>
                <label className="text-xs font-medium block mb-1">
                  Probabilidade (1 a 5)
                </label>
                <Select
                  value={probabilidade.toString()}
                  onChange={(e) => setProbabilidade(parseInt(e.target.value, 10))}
                  options={opcoesProbabilidade}
                />
              </div>

              <div>
                <label className="text-xs font-medium block mb-1">
                  Impacto (1 a 5)
                </label>
                <Select
                  value={impacto.toString()}
                  onChange={(e) => setImpacto(parseInt(e.target.value, 10))}
                  options={opcoesImpacto}
                />
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground block mb-1">
                Estratégia de Tratamento *
              </label>
              <Select
                value={estrategia}
                onChange={(e) => setEstrategia(e.target.value as EstrategiaRisco)}
                options={opcoesEstrategia}
              />
            </div>

            <div>
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground block mb-1">
                Responsável Técnico
              </label>
              <Input
                placeholder="Ex: Equipe de Redes / CISO"
                value={responsavelNome}
                onChange={(e) => setResponsavelNome(e.target.value)}
              />
            </div>
          </div>

          <div>
            <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground block mb-1">
              Controle ISO/IEC 27001 Vinculado
            </label>
            <Input
              list="sugestoes-controles-iso"
              placeholder="Ex: A.12.3 - Backup e Retenção Segura de Dados"
              value={controleISO}
              onChange={(e) => setControleISO(e.target.value)}
            />
            <datalist id="sugestoes-controles-iso">
              {CONTROLES_ISO27001_SUGESTOES.map((ctrl) => (
                <option key={ctrl} value={ctrl} />
              ))}
            </datalist>
          </div>

          <div>
            <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground block mb-1">
              Plano de Mitigação / Ação Preventiva *
            </label>
            <Textarea
              required
              rows={3}
              placeholder="Descreva as medidas corretivas, contingências e plano de ação aprovado pelo comitê de governança..."
              value={planoMitigacao}
              onChange={(e) => setPlanoMitigacao(e.target.value)}
            />
          </div>

          <div className="flex justify-end gap-2 pt-3 border-t">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setModalAberto(false)}
            >
              Cancelar
            </Button>
            <Button type="submit" disabled={salvando}>
              {salvando ? "Salvando..." : riscoEmEdicao ? "Atualizar Risco" : "Registrar Risco"}
            </Button>
          </div>
        </form>
      </Dialog>
    </div>
  );
}
