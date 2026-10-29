"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useMemo, useState } from "react";

import { FileSpreadsheet } from "lucide-react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import {
  ListaFaixas,
  NomeClicavel,
  ordenarColunas,
  useAparencia,
} from "@/components/plataforma/Faixas";
import { colunasVisiveis, rotuloQuadro, useQuadro } from "@/components/plataforma/useQuadro";
import { PainelDoQuadro, usePainelNaUrl } from "@/components/plataforma/PainelDoQuadro";
import { ProjetoDetalheConteudo } from "@/app/(protected)/projetos/[id]/page";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { useAction } from "@/components/plataforma/useAction";
import { ProjetosCronograma } from "@/components/projetos/ProjetosCronograma";
import { COLUNAS_PROJETO, ProjetosKanban } from "@/components/projetos/ProjetosKanban";
import { FiltroPorUnidade, filtrarPorUnidade } from "@/components/servicos/FiltroPorUnidade";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import { useActiveEntity } from "@/lib/plataforma/activeEntityStore";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { OrgTree } from "@/lib/plataforma/types";
import {
  ESTADO_PROJETO,
  PRIORIDADE_PROJETO,
  type EstadoProjeto,
  type PrioridadeProjeto,
  type Projeto,
} from "@/lib/projetos/projetos";

type Vista = "lista" | "kanban" | "cronograma";

/** Caminho legal a partir de planejado. O valor enviado é o id do estado. */
function passosDoProjeto(alvo: EstadoProjeto): EstadoProjeto[] {
  switch (alvo) {
    case "planejado":
      return [];
    case "em_andamento":
      return ["em_andamento"];
    case "cancelado":
      return ["cancelado"];
    case "suspenso":
      return ["em_andamento", "suspenso"];
    case "concluido":
      return ["em_andamento", "concluido"];
  }
}

export default function ProjetosPage() {
  return (
    <Suspense fallback={null}>
      <ProjetosConteudo />
    </Suspense>
  );
}

