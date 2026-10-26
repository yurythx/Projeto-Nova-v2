"use client";

import { colunasEmOrdem, type ColunaFaixa } from "@/components/plataforma/Faixas";
import { ConfiguracaoDeColunas } from "@/components/plataforma/ConfiguracaoDeColunas";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import { ESTADO_ATIVO, type EstadoAtivo } from "@/lib/servicos/servicos";

const FABRICA: ColunaFaixa[] = colunasEmOrdem(
  (Object.keys(ESTADO_ATIVO) as EstadoAtivo[]).map((id) => ({
    id,
    titulo: ESTADO_ATIVO[id].label,
    tom: ESTADO_ATIVO[id].tone,
  })),
);

export default function ColunasDeAtivosPage() {
  const { can } = usePlataforma();
  return (
    <ConfiguracaoDeColunas
      quadro="ativos"
      fabrica={FABRICA}
      titulo="Colunas de ativos"
      voltar="/ativos"
      pode={can("ativos:manage")}
    />
  );
}
