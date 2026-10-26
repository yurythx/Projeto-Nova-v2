"use client";

import { type FormEvent, useState } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Segredo {
  id: string;
  nome: string;
  usuario: string;
  endereco?: string;
}

interface Revelado extends Segredo {
  senha: string;
  notas: string;
}

/** O chamado guarda o id. A senha só volta pelo cofre, para quem já está autenticado. */
export function SegredoVinculadoCard({
  solicitacaoId,
  segredoId,
  tecnico,
  finalizada,
  onMudou,
}: {
  solicitacaoId: string;
  segredoId?: string;
  tecnico: boolean;
  finalizada: boolean;
  onMudou: () => void;
}) {
  const { can, enabled } = usePlataforma();
  const aberto = enabled("cofre") && tecnico;
  const ve = can("cofre:read") || can("cofre:reveal") || can("cofre:manage") || can("cofre:grant");
  const revela = can("cofre:reveal") || can("cofre:manage");
  const lista = useApiQuery<Segredo[]>(aberto && ve ? "v1/cofre/segredos" : null);
  const { run, pending } = useAction();
  const [escolhido, setEscolhido] = useState("");
  const [senha, setSenha] = useState<string | null>(null);

  if (!aberto) return null;

  const nome = (lista.data ?? []).find((item) => item.id === segredoId)?.nome;

  async function vincular(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const ok = await run(
      async () => {
        await apiClient.put(`v1/atendimento/solicitacoes/${solicitacaoId}/segredo`, {
          segredo_id: escolhido || null,
        });
        onMudou();
      },
      escolhido ? "Segredo vinculado" : "Vínculo removido",
    );
    if (ok === undefined) return;
    setSenha(null);
  }

  async function revelar() {
    if (!segredoId) return;
    const item = await run(
      () =>
        apiClient
          .post<Revelado>(`v1/cofre/segredos/${segredoId}/revelar`, {})
          .then((res) => res.data),
      "Senha revelada. A consulta ficou na auditoria.",
    );
    if (!item) return;
    setSenha(item.senha);
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Senha de serviço</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <p className="text-sm text-muted">
          {segredoId
            ? (nome ?? "Segredo fora do seu alcance")
            : "Nenhuma senha vinculada. O chamado não guarda o valor."}
        </p>
        {senha && (
          <Input
            id={`senha-do-chamado-${solicitacaoId}`}
            label="Senha revelada"
            readOnly
            value={senha}
            autoComplete="off"
          />
        )}
        {revela && segredoId && !senha && (
          <div>
            <Button
              type="button"
              variant="secondary"
              loading={pending}
              onClick={() => void revelar()}
            >
              Revelar senha do chamado
            </Button>
          </div>
        )}
        {ve && !finalizada && (
          <form
            className="flex flex-col gap-2 sm:flex-row sm:items-end"
            onSubmit={(e) => void vincular(e)}
          >
            <Select
              id={`segredo-do-chamado-${solicitacaoId}`}
              label="Segredo"
              placeholder="Nenhum"
              value={escolhido}
              onChange={(e) => setEscolhido(e.target.value)}
              options={(lista.data ?? []).map((item) => ({
                value: item.id,
                label: [item.nome, item.endereco || item.usuario].filter(Boolean).join(" · "),
              }))}
            />
            <Button type="submit" variant="secondary" loading={pending}>
              Vincular segredo
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  );
}
