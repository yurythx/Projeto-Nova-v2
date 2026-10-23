"use client";

import { ExternalLink, HardDrive, Link2, Unlink } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Select } from "@/components/ui/Select";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import { ESTADO_ATIVO, TIPO_ATIVO, type Ativo } from "@/lib/servicos/servicos";

export function AtivoVinculadoCard({
  solicitacaoId,
  tecnico,
  finalizada,
}: {
  solicitacaoId: string;
  tecnico: boolean;
  finalizada: boolean;
}) {
  const { can, enabled } = usePlataforma();
  const ativoModuloAtivo = enabled("ativos");
  const leAtivos = can("ativos:read") || can("ativos:manage");

  const ativosQuery = useApiQuery<Ativo[]>(ativoModuloAtivo && leAtivos ? "v1/ativos" : null);
  const ativos = ativosQuery.data ?? [];

  const storageKey = `atendimento:ativo:${solicitacaoId}`;
  const [ativoId, setAtivoId] = useState<string>(() => {
    if (typeof window !== "undefined") {
      return localStorage.getItem(storageKey) || "";
    }
    return "";
  });

  const [selecionado, setSelecionado] = useState("");

  if (!ativoModuloAtivo || !leAtivos) return null;

  const ativoAtual = ativos.find((a) => a.id === ativoId);

  function associar(id: string) {
    setAtivoId(id);
    if (typeof window !== "undefined") {
      localStorage.setItem(storageKey, id);
    }
    setSelecionado("");
  }

  function desassociar() {
    setAtivoId("");
    if (typeof window !== "undefined") {
      localStorage.removeItem(storageKey);
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
            href="/ativos"
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
                <span className="font-semibold text-foreground">{ativoAtual.nome}</span>
                <span className="font-mono text-xs text-muted">[{ativoAtual.identificador}]</span>
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
                  onClick={desassociar}
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
                      label: `${a.nome} (${a.identificador})`,
                    }))}
                  />
                </div>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  disabled={!selecionado}
                  onClick={() => associar(selecionado)}
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
