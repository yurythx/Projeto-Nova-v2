"use client";

import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { ModeloTarefa, Oferta } from "@/lib/atendimento/atendimento";

const MAX_TAREFAS = 20;

interface Rascunho {
  titulo: string;
  equipe_id: string;
}

/** Modelo de tarefas de uma oferta de requisição (ADR 068): cada linha
 * vira a tarefa de uma equipe quando o pedido chega à fila. É a lista da
 * admissão ou do desligamento. */
export function TarefasDaOferta({
  oferta,
  equipes,
  onFechar,
}: {
  oferta: Oferta;
  equipes: { id: string; nome: string }[];
  onFechar: () => void;
}) {
  const salvo = useApiQuery<ModeloTarefa[]>(`v1/atendimento/ofertas/${oferta.id}/tarefas`);
  const { run, pending } = useAction();
  const [rascunho, setRascunho] = useState<Rascunho[] | null>(null);
  const linhas =
    rascunho ?? (salvo.data ?? []).map((m) => ({ titulo: m.titulo, equipe_id: m.equipe_id }));
  const opcoes = equipes.map((e) => ({ value: e.id, label: e.nome }));

  function mudar(i: number, parcial: Partial<Rascunho>) {
    setRascunho(linhas.map((l, j) => (j === i ? { ...l, ...parcial } : l)));
  }

  async function salvar() {
    const ok = await run(
      () => apiClient.put(`v1/atendimento/ofertas/${oferta.id}/tarefas`, { tarefas: linhas }),
      linhas.length ? "Tarefas salvas" : "A oferta não tem mais tarefas",
    );
    if (ok) {
      setRascunho(null);
      void salvo.mutate();
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Tarefas: {oferta.nome}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4 pb-5">
        <p className="text-sm text-muted">
          Quando o pedido chega à fila, cada linha vira uma tarefa e a equipe é avisada. O pedido só
          é dado por atendido com todas concluídas ou dispensadas. Mudar a lista vale para os
          próximos pedidos.
        </p>
        {linhas.map((linha, i) => (
          <fieldset
            key={i}
            className="grid gap-2 rounded-md border border-surface-border p-3 sm:grid-cols-[1fr_14rem_auto] sm:items-end"
          >
            <legend className="px-1 text-sm font-semibold">Tarefa {i + 1}</legend>
            <Input
              id={`tarefa-${i}-titulo`}
              label="O que fazer"
              value={linha.titulo}
              maxLength={120}
              onChange={(e) => mudar(i, { titulo: e.target.value })}
            />
            <Select
              id={`tarefa-${i}-equipe`}
              label="Equipe"
              placeholder="Escolha a equipe"
              options={opcoes}
              value={linha.equipe_id}
              onChange={(e) => mudar(i, { equipe_id: e.target.value })}
            />
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={() => setRascunho(linhas.filter((_, j) => j !== i))}
            >
              <Trash2 size={14} aria-hidden="true" className="mr-1" /> Tirar a tarefa {i + 1}
            </Button>
          </fieldset>
        ))}
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            variant="secondary"
            disabled={linhas.length >= MAX_TAREFAS}
            onClick={() => setRascunho([...linhas, { titulo: "", equipe_id: "" }])}
          >
            <Plus size={14} aria-hidden="true" className="mr-1" /> Acrescentar tarefa
          </Button>
          <Button type="button" loading={pending} onClick={() => void salvar()}>
            Salvar tarefas
          </Button>
          <Button type="button" variant="ghost" onClick={onFechar}>
            Fechar
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
