// Tipos da API de integrações (ADR 076): conexão, área de preparo, tabela
// de-para e tarefas.

export type Modo = "direto" | "conector";
export type TipoTarefa = "extracao" | "simulacao" | "carga";
export type EstadoTarefa = "pendente" | "em_andamento" | "concluida" | "falhou" | "cancelada";

export interface Conexao {
  id: string;
  tipo: string;
  nome: string;
  modo: Modo;
  url: string;
  perfil: string;
  fuso: string;
  tem_credenciais: boolean;
  tem_token: boolean;
  token_em?: string;
  extraido_em?: string;
  created_at: string;
  created_by: string;
}

export interface Tarefa {
  id: string;
  conexao_id: string;
  tipo: TipoTarefa;
  estado: EstadoTarefa;
  feitos: number;
  total: number;
  relatorio?: Record<string, unknown>;
  erro?: string;
  pedida_por: string;
  criada_em: string;
  iniciada_em?: string;
  terminada_em?: string;
}

export interface GrupoInventario {
  secretaria: string;
  categoria: number;
  categoria_nome: string;
  chamados: number;
  com_prazo: number;
  nivel: string;
  classe: string;
  primeiro: string;
  ultimo: string;
}

export interface NivelSLA {
  id: number;
  nome: string;
  minutos: number;
  chamados: number;
  prioridade?: string;
}

export interface Inventario {
  itens: Record<string, number>;
  secretarias: Record<string, number>;
  grupos: GrupoInventario[];
  niveis: NivelSLA[];
  extraido_em?: string;
}

export interface Linha {
  secretaria: string;
  categoria: number;
  categoria_nome: string;
  chamados: number;
  com_prazo: number;
  nivel: string;
  unidade_id: string | null;
  unidade_nome?: string;
  oferta: string;
  tipo: "incidente" | "requisicao";
  prioridade: "critica" | "alta" | "media" | "baixa";
}

export const ROTULO_TAREFA: Record<TipoTarefa, string> = {
  extracao: "Extração",
  simulacao: "Simulação",
  carga: "Carga",
};

export const ROTULO_ESTADO: Record<EstadoTarefa, string> = {
  pendente: "Na fila",
  em_andamento: "Em andamento",
  concluida: "Concluída",
  falhou: "Falhou",
  cancelada: "Cancelada",
};

export const PRIORIDADES = [
  { value: "critica", label: "Crítica" },
  { value: "alta", label: "Alta" },
  { value: "media", label: "Média" },
  { value: "baixa", label: "Baixa" },
];

export const TIPOS = [
  { value: "incidente", label: "Incidente" },
  { value: "requisicao", label: "Requisição" },
];

export function ativa(t: Tarefa) {
  return t.estado === "pendente" || t.estado === "em_andamento";
}

/** "4 h" ou "30 min": o prazo da origem para quem lê. */
export function duracao(minutos: number) {
  if (minutos <= 0) return "—";
  if (minutos % 1440 === 0) return `${minutos / 1440} d`;
  if (minutos % 60 === 0) return `${minutos / 60} h`;
  return `${minutos} min`;
}
