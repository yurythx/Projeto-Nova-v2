// Cores, selos e prazo da mesa do protocolo, iguais ao Protocolo Digital
// original (client/src/components/ProtocolosManagement.js,
// KanbanBoardExternal.js e ProtocoloDetalhes.js). A paleta padrão do
// Tailwind tem os mesmos hexadecimais do original (blue-500 = #3b82f6...).

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
  decisao?: string;
  responsavel_id?: string;
  responsavel_nome?: string;
  exige_aceite?: boolean;
  aprovador_id?: string;
  processo_sei?: string;
  mensagem_nova?: boolean;
  complemento_prazo?: string;
  encaminhamento?: {
    unidade_id: string;
    unidade_nome?: string;
    justificativa?: string;
    prazo_dias?: number;
  } | null;
}

export const FINAIS = new Set(["concluido", "aprovado", "rejeitado", "cancelado"]);

/** Rótulos do original (com maiúscula em cada palavra). */
const ROTULO_STATUS: Record<string, string> = {
  novo: "Novo",
  em_andamento: "Em Andamento",
  em_analise: "Em Análise",
  concluido: "Concluído",
  pendente: "Pendente",
  cancelado: "Cancelado",
  aprovado: "Aprovado",
  rejeitado: "Rejeitado",
  em_complementacao: "Em Complementação",
  pending_approval: "Pendente de Aprovação",
};

export function rotuloDoStatus(status: string): string {
  return ROTULO_STATUS[status] ?? status;
}

/** Selo sólido do status na lista (StatusBadge da lista original). */
const SELO_STATUS: Record<string, string> = {
  novo: "bg-blue-500",
  em_andamento: "bg-amber-500",
  em_analise: "bg-cyan-500",
  concluido: "bg-emerald-500",
  pendente: "bg-red-500",
  cancelado: "bg-gray-500",
  em_complementacao: "bg-violet-500 animate-pulse",
  pending_approval: "bg-amber-500 animate-pulse",
  aprovado: "bg-emerald-500",
  rejeitado: "bg-red-500",
};

export function seloDoStatus(status: string): string {
  return SELO_STATUS[status] ?? "bg-gray-600";
}

/** Cor do cartão de estatística de cada status (o original pinta sólido). */
const COR_ESTATISTICA: Record<string, string> = {
  novo: "bg-blue-500",
  em_andamento: "bg-amber-500",
  em_analise: "bg-cyan-500",
  em_complementacao: "bg-violet-500",
  concluido: "bg-emerald-500",
  pendente: "bg-red-500",
  cancelado: "bg-gray-500",
  aprovado: "bg-emerald-500",
  rejeitado: "bg-red-500",
  pending_approval: "bg-amber-500",
};

export function corDaEstatistica(status: string): string {
  return COR_ESTATISTICA[status] ?? "bg-gray-600";
}

/** Selo de status na ficha (gradiente do ProtocoloDetalhes original). */
const GRADIENTE_STATUS: Record<string, string> = {
  novo: "bg-gradient-to-br from-[#4facfe] to-[#00f2fe] shadow-[0_4px_15px_rgba(79,172,254,0.3)] text-white",
  em_andamento:
    "bg-gradient-to-br from-[#43e97b] to-[#38f9d7] shadow-[0_4px_15px_rgba(67,233,123,0.3)] text-white",
  concluido:
    "bg-gradient-to-br from-[#fa709a] to-[#fee140] shadow-[0_4px_15px_rgba(250,112,154,0.3)] text-white",
  pendente:
    "bg-gradient-to-br from-[#a8edea] to-[#fed6e3] shadow-[0_4px_15px_rgba(168,237,234,0.3)] text-slate-800",
  cancelado:
    "bg-gradient-to-br from-[#ff9a9e] to-[#fecfef] shadow-[0_4px_15px_rgba(255,154,158,0.3)] text-slate-800",
};

