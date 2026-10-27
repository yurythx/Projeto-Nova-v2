"use client";

import { type FormEvent, useState } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { ContratoDeApoio, Solicitacao } from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

/** Contrato de apoio do chamado (ADR 071): o contrato do fornecedor que o
 * chamado espera. O tempo em "aguardando terceiro" se mede contra o prazo
 * copiado aqui. Só para quem atende, e só com o plug-in Contratos ligado. */
export function ContratoDeApoioCard({
  solicitacao,
  finalizada,
  onMudou,
}: {
  solicitacao: Solicitacao;
  finalizada: boolean;
  onMudou: () => void;
}) {
  const { enabled } = usePlataforma();
  const ligado = enabled("contratos");
  const lista = useApiQuery<ContratoDeApoio[]>(
    ligado && !finalizada ? "v1/atendimento/contratos-de-apoio" : null,
  );
  const { run, pending } = useAction();
  const [escolhido, setEscolhido] = useState(solicitacao.contrato_apoio_id ?? "");

  if (!ligado && !solicitacao.contrato_apoio_id) return null;

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const ok = await run(
      () =>
        apiClient.put(`v1/atendimento/solicitacoes/${solicitacao.id}/contrato`, {
          contrato_id: escolhido || null,
        }),
      escolhido ? "Contrato de apoio vinculado" : "Contrato de apoio retirado",
    );
    if (ok !== undefined) onMudou();
  }

  const opcoes = (lista.data ?? []).map((c) => ({
    value: c.id,
    label: c.prazo_horas ? `${c.rotulo} · prazo ${c.prazo_horas} h` : `${c.rotulo} · sem prazo`,
  }));

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Contrato de apoio</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 pb-5">
        {solicitacao.contrato_apoio ? (
          <p className="text-sm">
            {solicitacao.contrato_apoio}
            {solicitacao.contrato_prazo_horas
              ? ` · o fornecedor tem ${solicitacao.contrato_prazo_horas} h`
              : " · sem prazo no contrato"}
          </p>
        ) : (
          <p className="text-sm text-muted">
            Quando o chamado depender de um fornecedor, aponte o contrato: o tempo em
            &quot;aguardando terceiro&quot; passa a contar contra o prazo dele.
          </p>
        )}
        {ligado && !finalizada && (
          <form onSubmit={(e) => void salvar(e)} className="flex flex-wrap items-end gap-2">
            <Select
              id="contrato-apoio"
              label="Contrato"
              placeholder="Nenhum"
              options={opcoes}
              value={escolhido}
              onChange={(e) => setEscolhido(e.target.value)}
            />
            <Button type="submit" variant="secondary" loading={pending}>
              Salvar contrato
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  );
}
