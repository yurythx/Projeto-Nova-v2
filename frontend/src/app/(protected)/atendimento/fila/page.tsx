"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense } from "react";

import { FAIXAS_FILA, Kanban, faixaDaFila } from "@/components/atendimento/Kanban";
import {
  consultaDaLista,
  FiltrosDaLista,
  type FiltrosLista,
  lerFiltros,
  TabelaDeChamados,
} from "@/components/atendimento/ListaDeChamados";
import { colunasVisiveis, useQuadro } from "@/components/plataforma/useQuadro";
import { PainelDoQuadro, usePainelNaUrl } from "@/components/plataforma/PainelDoQuadro";
import { SolicitacaoDetalhe } from "@/app/(protected)/atendimento/solicitacoes/[id]/page";
import { useRealtimeState, useTopics } from "@/components/realtime/RealtimeProvider";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { useAction } from "@/components/plataforma/useAction";
import { buttonClass } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import {
  type Categoria,
  type Estado,
  rotuloEstado,
  rotuloTipo,
  type Solicitacao,
} from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { OrgTree } from "@/lib/plataforma/types";

type Vista = "lista" | "kanban";

export default function AtendimentoFilaPage() {
  return (
    <Suspense fallback={null}>
      <FilaDaEquipe />
    </Suspense>
  );
}

/** Fila da equipe: as solicitações das unidades que a pessoa atende, das
 * mais antigas para as mais novas, em lista ou em quadro. As duas vistas
 * leem a mesma consulta e mudam o estado pela mesma API. Quem está com a
 * fila aberta vê as mudanças dos colegas sem recarregar. A vista fica na
 * URL (?vista=lista ou ?vista=quadro). Sem parâmetro, abre o quadro. */
