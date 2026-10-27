// Atendimento: tipos da API e o vocabulário da tela. O servidor vê a
// oferta e o estado em linguagem simples; "incidente" e "requisição"
// aparecem só para a equipe.

export type TipoSolicitacao = "incidente" | "requisicao";

export type Estado =
  | "aguardando_aprovacao"
  | "aberto"
  | "em_atendimento"
  | "aguardando_servidor"
  | "aguardando_terceiro"
  | "resolvido"
  | "atendido"
  | "encerrado"
  | "cancelado";

export type Papel = "tecnico" | "solicitante";

export interface CampoOferta {
  chave: string;
  rotulo: string;
  tipo: "texto" | "numero" | "lista";
  obrigatorio: boolean;
  opcoes?: string[];
}

export interface Oferta {
  id: string;
  nome: string;
  descricao: string;
  tipo: TipoSolicitacao;
  produto_id?: string;
  unidade_id?: string;
  ativa: boolean;
  campos?: CampoOferta[];
}

export interface Solicitacao {
  id: string;
  numero: number;
  tipo: TipoSolicitacao;
  estado: Estado;
  titulo: string;
  descricao: string;
  oferta_id: string;
  oferta_nome: string;
  unidade_id?: string;
  solicitante_id: string;
  solicitante_nome: string;
  responsavel_id?: string;
  responsavel_nome?: string;
  resolvida_em?: string;
  encerrada_em?: string;
  created_at: string;
  updated_at: string;
  /** Passagens de quem consulta a fila (só na fila). */
  proximos?: Estado[];
  impacto: number;
  urgencia: number;
  prioridade?: Prioridade;
  meta_resposta_min?: number;
  meta_solucao_min?: number;
  respostas?: Record<string, string>;
  /** Id no cofre. A senha não viaja com o chamado. */
  cofre_segredo_id?: string;
  /** Medido na leitura, só para a equipe. */
  prazo?: Prazo;
}

export type Prioridade = "critica" | "alta" | "media" | "baixa";
export type Situacao = "no_prazo" | "atencao" | "critico" | "vencido" | "cumprido" | "descumprido";

export interface Relogio {
  meta_min: number;
  usado_min: number;
  percentual: number;
  estado: "correndo" | "pausado" | "parado";
  situacao: Situacao;
  vence_em?: string;
}

export interface Prazo {
  prioridade: Prioridade;
  resposta: Relogio;
  solucao: Relogio;
}

export interface Turno {
  dia: number;
  inicio: string;
  fim: string;
}

export interface Celula {
  impacto: number;
  urgencia: number;
  prioridade: Prioridade;
}

export interface Meta {
  prioridade: Prioridade;
  resposta_min: number;
  solucao_min: number;
}

export interface Politica {
  id: string;
  unidade_id?: string;
  fuso: string;
  turnos: Turno[];
  matriz: Celula[];
  metas: Meta[];
  herdada: boolean;
}

export type TipoFeriado = "nacional" | "estadual" | "municipal" | "ponto_facultativo";

export interface Feriado {
  id: string;
  unidade_id?: string;
  data: string;
  nome: string;
  tipo: TipoFeriado;
}

export interface Passagem {
  id: number;
  de?: Estado;
  para: Estado;
  autor_nome: string;
  em: string;
}

export interface Comentario {
  id: string;
  autor_id?: string;
  autor_nome: string;
  texto: string;
  interno: boolean;
  created_at: string;
}

export interface Avaliacao {
  id: string;
  solicitacao_id: string;
  autor_id: string;
  autor_nome: string;
  nota: number;
  comentario: string;
  created_at: string;
  numero?: number;
  titulo?: string;
}

export interface PontoIndicador {
  dia: string;
  mttr_horas: number | null;
  mtbf_horas: number | null;
  sla_percentual: number | null;
  fcr_percentual: number | null;
  csat_media: number | null;
  truncado: boolean;
}

export interface Governanca {
  csat?: { media: number; total: number; recentes: Avaliacao[] } | null;
  mttr_horas: number | null;
  mtbf_horas: number | null;
  sla_percentual: number | null;
  fcr_percentual: number | null;
  amostra: {
    incidentes: number;
    incidentes_resolvidos: number;
    com_prazo_concluido: number;
    concluidas: number;
    fcr: number;
    truncado: boolean;
  };
  deflexao?: { viram: number; sem_pedido: number; taxa: number } | null;
}