export function gradienteDoStatus(status: string): string {
  return GRADIENTE_STATUS[status] ?? `${seloDoStatus(status)} text-white`;
}

const ROTULO_PRIORIDADE: Record<string, string> = {
  baixa: "Baixa",
  normal: "Normal",
  alta: "Alta",
  urgente: "Urgente",
};

export function rotuloDaPrioridade(prioridade: string): string {
  return ROTULO_PRIORIDADE[prioridade] ?? prioridade;
}

/** Selo sólido da prioridade na lista. */
const SELO_PRIORIDADE: Record<string, string> = {
  baixa: "bg-violet-500",
  normal: "bg-blue-500",
  alta: "bg-orange-500",
  urgente: "bg-red-600",
};

export function seloDaPrioridade(prioridade: string): string {
  return SELO_PRIORIDADE[prioridade] ?? "bg-blue-500";
}

/** Selo da prioridade na ficha (gradiente). */
const GRADIENTE_PRIORIDADE: Record<string, string> = {
  baixa:
    "bg-gradient-to-br from-[#a8edea] to-[#fed6e3] shadow-[0_4px_15px_rgba(168,237,234,0.3)] text-slate-800",
  normal:
    "bg-gradient-to-br from-[#4facfe] to-[#00f2fe] shadow-[0_4px_15px_rgba(79,172,254,0.3)] text-white",
  alta: "bg-gradient-to-br from-[#fa709a] to-[#fee140] shadow-[0_4px_15px_rgba(250,112,154,0.3)] text-white",
  urgente:
    "bg-gradient-to-br from-[#ff9a9e] to-[#fecfef] shadow-[0_4px_15px_rgba(255,154,158,0.3)] text-slate-800",
};

export function gradienteDaPrioridade(prioridade: string): string {
  return GRADIENTE_PRIORIDADE[prioridade] ?? `${seloDaPrioridade(prioridade)} text-white`;
}

/** Borda esquerda do cartão da lista, pela situação do prazo. */
const BORDA_PRAZO: Record<string, string> = {
  no_prazo: "border-l-4 border-l-emerald-600",
  atencao: "border-l-4 border-l-amber-600",
  atrasado: "border-l-4 border-l-red-600",
  finalizado: "border-l-4 border-l-gray-500",
};

export function bordaDoPrazo(situacao?: string): string {
  return BORDA_PRAZO[situacao || "no_prazo"] ?? BORDA_PRAZO.no_prazo!;
}

/** Selo claro do prazo (SLABadge do original). */
const SELO_PRAZO: Record<string, string> = {
  no_prazo:
    "bg-emerald-50 text-emerald-600 border border-emerald-500/20 dark:bg-emerald-500/15 dark:text-emerald-300 dark:border-emerald-400/30",
  atencao:
    "bg-amber-50 text-amber-600 border border-amber-500/20 dark:bg-amber-500/15 dark:text-amber-300 dark:border-amber-400/30",
  atrasado:
    "bg-red-50 text-red-600 border border-red-500/20 dark:bg-red-500/15 dark:text-red-300 dark:border-red-400/30",
  finalizado:
    "bg-gray-100 text-gray-500 border border-gray-300/20 dark:bg-slate-700 dark:text-slate-300 dark:border-slate-500/30",
};

export function seloDoPrazo(situacao?: string): string {
  return SELO_PRAZO[situacao || "no_prazo"] ?? SELO_PRAZO.no_prazo!;
}

const ROTULO_PRAZO: Record<string, string> = {
  no_prazo: "No Prazo",
  atencao: "Atenção",
  atrasado: "Atrasado",
  finalizado: "Finalizado",
};

/** Dias úteis entre duas datas (sábado e domingo fora), contando o fim e não o início. */
export function diasUteis(inicio: Date, fim: Date): number {
  if (fim <= inicio) return 0;
  const atual = new Date(inicio.getTime());
  let conta = 0;
  for (;;) {
    atual.setDate(atual.getDate() + 1);
    if (atual > fim) break;
    const dia = atual.getDay();
    if (dia !== 0 && dia !== 6) conta++;
  }
  return conta;
}

