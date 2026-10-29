"use client";

import { useCallback, useMemo } from "react";
import { useApiQuery } from "@/lib/api/swr";
import { useActiveEntity } from "@/lib/plataforma/activeEntityStore";
import type { OrgTree } from "@/lib/plataforma/types";

/**
 * Hook central para isolamento absoluto de dados por Entidade Ativa (GLPI).
 *
 * Quando uma entidade está selecionada (Prefeitura, SMS ou SEMECEL), o filtro
 * é absoluto: apenas registros pertencentes à entidade (diretamente ou via unidade filha)
 * são retornados. Registros sem unidade ou pertencentes a outras secretarias são
 * omitidos.
 *
 * Quando "Todas as instituições" (visão global) está ativa, retorna todos os itens.
 */
export function useEntityFilter() {
  const activeEntity = useActiveEntity();
  const tree = useApiQuery<OrgTree[]>("v1/iam/org-tree");

  const unidadeParaEntidade = useMemo(() => {
    const mapa = new Map<string, string>();
    for (const ent of tree.data ?? []) {
      mapa.set(ent.id, ent.id);
      for (const u of ent.unidades) {
        mapa.set(u.id, ent.id);
      }
    }
    return mapa;
  }, [tree.data]);

  const filtrarPorEntidade = useCallback(
    <T extends { unidade_id?: string | null; secretaria_id?: string | null }>(itens: T[]): T[] => {
      if (!activeEntity) return itens;
      return itens.filter((item) => {
        const id = item.unidade_id || item.secretaria_id;
        if (!id) return false;
        const entId = unidadeParaEntidade.get(id);
        return id === activeEntity || entId === activeEntity;
      });
    },
    [activeEntity, unidadeParaEntidade],
  );

  return {
    activeEntity,
    tree,
    unidadeParaEntidade,
    filtrarPorEntidade,
  };
}

