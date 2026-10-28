"use client";

import { type FormEvent, useId, useState } from "react";

import { PrazoBadge } from "@/components/atendimento/PrazoBadge";
import { Pagination } from "@/components/plataforma/Pagination";
import { fmtDateTime } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from "@/components/ui/Table";
import type { PageMeta } from "@/lib/plataforma/types";
import {
  ESTADOS_FILA,
  PRIORIDADE,
  PRIORIDADES,
  rotuloEstado,
  rotuloTipo,
  type Solicitacao,
  tomEstado,
} from "@/lib/atendimento/atendimento";
import { gravarArmazenado, useArmazenado } from "@/lib/prefs/armazenado";

// A lista de chamados como a do GLPI (ADR 080): a busca sempre à mão, os
// demais filtros num painel recolhido e as colunas que ordenam. Os
// filtros moram na URL: o link leva a mesma lista.

/** Os filtros da lista, como ficam na URL. Vazio não filtra. */
export interface FiltrosLista {
  q: string;
  situacao: string;
  estado: string;
  tipo: string;
  responsavel: string;
  unidade: string;
  categoria: string;
  prioridade: string;
  de: string;
  ate: string;
  ordem: string;
  sentido: string;
  pagina: string;
  por: string;
}

const CHAVES: (keyof FiltrosLista)[] = [
  "q",
  "situacao",
  "estado",
  "tipo",
  "responsavel",
  "unidade",
  "categoria",
  "prioridade",
  "de",
  "ate",
  "ordem",
  "sentido",
  "pagina",
  "por",
];

/** Os filtros do painel: contam no botão e somem no "Limpar". */
const DO_PAINEL: (keyof FiltrosLista)[] = [
  "situacao",
  "estado",
  "tipo",
  "responsavel",
  "unidade",
  "categoria",
  "prioridade",
  "de",
  "ate",
];

export const POR_PAGINA = ["20", "50", "100"];

export function lerFiltros(params: URLSearchParams): FiltrosLista {
  const f = {} as FiltrosLista;
  for (const chave of CHAVES) f[chave] = params.get(chave) ?? "";
  return f;
}

/** Quantos filtros do painel estão ligados. */
export function filtrosAtivos(f: FiltrosLista) {
  return DO_PAINEL.filter((chave) => f[chave] !== "").length;
}

/** O começo do dia (YYYY-MM-DD) no fuso de quem olha, como instante. */
function inicioDoDia(dia: string, somar = 0) {
  const [a, m, d] = dia.split("-").map(Number);
  return new Date(a!, m! - 1, d! + somar).toISOString();
}

/** Os parâmetros da API para os filtros: o período vira [de, ate+1d). */
export function consultaDaLista(f: FiltrosLista) {
  return {
    q: f.q.trim() || undefined,
    situacao: f.situacao || undefined,
    estado: f.estado || undefined,
    tipo: f.tipo || undefined,
    responsavel: f.responsavel || undefined,
    unidade: f.unidade || undefined,
    categoria: f.categoria || undefined,
    prioridade: f.prioridade || undefined,
    aberta_de: f.de ? inicioDoDia(f.de) : undefined,
    aberta_ate: f.ate ? inicioDoDia(f.ate, 1) : undefined,
    ordem: f.ordem || "atualizada",
    sentido: f.ordem ? f.sentido || "asc" : "desc",
    page: Number(f.pagina) || 1,
    page_size: POR_PAGINA.includes(f.por) ? Number(f.por) : 50,
  };
}

type Opcao = { value: string; label: string };

const PAINEL_ABERTO = "nova-lista-chamados:filtros";

/**
 * A busca e o painel de filtros, recolhido por padrão como no GLPI. mudar
 * recebe só o que mudou; a página volta à primeira.
 */
