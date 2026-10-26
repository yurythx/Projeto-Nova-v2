"use client";

import { ConfiguracaoDeColunas } from "@/components/plataforma/ConfiguracaoDeColunas";
import { COLUNAS_MUDANCA } from "@/components/servicos/MudancasKanban";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

export default function ColunasDeMudancasPage() {
  const { can } = usePlataforma();
  return (
    <ConfiguracaoDeColunas
      quadro="mudancas"
      fabrica={COLUNAS_MUDANCA}
      titulo="Colunas de mudanças"
      voltar="/mudancas"
      pode={can("mudancas:manage")}
    />
  );
}
