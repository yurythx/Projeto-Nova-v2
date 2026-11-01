import { apiClient } from "@/lib/api/client";

export type CategoriaRisco =
  | "seguranca_informacao"
  | "infraestrutura"
  | "continuidade"
  | "operacional"
  | "conformidade_lgpd";

export type EstrategiaRisco = "mitigar" | "aceitar" | "transferir" | "evitar";

export type StatusRisco =
  | "identificado"
  | "em_tratamento"
  | "mitigado"
  | "monitorado"
  | "aceito";

export type NivelSeveridade = "baixo" | "medio" | "alto" | "critico";

export interface Risco {
  id: string;
  codigo: string;
  titulo: string;
  descricao: string;
  categoria: CategoriaRisco;
  probabilidade: number; // 1 a 5
  impacto: number; // 1 a 5
  severidade: number; // 1 a 25
  nivel_severidade: NivelSeveridade;
  estrategia: EstrategiaRisco;
  plano_mitigacao: string;
  controle_iso27001?: string;
  status: StatusRisco;
  responsavel_nome?: string;
  unidade_id: string;
  ativo_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface HeatmapItem {
  probabilidade: number;
  impacto: number;
  severidade: number;
  nivel: NivelSeveridade;
  total_riscos: number;
  riscos: {
    id: string;
    codigo: string;
    titulo: string;
    categoria: CategoriaRisco;
    status: StatusRisco;
  }[];
}

export interface CriarRiscoInput {
  titulo: string;
  descricao: string;
  categoria: CategoriaRisco;
  probabilidade: number;
  impacto: number;
  estrategia: EstrategiaRisco;
  plano_mitigacao: string;
  controle_iso27001?: string;
  responsavel_nome?: string;
  unidade_id?: string;
  ativo_id?: string | null;
}

export interface AtualizarRiscoInput {
  titulo?: string;
  descricao?: string;
  categoria?: CategoriaRisco;
  probabilidade?: number;
  impacto?: number;
  estrategia?: EstrategiaRisco;
  plano_mitigacao?: string;
  controle_iso27001?: string;
  status?: StatusRisco;
  responsavel_nome?: string;
  ativo_id?: string | null;
}

export const ROTULOS_CATEGORIA: Record<CategoriaRisco, string> = {
  seguranca_informacao: "Segurança da Informação",
  infraestrutura: "Infraestrutura & Redes",
  continuidade: "Continuidade de Negócio",
  operacional: "Operacional & Suporte",
  conformidade_lgpd: "Conformidade LGPD & GRC",
};

export const ROTULOS_STATUS: Record<StatusRisco, string> = {
  identificado: "Identificado",
  em_tratamento: "Em Tratamento",
  mitigado: "Mitigado",
  monitorado: "Monitorado",
  aceito: "Risco Aceito",
};

export const ROTULOS_ESTRATEGIA: Record<EstrategiaRisco, string> = {
  mitigar: "Mitigar (Reduzir)",
  aceitar: "Aceitar Risco",
  transferir: "Transferir (Terceirizar)",
  evitar: "Evitar / Eliminar",
};

export const CONTROLES_ISO27001_SUGESTOES = [
  "A.5.1 - Políticas de Segurança da Informação",
  "A.6.1 - Organização da Segurança da Informação",
  "A.7.2 - Conscientização e Treinamento em Segurança",
  "A.8.1 - Inventário e Posse Responsável de Ativos",
  "A.9.2 - Gestão de Acessos e Privilégios Mínimos",
  "A.9.4 - Controle de Acesso a Sistemas e Aplicações",
  "A.10.1 - Criptografia em Trânsito e Repouso",
  "A.11.1 - Segurança Física e Perímetros Protegidos",
  "A.12.1 - Procedimentos Operacionais e Responsabilidades",
  "A.12.3 - Backup e Retenção Segura de Dados",
  "A.12.4 - Registro de Logs e Auditoria Contínua",
  "A.12.6 - Gestão de Vulnerabilidades Técnicas",
  "A.13.1 - Segurança no Gerenciamento de Redes",
  "A.14.2 - Segurança nos Processos de Desenvolvimento",
  "A.16.1 - Gestão de Incidentes de Segurança",
  "A.17.1 - Continuidade da Segurança da Informação (DRP)",
  "A.18.1 - Conformidade com Requisitos Legais e LGPD",
];

export const riscosApi = {
  listar: async (params?: { status?: string; categoria?: string; severidade?: string }) => {
    const q = new URLSearchParams();
    if (params?.status) q.set("status", params.status);
    if (params?.categoria) q.set("categoria", params.categoria);
    if (params?.severidade) q.set("severidade", params.severidade);
    const qs = q.toString();
    return apiClient.get<Risco[]>(`v1/riscos${qs ? `?${qs}` : ""}`);
  },

  heatmap: async () => {
    return apiClient.get<HeatmapItem[]>("v1/riscos/heatmap");
  },

  obter: async (id: string) => {
    return apiClient.get<Risco>(`v1/riscos/${id}`);
  },

  criar: async (payload: CriarRiscoInput) => {
    return apiClient.post<Risco>("v1/riscos", payload);
  },

  atualizar: async (id: string, payload: AtualizarRiscoInput) => {
    return apiClient.put<Risco>(`v1/riscos/${id}`, payload);
  },
};