export interface Detalhe {
  solicitacao: Solicitacao;
  papeis: Papel[];
  proximos: Estado[];
  passagens: Passagem[];
  comentarios: Comentario[];
  /** Sem coordenadas: a posição só aparece na supervisão. */
  visitas: Visita[];
  avaliacao?: Avaliacao | null;
  /** Decisões já tomadas (ADR 068). */
  aprovacoes?: Aprovacao[];
  /** Nome da etapa que falta, quando o pedido espera aprovação. */
  etapa_atual?: string;
}

export interface Aprovacao {
  id: string;
  etapa: number;
  etapa_nome: string;
  aprovador_nome?: string;
  decisao: "aprovada" | "recusada";
  motivo?: string;
  created_at: string;
}

export interface EtapaAprovacao {
  ordem: number;
  nome: string;
  aprovadores: string[];
}

export interface Ponto {
  lat: number;
  lng: number;
  precisao_m: number;
}

export interface Destino {
  lat: number;
  lng: number;
  raio_m: number;
}

export interface Evidencia {
  id: string;
  visita_id: string;
  tipo: "foto" | "assinatura";
  content_type: string;
  tamanho: number;
  created_at: string;
}

export interface Visita {
  id: string;
  solicitacao_id: string;
  tecnico_id: string;
  tecnico_nome: string;
  estado: "em_andamento" | "concluida" | "cancelada";
  local: string;
  base_legal: string;
  inicio_em: string;
  inicio?: Ponto;
  distancia_m?: number;
  fora_da_area: boolean;
  fim_em?: string;
  assinante_nome: string;
  observacao: string;
  evidencias: Evidencia[];
}

export interface VisitaEmCampo extends Visita {
  numero: number;
  titulo: string;
  unidade_id?: string;
}

export const ESTADO_VISITA: Record<Visita["estado"], { label: string; tone: Tone }> = {
  em_andamento: { label: "Em andamento", tone: "info" },
  concluida: { label: "Concluída", tone: "success" },
  cancelada: { label: "Cancelada", tone: "neutral" },
};

/** Finalidade mostrada antes de ler a localização (LGPD, art. 9º). */
export const AVISO_LOCALIZACAO =
  "Para comprovar o atendimento no local, a sua localização é lida uma única vez agora e outra ao encerrar a visita — nunca em segundo plano. Base legal: execução de política pública (LGPD, art. 7º, III, e art. 23). As coordenadas são apagadas após um ano.";

/** Link para conferir um ponto no OpenStreetMap (sem embutir mapa). */
export function linkMapa(p: Ponto): string {
  return `https://www.openstreetmap.org/?mlat=${p.lat}&mlon=${p.lng}#map=18/${p.lat}/${p.lng}`;
}

/** Distância curta em texto ("850 m", "1,2 km"). */
export function distancia(m: number): string {
  return m < 1000 ? `${Math.round(m)} m` : `${(m / 1000).toFixed(1).replace(".", ",")} km`;
}

/** Frame de tempo real: só diz o que mudou; a tela busca de novo. */
export const FRAME_SOLICITACAO = "atendimento.solicitacao";

export interface Pessoa {
  id: string;
  nome: string;
}

export const TIPO: Record<TipoSolicitacao, string> = {
  incidente: "Incidente",
  requisicao: "Requisição",
};

type Tone = "neutral" | "success" | "danger" | "warning" | "info";

export const ESTADO: Record<Estado, { label: string; tone: Tone }> = {
  aguardando_aprovacao: { label: "Aguardando aprovação", tone: "warning" },
  aberto: { label: "Aberta", tone: "info" },
  em_atendimento: { label: "Em atendimento", tone: "info" },
  aguardando_servidor: { label: "Aguardando resposta do servidor", tone: "warning" },
  aguardando_terceiro: { label: "Aguardando terceiro", tone: "warning" },
  resolvido: { label: "Resolvida", tone: "success" },
  atendido: { label: "Atendida", tone: "success" },
  encerrado: { label: "Encerrada", tone: "neutral" },
  cancelado: { label: "Cancelada", tone: "danger" },
};

/** Rótulo e tom do estado. Um estado que a API passe a devolver antes da
 * tela conhecê-lo aparece pelo nome técnico, sem quebrar a página. */
export function rotuloEstado(estado: string): string {
  return ESTADO[estado as Estado]?.label ?? estado;
}

