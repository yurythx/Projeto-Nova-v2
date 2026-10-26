"use client";

import { useMemo } from "react";

import {
  ordenarColunas,
  rotuloDaColuna,
  type ColunaFaixa,
  type TomFaixa,
} from "@/components/plataforma/Faixas";
import { useAction } from "@/components/plataforma/useAction";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";

interface ColunaRemota {
  chave: string;
  nome: string;
  cor: string;
  tom: TomFaixa;
  ordem: number;
  entrada: boolean;
  ativa: boolean;
  de?: string[];
  para?: string[];
}

const TOM: TomFaixa[] = ["neutral", "info", "warning", "success", "danger"];

function tomDe(valor: string, reserva: TomFaixa): TomFaixa {
  return TOM.includes(valor as TomFaixa) ? (valor as TomFaixa) : reserva;
}

function mesclar(fabrica: readonly ColunaFaixa[], remotas: ColunaRemota[]): ColunaFaixa[] {
  const base = new Map(fabrica.map((coluna) => [coluna.id, coluna]));
  const vistas: ColunaFaixa[] = remotas.map((remota) => {
    const conhecida = base.get(remota.chave);
    return {
      id: remota.chave,
      titulo: remota.nome,
      tom: tomDe(remota.tom, conhecida?.tom ?? "neutral"),
      ordem: remota.ordem,
      cor: remota.cor || undefined,
      entrada: remota.entrada,
      ativa: remota.ativa,
      de: remota.de,
      para: remota.para,
    };
  });
  for (const coluna of fabrica) {
    if (!vistas.some((vista) => vista.id === coluna.id))
      vistas.push({ ...coluna, ativa: true, entrada: coluna.ordem === 0 });
  }
  return ordenarColunas(vistas);
}

/** Catálogo do quadro. Se a consulta falha, o quadro de fábrica continua. */
export function useQuadro(quadro: string, fabrica: readonly ColunaFaixa[]) {
  const consulta = useApiQuery<ColunaRemota[]>(`v1/quadros/${quadro}`);
  const oficial = Array.isArray(consulta.data) && consulta.data.length > 0;
  const colunas = useMemo(
    () => (oficial && consulta.data ? mesclar(fabrica, consulta.data) : [...fabrica]),
    [oficial, consulta.data, fabrica],
  );
  const { run } = useAction();

  async function criar(corpo: {
    nome: string;
    cor: string;
    ordem: number;
    de: string[];
    para: string[];
  }) {
    const ok = await run(async () => {
      await apiClient.post(`v1/quadros/${quadro}`, corpo);
      await consulta.mutate();
      return true;
    });
    return ok === true;
  }

  async function gravar(
    chave: string,
    corpo: { nome?: string; cor?: string; ordem?: number; ativa?: boolean },
  ) {
    const ok = await run(async () => {
      await apiClient.put(`v1/quadros/${quadro}/${chave}`, corpo);
      await consulta.mutate();
      return true;
    });
    return ok === true;
  }

  return { colunas, oficial, gravar, criar };
}

/** Colunas no quadro, mais a que ainda tem card mesmo retirada. */
export function colunasVisiveis<T>(
  colunas: readonly ColunaFaixa[],
  itens: readonly T[],
  colunaDe: (item: T) => string | null,
): ColunaFaixa[] {
  return colunas.filter(
    (coluna) => coluna.ativa !== false || itens.some((item) => colunaDe(item) === coluna.id),
  );
}

export function rotuloQuadro(
  oficial: boolean,
  colunas: readonly { id: string; titulo: string }[],
  gravado: { colunas: Record<string, { titulo?: string } | undefined> },
  id: string,
): string {
  if (oficial) return colunas.find((coluna) => coluna.id === id)?.titulo || id;
  return rotuloDaColuna(colunas, id, gravado);
}
