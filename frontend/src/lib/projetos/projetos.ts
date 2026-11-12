// Projetos e tarefas: tipos da API e vocabulário da tela.

export type Tone = "neutral" | "success" | "danger" | "warning" | "info";

export type EstadoProjeto = "planejado" | "em_andamento" | "suspenso" | "concluido" | "cancelado";

export type PrioridadeProjeto = "baixa" | "media" | "alta" | "critica";

export interface Projeto {
  id: string;
  numero: number;
  titulo: string;
  descricao: string;
  objetivo: string;
  estado: EstadoProjeto;
  prioridade: PrioridadeProjeto;
  data_inicio_planejada?: string;
  data_fim_planejada?: string;
  data_inicio_real?: string;
  data_fim_real?: string;
  orcamento: number;
  unidade_id?: string;
  produto_id?: string;
  gerente_id?: string;
  gerente_nome: string;
  tarefas_total?: number;
  tarefas_concluidas?: number;
  created_at: string;
  updated_at: string;
  pode_reabrir?: boolean;
  reabertura?: ReaberturaProjeto;
  cancelamento?: ReaberturaProjeto;
  tarefas?: Tarefa[];
  vinculos?: VinculoProjeto[];
}

export interface ReaberturaProjeto {
  id: string;
  projeto_id: string;
  motivo: string;
  aprovador_id: string;
  aprovador_nome: string;
  created_at: string;
}

export type TipoVinculoProjeto = "mudanca" | "problema" | "ativo" | "solicitacao";

export interface VinculoProjeto {
  id: string;
  projeto_id: string;
  tipo: TipoVinculoProjeto;
  alvo_id: string;
  rotulo: string;
  vinculado_por: string;
  created_at: string;
}

export const TIPO_VINCULO_PROJETO: Record<TipoVinculoProjeto, { label: string; tone: Tone }> = {
  mudanca: { label: "Mudança", tone: "info" },
  problema: { label: "Problema", tone: "warning" },
  ativo: { label: "Ativo de TI", tone: "neutral" },
  solicitacao: { label: "Chamado / Incidente", tone: "neutral" },
};

export type EstadoTarefa = "a_fazer" | "em_andamento" | "concluida" | "bloqueada";

export interface PessoaProjeto {
  id: string;
  nome: string;
  papel?: string;
}

/** Pessoas da tarefa. Sem a lista nova, usa o responsável único já gravado. */
export function pessoasDaTarefa(t: {
  pessoas?: PessoaProjeto[];
  responsavel_id?: string | null;
  responsavel_nome?: string;
}): PessoaProjeto[] {
  if (t.pessoas && t.pessoas.length > 0) return t.pessoas;
  if (t.responsavel_id || t.responsavel_nome?.trim()) {
    return [{ id: t.responsavel_id ?? "", nome: t.responsavel_nome ?? "" }];
  }
  return [];
}

export function tarefaInclui(
  t: {
    pessoas?: PessoaProjeto[];
    responsavel_id?: string | null;
    responsavel_nome?: string;
  },
  usuarioId?: string,
) {
  const pessoas = pessoasDaTarefa(t);
  if (!usuarioId) return pessoas.length > 0;
  return pessoas.some((pessoa) => pessoa.id === usuarioId);
}

export interface Tarefa {
  id: string;
  projeto_id: string;
  titulo: string;
  descricao: string;
  responsavel_id?: string;
  responsavel_nome: string;
  pessoas?: PessoaProjeto[];
  estado: EstadoTarefa;
  ordem: number;
  data_limite?: string;
  origem?: string;
  predecessora_id?: string;
  pai_id?: string;
  relato?: string;
  created_at: string;
  updated_at: string;
}

export interface TarefaMinha extends Tarefa {
  projeto_numero: number;
  projeto_titulo: string;
  projeto_estado: string;
}

export interface PrazoAgenda {
  id: string;
  titulo: string;
  estado: EstadoTarefa;
  data_limite: string;
  responsavel_id?: string;
  responsavel_nome: string;
  pessoas?: PessoaProjeto[];
  pai_id?: string;
  projeto_id: string;
  projeto_numero: number;
  projeto_titulo: string;
}

export interface AtualizacaoTarefa {
  id: string;
  tarefa_id: string;
  texto: string;
  autor_id?: string;
  autor_nome: string;
  created_at: string;
}

export interface CargaPessoa {
  id: string;
  nome: string;
  abertas: number;
  atrasadas: number;
}

export const ESTADO_PROJETO: Record<
  EstadoProjeto,
  { label: string; tone: Tone; acao: string; proximo?: EstadoProjeto }
> = {
  planejado: { label: "Planejado", tone: "info", acao: "Planejar", proximo: "em_andamento" },
  em_andamento: {
    label: "Em andamento",
    tone: "warning",
    acao: "Iniciar execução",
    proximo: "concluido",
  },
  suspenso: { label: "Suspenso", tone: "danger", acao: "Suspender" },
  concluido: { label: "Concluído", tone: "success", acao: "Concluir" },
  cancelado: { label: "Cancelado", tone: "neutral", acao: "Cancelar" },
};

export const PRIORIDADE_PROJETO: Record<PrioridadeProjeto, { label: string; tone: Tone }> = {
  baixa: { label: "Baixa", tone: "neutral" },
  media: { label: "Média", tone: "info" },
  alta: { label: "Alta", tone: "warning" },
  critica: { label: "Crítica", tone: "danger" },
};

/** Fatia das tarefas concluídas, como a coluna de progresso da lista. */
export function porcentagemDoProjeto(total?: number, concluidas?: number): number {
  const todas = total ?? 0;
  if (todas <= 0) return 0;
  return Math.round(((concluidas ?? 0) / todas) * 100);
}

export const ESTADO_TAREFA: Record<EstadoTarefa, { label: string; tone: Tone }> = {
  a_fazer: { label: "A fazer", tone: "neutral" },
  em_andamento: { label: "Em andamento", tone: "info" },
  concluida: { label: "Concluída", tone: "success" },
  bloqueada: { label: "Bloqueada", tone: "danger" },
};
export interface ApontamentoHora {
  id: string;
  projeto_id: string;
  tarefa_id?: string;
  tarefa_titulo?: string;
  usuario_id: string;
  usuario_nome: string;
  data_apontamento: string;
  horas: number;
  descricao: string;
  custo_hora: number;
  custo_total: number;
  created_at: string;
}

export interface ResumoHorasProjeto {
  total_horas: number;
  total_custo: number;
  por_usuario: Record<string, number>;
}

export interface ListarApontamentosResponse {
  items: ApontamentoHora[];
  resumo: ResumoHorasProjeto;
}
