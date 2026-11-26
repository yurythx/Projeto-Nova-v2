"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useMemo, useState } from "react";

import {
  AlertTriangle,
  Building2,
  CheckCircle2,
  Clock,
  Eye,
  FilterX,
  MoreVertical,
  Search,
  User,
} from "lucide-react";

import { DataState } from "@/components/plataforma/DataState";
import { useAction } from "@/components/plataforma/useAction";
import {
  BOTAO_SECUNDARIO,
  ModalDeAcao,
  type AcaoAberta,
} from "@/components/protocolos/AcoesDoPedido";
import {
  bordaDoPrazo,
  corDaEstatistica,
  dataCurta,
  FINAIS,
  nomeDaEtapa,
  rotuloDaPrioridade,
  rotuloDoStatus,
  seloDaPrioridade,
  seloDoPrazo,
  seloDoStatus,
  textoDoPrazo,
  type PedidoResumo,
} from "@/components/protocolos/estilo";
import { KanbanProtocolos } from "@/components/protocolos/KanbanProtocolos";
import { filtrarPedidos, paginar, STATUS } from "@/components/protocolos/painel";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

const POR_PAGINA = 10;

const ABAS_PRAZO = [
  { value: "todos", label: "Todos" },
  { value: "no_prazo", label: "No Prazo" },
  { value: "atencao", label: "Atenção" },
  { value: "atrasado", label: "Atrasados" },
  { value: "finalizado", label: "Finalizados" },
];

const CAMPO =
  "w-full min-w-[150px] rounded-lg border border-gray-300 bg-surface px-4 py-3 text-sm text-foreground transition-colors focus:border-blue-500 focus:outline-none focus:ring-[3px] focus:ring-blue-500/10";

export default function ProtocolosPage() {
  return (
    <Suspense fallback={null}>
      <PainelDeProtocolos />
    </Suspense>
  );
}

