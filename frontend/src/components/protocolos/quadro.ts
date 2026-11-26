import { colunasEmOrdem, type ColunaFaixa, type TomFaixa } from "@/components/plataforma/Faixas";

import { etapaLegivel, rotuloPrazo, rotuloPrioridade, rotuloStatus } from "./painel";

/** Linha da mesa: o que a listagem de pedidos devolve (o CPF vem mascarado). */
export interface PedidoResumo {
  id: string;
  numero: string;
  nome: string;
  cpf?: string;
  servico_id: string;
  servico_nome: string;
  unidade_id?: string;
  unidade_nome?: string;
  status: string;
  situacao_prazo?: string;
  etapa_id: string;
  prioridade: string;
  prazo_em?: string;
  created_at: string;
}

/**
 * Uma coluna por status, na ordem do fluxo. Pendente e rejeitado só
 * aparecem quando há pedido neles: o fluxo da carta quase não os usa.
 */
export const COLUNAS_PROTOCOLO: ColunaFaixa[] = colunasEmOrdem([
  { id: "novo", titulo: "Novos", tom: "info", entrada: true },
  { id: "em_analise", titulo: "Em análise", tom: "info" },
  { id: "em_andamento", titulo: "Em andamento", tom: "warning" },
  { id: "pendente", titulo: "Pendentes", tom: "neutral", ativa: false },
  { id: "em_complementacao", titulo: "Em complementação", tom: "danger" },
  { id: "pending_approval", titulo: "Aguardando aprovação", tom: "warning" },
  { id: "aprovado", titulo: "Aprovados", tom: "success" },
  { id: "concluido", titulo: "Concluídos", tom: "success" },
  { id: "rejeitado", titulo: "Rejeitados", tom: "danger", ativa: false },
  { id: "cancelado", titulo: "Cancelados", tom: "neutral" },
]);

const TOM_PRIORIDADE: Record<string, TomFaixa> = {
  baixa: "neutral",
  normal: "info",
  alta: "warning",
  urgente: "danger",
};

export function tomPrioridade(prioridade: string): TomFaixa {
  return TOM_PRIORIDADE[prioridade] ?? "neutral";
}

export function tomStatus(status: string): TomFaixa {
  return COLUNAS_PROTOCOLO.find((coluna) => coluna.id === status)?.tom ?? "neutral";
}

/** Semáforo do prazo no cartão e na linha, como o quadro do atendimento. */
export function molduraPrazo(pedido: Pick<PedidoResumo, "situacao_prazo">): string | undefined {
  if (pedido.situacao_prazo === "atrasado")
    return "border-danger/60 border-l-danger bg-danger/[0.02]";
  if (pedido.situacao_prazo === "atencao")
    return "border-warning/60 border-l-warning bg-warning/[0.02]";
  return undefined;
}

/** Dia do prazo no fuso do protocolo (Cuiabá). Vazio quando não há prazo. */
export function diaDoPrazo(iso?: string): string {
  if (!iso) return "";
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return "";
  return data.toLocaleDateString("pt-BR", { timeZone: "America/Cuiaba" });
}

/** Conta os pedidos por uma chave, do maior para o menor. */
export function contarPor<T>(itens: readonly T[], chave: (item: T) => string) {
  const mapa = new Map<string, number>();
  for (const item of itens) {
    const valor = chave(item) || "—";
    mapa.set(valor, (mapa.get(valor) ?? 0) + 1);
  }
  return [...mapa.entries()]
    .map(([nome, total]) => ({ nome, total }))
    .sort((a, b) => b.total - a.total || a.nome.localeCompare(b.nome, "pt-BR"));
}

/** Marca de ordem de bytes: o Excel abre o CSV como UTF-8. */
const BOM = String.fromCharCode(0xfeff);

function celula(texto: string) {
  return /[";\n]/.test(texto) ? `"${texto.replace(/"/g, '""')}"` : texto;
}

/** CSV da mesa (separado por ponto e vírgula, com BOM para o Excel). */
export function csvDosPedidos(pedidos: readonly PedidoResumo[]): string {
  const cabecalho = [
    "Numero",
    "Servico",
    "Requerente",
    "Unidade",
    "Status",
    "Etapa",
    "Prioridade",
    "Prazo",
    "SituacaoPrazo",
    "AbertoEm",
  ];
  const linhas = pedidos.map((p) =>
    [
      p.numero,
      p.servico_nome,
      p.nome,
      p.unidade_nome ?? "",
      rotuloStatus(p.status),
      p.etapa_id ? etapaLegivel(p.etapa_id) : "",
      rotuloPrioridade(p.prioridade),
      diaDoPrazo(p.prazo_em),
      p.situacao_prazo ? rotuloPrazo(p.situacao_prazo) : "",
      new Date(p.created_at).toLocaleDateString("pt-BR", { timeZone: "America/Cuiaba" }),
    ]
      .map(celula)
      .join(";"),
  );
  return BOM + [cabecalho.join(";"), ...linhas].join("\n");
}
