"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useMemo, useState } from "react";

import { FileSpreadsheet, Globe, Plus } from "lucide-react";

import { FichaConteudo } from "@/app/(protected)/protocolos/[id]/page";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { PainelDoQuadro, usePainelNaUrl } from "@/components/plataforma/PainelDoQuadro";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { FaixaIndicadores, IndicadoresDaMesa } from "@/components/protocolos/Indicadores";
import { filtrarPedidos, PRIORIDADES, STATUS } from "@/components/protocolos/painel";
import { ProtocolosLista, ProtocolosQuadro } from "@/components/protocolos/ProtocolosQuadro";
import { csvDosPedidos, type PedidoResumo } from "@/components/protocolos/quadro";
import { FiltroPorUnidade, filtrarPorUnidade } from "@/components/servicos/FiltroPorUnidade";
import { Button, buttonClass } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

type Vista = "quadro" | "lista" | "indicadores";

export default function ProtocolosPage() {
  return (
    <Suspense fallback={null}>
      <MesaDoProtocolo />
    </Suspense>
  );
}

function MesaDoProtocolo() {
  const { can } = usePlataforma();
  const router = useRouter();
  const params = useSearchParams();
  const painel = usePainelNaUrl();
  const paramVista = params.get("vista");
  const vista: Vista =
    paramVista === "lista" ? "lista" : paramVista === "indicadores" ? "indicadores" : "quadro";
  const gere = can("protocolos:gerir");
  const abre = can("protocolos:abrir");

  const lista = useApiQuery<PedidoResumo[]>("v1/protocolos/pedidos");
  const pedidos = useMemo(() => lista.data ?? [], [lista.data]);

  const [prazo, setPrazo] = useState("todos");
  const [status, setStatus] = useState("");
  const [busca, setBusca] = useState("");
  const [servico, setServico] = useState("");
  const [prioridade, setPrioridade] = useState("");
  const [unidade, setUnidade] = useState("todas");

  const servicos = useMemo(() => {
    const mapa = new Map<string, string>();
    for (const p of pedidos) mapa.set(p.servico_id, p.servico_nome);
    return [...mapa.entries()]
      .map(([value, label]) => ({ value, label }))
      .sort((a, b) => a.label.localeCompare(b.label, "pt-BR"));
  }, [pedidos]);

  const daUnidade = useMemo(() => filtrarPorUnidade(pedidos, unidade), [pedidos, unidade]);
  const visiveis = useMemo(
    () =>
      filtrarPedidos(daUnidade, {
        prazo,
        busca,
        servico,
        prioridade,
        // No quadro o status é a própria coluna; o filtro de status vale na lista e nos cartões de cima.
        status: vista === "quadro" && status !== "em_complementacao" ? "" : status,
      }),
    [daUnidade, prazo, busca, servico, status, prioridade, vista],
  );

  function trocarVista(proxima: Vista) {
    router.replace(painel.href({ vista: proxima === "quadro" ? null : proxima }));
  }

  function exportarCSV() {
    if (visiveis.length === 0) return;
    const blob = new Blob([csvDosPedidos(visiveis)], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `protocolos-${new Date().toISOString().slice(0, 10)}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }

  const abrir = (p: PedidoResumo) => painel.abrir(p.id);
  const filtrando =
    prazo !== "todos" ||
    status !== "" ||
    busca !== "" ||
    servico !== "" ||
    prioridade !== "" ||
    unidade !== "todas";

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Protocolo digital"
        title="Mesa do protocolo"
        description="Pedidos da carta de serviços que a sua unidade trata, com etapa, prazo e prioridade."
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              variant="secondary"
              onClick={exportarCSV}
              disabled={visiveis.length === 0}
              className="inline-flex items-center gap-1.5"
            >
              <FileSpreadsheet className="h-4 w-4" aria-hidden />
              Exportar CSV
            </Button>
            <Link
              href="/carta"
              className={`${buttonClass("secondary")} inline-flex items-center gap-1.5`}
            >
              <Globe className="h-4 w-4" aria-hidden />
              Carta pública
            </Link>
            {gere && (
              <>
                <Link href="/protocolos/servicos" className={buttonClass("secondary")}>
                  Serviços
                </Link>
                <Link href="/protocolos/formularios" className={buttonClass("secondary")}>
                  Formulários
                </Link>
                <Link href="/protocolos/responsaveis" className={buttonClass("secondary")}>
                  Responsáveis
                </Link>
              </>
            )}
            {abre && (
              <Link
                href="/protocolos/balcao"
                className={`${buttonClass("primary")} inline-flex items-center gap-1.5`}
              >
                <Plus className="h-4 w-4" aria-hidden />
                Novo pedido no balcão
              </Link>
            )}
          </div>
        }
      />

      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={pedidos.length === 0}
        emptyTitle="Nenhum pedido na sua mesa"
        emptyDescription="Os pedidos abertos pela carta pública ou no balcão aparecem aqui."
      >
        <FaixaIndicadores
          pedidos={daUnidade}
          filtro={{ prazo, status }}
          onFiltrar={(proximo) => {
            setPrazo(proximo.prazo);
            setStatus(proximo.status);
          }}
        />

        <SectionTabsInline
          label="Forma de ver os pedidos"
          value={vista}
          onChange={trocarVista}
          tabs={[
            { value: "quadro", label: "Quadro" },
            { value: "lista", label: "Lista" },
            { value: "indicadores", label: "Indicadores" },
          ]}
        />

        <FiltroPorUnidade
          rotulo="Filtrar pedidos por unidade"
          itens={pedidos}
          value={unidade}
          onChange={setUnidade}
        />

        <div
          className={`grid gap-3 sm:grid-cols-2 ${vista === "lista" ? "xl:grid-cols-4" : "xl:grid-cols-3"}`}
        >
          <Input
            id="busca-protocolo"
            label="Buscar"
            placeholder="Número, nome, serviço ou CPF"
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
          />
          <Select
            id="filtro-servico"
            label="Serviço"
            placeholder="Todos os serviços"
            value={servico}
            onChange={(e) => setServico(e.target.value)}
            options={servicos}
          />
          <Select
            id="filtro-prioridade"
            label="Prioridade"
            placeholder="Todas as prioridades"
            value={prioridade}
            onChange={(e) => setPrioridade(e.target.value)}
            options={PRIORIDADES}
          />
          {vista === "lista" && (
            <Select
              id="filtro-status"
              label="Status"
              placeholder="Todos os status"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              options={STATUS}
            />
          )}
        </div>

        {filtrando && (
          <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-muted">
            <span>
              {visiveis.length} de {pedidos.length} pedidos com os filtros atuais.
            </span>
            <button
              type="button"
              className="font-medium text-primary hover:underline"
              onClick={() => {
                setPrazo("todos");
                setStatus("");
                setBusca("");
                setServico("");
                setPrioridade("");
                setUnidade("todas");
              }}
            >
              Limpar filtros
            </button>
          </div>
        )}

        {vista === "indicadores" ? (
          <IndicadoresDaMesa pedidos={visiveis} />
        ) : visiveis.length === 0 ? (
          <p className="text-sm text-muted">Nenhum pedido com esses filtros.</p>
        ) : vista === "lista" ? (
          <ProtocolosLista itens={visiveis} aoAbrir={abrir} />
        ) : (
          <ProtocolosQuadro itens={visiveis} aoAbrir={abrir} />
        )}
      </DataState>

      <PainelDoQuadro
        aberto={painel.id !== null}
        id={painel.id}
        titulo="Detalhe do pedido"
        href={painel.id ? `/protocolos/${painel.id}` : "/protocolos"}
        onClose={() => {
          painel.fechar();
          void lista.mutate();
        }}
        folha
      >
        {painel.id && <FichaConteudo />}
      </PainelDoQuadro>
    </div>
  );
}