function PainelDeProtocolos() {
  const { can } = usePlataforma();
  const router = useRouter();
  const params = useSearchParams();
  const visao = params.get("vista") === "kanban" ? "kanban" : "lista";
  const trata = can("protocolos:tratar");
  const gere = can("protocolos:gerir");
  const abre = can("protocolos:abrir");

  const lista = useApiQuery<PedidoResumo[]>("v1/protocolos/pedidos");
  const pedidos = useMemo(() => lista.data ?? [], [lista.data]);
  const { run, pending } = useAction();

  const [prazo, setPrazo] = useState("todos");
  const [busca, setBusca] = useState("");
  const [servico, setServico] = useState("");
  const [status, setStatus] = useState("");
  const [prioridade, setPrioridade] = useState("");
  const [pagina, setPagina] = useState(1);
  const [menu, setMenu] = useState<string | null>(null);
  const [acao, setAcao] = useState<AcaoAberta | null>(null);

  const servicos = useMemo(() => {
    const mapa = new Map<string, string>();
    for (const p of pedidos) mapa.set(p.servico_id, p.servico_nome);
    return [...mapa.entries()].sort((a, b) => a[1].localeCompare(b[1], "pt-BR"));
  }, [pedidos]);

  const porStatus = useMemo(
    () =>
      STATUS.map((s) => ({
        status: s.value,
        total: pedidos.filter((p) => p.status === s.value).length,
      })).filter((s) => s.total > 0),
    [pedidos],
  );

  const contagemPrazo = useMemo(() => {
    const conta: Record<string, number> = { todos: pedidos.length };
    for (const p of pedidos) {
      const chave = p.situacao_prazo || "no_prazo";
      conta[chave] = (conta[chave] ?? 0) + 1;
    }
    return conta;
  }, [pedidos]);

  const visiveis = useMemo(
    () => filtrarPedidos(pedidos, { prazo, busca, servico, status, prioridade }),
    [pedidos, prazo, busca, servico, status, prioridade],
  );
  const paginaAtual = useMemo(() => paginar(visiveis, pagina, POR_PAGINA), [visiveis, pagina]);
  const complementacao = porStatus.find((s) => s.status === "em_complementacao")?.total ?? 0;

  function filtrar(mudar: () => void) {
    mudar();
    setPagina(1);
  }

  function trocarVisao(proxima: "lista" | "kanban") {
    router.replace(proxima === "kanban" ? "/protocolos?vista=kanban" : "/protocolos");
  }

  function abrir(p: PedidoResumo) {
    router.push(`/protocolos/${p.id}`);
  }

  async function avancar(p: PedidoResumo) {
    setMenu(null);
    const ok = await run(
      () => apiClient.post(`v1/protocolos/pedidos/${p.id}/avancar`, {}),
      "Protocolo avançou de etapa",
    );
    if (ok) void lista.mutate();
  }

  function escolher(p: PedidoResumo, tipo: AcaoAberta["tipo"]) {
    setMenu(null);
    setAcao({ tipo, pedidoId: p.id, numero: p.numero });
  }

  return (
    <div className="mx-auto flex w-full max-w-[1400px] flex-col">
      {/* Cabeçalho */}
      <div className="mb-8 flex flex-wrap items-center justify-between gap-4">
        <h1 className="text-3xl font-bold text-gray-800 dark:text-foreground">
          Painel de Protocolos
        </h1>
        <div className="flex flex-wrap items-center gap-2">
          {abre && (
            <Link href="/protocolos/balcao" className={BOTAO_SECUNDARIO}>
              Balcão
            </Link>
          )}
          {gere && (
            <>
              <Link href="/protocolos/servicos" className={BOTAO_SECUNDARIO}>
                Serviços
              </Link>
              <Link href="/protocolos/formularios" className={BOTAO_SECUNDARIO}>
                Formulários
              </Link>
              <Link href="/protocolos/responsaveis" className={BOTAO_SECUNDARIO}>
                Responsáveis
              </Link>
            </>
          )}
          <Link href="/carta" className={BOTAO_SECUNDARIO}>
            Carta pública
          </Link>
          <div
            className="flex gap-2 rounded-lg bg-slate-100 p-1 dark:bg-white/10"
            role="group"
            aria-label="Forma de ver os protocolos"
          >
            {(["lista", "kanban"] as const).map((v) => (
              <button
                key={v}
                type="button"
                aria-pressed={visao === v}
                onClick={() => trocarVisao(v)}
                className={`rounded-md px-4 py-2 text-sm font-medium transition-all ${
                  visao === v
                    ? "bg-white text-blue-500 shadow-[0_1px_3px_rgba(0,0,0,0.1)] dark:bg-surface"
                    : "text-slate-500 dark:text-muted"
                }`}
              >
                {v === "lista" ? "Lista" : "Kanban"}
              </button>
            ))}
          </div>
        </div>
      </div>

      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={false}
      >
        {/* Estatísticas */}
        <div className="mb-8 grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-4">
          <Estatistica
            titulo="Total de Protocolos"
            valor={pedidos.length}
            fundo="bg-gradient-to-br from-[#667eea] to-[#764ba2]"
          />
          {porStatus.map((s) => (
            <Estatistica
              key={s.status}
              titulo={rotuloDoStatus(s.status)}
              valor={s.total}
              fundo={corDaEstatistica(s.status)}
            />
          ))}
        </div>

        {/* Faixa de complementação */}
        {complementacao > 0 && (
          <div className="my-5 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-gradient-to-br from-[#f093fb] to-[#f5576c] px-5 py-4 text-white shadow-[0_4px_15px_rgba(240,147,251,0.3)]">
            <div className="flex items-center gap-3">
              <span className="flex h-10 w-10 items-center justify-center rounded-full bg-white/20">
                <AlertTriangle className="h-5 w-5" aria-hidden />
              </span>
              <div>
                <p className="text-base font-bold">
                  {complementacao} protocolo(s) aguardando complementação
                </p>
                <p className="text-sm opacity-90">
                  Estes protocolos precisam de documentos ou informações adicionais do cidadão
                </p>
              </div>
            </div>
            <button
              type="button"
              onClick={() => filtrar(() => setStatus("em_complementacao"))}
              className="rounded-lg border border-white/30 bg-white/20 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-white/30"
            >
              Ver Protocolos
            </button>
          </div>
        )}

        {/* Abas de prazo */}
        <div
          className="mb-6 mt-4 flex gap-6 overflow-x-auto border-b-2 border-gray-100 dark:border-surface-border"
          role="tablist"
          aria-label="Situação do prazo"
        >
          {ABAS_PRAZO.map((aba) => {
            const ativa = prazo === aba.value;
            return (
              <button
                key={aba.value}
                type="button"
                role="tab"
                aria-selected={ativa}
                onClick={() => filtrar(() => setPrazo(aba.value))}
                className={`relative -bottom-0.5 flex shrink-0 items-center gap-2 border-b-[3px] px-1 py-3 text-sm transition-all hover:text-blue-600 ${
                  ativa
                    ? "border-blue-600 font-semibold text-blue-600"
                    : "border-transparent font-medium text-gray-500 dark:text-muted"
                }`}
              >
                {aba.label}
                <span
                  className={`rounded-full px-2 py-px text-[0.7rem] font-semibold ${
                    ativa
                      ? "bg-blue-100 text-blue-600"
                      : "bg-gray-100 text-gray-500 dark:bg-white/10 dark:text-muted"
                  }`}
                >
                  {contagemPrazo[aba.value] ?? 0}
                </span>
              </button>
            );
          })}
        </div>

        {/* Filtros */}
        <div className="mb-8 rounded-xl border border-gray-200 bg-surface p-6 shadow-[0_1px_3px_rgba(0,0,0,0.1)] dark:border-surface-border">
          <div className="grid items-center gap-4 md:grid-cols-[1fr_1fr_1fr_auto]">
            <div className="relative md:col-span-4">
              <Search
                className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-500"
                aria-hidden
              />
              <input
                type="search"
                aria-label="Buscar"
                placeholder="Buscar por protocolo, CPF ou nome..."
                value={busca}
                onChange={(e) => filtrar(() => setBusca(e.target.value))}
                className="w-full rounded-lg border border-gray-300 bg-surface py-3.5 pl-11 pr-4 text-base text-foreground transition-colors focus:border-blue-500 focus:outline-none focus:ring-[3px] focus:ring-blue-500/10"
              />
            </div>
            <select
              aria-label="Serviço"
              value={servico}
              onChange={(e) => filtrar(() => setServico(e.target.value))}
              className={CAMPO}
            >
              <option value="">Todos os Serviços</option>
              {servicos.map(([id, nome]) => (
                <option key={id} value={id}>
                  {nome}
                </option>
              ))}
            </select>
            <select
              aria-label="Status"
              value={status}
              onChange={(e) => filtrar(() => setStatus(e.target.value))}
              className={CAMPO}
            >
              <option value="">Todos os Status</option>
              {STATUS.map((s) => (
                <option key={s.value} value={s.value}>
                  {rotuloDoStatus(s.value)}
                </option>
              ))}
            </select>
            <select
              aria-label="Prioridade"
              value={prioridade}
              onChange={(e) => filtrar(() => setPrioridade(e.target.value))}
              className={CAMPO}
            >
              <option value="">Todas as Prioridades</option>
              <option value="baixa">Baixa</option>
              <option value="normal">Normal</option>
              <option value="alta">Alta</option>
              <option value="urgente">Urgente</option>
            </select>
            <button
              type="button"
              onClick={() =>
                filtrar(() => {
                  setBusca("");
                  setServico("");
                  setStatus("");
                  setPrioridade("");
                  setPrazo("todos");
                })
              }
              className="flex items-center justify-center gap-2 rounded-lg bg-blue-500 px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-blue-600"
            >
              <FilterX className="h-4 w-4" aria-hidden />
              Limpar
            </button>
          </div>
        </div>

        {/* Kanban ou lista */}
        {visao === "kanban" && visiveis.length > 0 ? (
          <KanbanProtocolos pedidos={visiveis} onAbrir={abrir} />
        ) : (
          <div className="flex flex-col rounded-xl border border-gray-200 bg-surface shadow-[0_1px_3px_rgba(0,0,0,0.1)] dark:border-surface-border">
            {visiveis.length === 0 ? (
              <p className="p-12 text-center text-gray-500 dark:text-muted">
                Nenhum protocolo encontrado com os filtros aplicados.
              </p>
            ) : (
              <>
                {paginaAtual.itens.map((p) => {
                  const final = FINAIS.has(p.status);
                  const situacao = p.situacao_prazo || "no_prazo";
                  return (
                    <article
                      key={p.id}
                      className={`border-b border-gray-200 p-6 transition-colors last:border-b-0 hover:bg-slate-50 dark:border-surface-border dark:hover:bg-white/5 ${bordaDoPrazo(p.situacao_prazo)}`}
                    >
                      <div className="mb-4 flex flex-wrap items-start justify-between gap-4">
                        <div className="min-w-0 flex-1">
                          <h2 className="mb-1 text-lg font-semibold text-gray-800 dark:text-foreground">
                            Protocolo #{p.numero}
                          </h2>
                          <p className="mb-2 text-sm text-gray-500 dark:text-muted">
                            {p.servico_nome || "Serviço não encontrado"}
                          </p>
                          <div className="flex flex-wrap gap-4 text-xs text-gray-500 dark:text-muted">
                            <span className="flex items-center gap-1">
                              <span className="mr-1 font-semibold text-blue-500">📌 Etapa:</span>
                              {nomeDaEtapa(p.etapa_id)}
                            </span>
                            <span className="flex items-center gap-1">
                              <User className="h-3.5 w-3.5" aria-hidden />
                              {p.nome}
                            </span>
                            <span className="flex items-center gap-1">
                              <Building2 className="h-3.5 w-3.5" aria-hidden />
                              {p.unidade_nome || "Departamento não definido"}
                            </span>
                            <span className="flex items-center gap-1">
                              <Clock className="h-3.5 w-3.5" aria-hidden />
                              {dataCurta(p.created_at)}
                            </span>
                          </div>
                        </div>
                        <div className="flex flex-col items-end gap-1.5">
                          <span
                            className={`rounded-full px-3 py-1 text-xs font-semibold uppercase text-white ${seloDoStatus(p.status)}`}
                          >
                            {rotuloDoStatus(p.status)}
                          </span>
                          <span
                            className={`inline-flex items-center gap-1 rounded-full px-3 py-1 text-xs font-medium ${seloDoPrazo(p.situacao_prazo)}`}
                          >
                            {situacao === "atrasado" ? (
                              <AlertTriangle className="h-3.5 w-3.5" aria-hidden />
                            ) : situacao === "atencao" ? (
                              <Clock className="h-3.5 w-3.5" aria-hidden />
                            ) : (
                              <CheckCircle2 className="h-3.5 w-3.5" aria-hidden />
                            )}
                            {textoDoPrazo(p)}
                          </span>
                          <span
                            className={`rounded-full px-3 py-1 text-xs font-semibold uppercase text-white ${seloDaPrioridade(p.prioridade)}`}
                          >
                            {rotuloDaPrioridade(p.prioridade)}
                          </span>
                        </div>
                      </div>

                      <div className="relative flex items-center gap-2">
                        <button
                          type="button"
                          onClick={() => abrir(p)}
                          className="flex items-center gap-1.5 rounded-md border border-blue-500 bg-blue-500 px-3 py-2 text-xs font-medium text-white transition-colors hover:border-blue-600 hover:bg-blue-600"
                        >
                          <Eye className="h-4 w-4" aria-hidden />
                          Visualizar
                        </button>
                        {trata && !final && (
                          <div className="relative">
                            <button
                              type="button"
                              aria-label={`Ações do protocolo ${p.numero}`}
                              aria-expanded={menu === p.id}
                              onClick={() => setMenu(menu === p.id ? null : p.id)}
                              className="flex items-center rounded-md border border-gray-300 bg-surface p-2 text-gray-700 transition-colors hover:border-gray-400 hover:bg-gray-50 dark:text-foreground dark:hover:bg-white/5"
                            >
                              <MoreVertical className="h-4 w-4" aria-hidden />
                            </button>
                            {menu === p.id && (
                              <>
                                <button
                                  type="button"
                                  aria-label="Fechar menu"
                                  className="fixed inset-0 z-40 cursor-default"
                                  onClick={() => setMenu(null)}
                                />
                                <div
                                  role="menu"
                                  className="absolute left-0 top-full z-50 mt-2 min-w-[220px] overflow-hidden rounded-lg border border-slate-200 bg-white shadow-[0_10px_15px_-3px_rgba(0,0,0,0.1),0_4px_6px_-2px_rgba(0,0,0,0.05)] dark:border-surface-border dark:bg-surface"
                                >
                                  <ItemMenu cor="text-blue-500" onClick={() => abrir(p)}>
                                    Visualizar
                                  </ItemMenu>
                                  {p.etapa_id && p.status !== "em_complementacao" && (
                                    <ItemMenu
                                      cor="text-violet-600"
                                      disabled={pending}
                                      onClick={() => void avancar(p)}
                                    >
                                      Encaminhar para a próxima etapa
                                    </ItemMenu>
                                  )}
                                  <ItemMenu cor="text-cyan-600" onClick={() => escolher(p, "nota")}>
                                    Adicionar nota
                                  </ItemMenu>
                                  {p.status === "em_complementacao" ? (
                                    <ItemMenu
                                      cor="text-emerald-600"
                                      onClick={() => escolher(p, "resposta")}
                                    >
                                      Registrar resposta
                                    </ItemMenu>
                                  ) : (
                                    <ItemMenu
                                      cor="text-orange-600"
                                      onClick={() => escolher(p, "complemento")}
                                    >
                                      Solicitar complementação
                                    </ItemMenu>
                                  )}
                                  {p.status !== "em_complementacao" && (
                                    <ItemMenu
                                      cor="text-red-600"
                                      onClick={() => escolher(p, "concluir")}
                                    >
                                      Concluir
                                    </ItemMenu>
                                  )}
                                  <ItemMenu
                                    cor="text-red-600"
                                    onClick={() => escolher(p, "cancelar")}
                                  >
                                    Cancelar
                                  </ItemMenu>
                                </div>
                              </>
                            )}
                          </div>
                        )}
                      </div>
                    </article>
                  );
                })}
                {paginaAtual.paginas > 1 && (
                  <nav
                    aria-label="Páginas"
                    className="flex flex-wrap items-center justify-center gap-2 p-6"
                  >
                    <BotaoPagina
                      disabled={paginaAtual.pagina <= 1}
                      onClick={() => setPagina(paginaAtual.pagina - 1)}
                    >
                      Anterior
                    </BotaoPagina>
                    {Array.from({ length: paginaAtual.paginas }, (_, i) => i + 1).map((n) => (
                      <BotaoPagina
                        key={n}
                        ativa={n === paginaAtual.pagina}
                        onClick={() => setPagina(n)}
                      >
                        {n}
                      </BotaoPagina>
                    ))}
                    <BotaoPagina
                      disabled={paginaAtual.pagina >= paginaAtual.paginas}
                      onClick={() => setPagina(paginaAtual.pagina + 1)}
                    >
                      Próxima
                    </BotaoPagina>
                  </nav>
                )}
              </>
            )}
          </div>
        )}
      </DataState>

      <ModalDeAcao acao={acao} onFechar={() => setAcao(null)} onFeito={() => void lista.mutate()} />
    </div>
  );
}

