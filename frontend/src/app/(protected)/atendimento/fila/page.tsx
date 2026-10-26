"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";

import { FAIXAS_FILA, Kanban, faixaDaFila } from "@/components/atendimento/Kanban";
import { PrazoBadge } from "@/components/atendimento/PrazoBadge";
import {
  BarraFaixa,
  ordenarColunas,
  pontoFaixa,
  useAparencia,
  useFaixasRecolhidas,
} from "@/components/plataforma/Faixas";
import { colunasVisiveis, useQuadro } from "@/components/plataforma/useQuadro";
import { PainelDoQuadro, usePainelNaUrl } from "@/components/plataforma/PainelDoQuadro";
import { SolicitacaoDetalhe } from "@/app/(protected)/atendimento/solicitacoes/[id]/page";
import { useRealtimeState, useTopics } from "@/components/realtime/RealtimeProvider";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { Pagination } from "@/components/plataforma/Pagination";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";
import { Select } from "@/components/ui/Select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from "@/components/ui/Table";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import {
  ESTADOS_FILA,
  rotuloEstado,
  rotuloTipo,
  tomEstado,
  type Estado,
  type Solicitacao,
} from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

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
  const { recolhidas, alternar } = useFaixasRecolhidas("nova-faixas:fila-lista");
  const look = useAparencia("nova-faixas:fila");
  const catalogo = useQuadro("fila", FAIXAS_FILA);
  const controles = { oficial: catalogo.oficial };
  const params = useSearchParams();
  const router = useRouter();
  const painel = usePainelNaUrl();
  const vista: Vista = params.get("vista") === "lista" ? "lista" : "kanban";
  const [responsavel, setResponsavel] = useState("");
  const [tipo, setTipo] = useState("");
  const [estado, setEstado] = useState("");
  const [todas, setTodas] = useState(false);
  const [pagina, setPagina] = useState(1);
  const kanban = vista === "kanban";
  const fila = useApiPage<Solicitacao>(
    atende
      ? withQuery("v1/atendimento/fila", {
          responsavel,
          tipo,
          // O quadro mostra o fluxo em andamento, uma coluna por estado.
          estado: kanban ? undefined : estado,
          situacao: !kanban && todas ? "todas" : undefined,
          page: kanban ? 1 : pagina,
          page_size: kanban ? 100 : 50,
        })
      : null,
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

  function filtro<T>(set: (v: T) => void) {
    return (v: T) => {
      set(v);
      setPagina(1);
    };
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
  const colunasLista = colunasVisiveis(catalogo.colunas, itens, (s) => faixaDaFila(s.estado));

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

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Select
          id="fila-responsavel"
          label="Responsável"
          value={responsavel}
          onChange={(e) => filtro(setResponsavel)(e.target.value)}
          options={[
            { value: "", label: "Todos" },
            { value: "eu", label: "Comigo" },
            { value: "nenhum", label: "Sem responsável" },
          ]}
        />
        <Select
          id="fila-tipo"
          label="Tipo"
          value={tipo}
          onChange={(e) => filtro(setTipo)(e.target.value)}
          options={[
            { value: "", label: "Todos" },
            { value: "incidente", label: rotuloTipo("incidente") },
            { value: "requisicao", label: rotuloTipo("requisicao") },
          ]}
        />
        {!kanban && (
          <>
            <Select
              id="fila-estado"
              label="Estado"
              value={estado}
              onChange={(e) => filtro(setEstado)(e.target.value)}
              options={[
                { value: "", label: "Todos" },
                ...ESTADOS_FILA.map((v) => ({ value: v, label: rotuloEstado(v) })),
              ]}
            />
            <label className="flex items-center gap-2 self-end pb-2 text-sm">
              <input
                id="fila-todas"
                type="checkbox"
                checked={todas}
                onChange={(e) => filtro(setTodas)(e.target.checked)}
                className="h-4 w-4 accent-primary"
              />
              Incluir encerradas e canceladas
            </label>
          </>
        )}
      </div>

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
          <>
            <div className="flex flex-col gap-2">
              {ordenarColunas(colunasLista).map((coluna) => {
                const titulo = catalogo.oficial
                  ? coluna.titulo
                  : look.gravado.colunas[coluna.id]?.titulo || coluna.titulo;
                const cor = catalogo.oficial ? coluna.cor : look.gravado.colunas[coluna.id]?.cor;
                const total = itens.filter((s) => faixaDaFila(s.estado) === coluna.id).length;
                return (
                  <BarraFaixa
                    key={coluna.id}
                    titulo={titulo}
                    tituloId={`faixa-fila-${coluna.id}`}
                    total={total}
                    tom={coluna.tom}
                    cor={cor}
                    recolhida={recolhidas.includes(coluna.id)}
                    ordem={coluna.ordem}
                    corImediata={!catalogo.oficial}
                    onAlternar={() => alternar(coluna.id)}
                    onRenomear={
                      catalogo.oficial
                        ? undefined
                        : (nome) => look.salvarColuna(coluna.id, { titulo: nome })
                    }
                    onCor={
                      catalogo.oficial
                        ? undefined
                        : (valor) => look.salvarColuna(coluna.id, { cor: valor })
                    }
                  />
                );
              })}
            </div>
            {itens.length === 0 ? (
              <p className="text-sm text-muted">Fila vazia</p>
            ) : (
              <Table caption="Solicitações da fila" className="min-w-[980px]">
                <TableHead>
                  <TableRow>
                    <TableHeaderCell>Nº</TableHeaderCell>
                    <TableHeaderCell>Solicitação</TableHeaderCell>
                    <TableHeaderCell>Tipo</TableHeaderCell>
                    <TableHeaderCell>Estado</TableHeaderCell>
                    <TableHeaderCell>Responsável</TableHeaderCell>
                    <TableHeaderCell>Aberta por</TableHeaderCell>
                    <TableHeaderCell>Aberta em</TableHeaderCell>
                    <TableHeaderCell>Prazo</TableHeaderCell>
                    <TableHeaderCell>
                      <span className="sr-only">Ações</span>
                    </TableHeaderCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {itens
                    .filter((s) => !recolhidas.includes(faixaDaFila(s.estado)))
                    .map((s, indice) => ({ s, indice }))
                    .sort((a, b) => {
                      const ordemDe = (estado: string) =>
                        catalogo.colunas.find(
                          (coluna) => coluna.id === faixaDaFila(estado as Estado),
                        )?.ordem ?? 999;
                      return ordemDe(a.s.estado) - ordemDe(b.s.estado) || a.indice - b.indice;
                    })
                    .map(({ s }) => {
                      const faixa = faixaDaFila(s.estado);
                      const cor =
                        look.gravado.cartoes[s.id]?.cor ||
                        (catalogo.oficial
                          ? catalogo.colunas.find((coluna) => coluna.id === faixa)?.cor
                          : look.gravado.colunas[faixa]?.cor);
                      return (
                        <TableRow key={s.id}>
                          <TableCell className="tabular-nums text-muted">
                            <span
                              className={`mr-2 inline-block size-2.5 rounded-full align-middle ${cor ? "" : pontoFaixa(tomEstado(s.estado))}`}
                              style={cor ? { backgroundColor: cor } : undefined}
                              aria-hidden
                            />
                            #{s.numero}
                          </TableCell>
                          <TableCell>
                            <button
                              type="button"
                              className="font-medium text-primary hover:underline"
                              onClick={() => painel.abrir(s.id)}
                            >
                              {look.gravado.cartoes[s.id]?.titulo || s.titulo}
                            </button>
                            <p className="text-xs text-muted">{s.oferta_nome}</p>
                          </TableCell>
                          <TableCell>{rotuloTipo(s.tipo)}</TableCell>
                          <TableCell>
                            <Badge tone={tomEstado(s.estado)}>{rotuloEstado(s.estado)}</Badge>
                          </TableCell>
                          <TableCell>
                            {s.responsavel_nome || <span className="text-muted">—</span>}
                          </TableCell>
                          <TableCell>{s.solicitante_nome}</TableCell>
                          <TableCell className="whitespace-nowrap">
                            {fmtDateTime(s.created_at)}
                          </TableCell>
                          <TableCell>
                            <PrazoBadge prazo={s.prazo} />
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <button
                                type="button"
                                className="text-sm font-medium text-primary hover:underline"
                                onClick={() => painel.abrir(s.id)}
                              >
                                Editar
                              </button>
                              {!s.responsavel_id &&
                                s.estado !== "encerrado" &&
                                s.estado !== "cancelado" && (
                                  <Button
                                    type="button"
                                    size="sm"
                                    variant="secondary"
                                    disabled={pending}
                                    onClick={() => void assumir(s)}
                                    aria-label={`Assumir a solicitação #${s.numero}`}
                                  >
                                    Assumir
                                  </Button>
                                )}
                            </div>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                </TableBody>
              </Table>
            )}
            <Pagination meta={fila.data?.meta} onPage={setPagina} />
          </>
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
