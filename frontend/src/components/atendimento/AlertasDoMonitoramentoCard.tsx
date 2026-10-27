"use client";

import { BellRing } from "lucide-react";
import { useState, type FormEvent } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { Oferta } from "@/lib/atendimento/atendimento";

interface Configuracao {
  oferta_id: string | null;
}

/** Em que oferta os alertas do monitoramento abrem incidente (ADR 067).
 * Só quem gere o atendimento em toda a instituição escolhe; para os
 * demais a API responde 403 e o cartão não aparece. */
export function AlertasDoMonitoramentoCard({ ofertas }: { ofertas: Oferta[] }) {
  const cfg = useApiQuery<Configuracao>("v1/atendimento/eventos/configuracao");
  const { run, pending } = useAction();
  const [escolha, setEscolha] = useState<string | null>(null);

  if (!cfg.data) return null;
  const atual = escolha ?? cfg.data.oferta_id ?? "";
  const incidentes = ofertas.filter((o) => o.tipo === "incidente" && o.ativa);

  async function salvar(e: FormEvent) {
    e.preventDefault();
    const ok = await run(
      () => apiClient.put("v1/atendimento/eventos/configuracao", { oferta_id: atual || null }),
      atual
        ? "Os alertas passam a abrir incidente nesta oferta"
        : "Os alertas não abrem mais incidente",
    );
    if (ok) void cfg.mutate();
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2" className="flex items-center gap-2">
          <BellRing size={18} aria-hidden="true" className="text-primary" /> Alertas do
          monitoramento
        </CardTitle>
      </CardHeader>
      <CardContent className="pb-5">
        <form onSubmit={salvar} className="flex flex-col gap-3">
          <p className="text-sm text-muted">
            Cada alerta do monitoramento (API fora, disco cheio, backup atrasado) abre um incidente
            nesta oferta. O mesmo alerta de novo vira nota no mesmo chamado, e a volta ao normal não
            conclui nada sozinha.
          </p>
          <Select
            id="eventos-oferta"
            label="Oferta dos alertas"
            placeholder="Nenhuma: os alertas não abrem chamado"
            options={incidentes.map((o) => ({ value: o.id, label: o.nome }))}
            value={atual}
            onChange={(e) => setEscolha(e.target.value)}
          />
          <div>
            <Button type="submit" loading={pending}>
              Salvar
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
