"use client";

import { ArrowLeft, Clock, Link2, Plus, Printer, Trash2 } from "lucide-react";
import Link from "next/link";
import { useIdDaRota, usePainelEmbutido } from "@/components/plataforma/PainelDoQuadro";
import { type FormEvent, Suspense, useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import {
  NomeClicavel,
  Quadro,
  colunasEmOrdem,
  ordenarColunas,
  seloDoEstado,
  useAparencia,
  type ColunaFaixa,
} from "@/components/plataforma/Faixas";
import { colunasVisiveis, rotuloQuadro, useQuadro } from "@/components/plataforma/useQuadro";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { ProjetoGanttChart } from "@/components/projetos/ProjetoGanttChart";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import type { Solicitacao } from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import { useEntityFilter } from "@/lib/plataforma/useEntityFilter";
import type { Ativo, Mudanca, Problema } from "@/lib/servicos/servicos";
import {
  ESTADO_PROJETO,
  ESTADO_TAREFA,
  PRIORIDADE_PROJETO,
  TIPO_VINCULO_PROJETO,
  type ApontamentoHora,
  type EstadoProjeto,
  type EstadoTarefa,
  type ListarApontamentosResponse,
  type Projeto,
  type Tarefa,
  type TipoVinculoProjeto,
  type VinculoProjeto,
} from "@/lib/projetos/projetos";

export default function ProjetoDetalhePage() {
  return (
    <Suspense fallback={null}>
      <ProjetoDetalheConteudo />
    </Suspense>
  );
}

export function ProjetoDetalheConteudo() {
  const id = useIdDaRota();
  const embutido = usePainelEmbutido();
  const { can } = usePlataforma();
  const gere = can("projetos:manage");
  const le = gere || can("projetos:read");
  const { filtrarPorEntidade } = useEntityFilter();

  const { run, pending } = useAction();
  const lookTarefas = useAparencia(id ? `nova-faixas:tarefas:${id}` : "nova-faixas:tarefas:sem");
  const fabricaTarefa: ColunaFaixa[] = colunasEmOrdem(
    (Object.keys(ESTADO_TAREFA) as EstadoTarefa[]).map((estado) => ({
      id: estado,
      titulo: ESTADO_TAREFA[estado].label,
      tom: ESTADO_TAREFA[estado].tone,
    })),
  );
  const catalogoTarefas = useQuadro("tarefas", fabricaTarefa);
  const controlesTarefas = { oficial: catalogoTarefas.oficial };

  const projeto = useApiQuery<Projeto>(id && le ? `v1/projetos/${id}` : null);
  const tarefas = useApiQuery<Tarefa[]>(id && le ? `v1/projetos/${id}/tarefas` : null);
  const vinculos = useApiQuery<VinculoProjeto[]>(id && le ? `v1/projetos/${id}/vinculos` : null);
  const apontamentos = useApiQuery<ListarApontamentosResponse>(
    id && le ? `v1/projetos/${id}/apontamentos` : null,
  );

  const p = projeto.data;
  const listaTarefas = tarefas.data ?? [];
  const colunasTarefa = colunasVisiveis(catalogoTarefas.colunas, listaTarefas, (t) => t.estado);
  const listaVinculos = vinculos.data ?? [];
  const listaApontamentos = apontamentos.data?.items ?? [];
  const resumoHoras = apontamentos.data?.resumo ?? {
    total_horas: 0,
    total_custo: 0,
    por_usuario: {},
  };

  // Estados para nova tarefa
  const [modalTarefa, setModalTarefa] = useState(false);
  const [tarefaTitulo, setTarefaTitulo] = useState("");
  const [tarefaDescricao, setTarefaDescricao] = useState("");
  const [tarefaResponsavel, setTarefaResponsavel] = useState("");
  const [tarefaEstado, setTarefaEstado] = useState<EstadoTarefa>("a_fazer");

  // Estados para novo vínculo
  const [modalVinculo, setModalVinculo] = useState(false);
  const [vinculoTipo, setVinculoTipo] = useState<TipoVinculoProjeto>("mudanca");
  const [vinculoAlvoId, setVinculoAlvoId] = useState("");
  const [vinculoRotulo, setVinculoRotulo] = useState("");

  const mudancasOpcoes = useApiQuery<Mudanca[]>(
    modalVinculo &&
      vinculoTipo === "mudanca" &&
      (can("mudancas:manage") || can("mudancas:aprovar"))
      ? "v1/mudancas"
      : null,
  );
  const problemasOpcoes = useApiQuery<Problema[]>(
    modalVinculo &&
      vinculoTipo === "problema" &&
      (can("problemas:manage") || can("problemas:read"))
      ? "v1/problemas"
      : null,
  );
  const ativosOpcoes = useApiQuery<Ativo[]>(
    modalVinculo && vinculoTipo === "ativo" && (can("ativos:manage") || can("ativos:read"))
      ? "v1/ativos"
      : null,
  );
  const solicitacoesOpcoes = useApiPage<Solicitacao>(
    modalVinculo && vinculoTipo === "solicitacao" && can("atendimento:atender")
      ? withQuery("v1/atendimento/fila", { situacao: "todas", page_size: 100 })
      : null,
  );
  const jaVinculados = new Set(
    listaVinculos.filter((v) => v.tipo === vinculoTipo).map((v) => v.alvo_id),
  );
  const opcoesCatalogoVinculo = (
    vinculoTipo === "mudanca"
      ? filtrarPorEntidade(mudancasOpcoes.data ?? []).map((m) => ({
          value: m.id,
          label: `Mudança #${m.numero} · ${m.titulo}`,
        }))
      : vinculoTipo === "problema"
        ? filtrarPorEntidade(problemasOpcoes.data ?? []).map((pr) => ({
            value: pr.id,
            label: `Problema #${pr.numero} · ${pr.titulo}`,
          }))
        : vinculoTipo === "ativo"
          ? filtrarPorEntidade(ativosOpcoes.data ?? []).map((a) => ({
              value: a.id,
              label: `Ativo · ${a.nome}`,
            }))
          : filtrarPorEntidade(solicitacoesOpcoes.data?.items ?? []).map((s) => ({
              value: s.id,
              label: `Solicitação #${s.numero} · ${s.titulo}`,
            }))
  ).filter((op) => !jaVinculados.has(op.value));

  // Estados para novo apontamento de horas
  const [modalApontamento, setModalApontamento] = useState(false);
  const [apontamentoTarefaId, setApontamentoTarefaId] = useState("");
  const [apontamentoData, setApontamentoData] = useState(() =>
    new Date().toISOString().slice(0, 10),
  );
  const [apontamentoHoras, setApontamentoHoras] = useState("2");
  const [apontamentoCustoHora, setApontamentoCustoHora] = useState("");
  const [apontamentoDescricao, setApontamentoDescricao] = useState("");

  async function criarApontamento(e: FormEvent) {
    e.preventDefault();
    const horasNum = parseFloat(apontamentoHoras);
    const custoNum = apontamentoCustoHora ? parseFloat(apontamentoCustoHora) : 0;
    const ok = await run(
      () =>
        apiClient.post(`v1/projetos/${id}/apontamentos`, {
          tarefa_id: apontamentoTarefaId || null,
          data_apontamento: apontamentoData,
          horas: horasNum,
          custo_hora: custoNum,
          descricao: apontamentoDescricao.trim(),
        }),
      "Horas registradas com sucesso",
    );
    if (ok) {
      setModalApontamento(false);
      setApontamentoDescricao("");
      setApontamentoHoras("2");
      void apontamentos.mutate();
    }
  }

  async function excluirApontamento(ap: ApontamentoHora) {
    const ok = await run(
      () => apiClient.delete(`v1/projetos/${id}/apontamentos/${ap.id}`),
      "Apontamento excluído",
    );
    if (ok) void apontamentos.mutate();
  }

  async function mudarEstado(novoEstado: EstadoProjeto) {
    const ok = await run(
      () => apiClient.post(`v1/projetos/${id}/estado`, { estado: novoEstado }),
      `Projeto atualizado para ${ESTADO_PROJETO[novoEstado].label.toLowerCase()}`,
    );
    if (ok) void projeto.mutate();
  }

  async function criarTarefa(e: FormEvent) {
    e.preventDefault();
    const ok = await run(
      () =>
        apiClient.post(`v1/projetos/${id}/tarefas`, {
          titulo: tarefaTitulo,
          descricao: tarefaDescricao,
          responsavel_nome: tarefaResponsavel,
          estado: tarefaEstado,
          ordem: listaTarefas.length + 1,
        }),
      "Tarefa adicionada",
    );
    if (ok) {
      setModalTarefa(false);
      setTarefaTitulo("");
      setTarefaDescricao("");
      setTarefaResponsavel("");
      setTarefaEstado("a_fazer");
      void tarefas.mutate();
    }
  }

  async function atualizarTarefaEstado(t: Tarefa, novoEstado: EstadoTarefa) {
    const ok = await run(
      () =>
        apiClient.put(`v1/projetos/${id}/tarefas/${t.id}`, {
          titulo: t.titulo,
          descricao: t.descricao,
          responsavel_nome: t.responsavel_nome,
          estado: novoEstado,
          ordem: t.ordem,
        }),
      `Tarefa ${t.titulo}: ${seloDoEstado(catalogoTarefas.colunas, novoEstado, ESTADO_TAREFA[novoEstado]).label.toLowerCase()}`,
    );
    if (ok) void tarefas.mutate();
  }

  async function renomearTarefa(t: Tarefa, nome: string) {
    const ok = await run(
      () =>
        apiClient.put(`v1/projetos/${id}/tarefas/${t.id}`, {
          titulo: nome.slice(0, 200),
          descricao: t.descricao,
          responsavel_nome: t.responsavel_nome,
          estado: t.estado,
          ordem: t.ordem,
        }),
      "Nome atualizado",
    );
    if (ok) void tarefas.mutate();
  }

  async function excluirTarefa(t: Tarefa) {
    const ok = await run(
      () => apiClient.delete(`v1/projetos/${id}/tarefas/${t.id}`),
      `Tarefa removida`,
    );
    if (ok) void tarefas.mutate();
  }

  async function criarVinculo(e: FormEvent) {
    e.preventDefault();
    if (!vinculoAlvoId.trim()) return;
    const ok = await run(
      () =>
        apiClient.post(`v1/projetos/${id}/vinculos`, {
          tipo: vinculoTipo,
          alvo_id: vinculoAlvoId.trim(),
          rotulo: vinculoRotulo.trim(),
        }),
      "Item vinculado com sucesso",
    );
    if (ok) {
      setModalVinculo(false);
      setVinculoAlvoId("");
      setVinculoRotulo("");
      void vinculos.mutate();
    }
  }

  async function excluirVinculo(v: VinculoProjeto) {
    const ok = await run(
      () => apiClient.delete(`v1/projetos/${id}/vinculos/${v.id}`),
      "Vínculo removido",
    );
    if (ok) void vinculos.mutate();
  }

  const estadoMeta = p ? ESTADO_PROJETO[p.estado] : null;
  const prioridadeMeta = p ? PRIORIDADE_PROJETO[p.prioridade] : null;

  return (
    <div className="flex flex-col gap-6">
      {!embutido && (
        <div>
          <Link
            href="/projetos"
            className="inline-flex items-center gap-1.5 text-xs text-muted hover:text-foreground"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Voltar para Projetos
          </Link>
        </div>
      )}

      <DataState
        loading={projeto.isLoading}
        error={projeto.error}
        onRetry={() => void projeto.mutate()}
        empty={!p}
        emptyTitle="Projeto não encontrado"
      >
        {p && (
          <>
            <PageHeader
              eyebrow="Assistência"
              title={`Projeto #${p.numero}: ${p.titulo}`}
              description={p.descricao || "Sem descrição informada."}
              actions={
                <div className="flex flex-wrap items-center gap-2">
                  <Button
                    size="sm"
                    variant="secondary"
                    type="button"
                    onClick={() => window.print()}
                    className="inline-flex items-center gap-1.5"
                  >
                    <Printer className="h-4 w-4" />
                    Imprimir Ficha
                  </Button>
                  {prioridadeMeta && (
                    <Badge tone={prioridadeMeta.tone}>{prioridadeMeta.label}</Badge>
                  )}
                  {estadoMeta && <Badge tone={estadoMeta.tone}>{estadoMeta.label}</Badge>}

                  {gere && p.estado !== "cancelado" && (
                    <div className="flex items-center gap-1.5 ml-2">
                      {p.estado === "planejado" && (
                        <Button
                          size="sm"
                          disabled={pending}
                          onClick={() => void mudarEstado("em_andamento")}
                        >
                          Iniciar Projeto
                        </Button>
                      )}
                      {p.estado === "em_andamento" && (
                        <>
                          <Button
                            size="sm"
                            variant="secondary"
                            disabled={pending}
                            onClick={() => void mudarEstado("suspenso")}
                          >
                            Suspender
                          </Button>
                          <Button
                            size="sm"
                            disabled={pending}
                            onClick={() => void mudarEstado("concluido")}
                          >
                            Concluir Projeto
                          </Button>
                        </>
                      )}
                      {p.estado === "suspenso" && (
                        <Button
                          size="sm"
                          disabled={pending}
                          onClick={() => void mudarEstado("em_andamento")}
                        >
                          Retomar
                        </Button>
                      )}
                      {p.estado !== "concluido" && (
                        <Button
                          size="sm"
                          variant="danger"
                          disabled={pending}
                          onClick={() => void mudarEstado("cancelado")}
                        >
                          Cancelar
                        </Button>
                      )}
                    </div>
                  )}
                </div>
              }
            />

            {/* Informações e Escopo */}
            <div className="grid gap-4 md:grid-cols-3">
              <Card className="md:col-span-2">
                <CardHeader>
                  <CardTitle as="h2">Escopo e Objetivos</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4 text-sm pb-5">
                  <div>
                    <h3 className="text-xs font-semibold uppercase text-muted">Objetivo</h3>
                    <p className="mt-1 text-foreground">
                      {p.objetivo || "Nenhum objetivo específico registrado."}
                    </p>
                  </div>

                  <div>
                    <h3 className="text-xs font-semibold uppercase text-muted">Descrição</h3>
                    <p className="mt-1 text-foreground whitespace-pre-wrap">
                      {p.descricao || "Sem detalhes adicionais."}
                    </p>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle as="h2">Informações Gerais</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3 text-xs pb-5">
                  <div>
                    <span className="text-muted">Responsável:</span>
                    <p className="font-medium text-foreground">
                      {p.gerente_nome || "Não informado"}
                    </p>
                  </div>
                  <div>
                    <span className="text-muted">Orçamento:</span>
                    <p className="font-mono font-medium text-foreground">
                      R${" "}
                      {Number(p.orcamento).toLocaleString("pt-BR", {
                        minimumFractionDigits: 2,
                      })}
                    </p>
                  </div>
                  <div>
                    <span className="text-muted">Criado em:</span>
                    <p className="font-medium text-foreground">
                      {new Date(p.created_at).toLocaleString("pt-BR")}
                    </p>
                  </div>
                  {p.produto_id && (
                    <div>
                      <span className="text-muted">Produto Digital:</span>
                      <p className="mt-0.5">
                        <Link
                          href="/produtos"
                          className="font-medium text-primary hover:underline"
                        >
                          Ver no catálogo de produtos
                        </Link>
                      </p>
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>

            {/* Cronograma Visual e Diagrama de Gantt */}
            <ProjetoGanttChart
              projeto={p}
              tarefas={listaTarefas}
              vinculos={listaVinculos}
              podeEditar={gere && p.estado !== "cancelado"}
            />

            {/* Tarefas e Marcos */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-3">
                <div>
                  <CardTitle as="h2">Tarefas e Marcos</CardTitle>
                  <p className="text-xs text-muted mt-0.5">
                    Acompanhamento operacional das entregas e atividades vinculadas.
                  </p>
                </div>
                {gere && p.estado !== "cancelado" && (
                  <div className="flex items-center gap-2">
                    <Link
                      href="/projetos/tarefas/colunas"
                      className={buttonClass("secondary", "sm")}
                    >
                      Colunas
                    </Link>
                    <Button
                      size="sm"
                      type="button"
                      onClick={() => setModalTarefa(true)}
                      data-testid="adicionar-tarefa-btn"
                    >
                      Nova tarefa
                    </Button>
                  </div>
                )}
              </CardHeader>
              <CardContent className="pb-5">
                <DataState
                  loading={tarefas.isLoading}
                  error={tarefas.error}
                  onRetry={() => void tarefas.mutate()}
                  empty={false}
                >
                  {listaTarefas.length === 0 && (
                    <p className="mb-3 text-sm text-muted">
                      Nenhuma tarefa criada. Adicione a primeira tarefa para estruturar as entregas
                      deste projeto.
                    </p>
                  )}
                  <Quadro
                    rotulo="Quadro de tarefas"
                    memoria={`nova-faixas:tarefas:${p.id}`}
                    aparencia={`nova-faixas:tarefas:${p.id}`}
                    colunas={colunasTarefa}
                    {...controlesTarefas}
                    itens={listaTarefas}
                    colunaDe={(t) => t.estado}
                    idDe={(t) => t.id}
                    nomeDe={(t) => t.titulo}
                    pending={pending}
                    arrastavel={() => gere && p.estado !== "cancelado"}
                    podeSoltar={(t, coluna) =>
                      t.estado !== coluna &&
                      catalogoTarefas.colunas.find((item) => item.id === coluna)?.ativa !== false
                    }
                    aoSoltar={(t, coluna) => void atualizarTarefaEstado(t, coluna as EstadoTarefa)}
                    aoRenomear={
                      gere && p.estado !== "cancelado"
                        ? (t, nome) => void renomearTarefa(t, nome)
                        : undefined
                    }
                    render={(t, vista) => {
                      const meta = seloDoEstado(
                        catalogoTarefas.colunas,
                        t.estado,
                        ESTADO_TAREFA[t.estado],
                      );
                      return (
                        <div className="flex flex-col gap-2" data-testid={`tarefa-${t.id}`}>
                          <div className="flex flex-wrap items-center gap-2">
                            <NomeClicavel
                              nome={vista.titulo || t.titulo}
                              onSalvar={vista.salvarNome}
                              className="text-left text-sm font-medium text-foreground hover:underline"
                            />
                            <Badge tone={meta?.tone ?? "neutral"}>{meta?.label ?? t.estado}</Badge>
                          </div>
                          {t.descricao && <p className="text-xs text-muted">{t.descricao}</p>}
                          {t.responsavel_nome && (
                            <span className="text-[11px] text-muted">
                              Atribuído a: {t.responsavel_nome}
                            </span>
                          )}
                          {gere && p.estado !== "cancelado" && (
                            <div className="flex items-center gap-2">
                              <Select
                                value={t.estado}
                                onChange={(e) =>
                                  void atualizarTarefaEstado(t, e.target.value as EstadoTarefa)
                                }
                                aria-label={`Estado da tarefa ${t.titulo}`}
                                options={ordenarColunas(
                                  catalogoTarefas.colunas.filter(
                                    (coluna) => coluna.ativa !== false || coluna.id === t.estado,
                                  ),
                                ).map((coluna) => ({
                                  value: coluna.id,
                                  label: rotuloQuadro(
                                    catalogoTarefas.oficial,
                                    catalogoTarefas.colunas,
                                    lookTarefas.gravado,
                                    coluna.id,
                                  ),
                                }))}
                              />
                              <Button
                                size="sm"
                                variant="ghost"
                                onClick={() => void excluirTarefa(t)}
                                aria-label={`Excluir tarefa ${t.titulo}`}
                              >
                                &times;
                              </Button>
                            </div>
                          )}
                        </div>
                      );
                    }}
                  />
                </DataState>
              </CardContent>
            </Card>

            {/* Modal de Nova Tarefa */}
            <Dialog
              open={modalTarefa}
              size="full"
              onClose={() => setModalTarefa(false)}
              title="Nova Tarefa"
              description="Adicione uma atividade ou marco ao cronograma do projeto."
            >
              <form onSubmit={criarTarefa} className="mt-3 grid gap-3">
                <Input
                  id="tarefa-titulo"
                  label="Título da tarefa *"
                  value={tarefaTitulo}
                  onChange={(e) => setTarefaTitulo(e.target.value)}
                  placeholder="Ex.: Configurar banco de homologação"
                  required
                  maxLength={160}
                />

                <Textarea
                  id="tarefa-desc"
                  label="Descrição"
                  value={tarefaDescricao}
                  onChange={(e) => setTarefaDescricao(e.target.value)}
                  placeholder="Instruções ou critérios de aceite..."
                  rows={8}
                  className="min-h-48"
                  maxLength={4000}
                />

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <Input
                    id="tarefa-resp"
                    label="Responsável"
                    value={tarefaResponsavel}
                    onChange={(e) => setTarefaResponsavel(e.target.value)}
                    placeholder="Nome da pessoa responsável"
                  />

                  <Select
                    id="tarefa-estado"
                    label="Quadro"
                    value={tarefaEstado}
                    onChange={(e) => setTarefaEstado(e.target.value as EstadoTarefa)}
                    options={ordenarColunas(
                      catalogoTarefas.colunas.filter((coluna) => coluna.ativa !== false),
                    ).map((coluna) => ({
                      value: coluna.id,
                      label: rotuloQuadro(
                        catalogoTarefas.oficial,
                        catalogoTarefas.colunas,
                        lookTarefas.gravado,
                        coluna.id,
                      ),
                    }))}
                  />
                </div>

                <div className="flex justify-end gap-2 pt-2">
                  <Button type="button" variant="secondary" onClick={() => setModalTarefa(false)}>
                    Cancelar
                  </Button>
                  <Button type="submit" loading={pending}>
                    Adicionar tarefa
                  </Button>
                </div>
              </form>
            </Dialog>

            {/* Seção de Vínculos da Assistência (ITIL) */}
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-3">
                <div className="flex items-center gap-2">
                  <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-info/10 text-info">
                    <Link2 className="h-4 w-4" />
                  </div>
                  <div>
                    <CardTitle as="h2">Vínculos da Assistência</CardTitle>
                    <p className="text-xs text-muted mt-0.5">
                      Interconexão com mudanças, problemas, ativos de TI e incidentes relacionados.
                    </p>
                  </div>
                </div>
                {gere && p.estado !== "cancelado" && (
                  <Button
                    size="sm"
                    variant="secondary"
                    type="button"
                    onClick={() => setModalVinculo(true)}
                    data-testid="adicionar-vinculo-btn"
                    className="inline-flex items-center gap-1.5"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    Vincular item
                  </Button>
                )}
              </CardHeader>
              <CardContent className="pb-5">
                <DataState
                  loading={vinculos.isLoading}
                  error={vinculos.error}
                  onRetry={() => void vinculos.mutate()}
                  empty={listaVinculos.length === 0}
                  emptyTitle="Nenhum vínculo registrado"
                  emptyDescription="Vincule mudanças ou problemas estruturais associados a este projeto."
                >
                  <div className="divide-y divide-surface-border">
                    {listaVinculos.map((v) => {
                      const tipoMeta = TIPO_VINCULO_PROJETO[v.tipo];
                      const rota =
                        v.tipo === "mudanca"
                          ? `/mudancas/${v.alvo_id}`
                          : v.tipo === "problema"
                            ? `/problemas/${v.alvo_id}`
                            : v.tipo === "ativo"
                              ? `/ativos/${v.alvo_id}`
                              : v.tipo === "solicitacao"
                                ? `/atendimento/solicitacoes/${v.alvo_id}`
                                : null;

                      return (
                        <div
                          key={v.id}
                          className="py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                          data-testid={`vinculo-${v.id}`}
                        >
                          <div className="flex items-center gap-2.5">
                            <Badge tone={tipoMeta?.tone ?? "neutral"}>
                              {tipoMeta?.label ?? v.tipo}
                            </Badge>
                            {rota ? (
                              <Link
                                href={rota}
                                className="text-sm font-medium text-primary hover:underline"
                              >
                                {v.rotulo || v.alvo_id}
                              </Link>
                            ) : (
                              <span className="text-sm font-medium text-foreground">
                                {v.rotulo || v.alvo_id}
                              </span>
                            )}
                            <span className="text-[11px] text-muted">
                              • Vinculado por {v.vinculado_por || "sistema"} em{" "}
                              {new Date(v.created_at).toLocaleDateString("pt-BR")}
                            </span>
                          </div>

                          {gere && p.estado !== "cancelado" && (
                            <Button
                              size="sm"
                              variant="ghost"
                              onClick={() => void excluirVinculo(v)}
                              className="text-danger hover:text-danger"
                              aria-label={`Desvincular ${v.rotulo}`}
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </DataState>
              </CardContent>
            </Card>

            {/* Modal de Novo Vínculo */}
            <Dialog
              open={modalVinculo}
              size="full"
              onClose={() => setModalVinculo(false)}
              title="Vincular Item da Assistência"
              description="Conecte este projeto a uma mudança, problema ou ativo de TI."
            >
              <form onSubmit={criarVinculo} className="mt-3 grid gap-3">
                <Select
                  id="vinculo-tipo"
                  label="Tipo de vínculo *"
                  value={vinculoTipo}
                  onChange={(e) => {
                    setVinculoTipo(e.target.value as TipoVinculoProjeto);
                    setVinculoAlvoId("");
                    setVinculoRotulo("");
                  }}
                  options={[
                    { value: "mudanca", label: "Mudança (CAB / Janela)" },
                    { value: "problema", label: "Problema (Causa Raiz)" },
                    { value: "ativo", label: "Ativo de TI" },
                    { value: "solicitacao", label: "Chamado / Incidente" },
                  ]}
                />

                {opcoesCatalogoVinculo.length > 0 && (
                  <Select
                    id="vinculo-catalogo"
                    label="Selecionar item existente"
                    value={vinculoAlvoId}
                    onChange={(e) => {
                      const idEscolhido = e.target.value;
                      setVinculoAlvoId(idEscolhido);
                      const encontrado = opcoesCatalogoVinculo.find(
                        (op) => op.value === idEscolhido,
                      );
                      if (encontrado && !vinculoRotulo) {
                        setVinculoRotulo(encontrado.label);
                      }
                    }}
                    placeholder="Escolha um item da lista..."
                    options={opcoesCatalogoVinculo}
                  />
                )}

                <Input
                  id="vinculo-alvo-id"
                  label="ID do item (UUID) *"
                  value={vinculoAlvoId}
                  onChange={(e) => setVinculoAlvoId(e.target.value)}
                  placeholder="Ex.: 00000000-0000-0000-0000-000000000001"
                  required
                />

                <Input
                  id="vinculo-rotulo"
                  label="Rótulo descritivo (opcional)"
                  value={vinculoRotulo}
                  onChange={(e) => setVinculoRotulo(e.target.value)}
                  placeholder="Preenchido automaticamente se deixado em branco"
                  maxLength={200}
                />

                <div className="flex justify-end gap-2 pt-2">
                  <Button type="button" variant="secondary" onClick={() => setModalVinculo(false)}>
                    Cancelar
                  </Button>
                  <Button type="submit" loading={pending}>
                    Vincular
                  </Button>
                </div>
              </form>
            </Dialog>

            {/* Card de Apontamento de Horas & Timesheet */}
            <Card data-testid="card-timesheet">
              <CardHeader className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-surface-border">
                <div className="flex items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Clock className="h-4 w-4" />
                  </div>
                  <div>
                    <CardTitle as="h2">Apontamento de Horas & Timesheet</CardTitle>
                    <p className="text-xs text-muted mt-0.5">
                      Registro de horas dedicadas pela equipe e controle de queima do orçamento.
                    </p>
                  </div>
                </div>
                {gere && p.estado !== "cancelado" && (
                  <Button
                    size="sm"
                    variant="secondary"
                    type="button"
                    onClick={() => setModalApontamento(true)}
                    data-testid="lancar-horas-btn"
                    className="inline-flex items-center gap-1.5"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    Lançar horas
                  </Button>
                )}
              </CardHeader>

              {/* Métricas Consolidadas do Timesheet */}
              <div className="grid grid-cols-1 sm:grid-cols-3 divide-y sm:divide-y-0 sm:divide-x divide-surface-border border-b border-surface-border bg-black/[0.01] dark:bg-white/[0.01]">
                <div className="p-4">
                  <span className="text-xs text-muted block">Total de Horas Lançadas</span>
                  <span className="text-xl font-bold text-foreground">
                    {resumoHoras.total_horas.toFixed(1)} h
                  </span>
                </div>
                <div className="p-4">
                  <span className="text-xs text-muted block">Custo Realizado Acumulado</span>
                  <span className="text-xl font-bold text-foreground">
                    R${" "}
                    {resumoHoras.total_custo.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
                  </span>
                </div>
                <div className="p-4">
                  <span className="text-xs text-muted block">Consumo do Orçamento</span>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="text-sm font-semibold text-foreground">
                      {p.orcamento > 0
                        ? `${Math.round((resumoHoras.total_custo / p.orcamento) * 100)}%`
                        : "N/A"}
                    </span>
                    {p.orcamento > 0 && (
                      <div className="h-2 flex-1 rounded-full bg-surface-border overflow-hidden">
                        <div
                          className={`h-full transition-all ${
                            resumoHoras.total_custo > p.orcamento
                              ? "bg-danger"
                              : resumoHoras.total_custo > p.orcamento * 0.8
                                ? "bg-warning"
                                : "bg-success"
                          }`}
                          style={{
                            width: `${Math.min(100, Math.round((resumoHoras.total_custo / p.orcamento) * 100))}%`,
                          }}
                        />
                      </div>
                    )}
                  </div>
                </div>
              </div>

              <CardContent className="pb-5 pt-4">
                <DataState
                  loading={apontamentos.isLoading}
                  error={apontamentos.error}
                  onRetry={() => void apontamentos.mutate()}
                  empty={listaApontamentos.length === 0}
                  emptyTitle="Nenhum apontamento registrado"
                  emptyDescription="Lance as primeiras horas trabalhadas neste projeto."
                >
                  <div className="divide-y divide-surface-border">
                    {listaApontamentos.map((ap) => (
                      <div
                        key={ap.id}
                        className="py-3 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                        data-testid={`apontamento-${ap.id}`}
                      >
                        <div className="space-y-1">
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-xs text-foreground">
                              {ap.usuario_nome}
                            </span>
                            <Badge tone="info">{ap.horas}h</Badge>
                            {ap.tarefa_titulo && <Badge tone="neutral">{ap.tarefa_titulo}</Badge>}
                            <span className="text-xs text-muted">
                              em{" "}
                              {new Date(ap.data_apontamento + "T00:00:00").toLocaleDateString(
                                "pt-BR",
                              )}
                            </span>
                          </div>
                          <p className="text-xs text-muted">{ap.descricao}</p>
                          {ap.custo_total > 0 && (
                            <span className="text-[11px] text-muted">
                              Custo calculado: R${" "}
                              {ap.custo_total.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
                            </span>
                          )}
                        </div>

                        {gere && p.estado !== "cancelado" && (
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => void excluirApontamento(ap)}
                            className="text-danger hover:text-danger self-end sm:self-center"
                            aria-label={`Excluir apontamento de ${ap.usuario_nome}`}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        )}
                      </div>
                    ))}
                  </div>
                </DataState>
              </CardContent>
            </Card>

            {/* Modal de Lançar Horas */}
            <Dialog
              open={modalApontamento}
              size="full"
              onClose={() => setModalApontamento(false)}
              title="Lançar Horas no Projeto"
              description="Registre o esforço despendido em tarefas ou marcos do projeto."
            >
              <form onSubmit={criarApontamento} className="mt-3 grid gap-3">
                <Select
                  id="apontamento-tarefa"
                  label="Tarefa associada (opcional)"
                  value={apontamentoTarefaId}
                  onChange={(e) => setApontamentoTarefaId(e.target.value)}
                  options={[
                    { value: "", label: "Nenhuma (esforço geral do projeto)" },
                    ...listaTarefas.map((t) => ({
                      value: t.id,
                      label: t.titulo,
                    })),
                  ]}
                />

                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <Input
                    id="apontamento-data"
                    label="Data *"
                    type="date"
                    value={apontamentoData}
                    onChange={(e) => setApontamentoData(e.target.value)}
                    required
                  />

                  <Input
                    id="apontamento-horas"
                    label="Horas trabalhadas *"
                    type="number"
                    step="0.5"
                    min="0.5"
                    max="24"
                    value={apontamentoHoras}
                    onChange={(e) => setApontamentoHoras(e.target.value)}
                    required
                  />
                </div>

                <Input
                  id="apontamento-custo"
                  label="Custo por hora (R$, opcional)"
                  type="number"
                  step="0.01"
                  min="0"
                  value={apontamentoCustoHora}
                  onChange={(e) => setApontamentoCustoHora(e.target.value)}
                  placeholder="Ex.: 85.00"
                />

                <Textarea
                  id="apontamento-desc"
                  label="Descrição das atividades realizadas *"
                  value={apontamentoDescricao}
                  onChange={(e) => setApontamentoDescricao(e.target.value)}
                  placeholder="Descreva o que foi desenvolvido, configurado ou homologado..."
                  required
                  rows={8}
                  className="min-h-48"
                  maxLength={2000}
                />

                <div className="flex justify-end gap-2 pt-2">
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => setModalApontamento(false)}
                  >
                    Cancelar
                  </Button>
                  <Button type="submit" loading={pending}>
                    Registrar horas
                  </Button>
                </div>
              </form>
            </Dialog>
          </>
        )}
      </DataState>
    </div>
  );
}
