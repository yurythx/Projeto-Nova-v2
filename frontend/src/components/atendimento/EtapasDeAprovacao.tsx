"use client";

import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { EtapaAprovacao, Oferta, Pessoa } from "@/lib/atendimento/atendimento";

const MAX_ETAPAS = 5;

interface Rascunho {
  nome: string;
  aprovadores: string[];
}

/** Etapas de aprovação de uma oferta de requisição (ADR 068): o pedido só
 * chega à fila depois de passar por todas. Basta um aprovador por etapa. */
export function EtapasDeAprovacao({ oferta, onFechar }: { oferta: Oferta; onFechar: () => void }) {
  const salvas = useApiQuery<EtapaAprovacao[]>(`v1/atendimento/ofertas/${oferta.id}/etapas`);
  const pessoas = useApiQuery<Pessoa[]>(
    oferta.unidade_id ? `v1/atendimento/candidatos?unidade_id=${oferta.unidade_id}` : null,
  );
  const { run, pending } = useAction();
  const [rascunho, setRascunho] = useState<Rascunho[] | null>(null);
  const etapas =
    rascunho ?? (salvas.data ?? []).map((e) => ({ nome: e.nome, aprovadores: e.aprovadores }));

  function mudar(i: number, parcial: Partial<Rascunho>) {
    setRascunho(etapas.map((e, j) => (j === i ? { ...e, ...parcial } : e)));
  }

  async function salvar() {
    const ok = await run(
      () => apiClient.put(`v1/atendimento/ofertas/${oferta.id}/etapas`, { etapas }),
      etapas.length ? "Etapas de aprovação salvas" : "A oferta não pede mais aprovação",
    );
    if (ok) {
      setRascunho(null);
      void salvas.mutate();
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Etapas de aprovação: {oferta.nome}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4 pb-5">
        <p className="text-sm text-muted">
          O pedido desta oferta só chega à fila depois de aprovado em todas as etapas, em ordem.
          Basta um aprovador por etapa; quem pediu não aprova o próprio pedido. Sem etapas, o pedido
          vai direto para a fila.
        </p>
        {etapas.map((etapa, i) => (
          <fieldset
            key={i}
            className="flex flex-col gap-2 rounded-md border border-surface-border p-3"
          >
            <legend className="px-1 text-sm font-semibold">Etapa {i + 1}</legend>
            <Input
              id={`etapa-${i}-nome`}
              label="Nome da etapa"
              value={etapa.nome}
              maxLength={80}
              onChange={(e) => mudar(i, { nome: e.target.value })}
            />
            <fieldset className="flex flex-col gap-1">
              <legend className="text-sm font-medium">Quem aprova</legend>
              {(pessoas.data ?? []).length === 0 && (
                <p className="text-xs text-muted">Ninguém lotado na unidade da oferta.</p>
              )}
              {(pessoas.data ?? []).map((p) => (
                <label key={p.id} className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    className="h-4 w-4 accent-primary"
                    checked={etapa.aprovadores.includes(p.id)}
                    onChange={(e) =>
                      mudar(i, {
                        aprovadores: e.target.checked
                          ? [...etapa.aprovadores, p.id]
                          : etapa.aprovadores.filter((a) => a !== p.id),
                      })
                    }
                  />
                  {p.nome}
                </label>
              ))}
            </fieldset>
            <div>
              <Button
                type="button"
                size="sm"
                variant="ghost"
                onClick={() => setRascunho(etapas.filter((_, j) => j !== i))}
              >
                <Trash2 size={14} aria-hidden="true" className="mr-1" /> Tirar a etapa {i + 1}
              </Button>
            </div>
          </fieldset>
        ))}
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="secondary"
            disabled={etapas.length >= MAX_ETAPAS}
            onClick={() => setRascunho([...etapas, { nome: "", aprovadores: [] }])}
          >
            <Plus size={14} aria-hidden="true" className="mr-1" /> Acrescentar etapa
          </Button>
          <Button type="button" loading={pending} onClick={() => void salvar()}>
            Salvar etapas
          </Button>
          <Button type="button" variant="ghost" onClick={onFechar}>
            Fechar
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
