import type { OrgTree } from "@/lib/plataforma/types";

/** Unidade sugerida para um projeto aberto com uma instituição no cabeçalho.
 * Uma unidade só vira o padrão. Várias exigem escolha, para o projeto não
 * ser gravado sem dono e sumir da lista dessa instituição. */
export function unidadeDaEntidadeAtiva(tree: OrgTree[] | undefined, activeEntity: string): string {
  if (!activeEntity || !tree?.length) return "";
  for (const ent of tree) {
    if (ent.id === activeEntity) {
      const unica = ent.unidades[0];
      return ent.unidades.length === 1 && unica ? unica.id : "";
    }
    if (ent.unidades.some((u) => u.id === activeEntity)) return activeEntity;
  }
  return "";
}
