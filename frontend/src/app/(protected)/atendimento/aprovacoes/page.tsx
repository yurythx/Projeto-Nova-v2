"use client";

import { useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { Solicitacao } from "@/lib/atendimento/atendimento";

const MAX_MOTIVO = 1000;

/** Pedidos que esperam a decisão de quem consulta (ADR 068). Não pede
 * permissão do atendimento: a chefia aprova sem atender. Recusar exige
 * motivo, que vira a resposta a quem pediu. */
export default function AtendimentoAprovacoesPage() {
  const lista = useApiQuery<Solicitacao[]>("v1/atendimento/aprovacoes");
  const { run, pending } = useAction();
  const [recusando, setRecusando] = useState<string | null>(null);
  const [motivo, setMotivo] = useState("");

  async function decidir(s: Solicitacao, aprovar: boolean) {
    const ok = await run(
      () =>
        apiClient.post(`v1/atendimento/solicitacoes/${s.id}/aprovacao`, {
          aprovar,
          motivo: aprovar ? "" : motivo.trim(),
        }),
      aprovar ? `Pedido #${s.numero} aprovado` : `Pedido #${s.numero} recusado`,
    );
    if (ok) {
      setRecusando(null);
      setMotivo("");
      void lista.mutate();
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Aprovações"
        description="Pedidos que esperam a sua decisão. Aprovado na última etapa, o pedido vai para a fila da equipe."
      />
      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={(lista.data ?? []).length === 0}
        emptyTitle="Nada esperando a sua aprovação"
        emptyDescription="Quando um pedido depender de você, ele aparece aqui e você recebe um aviso."
      >
        <ul className="flex flex-col gap-4">
          {(lista.data ?? []).map((s) => (
            <li key={s.id}>
              <Card>
                <CardHeader>
                  <CardTitle as="h2">
                    #{s.numero} {s.titulo}
                  </CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-3 pb-5">
                  <p className="text-sm text-muted">
                    {s.oferta_nome} · pedido por {s.solicitante_nome} em {fmtDateTime(s.created_at)}
                  </p>
                  {s.descricao ? (
                    <p className="whitespace-pre-wrap text-sm">{s.descricao}</p>
                  ) : null}
                  {recusando === s.id ? (
                    <div className="flex flex-col gap-2">
                      <Textarea
                        id={`motivo-${s.id}`}
                        label="Motivo da recusa"
                        value={motivo}
                        maxLength={MAX_MOTIVO}
                        rows={3}
                        onChange={(e) => setMotivo(e.target.value)}
                      />
                      <div className="flex flex-wrap gap-2">
                        <Button
                          type="button"
                          variant="danger"
                          loading={pending}
                          disabled={!motivo.trim()}
                          onClick={() => void decidir(s, false)}
                        >
                          Confirmar recusa
                        </Button>
                        <Button type="button" variant="ghost" onClick={() => setRecusando(null)}>
                          Voltar
                        </Button>
                      </div>
                    </div>
                  ) : (
                    <div className="flex flex-wrap gap-2">
                      <Button type="button" loading={pending} onClick={() => void decidir(s, true)}>
                        Aprovar
                      </Button>
                      <Button
                        type="button"
                        variant="secondary"
                        onClick={() => {
                          setRecusando(s.id);
                          setMotivo("");
                        }}
                      >
                        Recusar
                      </Button>
                    </div>
                  )}
                </CardContent>
              </Card>
            </li>
          ))}
        </ul>
      </DataState>
    </div>
  );
}
