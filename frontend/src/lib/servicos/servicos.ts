// Ativos, problemas e mudanças (fases 6 e 7): tipos da API e vocabulário
// da tela.

type Tone = "neutral" | "success" | "danger" | "warning" | "info";

export type TipoAtivo = "hardware" | "software" | "servico" | "rede" | "outro";
export type EstadoAtivo = "planejado" | "em_uso" | "manutencao" | "desativado";

export interface DiscoLocal {
  nome: string;
  bytes: number;
}

export interface ProgramaInstalado {
  nome: string;
  versao?: string;
}

export interface InventarioMaquina {
  sistema: string;
  processadores: number;
  memoria_bytes?: number;
  discos?: DiscoLocal[];
  programas?: ProgramaInstalado[];
  programas_truncado?: boolean;
}

export interface CredencialAgente {
  id: string;
  nome: string;
  unidade_id: string;
  created_at: string;
  token?: string;
}

export interface Ativo {
  id: string;
  nome: string;
  tipo: TipoAtivo;
  estado: EstadoAtivo;
  identificador: string;
  descricao: string;
  produto_id?: string;
  unidade_id?: string;
  origem?: "manual" | "agente";
  inventario?: InventarioMaquina;
  created_at?: string;
  updated_at?: string;
}

export const TIPO_ATIVO: Record<TipoAtivo, string> = {
  hardware: "Equipamento",
  software: "Sistema ou licença",
  servico: "Serviço contratado",
  rede: "Rede",
  outro: "Outro",
};

export const ESTADO_ATIVO: Record<EstadoAtivo, { label: string; tone: Tone }> = {
  planejado: { label: "Planejado", tone: "info" },
  em_uso: { label: "Em uso", tone: "success" },
  manutencao: { label: "Em manutenção", tone: "warning" },
  desativado: { label: "Desativado", tone: "neutral" },
};

export type EstadoProblema =
  "registrado" | "em_analise" | "erro_conhecido" | "resolvido" | "encerrado";

export interface Problema {
  id: string;
  numero: number;
  titulo: string;
  descricao: string;
  estado: EstadoProblema;
  causa_raiz: string;
  contorno: string;
  produto_id?: string;
  unidade_id?: string;
  resolvido_em?: string;
  created_at: string;
  incidentes: { solicitacao_id: string; numero: number; titulo: string; vinculado_em: string }[];
  ativos: { ativo_id: string; nome: string; vinculado_em: string }[];
  proximos: EstadoProblema[];
}

export interface Recorrencia {
  oferta_id: string;
  oferta_nome: string;
  unidade_id?: string;
  total: number;
  ultimo_em: string;
  solicitacoes: string[];
}

export const ESTADO_PROBLEMA: Record<EstadoProblema, { label: string; tone: Tone; acao: string }> =
  {
    registrado: { label: "Registrado", tone: "info", acao: "Registrar" },
    em_analise: { label: "Em análise", tone: "info", acao: "Analisar" },
    erro_conhecido: {
      label: "Erro conhecido",
      tone: "warning",
      acao: "Publicar como erro conhecido",
    },
    resolvido: { label: "Resolvido", tone: "success", acao: "Marcar como resolvido" },
    encerrado: { label: "Encerrado", tone: "neutral", acao: "Encerrar" },
  };

export type TipoMudanca = "padrao" | "normal" | "emergencial";
export type Risco = "baixo" | "medio" | "alto";
export type EstadoMudanca =
  | "rascunho"
  | "em_avaliacao"
  | "aprovada"
  | "rejeitada"
  | "em_execucao"
  | "concluida"
  | "falhou"
  | "cancelada";

export interface Mudanca {
  id: string;
  numero: number;
  titulo: string;
  descricao: string;
  justificativa: string;
  tipo: TipoMudanca;
  risco: Risco;
  risco_score?: number;
  estado: EstadoMudanca;
  plano_reversao: string;
  inicio_planejado?: string;
  fim_planejado?: string;
  resultado: string;
  executada_inicio?: string;
  executada_fim?: string;
  produto_id?: string;
  unidade_id?: string;
  solicitante_id: string;
  solicitante_nome: string;
  created_at: string;
  aprovacoes: {
    id: string;
    aprovador: string;
    decisao: "aprovada" | "rejeitada" | "automatica";
    parecer: string;
    em: string;
  }[];
  vinculos: {
    tipo: "problema" | "incidente" | "ativo";
    alvo_id: string;
    rotulo: string;
    vinculado_em: string;
  }[];
  proximos: EstadoMudanca[];
}

export const TIPO_MUDANCA: Record<TipoMudanca, string> = {
  padrao: "Padrão (pré-aprovada)",
  normal: "Normal",
  emergencial: "Emergencial",
};

export const RISCO: Record<Risco, { label: string; tone: Tone }> = {
  baixo: { label: "Risco baixo", tone: "neutral" },
  medio: { label: "Risco médio", tone: "warning" },
  alto: { label: "Risco alto", tone: "danger" },
};

export const ESTADO_MUDANCA: Record<EstadoMudanca, { label: string; tone: Tone; acao: string }> = {
  rascunho: { label: "Rascunho", tone: "neutral", acao: "Voltar a rascunho" },
  em_avaliacao: { label: "Em avaliação", tone: "info", acao: "Enviar para aprovação" },
  aprovada: { label: "Aprovada", tone: "success", acao: "Aprovar" },
  rejeitada: { label: "Rejeitada", tone: "danger", acao: "Rejeitar" },
  em_execucao: { label: "Em execução", tone: "info", acao: "Iniciar execução" },
  concluida: { label: "Concluída", tone: "success", acao: "Concluir" },
  falhou: { label: "Falhou", tone: "danger", acao: "Registrar falha" },
  cancelada: { label: "Cancelada", tone: "neutral", acao: "Cancelar" },
};

/** Passagens que pedem texto: parecer (rejeitar exige; aprovar aceita) e
 * resultado (falhar exige; concluir aceita). */
export const TEXTO_DA_PASSAGEM: Partial<
  Record<EstadoMudanca, { rotulo: string; obrigatorio: boolean }>
> = {
  aprovada: { rotulo: "Parecer", obrigatorio: false },
  rejeitada: { rotulo: "Parecer", obrigatorio: true },
  concluida: { rotulo: "Resultado", obrigatorio: false },
  falhou: { rotulo: "O que foi revertido", obrigatorio: true },
};

/** Valor para <input type="datetime-local"> a partir de ISO (hora local). */
export function paraCampoData(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** ISO a partir do campo datetime-local (vazio = sem data). */
export function deCampoData(valor: string): string | null {
  return valor ? new Date(valor).toISOString() : null;
}
