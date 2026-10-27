"use client";

import { useState } from "react";

import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import type { Tarefa } from "@/lib/atendimento/atendimento";

const ESTADO_TAREFA: Record<
  Tarefa["estado"],
  { label: string; tone: "warning" | "success" | "neutral" }
> = {
  pendente: { label: "Pendente", tone: "warning" },
  concluida: { label: "Concluída", tone: "success" },
  dispensada: { label: "Dispensada", tone: "neutral" },
};

/** Tarefas de cada equipe no pedido (ADR 068). Conclui quem é da equipe;
 * dispensa, com motivo, quem atende o chamado. Enquanto houver pendente,
 * o pedido não é dado por atendido. */
export function TarefasCard({
  solicitacao,
  tarefas,
  podeAgir,
  onMudou,
}: {
  solicitacao: string;
  tarefas: Tarefa[];
  podeAgir: boolean;
  onMudou: () => void;
}) {
  const { run, pending } = useAction();
  const [aberta, setAberta] = useState<{ id: string; concluir: boolean } | null>(null);
  const [texto, setTexto] = useState("");
  if (tarefas.length === 0) return null;
  const pendentes = tarefas.filter((t) => t.estado === "pendente").length;

  async function fazer(t: Tarefa, concluir: boolean) {
    const ok = await run(
      () =>
        apiClient.post(`v1/atendimento/solicitacoes/${solicitacao}/tarefas/${t.id}`, {
          concluir,
          observacao: texto.trim(),
        }),
      concluir ? "Tarefa concluída" : "Tarefa dispensada",
    );
    if (ok) {
      setAberta(null);
      setTexto("");
      onMudou();
    }
  }

  function abrir(id: string, concluir: boolean) {
    setAberta({ id, concluir });
    setTexto("");
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Tarefas</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 pb-5">
        <p className="text-sm text-muted">
          {pendentes > 0
            ? `${pendentes} de ${tarefas.length} pendente${pendentes > 1 ? "s" : ""}. O pedido só é dado por atendido depois de todas.`
            : "Todas as tarefas foram feitas."}
        </p>
        <ol className="flex flex-col gap-3">
          {tarefas.map((t) => (
            <li
              key={t.id}
              className="flex flex-col gap-2 border-t border-surface-border pt-3 first:border-0 first:pt-0"
            >
              <div className="flex flex-wrap items-center gap-2 text-sm">
                <Badge tone={ESTADO_TAREFA[t.estado].tone}>{ESTADO_TAREFA[t.estado].label}</Badge>
                <span className="font-medium">{t.titulo}</span>
                <span className="text-muted">· {t.equipe_nome}</span>
              </div>
              {t.estado !== "pendente" && (
                <p className="text-xs text-muted">
                  {t.feita_por_nome || "Alguém"}
                  {t.feita_em ? ` em ${fmtDateTime(t.feita_em)}` : ""}
                  {t.observacao ? `: ${t.observacao}` : ""}
                </p>
              )}
              {t.estado === "pendente" && podeAgir && aberta?.id !== t.id && (
                <div className="flex flex-wrap gap-2">
                  {t.minha && (
                    <Button type="button" size="sm" onClick={() => abrir(t.id, true)}>
                      Concluir {t.titulo}
                    </Button>
                  )}
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    onClick={() => abrir(t.id, false)}
                  >
                    Dispensar {t.titulo}
                  </Button>
                </div>
              )}
              {aberta?.id === t.id && (
                <div className="flex flex-col gap-2">
                  <Textarea
                    id={`tarefa-${t.id}-texto`}
                    label={aberta.concluir ? "Observação (opcional)" : "Motivo da dispensa"}
                    value={texto}
                    maxLength={1000}
                    rows={2}
                    onChange={(e) => setTexto(e.target.value)}
                  />
                  <div className="flex flex-wrap gap-2">
                    <Button
                      type="button"
                      size="sm"
                      loading={pending}
                      disabled={!aberta.concluir && !texto.trim()}
                      onClick={() => void fazer(t, aberta.concluir)}
                    >
                      {aberta.concluir ? "Confirmar conclusão" : "Confirmar dispensa"}
                    </Button>
                    <Button type="button" size="sm" variant="ghost" onClick={() => setAberta(null)}>
                      Voltar
                    </Button>
                  </div>
                </div>
              )}
            </li>
          ))}
        </ol>
      </CardContent>
    </Card>
  );
}
