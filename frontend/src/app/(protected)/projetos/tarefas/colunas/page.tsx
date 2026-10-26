"use client";

import { colunasEmOrdem, type ColunaFaixa } from "@/components/plataforma/Faixas";
import { ConfiguracaoDeColunas } from "@/components/plataforma/ConfiguracaoDeColunas";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import { ESTADO_TAREFA, type EstadoTarefa } from "@/lib/projetos/projetos";

const FABRICA: ColunaFaixa[] = colunasEmOrdem(
  (Object.keys(ESTADO_TAREFA) as EstadoTarefa[]).map((id) => ({
    id,
    titulo: ESTADO_TAREFA[id].label,
    tom: ESTADO_TAREFA[id].tone,
  })),
);

export default function ColunasDeTarefasPage() {
  const { can } = usePlataforma();
  return (
    <ConfiguracaoDeColunas
      quadro="tarefas"
      fabrica={FABRICA}
      titulo="Colunas de tarefas"
      voltar="/projetos"
      pode={can("projetos:manage")}
    />
  );
}
