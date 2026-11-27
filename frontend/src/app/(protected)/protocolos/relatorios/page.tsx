"use client";

import Link from "next/link";
import { useMemo, useState } from "react";

import { ArrowLeft, BarChart3, Download } from "lucide-react";

import { DataState } from "@/components/plataforma/DataState";
import { dataCurta } from "@/components/protocolos/estilo";
import { hojeEmCuiaba } from "@/components/protocolos/HorarioLivre";
import {
  agrupar,
  atrasado,
  baixarCsv,
  csv,
  dias,
  noPeriodo,
  resumo,
  type Linha,
  type PedidoDoRelatorio,
} from "@/components/protocolos/relatorios";
import {
  BOTAO_SECUNDARIO,
  CabecalhoDaTela,
  CAMPO,
  CARTAO,
  Rotulado,
  TEXTO_FORTE,
  TEXTO_FRACO,
  Vazio,
} from "@/components/protocolos/ui";
import { useApiQuery } from "@/lib/api/swr";

const COLUNAS = [
  "Nome",
  "Pedidos",
  "Em aberto",
  "Atendidos",
  "Cancelados",
  "Atrasados",
  "Tempo médio (dias)",
];

function linhaCsv(l: Linha): (string | number)[] {
  return [
    l.nome,
    l.total,
    l.abertos,
    l.atendidos,
    l.cancelados,
    l.atrasados,
    l.tempoMedio === null ? "" : l.tempoMedio.toFixed(1).replace(".", ","),
  ];
}

