// Ocorrências e página de status (ADR 066).

type Tone = "neutral" | "success" | "danger" | "warning" | "info";

export type Impacto = "degradado" | "parcial" | "fora";
export type EstadoOcorrencia = "investigando" | "identificado" | "monitorando" | "resolvido";
export type SituacaoGeral = Impacto | "operacional";

export interface AtualizacaoPublica {
  estado: EstadoOcorrencia;
  texto: string;
  em: string;
}

export interface OcorrenciaPublica {
  id: string;
  titulo: string;
  impacto: Impacto;
  estado: EstadoOcorrencia;
  servicos: string[];
  iniciada_em: string;
  resolvida_em?: string;
  atualizacoes: AtualizacaoPublica[];
}

export interface Situacao {
  geral: SituacaoGeral;
  abertas: OcorrenciaPublica[];
  recentes: OcorrenciaPublica[];
}

export interface Atualizacao {
  id: string;
  estado: EstadoOcorrencia;
  impacto: Impacto;
  texto: string;
  autor_nome?: string;
  created_at: string;
}

export interface Ocorrencia {
  id: string;
  titulo: string;
  impacto: Impacto;
  estado: EstadoOcorrencia;
  servicos: string[];
  chamado_id?: string;
  coordenador_id?: string;
  coordenador_nome?: string;
  iniciada_em: string;
  resolvida_em?: string;
  revisao?: string;
  revisada_em?: string;
  updated_at: string;
  atualizacoes: Atualizacao[];
}

export const IMPACTO: Record<Impacto, { label: string; tone: Tone }> = {
  degradado: { label: "Lentidão ou falhas", tone: "warning" },
  parcial: { label: "Parte fora do ar", tone: "danger" },
  fora: { label: "Fora do ar", tone: "danger" },
};

export const ESTADO: Record<EstadoOcorrencia, string> = {
  investigando: "Investigando",
  identificado: "Causa identificada",
  monitorando: "Em observação",
  resolvido: "Resolvido",
};

export const SITUACAO: Record<SituacaoGeral, { titulo: string; tone: Tone }> = {
  operacional: { titulo: "Todos os serviços funcionando", tone: "success" },
  degradado: { titulo: "Alguns serviços com lentidão ou falhas", tone: "warning" },
  parcial: { titulo: "Parte dos serviços fora do ar", tone: "danger" },
  fora: { titulo: "Serviços fora do ar", tone: "danger" },
};

export const ESTADOS: EstadoOcorrencia[] = [
  "investigando",
  "identificado",
  "monitorando",
  "resolvido",
];
export const IMPACTOS: Impacto[] = ["degradado", "parcial", "fora"];

/** Minutos desde a última notícia: o painel mostra quem está sem
 * atualização há muito tempo. */
export function minutosSemNoticia(o: Ocorrencia, agora: number): number {
  const ultima = o.atualizacoes.at(-1)?.created_at ?? o.iniciada_em;
  return Math.max(0, Math.floor((agora - new Date(ultima).getTime()) / 60000));
}

/** Fuso da instituição. A página de status é renderizada no servidor (em
 * UTC): sem fuso explícito, quem lê veria a hora errada. */
export const FUSO = "America/Cuiaba";

const horario = new Intl.DateTimeFormat("pt-BR", {
  dateStyle: "short",
  timeStyle: "short",
  timeZone: FUSO,
});

/** Data e hora no fuso da instituição. */
export function fmtHorario(iso: string): string {
  return horario.format(new Date(iso));
}