/**
 * Texto do selo de prazo: "No Prazo (3/15d)". Os dias corridos são úteis
 * desde a abertura; o total, os úteis até o prazo. Sem prazo, só o rótulo.
 */
export function textoDoPrazo(
  pedido: Pick<PedidoResumo, "situacao_prazo" | "created_at" | "prazo_em">,
  agora: Date = new Date(),
): string {
  const situacao = pedido.situacao_prazo || "no_prazo";
  const rotulo = ROTULO_PRAZO[situacao] ?? situacao;
  if (situacao === "finalizado" || !pedido.prazo_em) return rotulo;
  const aberto = new Date(pedido.created_at);
  const prazo = new Date(pedido.prazo_em);
  if (Number.isNaN(aberto.getTime()) || Number.isNaN(prazo.getTime())) return rotulo;
  return `${rotulo} (${diasUteis(aberto, agora)}/${diasUteis(aberto, prazo)}d)`;
}

/** Coluna do kanban; o original junta concluído com aprovado e cancelado com rejeitado. */
export interface ColunaKanban {
  id: string;
  titulo: string;
  cor: string;
  borda: string;
}

export const COLUNAS_KANBAN: ColunaKanban[] = [
  { id: "novo", titulo: "Novos", cor: "bg-blue-500", borda: "border-b-blue-500" },
  { id: "em_analise", titulo: "Em Análise", cor: "bg-cyan-500", borda: "border-b-cyan-500" },
  { id: "pendente", titulo: "Pendente", cor: "bg-violet-500", borda: "border-b-violet-500" },
  {
    id: "em_andamento",
    titulo: "Em Andamento",
    cor: "bg-amber-500",
    borda: "border-b-amber-500",
  },
  {
    id: "pending_approval",
    titulo: "Aguardando Aprovação",
    cor: "bg-orange-500",
    borda: "border-b-orange-500",
  },
  {
    id: "em_complementacao",
    titulo: "Aguardando Comp.",
    cor: "bg-pink-500",
    borda: "border-b-pink-500",
  },
  { id: "concluido", titulo: "Concluídos", cor: "bg-emerald-500", borda: "border-b-emerald-500" },
  {
    id: "rejeitado",
    titulo: "Cancelados/Rejeitados",
    cor: "bg-red-500",
    borda: "border-b-red-500",
  },
];

export function colunaDoKanban(status: string): string {
  if (status === "cancelado" || status === "rejeitado") return "rejeitado";
  if (status === "concluido" || status === "aprovado") return "concluido";
  return status;
}

/** Borda esquerda do cartão do kanban, pela prioridade. */
export function bordaDaPrioridade(prioridade: string): string {
  return prioridade === "alta" || prioridade === "urgente"
    ? "border-l-red-500"
    : "border-l-blue-500";
}

/** Selo pequeno da prioridade no cartão do kanban. */
export function seloPequenoDaPrioridade(prioridade: string): string {
  return prioridade === "alta" || prioridade === "urgente"
    ? "bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-300"
    : "bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-300";
}

/** Data curta em pt-BR no fuso do protocolo (Cuiabá). */
export function dataCurta(iso?: string): string {
  if (!iso) return "Data não disponível";
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return "Data inválida";
  return data.toLocaleDateString("pt-BR", { timeZone: "America/Cuiaba" });
}

/** Etapa legível: "analise_tecnica" vira "Analise tecnica". */
export function nomeDaEtapa(etapa?: string): string {
  const texto = (etapa ?? "").trim().replaceAll("_", " ");
  if (!texto) return "Etapa não definida";
  return texto.charAt(0).toUpperCase() + texto.slice(1);
}