export default function RelatoriosPage() {
  const hoje = hojeEmCuiaba();
  const [de, setDe] = useState(`${hoje.slice(0, 4)}-01-01`);
  const [ate, setAte] = useState(hoje);
  const lista = useApiQuery<PedidoDoRelatorio[]>("v1/protocolos/pedidos");

  const pedidos = useMemo(() => noPeriodo(lista.data ?? [], de, ate), [lista.data, de, ate]);
  const total = useMemo(() => resumo(pedidos), [pedidos]);
  const porServico = useMemo(() => agrupar(pedidos, "servico"), [pedidos]);
  const porUnidade = useMemo(() => agrupar(pedidos, "unidade"), [pedidos]);
  const atrasados = useMemo(
    () =>
      pedidos.filter(atrasado).sort((a, b) => (a.prazo_em ?? "").localeCompare(b.prazo_em ?? "")),
    [pedidos],
  );
  const sufixo = `${de || "inicio"}_a_${ate || "hoje"}`;

  return (
    <div className="mx-auto w-full max-w-[1400px]">
      <CabecalhoDaTela
        icone={BarChart3}
        titulo="Relatórios"
        descricao="Pedidos abertos no período, dentro da sua visão. O tempo médio vai da abertura ao encerramento, em dias corridos, e não conta os cancelados."
      >
        <Link href="/protocolos" className={BOTAO_SECUNDARIO}>
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Voltar à mesa
        </Link>
      </CabecalhoDaTela>

      <div className={`${CARTAO} mb-6 flex flex-wrap items-end gap-4 p-5`}>
        <Rotulado id="de" rotulo="Abertos de">
          <input
            id="de"
            type="date"
            value={de}
            max={ate || undefined}
            onChange={(e) => setDe(e.target.value)}
            className={CAMPO}
          />
        </Rotulado>
        <Rotulado id="ate" rotulo="até">
          <input
            id="ate"
            type="date"
            value={ate}
            min={de || undefined}
            onChange={(e) => setAte(e.target.value)}
            className={CAMPO}
          />
        </Rotulado>
      </div>

      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={false}
      >
        <dl className="mb-6 grid grid-cols-2 gap-4 md:grid-cols-5">
          <Numero rotulo="Pedidos" valor={String(total.total)} />
          <Numero rotulo="Em aberto" valor={String(total.abertos)} />
          <Numero rotulo="Atendidos" valor={String(total.atendidos)} />
          <Numero rotulo="Atrasados" valor={String(total.atrasados)} alerta={total.atrasados > 0} />
          <Numero rotulo="Tempo médio" valor={dias(total.tempoMedio)} />
        </dl>

        {pedidos.length === 0 ? (
          <Vazio titulo="Nenhum pedido" texto="Nenhum pedido aberto nesse período." />
        ) : (
          <div className="flex flex-col gap-6">
            <Tabela
              titulo="Por serviço"
              linhas={porServico}
              onCsv={() =>
                baixarCsv(
                  `protocolos_por_servico_${sufixo}.csv`,
                  csv(COLUNAS, porServico.map(linhaCsv)),
                )
              }
            />
            <Tabela
              titulo="Por unidade"
              linhas={porUnidade}
              onCsv={() =>
                baixarCsv(
                  `protocolos_por_unidade_${sufixo}.csv`,
                  csv(COLUNAS, porUnidade.map(linhaCsv)),
                )
              }
            />

            <section className={`${CARTAO} p-5`}>
              <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                <h2 className={`text-lg font-semibold ${TEXTO_FORTE}`}>
                  Atrasados ({atrasados.length})
                </h2>
                <button
                  type="button"
                  disabled={atrasados.length === 0}
                  className={BOTAO_SECUNDARIO}
                  onClick={() =>
                    baixarCsv(
                      `protocolos_atrasados_${sufixo}.csv`,
                      csv(
                        ["Número", "Requerente", "Serviço", "Unidade", "Responsável", "Prazo"],
                        atrasados.map((p) => [
                          p.numero,
                          p.nome,
                          p.servico_nome,
                          p.unidade_nome ?? "",
                          p.responsavel_nome ?? "",
                          p.prazo_em ? dataCurta(p.prazo_em) : "",
                        ]),
                      ),
                    )
                  }
                >
                  <Download className="h-4 w-4" aria-hidden />
                  Baixar CSV
                </button>
              </div>
              {atrasados.length === 0 ? (
                <p className={`text-sm ${TEXTO_FRACO}`}>Nenhum pedido atrasado. 🎉</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[640px] text-left text-sm">
                    <thead className={`text-xs uppercase ${TEXTO_FRACO}`}>
                      <tr>
                        <th className="py-2 pr-3 font-medium">Número</th>
                        <th className="py-2 pr-3 font-medium">Serviço</th>
                        <th className="py-2 pr-3 font-medium">Unidade</th>
                        <th className="py-2 pr-3 font-medium">Responsável</th>
                        <th className="py-2 font-medium">Prazo</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100 dark:divide-slate-700">
                      {atrasados.map((p) => (
                        <tr key={p.id} className="text-gray-700 dark:text-slate-200">
                          <td className="py-2 pr-3">
                            <Link
                              href={`/protocolos/${p.id}`}
                              className="font-mono font-semibold text-blue-600 hover:underline dark:text-sky-400"
                            >
                              {p.numero}
                            </Link>
                          </td>
                          <td className="py-2 pr-3">{p.servico_nome}</td>
                          <td className="py-2 pr-3">{p.unidade_nome}</td>
                          <td className="py-2 pr-3">{p.responsavel_nome || "Sem responsável"}</td>
                          <td className="py-2 font-medium text-red-600 dark:text-red-400">
                            {p.prazo_em ? dataCurta(p.prazo_em) : "—"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </section>
          </div>
        )}
      </DataState>
    </div>
  );
}

function Numero({ rotulo, valor, alerta }: { rotulo: string; valor: string; alerta?: boolean }) {
  return (
    <div className={`${CARTAO} p-4`}>
      <dt className={`text-xs font-medium uppercase ${TEXTO_FRACO}`}>{rotulo}</dt>
      <dd
        className={`mt-1 text-2xl font-bold ${alerta ? "text-red-600 dark:text-red-400" : TEXTO_FORTE}`}
      >
        {valor}
      </dd>
    </div>
  );
}

function Tabela({ titulo, linhas, onCsv }: { titulo: string; linhas: Linha[]; onCsv: () => void }) {
  return (
    <section className={`${CARTAO} p-5`}>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h2 className={`text-lg font-semibold ${TEXTO_FORTE}`}>{titulo}</h2>
        <button type="button" className={BOTAO_SECUNDARIO} onClick={onCsv}>
          <Download className="h-4 w-4" aria-hidden />
          Baixar CSV
        </button>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[640px] text-left text-sm" aria-label={titulo}>
          <thead className={`text-xs uppercase ${TEXTO_FRACO}`}>
            <tr>
              <th className="py-2 pr-3 font-medium">Nome</th>
              <th className="py-2 pr-3 text-right font-medium">Pedidos</th>
              <th className="py-2 pr-3 text-right font-medium">Em aberto</th>
              <th className="py-2 pr-3 text-right font-medium">Atendidos</th>
              <th className="py-2 pr-3 text-right font-medium">Cancelados</th>
              <th className="py-2 pr-3 text-right font-medium">Atrasados</th>
              <th className="py-2 text-right font-medium">Tempo médio</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100 tabular-nums dark:divide-slate-700">
            {linhas.map((l) => (
              <tr key={l.chave} className="text-gray-700 dark:text-slate-200">
                <td className="py-2 pr-3 font-medium">{l.nome}</td>
                <td className="py-2 pr-3 text-right">{l.total}</td>
                <td className="py-2 pr-3 text-right">{l.abertos}</td>
                <td className="py-2 pr-3 text-right">{l.atendidos}</td>
                <td className="py-2 pr-3 text-right">{l.cancelados}</td>
                <td
                  className={`py-2 pr-3 text-right ${l.atrasados > 0 ? "font-semibold text-red-600 dark:text-red-400" : ""}`}
                >
                  {l.atrasados}
                </td>
                <td className="py-2 text-right">{dias(l.tempoMedio)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