export function FiltrosDaLista({
  filtros: f,
  unidades,
  categorias,
  mudar,
}: {
  filtros: FiltrosLista;
  unidades: Opcao[];
  categorias: Opcao[];
  mudar: (novos: Partial<FiltrosLista>) => void;
}) {
  const painelId = useId();
  // Recolhido por padrão; quem abre o painel o encontra aberto da próxima vez.
  const aberto = useArmazenado(PAINEL_ABERTO) === "1";
  const [busca, setBusca] = useState(f.q);
  // A URL manda: voltar no navegador traz a busca de antes.
  const [qVisto, setQVisto] = useState(f.q);
  if (qVisto !== f.q) {
    setQVisto(f.q);
    setBusca(f.q);
  }
  const ativos = filtrosAtivos(f);

  function alternar() {
    gravarArmazenado(PAINEL_ABERTO, aberto ? null : "1");
  }

  function buscar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    mudar({ q: busca.trim() });
  }

  function limpar() {
    const vazio: Partial<FiltrosLista> = {};
    for (const chave of DO_PAINEL) vazio[chave] = "";
    mudar(vazio);
  }

  const campo = (chave: keyof FiltrosLista) => ({
    value: f[chave],
    onChange: (e: { target: { value: string } }) => mudar({ [chave]: e.target.value }),
  });

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <form role="search" onSubmit={buscar} className="flex min-w-0 flex-1 gap-2">
          <label htmlFor="lista-busca" className="sr-only">
            Buscar chamados
          </label>
          <input
            id="lista-busca"
            type="search"
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
            placeholder="Buscar por número, título ou descrição"
            className="min-w-0 flex-1 rounded-lg border border-surface-border bg-surface px-3 py-2 text-sm"
          />
          <Button type="submit" variant="secondary">
            Buscar
          </Button>
        </form>
        <Button
          type="button"
          variant="secondary"
          aria-expanded={aberto}
          aria-controls={painelId}
          onClick={alternar}
        >
          Filtros{ativos > 0 ? ` (${ativos})` : ""} {aberto ? "▴" : "▾"}
        </Button>
        {ativos > 0 && (
          <Button type="button" variant="ghost" onClick={limpar}>
            Limpar filtros
          </Button>
        )}
      </div>
      <div
        id={painelId}
        hidden={!aberto}
        className="grid gap-3 rounded-lg border border-surface-border p-3 sm:grid-cols-2 lg:grid-cols-4"
      >
        <Select
          id="lista-situacao"
          label="Situação"
          {...campo("situacao")}
          options={[
            { value: "", label: "Em andamento" },
            { value: "todas", label: "Todas, com as encerradas" },
          ]}
        />
        <Select
          id="lista-estado"
          label="Status"
          {...campo("estado")}
          options={[
            { value: "", label: "Todos" },
            ...ESTADOS_FILA.map((v) => ({ value: v, label: rotuloEstado(v) })),
          ]}
        />
        <Select
          id="lista-tipo"
          label="Tipo"
          {...campo("tipo")}
          options={[
            { value: "", label: "Todos" },
            { value: "incidente", label: rotuloTipo("incidente") },
            { value: "requisicao", label: rotuloTipo("requisicao") },
          ]}
        />
        <Select
          id="lista-responsavel"
          label="Técnico"
          {...campo("responsavel")}
          options={[
            { value: "", label: "Todos" },
            { value: "eu", label: "Comigo" },
            { value: "nenhum", label: "Sem responsável" },
          ]}
        />
        <Select
          id="lista-unidade"
          label="Unidade"
          {...campo("unidade")}
          options={[{ value: "", label: "Todas" }, ...unidades]}
        />
        <Select
          id="lista-categoria"
          label="Categoria"
          {...campo("categoria")}
          options={[{ value: "", label: "Todas" }, ...categorias]}
        />
        <Select
          id="lista-prioridade"
          label="Prioridade"
          {...campo("prioridade")}
          options={[
            { value: "", label: "Todas" },
            ...PRIORIDADES.map((p) => ({ value: p, label: PRIORIDADE[p].label })),
          ]}
        />
        <div className="grid grid-cols-2 gap-2">
          <Input id="lista-de" type="date" label="Aberta de" {...campo("de")} />
          <Input id="lista-ate" type="date" label="até" {...campo("ate")} />
        </div>
      </div>
    </div>
  );
}

const COLUNAS: { id: string; titulo: string }[] = [
  { id: "numero", titulo: "ID" },
  { id: "titulo", titulo: "Título" },
  { id: "unidade", titulo: "Unidade" },
  { id: "estado", titulo: "Status" },
  { id: "prioridade", titulo: "Prioridade" },
  { id: "atualizada", titulo: "Última atualização" },
  { id: "aberta", titulo: "Abertura" },
  { id: "solicitante", titulo: "Requerente" },
  { id: "responsavel", titulo: "Técnico" },
  { id: "categoria", titulo: "Categoria" },
];

/**
 * A tabela: um clique no título ordena pela coluna, outro inverte. Sem
 * escolha, a mais recém-atualizada vem primeiro, como no GLPI.
 */
