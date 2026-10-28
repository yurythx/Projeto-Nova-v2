"use client";

import type { ReactNode } from "react";

import { ModuleTabs } from "@/components/layout/ModuleTabs";
import { soSolicitante } from "@/lib/plataforma/central";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

/** Abas do atendimento: o portal de quem abre (índice), a fila de quem
 * atende e o cadastro de quem gere — cada uma só com a permissão. */
export default function AtendimentoLayout({ children }: { children: ReactNode }) {
  const { me } = usePlataforma();
  return (
    <>
      {/* Na central, quem só pede não tem abas: tudo sai dos blocos. */}
      {!soSolicitante(me) && (
        <ModuleTabs
          module="atendimento"
          index={{ href: "/atendimento", label: "Central de atendimento" }}
          ariaLabel="Seções do atendimento"
        />
      )}
      {children}
    </>
  );
}
