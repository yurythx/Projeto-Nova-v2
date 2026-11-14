import { emCuiaba } from "./horario";

export type Tom = "neutral" | "success" | "danger" | "warning" | "info";

const PROGRAMAS: Record<string, string> = {
  regulariza: "Regulariza Rondonópolis",
  avanca: "Avança Rondonópolis",
};

const SITUACOES: { value: string; label: string; tom: Tom }[] = [
  { value: "protocolado", label: "Protocolado", tom: "info" },
  { value: "em_exigencia", label: "Em exigência", tom: "warning" },
  { value: "em_analise", label: "Em análise", tom: "info" },
  { value: "aguardando_pagamento", label: "Aguardando pagamento", tom: "warning" },
  { value: "parcelado", label: "Parcelado", tom: "success" },
  { value: "decidido", label: "Decidido", tom: "success" },
  { value: "recurso", label: "Recurso", tom: "warning" },
  { value: "arquivado", label: "Arquivado", tom: "neutral" },
];

/** Opções do filtro da lista. A situação vazia significa todas. */
export const OPCOES_SITUACAO = SITUACOES.map(({ value, label }) => ({ value, label }));

export function nomePrograma(id: string): string {
  return PROGRAMAS[id] ?? id;
}

export function nomeSituacao(id: string): string {
  return SITUACOES.find((item) => item.value === id)?.label ?? id;
}

export function tomSituacao(id: string): Tom {
  return SITUACOES.find((item) => item.value === id)?.tom ?? "neutral";
}

export interface Marco {
  rotulo: string;
  quando: string;
}

/** Andamento com as datas já gravadas no processo. Data vazia não entra. */
export function andamento(item: {
  recibo_em?: string;
  exigencia_aberta?: string;
  despacho_em?: string;
}): Marco[] {
  const candidatos = [
    { rotulo: "Protocolo registrado", iso: item.recibo_em },
    { rotulo: "Exigência aberta", iso: item.exigencia_aberta },
    { rotulo: "Decisão registrada", iso: item.despacho_em },
  ];
  return candidatos
    .flatMap((candidato) => {
      const quando = emCuiaba(candidato.iso);
      return quando ? [{ rotulo: candidato.rotulo, quando, iso: candidato.iso ?? "" }] : [];
    })
    .sort((a, b) => new Date(a.iso).getTime() - new Date(b.iso).getTime())
    .map(({ rotulo, quando }) => ({ rotulo, quando }));
}
