"use client";

import { ExternalLink, HardDrive, Link2, Unlink } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import { ESTADO_ATIVO, TIPO_ATIVO, type Ativo } from "@/lib/servicos/servicos";

export function AtivoVinculadoCard({
  solicitacaoId,
  ativoId: ativoIdProp,
  ativoNome: ativoNomeProp,
  tecnico,
  finalizada,
  onMudou,
}: {
  solicitacaoId: string;
  ativoId?: string;
  ativoNome?: string;
  tecnico: boolean;
  finalizada: boolean;
  onMudou?: () => void;
}) {
  const { can, enabled } = usePlataforma();
  const ativoModuloAtivo = enabled("ativos");
  const leAtivos = can("ativos:read") || can("ativos:manage") || can("atendimento:atender");

  const ativosQuery = useApiQuery<Ativo[]>(ativoModuloAtivo && leAtivos ? "v1/ativos" : null);
  const ativos = ativosQuery.data ?? [];

  const [ativoIdLocal, setAtivoIdLocal] = useState<string | null>(null);
  const ativoId = ativoIdLocal !== null ? ativoIdLocal : (ativoIdProp ?? "");

  const [selecionado, setSelecionado] = useState("");
  const { run, pending } = useAction();

  if (!ativoModuloAtivo || !leAtivos) return null;

  const ativoAtual =
    ativos.find((a) => a.id === ativoId) ??
    (ativoId && ativoNomeProp
      ? ({
          id: ativoId,
          nome: ativoNomeProp,
          tipo: "hardware",
          estado: "em_uso",
          identificador: "",
          descricao: "",
        } as Ativo)
      : undefined);

  async function associar(id: string) {
    setAtivoIdLocal(id);
    setSelecionado("");
    const ok = await run(
      () => apiClient.put(`v1/atendimento/solicitacoes/${solicitacaoId}/ativo`, { ativo_id: id }),
      "Ativo vinculado ao chamado",
    );
    if (ok) {
      onMudou?.();
    }
  }

  async function desassociar() {
    setAtivoIdLocal("");
    const ok = await run(
      () => apiClient.put(`v1/atendimento/solicitacoes/${solicitacaoId}/ativo`, { ativo_id: null }),
      "Ativo desvinculado do chamado",
    );
    if (ok) {
      onMudou?.();
    }
  }

  return (
    <Card data-testid="card-ativo-vinculado">
      <CardHeader className="flex flex-row items-center justify-between">
        <div className="flex items-center gap-2">
          <HardDrive className="h-5 w-5 text-primary" />
          <CardTitle as="h2">Item de Configuração / Ativo (CMDB)</CardTitle>
        </div>
        {ativoAtual && (
          <Link
            href={`/ativos/${ativoAtual.id}`}
            className="flex items-center gap-1 text-xs text-primary hover:underline font-medium"
          >
            <span>Ver no parque</span>
            <ExternalLink className="h-3 w-3" />
          </Link>
        )}
      </CardHeader>
      <CardContent className="flex flex-col gap-3 pb-5">
        {ativoAtual ? (
          <div className="flex flex-col gap-2 rounded-lg border border-surface-border bg-surface p-3 text-sm">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <Link
                  href={`/ativos/${ativoAtual.id}`}
                  className="font-semibold text-foreground hover:text-primary hover:underline"
                >
                  {ativoAtual.nome}
                </Link>
                {ativoAtual.identificador && (
                  <span className="font-mono text-xs text-muted">[{ativoAtual.identificador}]</span>
                )}
              </div>
              <div className="flex items-center gap-2">
                <Badge tone="info">{TIPO_ATIVO[ativoAtual.tipo] || ativoAtual.tipo}</Badge>
                <Badge tone={ESTADO_ATIVO[ativoAtual.estado]?.tone ?? "neutral"}>
                  {ESTADO_ATIVO[ativoAtual.estado]?.label ?? ativoAtual.estado}
                </Badge>
              </div>
            </div>
            {ativoAtual.descricao && (
              <p className="text-xs text-muted line-clamp-2">{ativoAtual.descricao}</p>
            )}
            {tecnico && !finalizada && (
              <div className="pt-2 flex justify-end">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  disabled={pending}
                  onClick={() => void desassociar()}
                  className="text-danger hover:text-danger flex items-center gap-1 text-xs"
                >
                  <Unlink className="h-3.5 w-3.5" />
                  Desvincular ativo
                </Button>
              </div>
            )}
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            <p className="text-sm text-muted">
              Nenhum ativo ou equipamento vinculado diretamente a esta solicitação.
            </p>
            {tecnico && !finalizada && ativos.length > 0 && (
              <div className="flex flex-wrap items-end gap-2 pt-1">
                <div className="min-w-64">
                  <Select
                    id="solicitacao-ativo-select"
                    label="Vincular equipamento ou sistema"
                    value={selecionado}
                    onChange={(e) => setSelecionado(e.target.value)}
                    placeholder="Escolha um ativo do catálogo..."
                    options={ativos.map((a) => ({
                      value: a.id,
                      label: a.identificador ? `${a.nome} (${a.identificador})` : a.nome,
                    }))}
                  />
                </div>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  disabled={!selecionado || pending}
                  onClick={() => void associar(selecionado)}
                  className="flex items-center gap-1.5"
                >
                  <Link2 className="h-3.5 w-3.5" />
                  Associar ao chamado
                </Button>
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
