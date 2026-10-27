// IA assistiva do atendimento (ADR 065): sugere a quem atende, que decide.

export interface EstadoIA {
  disponivel: boolean;
}

export interface ClassificacaoSugerida {
  impacto: number;
  urgencia: number;
  motivo: string;
}

export type TarefaIA = "classificacao" | "resumo";
export type DecisaoIA = "aceita" | "recusada";

export interface SugestaoIA {
  id: string;
  solicitacao_id: string;
  tipo: TarefaIA;
  conexao: string;
  classificacao?: ClassificacaoSugerida;
  resumo?: string;
  decisao?: DecisaoIA;
  created_at: string;
}

export interface IndicadorIA {
  tipo: TarefaIA;
  total: number;
  aceitas: number;
  recusadas: number;
  pendentes: number;
}

export const TAREFA_IA: Record<TarefaIA, string> = {
  classificacao: "Classificação",
  resumo: "Resumo",
};

/** Parte das sugestões decididas que foi aceita, de 0 a 100. Sem nenhuma
 * decisão, null: indicador sem amostra fica vazio. */
export function taxaDeAceite(i: IndicadorIA): number | null {
  const decididas = i.aceitas + i.recusadas;
  if (decididas === 0) return null;
  return Math.round((i.aceitas / decididas) * 100);
}
