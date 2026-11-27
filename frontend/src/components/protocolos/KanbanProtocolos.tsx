"use client";

import { CalendarDays, ChevronLeft, ChevronRight, MapPin, UserCheck } from "lucide-react";
import { useState } from "react";

import {
  bordaDaPrioridade,
  COLUNAS_KANBAN,
  colunaDoKanban,
  dataCurta,
  seloPequenoDaPrioridade,
  type PedidoResumo,
} from "./estilo";

const VISIVEIS = 3;

/** Kanban da mesa, como o KanbanBoardExternal do Protocolo Digital. */
export function KanbanProtocolos({
  pedidos,
  onAbrir,
}: {
  pedidos: PedidoResumo[];
  onAbrir: (p: PedidoResumo) => void;
}) {
  const [inicio, setInicio] = useState(0);
  const colunas = COLUNAS_KANBAN.slice(inicio, inicio + VISIVEIS);
  const temAnterior = inicio > 0;
  const temProxima = inicio + VISIVEIS < COLUNAS_KANBAN.length;

  return (
    <div className="relative flex items-center" data-testid="protocolos-kanban">
      {temAnterior && (
        <button
          type="button"
          aria-label="Colunas anteriores"
          onClick={() => setInicio((atual) => Math.max(0, atual - 1))}
          className="absolute -left-5 z-10 flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-blue-500 shadow-[0_2px_4px_rgba(0,0,0,0.1)] dark:border-slate-600 dark:bg-slate-800 dark:text-sky-400"
        >
          <ChevronLeft className="h-6 w-6" aria-hidden />
        </button>
      )}

      <div className="flex min-h-[60vh] flex-1 gap-6 overflow-x-auto px-2 py-4">
        {colunas.map((coluna) => {
          const cartoes = pedidos.filter((p) => colunaDoKanban(p.status) === coluna.id);
          return (
            <section
              key={coluna.id}
              aria-label={coluna.titulo}
              className="flex max-h-full flex-[0_0_350px] flex-col rounded-xl border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900/50"
            >
              <header
                className={`flex items-center justify-between rounded-t-xl border-b-2 bg-white px-5 py-4 dark:bg-surface ${coluna.borda}`}
              >
                <h3 className="flex items-center gap-2 text-base font-bold text-slate-800 dark:text-slate-100">
                  <span className={`h-3 w-3 rounded-full ${coluna.cor}`} aria-hidden />
                  {coluna.titulo}
                </h3>
                <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-500 dark:bg-slate-700 dark:text-slate-400">
                  {cartoes.length}
                </span>
              </header>
              <div className="flex flex-1 flex-col gap-4 overflow-y-auto p-4">
                {cartoes.map((p) => (
                  <button
                    key={p.id}
                    type="button"
                    onClick={() => onAbrir(p)}
                    className={`rounded-lg border border-l-4 border-slate-200 bg-white p-4 text-left shadow-[0_1px_3px_rgba(0,0,0,0.1)] transition hover:-translate-y-0.5 hover:border-slate-300 hover:shadow-md dark:hover:border-slate-500 dark:border-slate-700 dark:bg-surface ${bordaDaPrioridade(p.prioridade)}`}
                  >
                    <div className="mb-2 flex items-start justify-between gap-2">
                      <span className="rounded bg-indigo-100 px-2 py-0.5 text-xs font-bold text-indigo-500 dark:bg-indigo-500/20 dark:text-indigo-300">
                        {p.numero}
                      </span>
                      <span
                        className={`rounded px-1.5 py-0.5 text-[0.7rem] font-semibold uppercase ${seloPequenoDaPrioridade(p.prioridade)}`}
                      >
                        {p.prioridade || "Normal"}
                      </span>
                    </div>
                    <h4 className="mb-2 line-clamp-2 text-sm font-semibold leading-snug text-slate-900 dark:text-slate-100">
                      {p.servico_nome}
                    </h4>
                    <div className="mt-3 flex flex-col gap-1 border-t border-slate-100 pt-3 dark:border-slate-700">
                      <span
                        className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400"
                        title="Origem"
                      >
                        <MapPin className="h-3.5 w-3.5 shrink-0" aria-hidden />
                        <span className="truncate">{p.nome || "Cidadão"}</span>
                      </span>
                      <span
                        className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400"
                        title="Data de Criação"
                      >
                        <CalendarDays className="h-3.5 w-3.5 shrink-0" aria-hidden />
                        {dataCurta(p.created_at)}
                      </span>
                      <span
                        className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400"
                        title="Responsável"
                      >
                        <UserCheck className="h-3.5 w-3.5 shrink-0" aria-hidden />
                        <span className="truncate">
                          {p.responsavel_nome || "Aguardando aceite"}
                        </span>
                      </span>
                      {p.encaminhamento && (
                        <span className="text-xs font-medium text-amber-700 dark:text-amber-300">
                          ⏳ Encaminhado para {p.encaminhamento.unidade_nome || "outra unidade"}
                        </span>
                      )}
                    </div>
                  </button>
                ))}
                {cartoes.length === 0 && (
                  <p className="px-4 py-8 text-center text-sm text-slate-400 dark:text-slate-500">
                    Nenhum protocolo.
                  </p>
                )}
              </div>
            </section>
          );
        })}
      </div>

      {temProxima && (
        <button
          type="button"
          aria-label="Próximas colunas"
          onClick={() => setInicio((atual) => atual + 1)}
          className="absolute -right-5 z-10 flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 bg-white text-blue-500 shadow-[0_2px_4px_rgba(0,0,0,0.1)] dark:border-slate-600 dark:bg-slate-800 dark:text-sky-400"
        >
          <ChevronRight className="h-6 w-6" aria-hidden />
        </button>
      )}
    </div>
  );
}