function ProjetosConteudo() {
  const { can, loading: carregandoSessao } = usePlataforma();
  const router = useRouter();
  const params = useSearchParams();
  const painel = usePainelNaUrl();
  const paramVista = params.get("vista");
  const paramProduto = params.get("produto") ?? "";
  const vista: Vista =
    paramVista === "lista" ? "lista" : paramVista === "cronograma" ? "cronograma" : "kanban";
  const gere = can("projetos:manage");
  const le = gere || can("projetos:read");
  const [estado, setEstado] = useState("");
  const [busca, setBusca] = useState("");
  const [unidadeFiltro, setUnidadeFiltro] = useState("todas");
  const somenteFiltroLista = vista === "lista";

  const { run, pending } = useAction();

  const projetos = useApiQuery<Projeto[]>(
    le
      ? withQuery("v1/projetos", {
          estado: somenteFiltroLista ? estado : undefined,
          q: busca || undefined,
          produto_id: paramProduto || undefined,
        })
      : null,
  );

  const [modalAberto, setModalAberto] = useState(false);
  const [titulo, setTitulo] = useState("");
  const [descricao, setDescricao] = useState("");
  const [objetivo, setObjetivo] = useState("");
  const [prioridade, setPrioridade] = useState<PrioridadeProjeto>("media");
  const [gerenteNome, setGerenteNome] = useState("");
  const [orcamento, setOrcamento] = useState("0");
  const [unidadeId, setUnidadeId] = useState("");
  const [produtoId, setProdutoId] = useState("");
  const [quadro, setQuadro] = useState<EstadoProjeto>("planejado");
  const produtos = useApiQuery<{ id: string; nome: string; unidade_id?: string }[]>(
    modalAberto && (can("produtos:manage") || can("produtos:read")) ? "v1/produtos" : null,
  );
  const look = useAparencia("nova-faixas:projetos");
  const catalogo = useQuadro("projetos", COLUNAS_PROJETO);
  const controles = { oficial: catalogo.oficial };

  function trocarVista(proxima: string) {
    const vista =
      proxima === "lista" ? "lista" : proxima === "cronograma" ? "cronograma" : "quadro";
    router.replace(painel.href({ vista }));
  }

  function exportarCSV() {
    if (lista.length === 0) return;
    const cabecalho = [
      "Numero",
      "Titulo",
      "Estado",
      "Prioridade",
      "Gerente",
      "Orcamento",
      "CriadoEm",
    ];
    const linhas = lista.map((p) => [
      p.numero,
      `"${p.titulo.replace(/"/g, '""')}"`,
      ESTADO_PROJETO[p.estado]?.label ?? p.estado,
      PRIORIDADE_PROJETO[p.prioridade]?.label ?? p.prioridade,
      `"${(p.gerente_nome || "").replace(/"/g, '""')}"`,
      Number(p.orcamento) || 0,
      new Date(p.created_at).toLocaleDateString("pt-BR"),
    ]);

    const conteudo = "\uFEFF" + [cabecalho.join(";"), ...linhas.map((l) => l.join(";"))].join("\n");
    const blob = new Blob([conteudo], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.setAttribute(
      "download",
      `relatorio-projetos-${new Date().toISOString().slice(0, 10)}.csv`,
    );
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }

  async function criar(e: FormEvent) {
    e.preventDefault();
    const alvo = quadro;
    const criado = await run(async () => {
      const res = await apiClient.post<Projeto>("v1/projetos", {
        titulo,
        descricao,
        objetivo,
        prioridade,
        gerente_nome: gerenteNome,
        orcamento: Number(orcamento) || 0,
        unidade_id: unidadeId || undefined,
        produto_id: produtoId || undefined,
      });
      for (const passo of passosDoProjeto(alvo)) {
        await apiClient.post(`v1/projetos/${res.data.id}/estado`, { estado: passo });
      }
      return res.data;
    }, "Projeto criado com sucesso");
    if (criado) {
      setModalAberto(false);
      setTitulo("");
      setDescricao("");
      setObjetivo("");
      setGerenteNome("");
      setOrcamento("0");
      setProdutoId("");
      setQuadro("planejado");
      void projetos.mutate();
      router.push(`/projetos/${criado.id}`);
    }
  }

  async function mover(p: Projeto, para: EstadoProjeto) {
    const ok = await run(
      () => apiClient.post(`v1/projetos/${p.id}/estado`, { estado: para }),
      `Projeto #${p.numero}: ${ESTADO_PROJETO[para].label.toLowerCase()}`,
    );
    if (ok) void projetos.mutate();
  }

  async function renomearProjeto(p: Projeto, nome: string) {
    const ok = await run(
      () =>
        apiClient.put(`v1/projetos/${p.id}`, {
          titulo: nome.slice(0, 160),
          descricao: p.descricao,
          objetivo: p.objetivo,
          prioridade: p.prioridade,
          data_inicio_planejada: p.data_inicio_planejada ?? null,
          data_fim_planejada: p.data_fim_planejada ?? null,
          data_inicio_real: p.data_inicio_real ?? null,
          data_fim_real: p.data_fim_real ?? null,
          orcamento: Number(p.orcamento) || 0,
          unidade_id: p.unidade_id ?? null,
          produto_id: p.produto_id ?? null,
          gerente_id: p.gerente_id ?? null,
          gerente_nome: p.gerente_nome,
        }),
      "Nome atualizado",
    );
    if (ok) void projetos.mutate();
  }

  const activeEntity = useActiveEntity();
  const arvore = useApiQuery<OrgTree[]>(le ? "v1/iam/org-tree" : null);
  const unidadeParaEntidade = useMemo(() => {
    const mapa = new Map<string, string>();
    for (const ent of arvore.data ?? []) {
      for (const uni of ent.unidades ?? []) {
        mapa.set(uni.id, ent.id);
      }
    }
    return mapa;
  }, [arvore.data]);

  const projetosFiltrados = useMemo(() => {
    const list = projetos.data ?? [];
    if (!activeEntity) return list;
    return list.filter((p) => {
      if (!p.unidade_id) return true;
      const entId = unidadeParaEntidade.get(p.unidade_id);
      return entId === activeEntity;
    });
  }, [projetos.data, activeEntity, unidadeParaEntidade]);

  const lista = filtrarPorUnidade(projetosFiltrados, unidadeFiltro);
  const colunas = colunasVisiveis(catalogo.colunas, lista, (p) => p.estado);

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Assistência"
        title="Projetos"
        description="Planejamento, acompanhamento de marcos e entregas de projetos estratégicos e operacionais da assistência."
        actions={
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="secondary"
              onClick={exportarCSV}
              disabled={lista.length === 0}
              className="inline-flex items-center gap-1.5"
            >
              <FileSpreadsheet className="h-4 w-4" />
              Exportar CSV
            </Button>
            {gere && (
              <Link href="/projetos/colunas" className={buttonClass("secondary")}>
                Colunas
              </Link>
            )}
            {gere && (
              <Button
                type="button"
                onClick={() => setModalAberto(true)}
                data-testid="novo-projeto-btn"
              >
                Novo projeto
              </Button>
            )}
          </div>
        }
      />

      <SectionTabsInline
        label="Forma de ver os projetos"
        value={vista}
        onChange={trocarVista}
        tabs={[
          { value: "kanban", label: "Quadro" },
          { value: "lista", label: "Lista" },
          { value: "cronograma", label: "Cronograma" },
        ]}
      />

      <FiltroPorUnidade
        rotulo="Filtrar projetos por unidade"
        itens={projetosFiltrados}
        value={unidadeFiltro}
        onChange={setUnidadeFiltro}
      />

      <div className={`grid gap-3 ${somenteFiltroLista ? "sm:grid-cols-2" : "sm:grid-cols-1"}`}>
        {somenteFiltroLista && (
          <Select
            id="projetos-estado"
            label="Estado"
            value={estado}
            onChange={(e) => setEstado(e.target.value)}
            options={[
              { value: "", label: "Todos os estados" },
              ...(Object.keys(ESTADO_PROJETO) as EstadoProjeto[]).map((e) => ({
                value: e,
                label: ESTADO_PROJETO[e].label,
              })),
            ]}
          />
        )}
        <Input
          id="projetos-busca"
          label="Buscar no título ou objetivo"
          value={busca}
          onChange={(e) => setBusca(e.target.value)}
        />
      </div>

      <DataState
        loading={projetos.isLoading || carregandoSessao}
        error={projetos.error}
        onRetry={() => void projetos.mutate()}
        empty={vista === "cronograma" && lista.length === 0}
        emptyTitle="Nenhum projeto encontrado"
        emptyDescription={
          gere
            ? "Crie seu primeiro projeto para acompanhar entregas, marcos e cronogramas."
            : "Nenhum projeto registrado para exibição."
        }
      >
        {lista.length === 0 && vista !== "cronograma" && (
          <p className="text-sm text-muted">Nenhum projeto encontrado</p>
        )}
        {vista === "kanban" ? (
          <ProjetosKanban
            itens={lista}
            onMover={mover}
            pending={pending}
            aoRenomear={gere ? (p, nome) => void renomearProjeto(p, nome) : undefined}
            aoAbrir={(p) => painel.abrir(p.id)}
            colunas={colunas}
            controles={controles}
          />
        ) : vista === "cronograma" ? (
          <ProjetosCronograma projetos={lista} />
        ) : (
          <ListaFaixas
            rotulo="Lista de projetos"
            memoria="nova-faixas:projetos-lista"
            aparencia="nova-faixas:projetos"
            colunas={colunas}
            {...controles}
            itens={lista}
            colunaDe={(p) => p.estado}
            idDe={(p) => p.id}
            nomeDe={(p) => p.titulo}
            aoRenomear={gere ? (p, nome) => void renomearProjeto(p, nome) : undefined}
            aoAbrir={(p) => painel.abrir(p.id)}
            render={(p, vista) => {
              const estadoMeta = ESTADO_PROJETO[p.estado];
              const prioridadeMeta = PRIORIDADE_PROJETO[p.prioridade];
              return (
                <div className="flex flex-col gap-2">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="flex items-center gap-3">
                      <span className="font-mono text-sm font-semibold text-primary">
                        #{p.numero}
                      </span>
                      <NomeClicavel
                        nome={vista.titulo || p.titulo}
                        href={`/projetos/${p.id}`}
                        onSalvar={vista.salvarNome}
                        className="text-left text-base font-semibold text-foreground hover:underline"
                      />
                    </div>
                    <div className="flex items-center gap-2">
                      {prioridadeMeta && (
                        <Badge tone={prioridadeMeta.tone}>{prioridadeMeta.label}</Badge>
                      )}
                      <Badge tone={estadoMeta?.tone ?? "neutral"}>
                        {estadoMeta?.label ?? p.estado}
                      </Badge>
                    </div>
                  </div>
                  {p.descricao && <p className="line-clamp-2 text-sm text-muted">{p.descricao}</p>}
                  <div className="flex flex-wrap items-center gap-4 border-t border-surface-border pt-2 text-xs text-muted">
                    <span>
                      Gerente: <strong>{p.gerente_nome || "Não definido"}</strong>
                    </span>
                    {Number(p.orcamento) > 0 && (
                      <span>
                        Orçamento:{" "}
                        <strong className="font-mono">
                          R${" "}
                          {Number(p.orcamento).toLocaleString("pt-BR", {
                            minimumFractionDigits: 2,
                          })}
                        </strong>
                      </span>
                    )}
                    <span>
                      Criado em:{" "}
                      <strong>{new Date(p.created_at).toLocaleDateString("pt-BR")}</strong>
                    </span>
                    <button
                      type="button"
                      className="font-medium text-primary hover:underline"
                      onClick={() => painel.abrir(p.id)}
                    >
                      Editar
                    </button>
                  </div>
                </div>
              );
            }}
          />
        )}
      </DataState>

      {/* Modal de Criação de Projeto */}
      <Dialog
        open={modalAberto}
        onClose={() => setModalAberto(false)}
        title="Novo Projeto"
        description="O projeto entra no quadro escolhido. O nome da coluna é só o rótulo; o sistema guarda o estado."
        size="full"
      >
        <form onSubmit={criar} className="mt-3 grid gap-3">
          <Input
            id="proj-titulo"
            label="Título do projeto *"
            value={titulo}
            onChange={(e) => setTitulo(e.target.value)}
            placeholder="Ex.: Modernização do Data Center"
            required
            maxLength={160}
          />

          <Textarea
            id="proj-desc"
            label="Descrição"
            value={descricao}
            onChange={(e) => setDescricao(e.target.value)}
            placeholder="Contexto, escopo e premissas gerais..."
            rows={8}
            className="min-h-48"
            maxLength={4000}
          />

          <Textarea
            id="proj-obj"
            label="Objetivo"
            value={objetivo}
            onChange={(e) => setObjetivo(e.target.value)}
            placeholder="Resultados esperados e valor entregue..."
            rows={8}
            className="min-h-48"
            maxLength={4000}
          />

          <Select
            id="proj-quadro"
            label="Quadro"
            value={quadro}
            onChange={(e) => setQuadro(e.target.value as EstadoProjeto)}
            options={ordenarColunas(
              catalogo.colunas.filter((coluna) => coluna.ativa !== false),
            ).map((coluna) => ({
              value: coluna.id,
              label: rotuloQuadro(catalogo.oficial, catalogo.colunas, look.gravado, coluna.id),
            }))}
          />

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Select
              id="proj-prio"
              label="Prioridade"
              value={prioridade}
              onChange={(e) => setPrioridade(e.target.value as PrioridadeProjeto)}
              options={Object.entries(PRIORIDADE_PROJETO).map(([k, v]) => ({
                value: k,
                label: v.label,
              }))}
            />

            <Input
              id="proj-gerente"
              label="Responsável / Gerente"
              value={gerenteNome}
              onChange={(e) => setGerenteNome(e.target.value)}
              placeholder="Nome do gerente"
            />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Input
              id="proj-orc"
              label="Orçamento previsto (R$)"
              type="number"
              step="0.01"
              value={orcamento}
              onChange={(e) => setOrcamento(e.target.value)}
            />

            <UnidadeDonaSelect
              id="proj-unidade"
              label="Unidade"
              permission="projetos:manage"
              defaultValue={unidadeId}
              onChange={(uid) => setUnidadeId(uid)}
            />
          </div>

          {(produtos.data ?? []).length > 0 && (
            <Select
              id="proj-produto"
              label="Produto digital (opcional)"
              value={produtoId}
              onChange={(e) => setProdutoId(e.target.value)}
              options={[
                { value: "", label: "Nenhum produto associado" },
                ...(produtos.data ?? [])
                  .filter((prod) => !unidadeId || !prod.unidade_id || prod.unidade_id === unidadeId)
                  .map((prod) => ({ value: prod.id, label: prod.nome })),
              ]}
            />
          )}

          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setModalAberto(false)}>
              Cancelar
            </Button>
            <Button type="submit" loading={pending}>
              Criar projeto
            </Button>
          </div>
        </form>
      </Dialog>
      <PainelDoQuadro
        aberto={painel.id !== null}
        id={painel.id}
        titulo="Detalhe do projeto"
        href={painel.id ? `/projetos/${painel.id}` : "/projetos"}
        onClose={painel.fechar}
      >
        {painel.id && <ProjetoDetalheConteudo />}
      </PainelDoQuadro>
    </div>
  );
}
