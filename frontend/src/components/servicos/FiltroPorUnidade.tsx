"use client";

import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { useApiQuery } from "@/lib/api/swr";
import type { OrgTree } from "@/lib/plataforma/types";

export function filtrarPorUnidade<T extends { unidade_id?: string | null }>(
  itens: T[],
  unidadeFiltro: string,
): T[] {
  if (!unidadeFiltro || unidadeFiltro === "todas") return itens;
  return itens.filter((item) => (item.unidade_id ?? "institucional") === unidadeFiltro);
}

/** Recorte rápido por secretaria/unidade (SECITI, SMS, SEMED) nos quadros
 * e listas da cadeia ITIL v5 quando o escopo traz mais de uma unidade. */
export function FiltroPorUnidade<T extends { unidade_id?: string | null }>({
  rotulo = "Filtrar por unidade",
  itens,
  value,
  onChange,
}: {
  rotulo?: string;
  itens: T[];
  value: string;
  onChange: (proxima: string) => void;
}) {
  const chaves = Array.from(new Set(itens.map((i) => i.unidade_id ?? "institucional")));
  const multiUnidade = chaves.length > 1;
  const tree = useApiQuery<OrgTree[]>(multiUnidade ? "v1/iam/org-tree" : null);

  if (!multiUnidade) return null;

  const nomes = new Map<string, string>();
  for (const ent of tree.data ?? []) {
    for (const u of ent.unidades) {
      nomes.set(u.id, u.sigla ? `${u.sigla} — ${u.nome}` : u.nome);
    }
  }

  return (
    <SectionTabsInline
      label={rotulo}
      value={value}
      onChange={onChange}
      tabs={[
        { value: "todas", label: `Todas (${itens.length})` },
        ...chaves.map((chave) => ({
          value: chave,
          label:
            chave === "institucional"
              ? "Institucional"
              : (nomes.get(chave) ?? `Unidade ${chave.slice(0, 8)}`),
        })),
      ]}
    />
  );
}

