"use client";

import { SectionTabs } from "@/components/layout/SectionTabs";
import { MODULE_CHILDREN, visibleTabs } from "@/lib/nav/sectionTabs";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

/** Abas de um módulo no topo das páginas — as mesmas subpáginas do
 * submenu da barra lateral (MODULE_CHILDREN), precedidas da rota índice.
 * Esconder a aba é só UX: a API revalida cada permissão (A01). */
export function ModuleTabs({ module, index, ariaLabel }: { module: string; index: { href: string; label: string }; ariaLabel: string }) {
  const { can, canGlobal, enabled } = usePlataforma();
  const tabs = [index, ...visibleTabs(MODULE_CHILDREN[module] ?? [], can, enabled, canGlobal)];
  return <SectionTabs tabs={tabs} ariaLabel={ariaLabel} className="mb-6 print:hidden" />;
}