export function TabelaDeChamados({
  itens,
  meta,
  filtros: f,
  mudar,
  aoAbrir,
  onAssumir,
  pending,
}: {
  itens: Solicitacao[];
  meta?: PageMeta;
  filtros: FiltrosLista;
  mudar: (novos: Partial<FiltrosLista>) => void;
  aoAbrir: (s: Solicitacao) => void;
  onAssumir: (s: Solicitacao) => void;
  pending: boolean;
}) {
  const ordem = f.ordem || "atualizada";
  const desc = f.ordem ? f.sentido === "desc" : true;
  const total = meta?.total_items ?? itens.length;
  const pagina = meta?.page ?? 1;
  const tamanho = meta?.page_size ?? itens.length;
  const primeiro = total === 0 ? 0 : (pagina - 1) * tamanho + 1;
  const ultimo = Math.min(total, (pagina - 1) * tamanho + itens.length);

  function ordenar(coluna: string) {
    const inverter = coluna === ordem;
    mudar({ ordem: coluna, sentido: inverter && !desc ? "desc" : "asc" });
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center justify-between gap-2 text-sm text-muted">
        <span role="status">
          {total === 0
            ? "Nenhum chamado"
            : `Mostrando ${primeiro}–${ultimo} de ${total.toLocaleString("pt-BR")}`}
        </span>
        <label className="flex items-center gap-2">
          Por página
          <select
            value={POR_PAGINA.includes(f.por) ? f.por : "50"}
            onChange={(e) => mudar({ por: e.target.value })}
            className="rounded-md border border-surface-border bg-surface px-2 py-1"
          >
            {POR_PAGINA.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </label>
      </div>
      {itens.length === 0 ? (
        <p className="text-sm text-muted">Fila vazia</p>
      ) : (
        <Table caption="Solicitações da fila" className="min-w-[1100px]">
          <TableHead>
            <TableRow>
              {COLUNAS.map((c) => {
                const ativa = c.id === ordem;
                return (
                  <TableHeaderCell
                    key={c.id}
                    aria-sort={ativa ? (desc ? "descending" : "ascending") : "none"}
                  >
                    <button
                      type="button"
                      onClick={() => ordenar(c.id)}
                      className="inline-flex items-center gap-1 font-semibold hover:underline"
                    >
                      {c.titulo}
                      <span aria-hidden className={ativa ? "" : "opacity-30"}>
                        {ativa && !desc ? "▲" : "▼"}
                      </span>
                    </button>
                  </TableHeaderCell>
                );
              })}
              <TableHeaderCell>Prazo</TableHeaderCell>
              <TableHeaderCell>
                <span className="sr-only">Ações</span>
              </TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {itens.map((s) => (
              <TableRow key={s.id}>
                <TableCell className="tabular-nums text-muted">#{s.numero}</TableCell>
                <TableCell>
                  <button
                    type="button"
                    className="text-left font-medium text-primary hover:underline"
                    onClick={() => aoAbrir(s)}
                  >
                    {s.titulo}
                  </button>
                  <p className="text-xs text-muted">
                    {rotuloTipo(s.tipo)} · {s.oferta_nome}
                  </p>
                </TableCell>
                <TableCell>{s.unidade_nome || <span className="text-muted">—</span>}</TableCell>
                <TableCell>
                  <Badge tone={tomEstado(s.estado)}>{rotuloEstado(s.estado)}</Badge>
                </TableCell>
                <TableCell>
                  {s.prioridade ? (
                    <Badge tone={PRIORIDADE[s.prioridade].tone}>
                      {PRIORIDADE[s.prioridade].label}
                    </Badge>
                  ) : (
                    <span className="text-muted">—</span>
                  )}
                </TableCell>
                <TableCell className="whitespace-nowrap">{fmtDateTime(s.updated_at)}</TableCell>
                <TableCell className="whitespace-nowrap">{fmtDateTime(s.created_at)}</TableCell>
                <TableCell>{s.solicitante_nome}</TableCell>
                <TableCell>{s.responsavel_nome || <span className="text-muted">—</span>}</TableCell>
                <TableCell>{s.categoria_nome || <span className="text-muted">—</span>}</TableCell>
                <TableCell>
                  <PrazoBadge prazo={s.prazo} />
                </TableCell>
                <TableCell>
                  {!s.responsavel_id && s.estado !== "encerrado" && s.estado !== "cancelado" && (
                    <Button
                      type="button"
                      size="sm"
                      variant="secondary"
                      disabled={pending}
                      onClick={() => onAssumir(s)}
                      aria-label={`Assumir a solicitação #${s.numero}`}
                    >
                      Assumir
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
      <Pagination meta={meta} onPage={(p) => mudar({ pagina: String(p) })} />
    </div>
  );
}