function FilaDaEquipe() {
  const { can, me } = usePlataforma();
  const atende = can("atendimento:atender");
  const abre = can("atendimento:abrir");
  const gereColunas = can("atendimento:manage");
  const catalogo = useQuadro("fila", FAIXAS_FILA);
  const controles = { oficial: catalogo.oficial };
  const params = useSearchParams();
  const router = useRouter();
  const painel = usePainelNaUrl();
  const vista: Vista = params.get("vista") === "lista" ? "lista" : "kanban";
  const filtros = lerFiltros(params);
  const kanban = vista === "kanban";
  const fila = useApiPage<Solicitacao>(
    atende
      ? withQuery(
          "v1/atendimento/fila",
          kanban
            ? // O quadro mostra o fluxo em andamento, uma coluna por estado.
              {
                responsavel: filtros.responsavel || undefined,
                tipo: filtros.tipo || undefined,
                page: 1,
                page_size: 100,
              }
            : consultaDaLista(filtros),
        )
      : null,
  );
  const arvore = useApiQuery<OrgTree[]>(atende && !kanban ? "v1/iam/org-tree" : null);
  const categorias = useApiQuery<Categoria[]>(
    atende && !kanban ? "v1/atendimento/catalogo/categorias" : null,
  );
  const unidades = (arvore.data ?? []).flatMap((e) =>
    e.unidades.map((u) => ({ value: u.id, label: `${e.nome} — ${u.nome}` })),
  );
  const topicos = useApiQuery<{ topicos: string[] }>(atende ? "v1/atendimento/fila/topicos" : null);
  const conexao = useRealtimeState();
  useTopics(topicos.data?.topicos ?? [], () => void fila.mutate());
  const { run, pending } = useAction();

  if (!atende) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Atendimento" title="Fila da equipe" />
        <EmptyState
          title="Seu perfil não atende a fila"
          description="A fila mostra as solicitações das unidades em que você atende."
        />
      </div>
    );
  }

  /** Troca os filtros na URL; mudar qualquer um que não a página volta à primeira. */
  function mudar(novos: Partial<FiltrosLista>) {
    const extra: Record<string, string | null> = { pagina: null };
    for (const [chave, valor] of Object.entries(novos)) extra[chave] = valor ? valor : null;
    router.replace(painel.href(extra), { scroll: false });
  }

  function trocarVista(v: Vista) {
    router.replace(painel.href({ vista: v === "lista" ? "lista" : "quadro" }));
  }

  async function mover(s: Solicitacao, para: Estado) {
    await run(
      () => apiClient.post(`v1/atendimento/solicitacoes/${s.id}/estado`, { estado: para }),
      `Solicitação #${s.numero}: ${rotuloEstado(para).toLowerCase()}`,
    );
    // Recusada ou não, a fila volta a refletir o banco.
    void fila.mutate();
  }

  async function assumir(s: Solicitacao) {
    if (!me) return;
    const ok = await run(
      () =>
        apiClient.put(`v1/atendimento/solicitacoes/${s.id}/responsavel`, { responsavel_id: me.id }),
      `Solicitação #${s.numero} assumida`,
    );
    if (ok) void fila.mutate();
  }

  const itens = fila.data?.items ?? [];
  const colunasQuadro = colunasVisiveis(
    catalogo.colunas.filter((coluna) => coluna.id !== "encerrado" && coluna.id !== "cancelado"),
    itens,
    (s) => {
      const faixa = faixaDaFila(s.estado);
      return faixa === "encerrado" || faixa === "cancelado" ? null : faixa;
    },
  );

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Fila da equipe"
        description="Solicitações das unidades que você atende. Assumir uma solicitação aberta já a coloca em atendimento."
        actions={
          <div className="flex items-center gap-2">
            {gereColunas && (
              <Link href="/atendimento/fila/colunas" className={buttonClass("secondary", "sm")}>
                Colunas
              </Link>
            )}
            {abre && (
              <Link href="/atendimento" className={buttonClass("primary", "sm")}>
                Nova solicitação
              </Link>
            )}
            <span className="text-xs text-muted" role="status">
              {conexao === "open"
                ? "Atualização automática ligada"
                : "Atualização automática indisponível"}
            </span>
          </div>
        }
      />

      <SectionTabsInline
        label="Forma de ver a fila"
        value={vista}
        onChange={trocarVista}
        tabs={[
          { value: "kanban", label: "Quadro" },
          { value: "lista", label: "Lista" },
        ]}
      />

      {kanban ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Select
            id="fila-responsavel"
            label="Responsável"
            value={filtros.responsavel}
            onChange={(e) => mudar({ responsavel: e.target.value })}
            options={[
              { value: "", label: "Todos" },
              { value: "eu", label: "Comigo" },
              { value: "nenhum", label: "Sem responsável" },
            ]}
          />
          <Select
            id="fila-tipo"
            label="Tipo"
            value={filtros.tipo}
            onChange={(e) => mudar({ tipo: e.target.value })}
            options={[
              { value: "", label: "Todos" },
              { value: "incidente", label: rotuloTipo("incidente") },
              { value: "requisicao", label: rotuloTipo("requisicao") },
            ]}
          />
        </div>
      ) : (
        <FiltrosDaLista
          filtros={filtros}
          unidades={unidades}
          categorias={(categorias.data ?? []).map((c) => ({ value: c.id, label: c.nome }))}
          mudar={mudar}
        />
      )}

      <DataState
        loading={fila.isLoading}
        error={fila.error}
        onRetry={() => void fila.mutate()}
        empty={false}
      >
        {kanban ? (
          <>
            <Kanban
              itens={itens}
              onMover={(s, para) => void mover(s, para)}
              pending={pending}
              aoAbrir={(s) => painel.abrir(s.id)}
              colunas={colunasQuadro}
              controles={controles}
            />
            {(fila.data?.meta?.total_items ?? 0) > itens.length && (
              <p className="text-sm text-muted">
                O quadro mostra as {itens.length} mais antigas de {fila.data?.meta?.total_items}.
                Use os filtros ou a lista para ver as demais.
              </p>
            )}
          </>
        ) : (
          <TabelaDeChamados
            itens={itens}
            meta={fila.data?.meta}
            filtros={filtros}
            mudar={mudar}
            aoAbrir={(s) => painel.abrir(s.id)}
            onAssumir={(s) => void assumir(s)}
            pending={pending}
          />
        )}
      </DataState>
      <PainelDoQuadro
        aberto={painel.id !== null}
        id={painel.id}
        titulo="Detalhe da solicitação"
        href={painel.id ? `/atendimento/solicitacoes/${painel.id}` : "/atendimento/fila"}
        onClose={painel.fechar}
      >
        {painel.id && <SolicitacaoDetalhe />}
      </PainelDoQuadro>
    </div>
  );
}
