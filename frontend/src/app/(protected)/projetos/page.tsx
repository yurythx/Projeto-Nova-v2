"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useMemo, useState } from "react";

import { FileSpreadsheet } from "lucide-react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { PessoaComAvatar } from "@/components/plataforma/AvatarPessoa";
import { DataState } from "@/components/plataforma/DataState";
import {
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
import { CargaEquipe, MeuTrabalho } from "@/components/projetos/MeuTrabalho";
import { ListaTabela } from "@/components/projetos/ListaTabela";
import { ProgressoProjeto } from "@/components/projetos/ProgressoProjeto";
import { TarefasNaLista } from "@/components/projetos/TarefasNaLista";
import { diaBrasil } from "@/components/projetos/datas";
import { PessoaDaUnidade } from "@/components/projetos/PessoaDaUnidade";
import { ProjetosCronograma } from "@/components/projetos/ProjetosCronograma";
import { COLUNAS_PROJETO, ProjetosKanban } from "@/components/projetos/ProjetosKanban";
import { Wizard5W2H, type Corpo5W2H } from "@/components/projetos/Wizard5W2H";
import { FiltroPorUnidade, filtrarPorUnidade } from "@/components/servicos/FiltroPorUnidade";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { TableCell } from "@/components/ui/Table";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import { useEntityFilter } from "@/lib/plataforma/useEntityFilter";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import {
  ESTADO_PROJETO,
  PRIORIDADE_PROJETO,
  type EstadoProjeto,
  type PrioridadeProjeto,
  type CargaPessoa,
  type Projeto,
  type TarefaMinha,
} from "@/lib/projetos/projetos";

type Vista = "lista" | "kanban" | "cronograma" | "minhas" | "carga";

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
    paramVista === "lista"
      ? "lista"
      : paramVista === "cronograma"
        ? "cronograma"
        : paramVista === "minhas"
          ? "minhas"
          : paramVista === "carga"
            ? "carga"
            : "kanban";
  const gere = can("projetos:manage");
  const le = gere || can("projetos:read");
  const [estado, setEstado] = useState("");
  const [busca, setBusca] = useState("");
  const [unidadeFiltro, setUnidadeFiltro] = useState("todas");
  const somenteFiltroLista = vista === "lista";

  const { run, pending } = useAction();

  const projetos = useApiQuery<Projeto[]>(
    le && vista !== "minhas" && vista !== "carga"
      ? withQuery("v1/projetos", {
          estado: somenteFiltroLista ? estado : undefined,
          q: busca || undefined,
          produto_id: paramProduto || undefined,
        })
      : null,
  );
  const minhas = useApiQuery<TarefaMinha[]>(vista === "minhas" && le ? "v1/projetos/minhas" : null);
  const carga = useApiQuery<CargaPessoa[]>(vista === "carga" && le ? "v1/projetos/carga" : null);

  const [modalAberto, setModalAberto] = useState(false);
  const [modo, setModo] = useState<"branco" | "5w2h">("branco");
  const [titulo, setTitulo] = useState("");
  const [descricao, setDescricao] = useState("");
  const [objetivo, setObjetivo] = useState("");
  const [prioridade, setPrioridade] = useState<PrioridadeProjeto>("media");
  const [gerente, setGerente] = useState({ id: "", nome: "" });
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
      proxima === "lista" || proxima === "cronograma" || proxima === "minhas" || proxima === "carga"
        ? proxima
        : "quadro";
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
      "Prazo",
      "Orcamento",
      "CriadoEm",
    ];
    const linhas = lista.map((p) => [
      p.numero,
      `"${p.titulo.replace(/"/g, '""')}"`,
      ESTADO_PROJETO[p.estado]?.label ?? p.estado,
      PRIORIDADE_PROJETO[p.prioridade]?.label ?? p.prioridade,
      `"${(p.gerente_nome || "").replace(/"/g, '""')}"`,
      diaBrasil(p.data_fim_planejada),
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

  function fecharModal() {
    setModalAberto(false);
    setModo("branco");
  }

  async function criarProjeto(corpo: Record<string, unknown>, alvo: EstadoProjeto, carta: boolean) {
    const criado = await run(async () => {
      const res = await apiClient.post<Projeto>("v1/projetos", corpo);
      for (const passo of passosDoProjeto(alvo)) {
        await apiClient.post(`v1/projetos/${res.data.id}/estado`, { estado: passo });
      }
      return res.data;
    }, "Projeto criado com sucesso");
    if (criado) {
      fecharModal();
      setTitulo("");
      setDescricao("");
      setObjetivo("");
      setGerente({ id: "", nome: "" });
      setOrcamento("0");
      setProdutoId("");
      setQuadro("planejado");
      void projetos.mutate();
      router.push(carta ? `/projetos/${criado.id}?vista=5w2h` : `/projetos/${criado.id}`);
    }
  }

  function criar(e: FormEvent) {
    e.preventDefault();
    void criarProjeto(
      {
        titulo,
        descricao,
        objetivo,
        prioridade,
        gerente_id: gerente.id || undefined,
        gerente_nome: gerente.id ? gerente.nome : "",
        orcamento: Number(orcamento) || 0,
        unidade_id: unidadeId || undefined,
        produto_id: produtoId || undefined,
      },
      quadro,
      false,
    );
  }

  function criarGuiado(corpo: Corpo5W2H, alvo: EstadoProjeto) {
    void criarProjeto(corpo as unknown as Record<string, unknown>, alvo, true);
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

  const { filtrarPorEntidade } = useEntityFilter();
  const todosProjetos = useMemo(
    () => filtrarPorEntidade(projetos.data ?? []),
    [filtrarPorEntidade, projetos.data],
  );

  const lista = filtrarPorUnidade(todosProjetos, unidadeFiltro);
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
          { value: "minhas", label: "Meu trabalho" },
          { value: "carga", label: "Carga" },
        ]}
      />

      {vista !== "minhas" && vista !== "carga" && (
      <>
      <FiltroPorUnidade
        rotulo="Filtrar projetos por unidade"
        itens={todosProjetos}
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
      </>
      )}

      {vista === "minhas" ? (
        <DataState
          loading={minhas.isLoading || carregandoSessao}
          error={minhas.error}
          empty={false}
          onRetry={() => void minhas.mutate()}
        >
          <MeuTrabalho tarefas={minhas.data ?? []} />
        </DataState>
      ) : vista === "carga" ? (
        <DataState
          loading={carga.isLoading || carregandoSessao}
          error={carga.error}
          empty={false}
          onRetry={() => void carga.mutate()}
        >
          <CargaEquipe pessoas={carga.data ?? []} />
        </DataState>
      ) : (
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
          <ListaTabela
            rotulo="Lista de projetos"
            memoria="nova-faixas:projetos-lista"
            aparencia="nova-faixas:projetos"
            colunas={colunas}
            oficial={controles.oficial}
            itens={lista}
            colunaDe={(p) => p.estado}
            idDe={(p) => p.id}
            nomeDe={(p) => p.titulo}
            aoRenomear={gere ? (p, nome) => void renomearProjeto(p, nome) : undefined}
            detalhe={(p) => <TarefasNaLista projetoId={p.id} titulo={p.titulo} podeCriar={gere} />}
            cabecalhos={[
              { id: "numero", rotulo: "#" },
              { id: "projeto", rotulo: "Projeto" },
              { id: "responsavel", rotulo: "Responsável" },
              { id: "prioridade", rotulo: "Prioridade" },
              { id: "progresso", rotulo: "Progresso" },
              { id: "prazo", rotulo: "Prazo" },
            ]}
            render={(p, vista) => {
              const prioridadeMeta = PRIORIDADE_PROJETO[p.prioridade];
              return (
                <>
                  <TableCell>
                    <span className="font-mono text-sm font-semibold text-primary">#{p.numero}</span>
                  </TableCell>
                  <TableCell>
                    <NomeClicavel
                      nome={vista.titulo || p.titulo}
                      href={`/projetos/${p.id}`}
                      onSalvar={vista.salvarNome}
                      className="truncate text-left text-sm font-semibold text-foreground hover:underline"
                    />
                  </TableCell>
                  <TableCell>
                    <PessoaComAvatar
                      id={p.gerente_id}
                      nome={p.gerente_nome}
                      className="text-sm text-foreground"
                    />
                  </TableCell>
                  <TableCell>
                    {prioridadeMeta ? (
                      <Badge tone={prioridadeMeta.tone}>{prioridadeMeta.label}</Badge>
                    ) : (
                      <span className="text-sm text-muted">—</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <ProgressoProjeto
                      titulo={p.titulo}
                      total={p.tarefas_total}
                      concluidas={p.tarefas_concluidas}
                    />
                  </TableCell>
                  <TableCell>
                    <span className="text-sm text-muted">
                      {diaBrasil(p.data_fim_planejada) || "Sem prazo"}
                    </span>
                  </TableCell>
                </>
              );
            }}
          />
        )}
      </DataState>
      )}

      {/* Modal de Criação de Projeto */}
      <Dialog
        open={modalAberto}
        onClose={fecharModal}
        title="Novo Projeto"
        description={
          modo === "5w2h"
            ? "Sete passos para o plano diretor. Só o nome do projeto é obrigatório."
            : "O projeto entra no quadro escolhido. O nome da coluna é só o rótulo; o sistema guarda o estado."
        }
        size="full"
      >
        <div className="mt-3 flex gap-2" role="group" aria-label="Tipo de criação">
          <Button
            type="button"
            variant={modo === "branco" ? "primary" : "secondary"}
            onClick={() => setModo("branco")}
          >
            Em branco
          </Button>
          <Button
            type="button"
            variant={modo === "5w2h" ? "primary" : "secondary"}
            onClick={() => setModo("5w2h")}
            data-testid="modo-5w2h"
          >
            5W2H
          </Button>
        </div>
        {modo === "5w2h" ? (
          <Wizard5W2H
            produtos={produtos.data ?? []}
            colunas={ordenarColunas(
              catalogo.colunas.filter((coluna) => coluna.ativa !== false),
            ).map((coluna) => ({
              value: coluna.id,
              label: rotuloQuadro(catalogo.oficial, catalogo.colunas, look.gravado, coluna.id),
            }))}
            pending={pending}
            onCancel={fecharModal}
            onCriar={criarGuiado}
          />
        ) : (
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

            <PessoaDaUnidade
              id="proj-gerente"
              label="Responsável / Gerente"
              unidadeId={unidadeId}
              value={gerente}
              onChange={setGerente}
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
        )}
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
