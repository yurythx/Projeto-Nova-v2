import type { ReactNode } from "react";

import { ModuleTabs } from "@/components/layout/ModuleTabs";

/** Abas do atendimento: o portal de quem abre (índice), a fila de quem
 * atende e o cadastro de quem gere — cada uma só com a permissão. */
export default function AtendimentoLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <ModuleTabs
        module="atendimento"
        index={{ href: "/atendimento", label: "Minhas solicitações" }}
        ariaLabel="Seções do atendimento"
      />
      {children}
    </>
  );
}
