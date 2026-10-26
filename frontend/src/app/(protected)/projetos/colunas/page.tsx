"use client";

import { ConfiguracaoDeColunas } from "@/components/plataforma/ConfiguracaoDeColunas";
import { COLUNAS_PROJETO } from "@/components/projetos/ProjetosKanban";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

export default function ColunasDeProjetosPage() {
  const { can } = usePlataforma();
  return (
    <ConfiguracaoDeColunas
      quadro="projetos"
      fabrica={COLUNAS_PROJETO}
      titulo="Colunas de projetos"
      voltar="/projetos"
      pode={can("projetos:manage")}
    />
  );
}
