"use client";

import { FAIXAS_FILA } from "@/components/atendimento/Kanban";
import { ConfiguracaoDeColunas } from "@/components/plataforma/ConfiguracaoDeColunas";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

export default function ColunasDaFilaPage() {
  const { can } = usePlataforma();
  return (
    <ConfiguracaoDeColunas
      quadro="fila"
      fabrica={FAIXAS_FILA}
      titulo="Colunas da fila"
      voltar="/atendimento/fila"
      pode={can("atendimento:manage")}
    />
  );
}