function Estatistica({ titulo, valor, fundo }: { titulo: string; valor: number; fundo: string }) {
  return (
    <div
      className={`relative overflow-hidden rounded-2xl p-6 shadow-[0_8px_32px_rgba(0,0,0,0.1)] transition hover:-translate-y-0.5 hover:shadow-[0_12px_40px_rgba(0,0,0,0.15)] ${fundo}`}
    >
      <span
        className="pointer-events-none absolute inset-0 bg-gradient-to-br from-white/10 to-white/0"
        aria-hidden
      />
      <p className="relative mb-2 text-sm font-medium text-white/90">{titulo}</p>
      <p className="relative text-[2rem] font-bold text-white [text-shadow:0_2px_4px_rgba(0,0,0,0.1)]">
        {valor}
      </p>
    </div>
  );
}

function ItemMenu({
  cor,
  disabled,
  onClick,
  children,
}: {
  cor: string;
  disabled?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      disabled={disabled}
      onClick={onClick}
      className={`flex w-full items-center gap-3 px-4 py-3 text-left text-sm font-medium transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-white/5 ${cor}`}
    >
      {children}
    </button>
  );
}

function BotaoPagina({
  ativa,
  disabled,
  onClick,
  children,
}: {
  ativa?: boolean;
  disabled?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      aria-current={ativa ? "page" : undefined}
      onClick={onClick}
      className={`rounded-md border px-4 py-2 text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
        ativa
          ? "border-blue-500 bg-blue-500 text-white"
          : "border-gray-300 bg-surface text-gray-700 hover:border-gray-400 hover:bg-gray-50 dark:text-foreground dark:hover:bg-white/5"
      }`}
    >
      {children}
    </button>
  );
}
