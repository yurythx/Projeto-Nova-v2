"use client";

import { type FormEvent, useState } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { apiClient } from "@/lib/api/client";
import type { Experiencia, MedidaXLA } from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

const MEDIDAS: {
  chave: keyof Experiencia;
  rotulo: string;
  explica: string;
  sufixo: string;
  meta: "csat_min" | "facilidade_min" | "reabertura_max" | "idas_max" | "espera_max_horas";
}[] = [
  { chave: "csat", rotulo: "Nota", explica: "de 1 a 5", sufixo: "", meta: "csat_min" },
  {
    chave: "facilidade",
    rotulo: "Facilidade",
    explica: "“foi fácil?”, de 1 a 5",
    sufixo: "",
    meta: "facilidade_min",
  },
  {
    chave: "reabertura",
    rotulo: "Reabertura",
    explica: "pedidos que precisaram ser reabertos",
    sufixo: "%",
    meta: "reabertura_max",
  },
  {
    chave: "idas_e_voltas",
    rotulo: "Idas e voltas",
    explica: "vezes em que a pessoa teve de responder, por pedido",
    sufixo: "",
    meta: "idas_max",
  },
  {
    chave: "espera_horas",
    rotulo: "Espera",
    explica: "mediana, em horas corridas, da abertura à conclusão",
    sufixo: " h",
    meta: "espera_max_horas",
  },
];

function numero(v: number) {
  return v.toLocaleString("pt-BR");
}

function Situacao({ m, sufixo }: { m: MedidaXLA; sufixo: string }) {
  if (m.meta == null) return <span className="text-xs text-muted">sem meta</span>;
  const alvo = `${m.maior_melhor ? "≥" : "≤"} ${numero(m.meta)}${sufixo}`;
  if (m.atingida == null) return <span className="text-xs text-muted">meta {alvo}</span>;
  return (
    <Badge tone={m.atingida ? "success" : "danger"}>
      {m.atingida ? "Meta atingida" : "Abaixo da meta"} ({alvo})
    </Badge>
  );
}

/** Experiência de quem pede (XLA, ADR 069): como foi para a pessoa, não
 * só se o processo cumpriu o prazo. A gestão de toda a instituição define
 * as metas. */
export function ExperienciaCard({
  experiencia,
  onMetasSalvas,
}: {
  experiencia: Experiencia;
  onMetasSalvas?: () => void;
}) {
  const { canGlobal } = usePlataforma();
  const { run, pending } = useAction();
  const [editando, setEditando] = useState(false);

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const corpo: Record<string, number | null> = {};
    for (const m of MEDIDAS) {
      const v = String(fd.get(m.meta) ?? "").trim();
      corpo[m.meta] = v === "" ? null : Number(v.replace(",", "."));
    }
    const ok = await run(
      () => apiClient.put("v1/atendimento/xla", corpo),
      "Metas de experiência salvas",
    );
    if (ok) {
      setEditando(false);
      onMetasSalvas?.();
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Experiência de quem pede</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4 pb-5">
        <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          {MEDIDAS.map((d) => {
            const m = experiencia[d.chave];
            return (
              <div key={d.chave} className="flex flex-col gap-1">
                <dt className="text-xs text-muted" title={d.explica}>
                  {d.rotulo}
                </dt>
                <dd className="flex flex-col gap-1">
                  <span className="text-lg font-semibold">
                    {m.valor == null ? "sem amostra" : `${numero(m.valor)}${d.sufixo}`}
                  </span>
                  <span className="text-xs text-muted">
                    {d.explica} · {m.amostra} {m.amostra === 1 ? "registro" : "registros"}
                  </span>
                  <Situacao m={m} sufixo={d.sufixo} />
                </dd>
              </div>
            );
          })}
        </dl>
        {canGlobal("atendimento:manage") &&
          (editando ? (
            <form onSubmit={(e) => void salvar(e)} className="flex flex-col gap-3">
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
                {MEDIDAS.map((d) => (
                  <Input
                    key={d.meta}
                    id={`meta-${d.meta}`}
                    name={d.meta}
                    label={`${d.rotulo}: ${experiencia[d.chave].maior_melhor ? "mínimo" : "máximo"}`}
                    inputMode="decimal"
                    defaultValue={experiencia[d.chave].meta?.toString() ?? ""}
                  />
                ))}
              </div>
              <p className="text-xs text-muted">Em branco, a medida fica sem meta.</p>
              <div className="flex flex-wrap gap-2">
                <Button type="submit" loading={pending}>
                  Salvar metas
                </Button>
                <Button type="button" variant="ghost" onClick={() => setEditando(false)}>
                  Cancelar
                </Button>
              </div>
            </form>
          ) : (
            <div>
              <Button type="button" variant="secondary" onClick={() => setEditando(true)}>
                Definir metas
              </Button>
            </div>
          ))}
      </CardContent>
    </Card>
  );
}