/** O que a pessoa que abriu o pedido precisa saber, sem o vocabulário da fila. */
export function proximoPasso(estado: string): string {
  switch (estado) {
    case "aguardando_servidor":
      return "A equipe pediu uma resposta sua";
    case "resolvido":
    case "atendido":
      return "Avalie o atendimento";
    case "encerrado":
      return "Este pedido foi encerrado";
    case "cancelado":
      return "Este pedido foi cancelado";
    default:
      return "A equipe está cuidando deste pedido";
  }
}

export function tomEstado(estado: string): Tone {
  return ESTADO[estado as Estado]?.tone ?? "neutral";
}

export function rotuloTipo(tipo: string): string {
  return TIPO[tipo as TipoSolicitacao] ?? tipo;
}

export const PRIORIDADES: Prioridade[] = ["critica", "alta", "media", "baixa"];

export const PRIORIDADE: Record<Prioridade, { label: string; tone: Tone }> = {
  critica: { label: "Crítica", tone: "danger" },
  alta: { label: "Alta", tone: "warning" },
  media: { label: "Média", tone: "info" },
  baixa: { label: "Baixa", tone: "neutral" },
};

/** Níveis de impacto (o impacto é baixo) e de urgência (a urgência é baixa). */
export const NIVEIS = [
  { value: 1, label: "Baixo", feminino: "Baixa" },
  { value: 2, label: "Médio", feminino: "Média" },
  { value: 3, label: "Alto", feminino: "Alta" },
];

export const SITUACAO: Record<Situacao, { label: string; tone: Tone }> = {
  no_prazo: { label: "No prazo", tone: "success" },
  atencao: { label: "Atenção", tone: "warning" },
  critico: { label: "Crítico", tone: "danger" },
  vencido: { label: "Vencido", tone: "danger" },
  cumprido: { label: "Cumprido", tone: "success" },
  descumprido: { label: "Descumprido", tone: "danger" },
};

export const TIPO_FERIADO: Record<TipoFeriado, string> = {
  nacional: "Nacional",
  estadual: "Estadual",
  municipal: "Municipal",
  ponto_facultativo: "Ponto facultativo",
};

export const DIAS_DA_SEMANA = [
  "Domingo",
  "Segunda",
  "Terça",
  "Quarta",
  "Quinta",
  "Sexta",
  "Sábado",
];

/** O relógio que importa agora: a resposta enquanto ninguém começou; a
 * solução depois. */
export function relogioEmFoco(p: Prazo): { nome: "Resposta" | "Solução"; relogio: Relogio } {
  return p.resposta.estado === "correndo"
    ? { nome: "Resposta", relogio: p.resposta }
    : { nome: "Solução", relogio: p.solucao };
}

/** Minutos úteis em texto curto ("1h30", "45 min", "2h"). */
export function duracao(min: number): string {
  if (min < 60) return `${min} min`;
  const h = Math.floor(min / 60);
  const m = min % 60;
  return m ? `${h}h${String(m).padStart(2, "0")}` : `${h}h`;
}

/** Estados da fila, na ordem do fluxo, para o filtro. */
export const ESTADOS_FILA: Estado[] = [
  "aguardando_aprovacao",
  "aberto",
  "em_atendimento",
  "aguardando_servidor",
  "aguardando_terceiro",
  "resolvido",
  "atendido",
  "encerrado",
  "cancelado",
];

/** Rótulo do botão de cada passagem. A mesma passagem tem nome diferente
 * para quem abriu (confirmar, reabrir) e para a equipe. */
export function rotuloAcao(estado: Estado, de: Estado, solicitante: boolean): string {
  const concluida = de === "resolvido" || de === "atendido";
  if (solicitante && concluida) {
    return estado === "encerrado" ? "Confirmar e encerrar" : "Não resolveu, reabrir";
  }
  switch (estado) {
    case "em_atendimento":
      return concluida ? "Reabrir" : "Retomar atendimento";
    case "aguardando_servidor":
      return "Pedir informação ao servidor";
    case "aguardando_terceiro":
      return "Aguardar terceiro";
    case "resolvido":
      return "Marcar como resolvida";
    case "atendido":
      return "Marcar como atendida";
    case "encerrado":
      return "Encerrar";
    case "cancelado":
      return "Cancelar solicitação";
    default:
      return ESTADO[estado].label;
  }
}

export interface ApontamentoAtendimento {
  id: string;
  solicitacao_id: string;
  usuario_id: string;
  usuario_nome: string;
  data_apontamento: string;
  horas: number;
  descricao: string;
  created_at: string;
}

export interface ResumoApontamentosAtendimento {
  total_horas: number;
  apontamentos: ApontamentoAtendimento[];
}
