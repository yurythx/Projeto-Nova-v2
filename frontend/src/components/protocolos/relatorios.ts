// Contas dos relatórios do protocolo, feitas sobre a lista da mesa (a visão da pessoa).

import { FINAIS, type PedidoResumo } from "./estilo";

/** Linha da mesa com a data da última mudança: no pedido encerrado, é o encerramento. */
export type PedidoDoRelatorio = PedidoResumo & { updated_at?: string };

const DIA = 24 * 60 * 60 * 1000;

/** Encerrado com decisão: o cancelado sai da conta do tempo de atendimento. */
const ATENDIDOS = new Set(["concluido", "aprovado", "rejeitado"]);

export interface Linha {
  chave: string;
  nome: string;
  total: number;
  abertos: number;
  atendidos: number;
  cancelados: number;
  atrasados: number;
  /** Dias corridos, da abertura ao encerramento; null sem pedido atendido. */
  tempoMedio: number | null;
}

/** Dia (AAAA-MM-DD) em Cuiabá. */
export function diaDe(iso: string): string {
  return new Date(iso).toLocaleDateString("en-CA", { timeZone: "America/Cuiaba" });
}

/** Pedidos abertos entre "de" e "ate" (inclusive, AAAA-MM-DD); vazio não limita. */
export function noPeriodo(pedidos: PedidoDoRelatorio[], de: string, ate: string) {
  return pedidos.filter((p) => {
    const dia = diaDe(p.created_at);
    return (!de || dia >= de) && (!ate || dia <= ate);
  });
}

export function atrasado(p: PedidoResumo): boolean {
  return !FINAIS.has(p.status) && p.situacao_prazo === "atrasado";
}

/** Dias corridos do atendimento, ou null se o pedido não foi atendido. */
export function diasDeAtendimento(p: PedidoDoRelatorio): number | null {
  if (!ATENDIDOS.has(p.status) || !p.updated_at) return null;
  const dias = (new Date(p.updated_at).getTime() - new Date(p.created_at).getTime()) / DIA;
  return dias < 0 ? 0 : dias;
}

export function resumo(pedidos: PedidoDoRelatorio[], chave = "", nome = "Total"): Linha {
  const tempos = pedidos.map(diasDeAtendimento).filter((d): d is number => d !== null);
  return {
    chave,
    nome,
    total: pedidos.length,
    abertos: pedidos.filter((p) => !FINAIS.has(p.status)).length,
    atendidos: tempos.length,
    cancelados: pedidos.filter((p) => p.status === "cancelado").length,
    atrasados: pedidos.filter(atrasado).length,
    tempoMedio: tempos.length ? tempos.reduce((a, b) => a + b, 0) / tempos.length : null,
  };
}

/** Uma linha por serviço ou por unidade, da mais pedida para a menos. */
export function agrupar(pedidos: PedidoDoRelatorio[], por: "servico" | "unidade"): Linha[] {
  const grupos = new Map<string, { nome: string; itens: PedidoDoRelatorio[] }>();
  for (const p of pedidos) {
    const chave = (por === "servico" ? p.servico_id : p.unidade_id) ?? "";
    const nome = (por === "servico" ? p.servico_nome : p.unidade_nome) || "Sem identificação";
    const grupo = grupos.get(chave) ?? { nome, itens: [] };
    grupo.itens.push(p);
    grupos.set(chave, grupo);
  }
  return [...grupos.entries()]
    .map(([chave, g]) => resumo(g.itens, chave, g.nome))
    .sort((a, b) => b.total - a.total || a.nome.localeCompare(b.nome, "pt-BR"));
}

/** Dias com uma casa decimal, ou travessão. */
export function dias(valor: number | null): string {
  if (valor === null) return "—";
  return `${valor.toLocaleString("pt-BR", { maximumFractionDigits: 1, minimumFractionDigits: 1 })} d`;
}

function celula(valor: string | number): string {
  const texto = String(valor);
  // Fórmula no início vira texto no Excel (injeção de CSV).
  const seguro = /^[=+\-@\t\r]/.test(texto) ? `'${texto}` : texto;
  return /[";\n\r]/.test(seguro) ? `"${seguro.replace(/"/g, '""')}"` : seguro;
}

/** CSV com ponto e vírgula e BOM, que o Excel em português abre direto. */
export function csv(cabecalho: string[], linhas: (string | number)[][]): string {
  return "﻿" + [cabecalho, ...linhas].map((l) => l.map(celula).join(";")).join("\r\n");
}

export function baixarCsv(nome: string, conteudo: string) {
  const url = URL.createObjectURL(new Blob([conteudo], { type: "text/csv;charset=utf-8" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = nome;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
