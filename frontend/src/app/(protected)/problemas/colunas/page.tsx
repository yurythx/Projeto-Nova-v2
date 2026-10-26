"use client";

import { ConfiguracaoDeColunas } from "@/components/plataforma/ConfiguracaoDeColunas";
import { COLUNAS_PROBLEMA } from "@/components/servicos/ProblemasKanban";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

export default function ColunasDeProblemasPage() {
  const { can } = usePlataforma();
  return (
    <ConfiguracaoDeColunas
      quadro="problemas"
      fabrica={COLUNAS_PROBLEMA}
      titulo="Colunas de problemas"
      voltar="/problemas"
      pode={can("problemas:manage")}
    />
  );
}
