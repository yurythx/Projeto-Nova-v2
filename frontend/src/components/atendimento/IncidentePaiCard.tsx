"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { GitFork, Link2, Plus, Unlink, AlertCircle, CheckCircle2 } from "lucide-react";

import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { apiClient } from "@/lib/api/client";
import {
  type Solicitacao,
  type Detalhe,
  rotuloEstado,
  tomEstado,
  incidentePaiApi,
} from "@/lib/atendimento/atendimento";

export function IncidentePaiCard({
  solicitacao,
  podeEditar,
  onAtualizado,
}: {
  solicitacao: Solicitacao;
  podeEditar: boolean;
  onAtualizado: () => void;
}) {
  const { run, pending } = useAction();
  const [filhos, setFilhos] = useState<Solicitacao[]>([]);
  const [carregandoFilhos, setCarregandoFilhos] = useState(false);
  const [modalVincularAberto, setModalVincularAberto] = useState(false);
  const [idOuNumeroFilho, setIdOuNumeroFilho] = useState("");
  const [erroVincular, setErroVincular] = useState<string | null>(null);

  // Busca dados do chamado pai se este chamado for filho
  const [chamadoPai, setChamadoPai] = useState<Solicitacao | null>(null);

  useEffect(() => {
    if (solicitacao.incidente_pai_id) {
      void apiClient
        .get<Detalhe>(`v1/atendimento/solicitacoes/${solicitacao.incidente_pai_id}`)
        .then((res) => {
          if (res.data) setChamadoPai(res.data.solicitacao);
        })
        .catch(() => setChamadoPai(null));
    } else {
      setChamadoPai(null);
    }
  }, [solicitacao.incidente_pai_id]);

  // Carrega lista de chamados filhos agrupados sob este chamado
  const carregarFilhos = () => {
    setCarregandoFilhos(true);
    void incidentePaiApi
      .listarFilhos(solicitacao.id)
      .then((res) => {
        setFilhos(res.data ?? []);
      })
      .catch(() => setFilhos([]))
      .finally(() => setCarregandoFilhos(false));
  };

  useEffect(() => {
    carregarFilhos();
  }, [solicitacao.id]);

  const desvincularPai = async () => {
    await run(
      () => incidentePaiApi.vincular(solicitacao.id, null),
      "Chamado desvinculado do incidente pai com sucesso.",
    );
    onAtualizado();
  };

  const vincularFilho = async (e: React.FormEvent) => {
    e.preventDefault();
    setErroVincular(null);
    const alvo = idOuNumeroFilho.trim();
    if (!alvo) return;

    try {
      // Se for número ou ID, resolve primeiro
      let filhoId = alvo;
      if (!alvo.includes("-") && !isNaN(Number(alvo))) {
        // É número do chamado: busca a solicitação pelo número na fila
        const busca = await apiClient.get<any>(`v1/atendimento/fila?q=${alvo}`);
        const achado = (busca.data?.items ?? []).find((s: Solicitacao) => s.numero === Number(alvo));
        if (!achado) {
          setErroVincular(`Chamado #${alvo} não encontrado.`);
          return;
        }
        filhoId = achado.id;
      }

      await run(
        () => incidentePaiApi.vincular(filhoId, solicitacao.id),
        "Chamado correlato vinculado com sucesso.",
      );
      setModalVincularAberto(false);
      setIdOuNumeroFilho("");
      carregarFilhos();
      onAtualizado();
    } catch (err: any) {
      setErroVincular(err.message || "Não foi possível vincular o chamado.");
    }
  };

  const desvincularFilho = async (filhoId: string) => {
    await run(
      () => incidentePaiApi.vincular(filhoId, null),
      "Chamado filho desvinculado.",
    );
    carregarFilhos();
  };

  return (
    <Card className="border-l-4 border-l-blue-600">
      <CardHeader className="pb-3 flex flex-row items-center justify-between">
        <CardTitle className="text-sm font-semibold flex items-center gap-2">
          <GitFork size={16} className="text-blue-600" />
          Agrupamento de Incidentes (AIOps)
        </CardTitle>
        {podeEditar && !solicitacao.incidente_pai_id && (
          <Button
            size="sm"
            variant="secondary"
            className="text-xs h-7 gap-1"
            onClick={() => setModalVincularAberto(true)}
          >
            <Plus size={13} /> Vincular chamado filho
          </Button>
        )}
      </CardHeader>

      <CardContent className="space-y-3">
        {/* Caso este chamado seja um FILHO */}
        {solicitacao.incidente_pai_id && (
          <div className="p-3 bg-blue-50 dark:bg-blue-950/40 rounded-lg border border-blue-200 dark:border-blue-900 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div className="flex items-start gap-2.5">
              <Link2 size={16} className="text-blue-600 mt-0.5 shrink-0" />
              <div>
                <div className="text-xs font-semibold text-blue-950 dark:text-blue-200">
                  Chamado dependente vinculado ao Incidente Pai:
                </div>
                {chamadoPai ? (
                  <Link
                    href={`/atendimento/solicitacoes/${chamadoPai.id}`}
                    className="text-sm font-bold text-primary hover:underline flex items-center gap-1.5 mt-0.5"
                  >
                    <span>#{chamadoPai.numero} — {chamadoPai.titulo}</span>
                    <Badge tone={tomEstado(chamadoPai.estado)} className="text-[10px]">
                      {rotuloEstado(chamadoPai.estado)}
                    </Badge>
                  </Link>
                ) : (
                  <span className="text-xs text-muted font-mono">{solicitacao.incidente_pai_id}</span>
                )}
                <p className="text-[11px] text-muted-foreground mt-0.5">
                  A solução técnica deste chamado é coordenada a partir do incidente pai.
                </p>
              </div>
            </div>

            {podeEditar && (
              <Button
                variant="ghost"
                size="sm"
                className="text-xs text-danger hover:text-danger h-7 shrink-0 gap-1"
                disabled={pending}
                onClick={desvincularPai}
              >
                <Unlink size={13} /> Desvincular
              </Button>
            )}
          </div>
        )}

        {/* Caso este chamado possua FILHOS agrupados */}
        {filhos.length > 0 && (
          <div className="space-y-2">
            <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider flex items-center gap-1.5">
              <span>Chamados filhos correlatos ({filhos.length}):</span>
            </div>
            <div className="divide-y rounded-lg border border-surface-border bg-surface overflow-hidden">
              {filhos.map((filho) => (
                <div
                  key={filho.id}
                  className="p-2.5 flex items-center justify-between gap-3 text-xs hover:bg-muted/30 transition-colors"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <span className="font-mono font-bold text-primary">#{filho.numero}</span>
                    <Link
                      href={`/atendimento/solicitacoes/${filho.id}`}
                      className="font-medium text-foreground hover:underline truncate max-w-xs"
                    >
                      {filho.titulo}
                    </Link>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    <span className="text-muted text-[11px] hidden sm:inline">
                      {filho.solicitante_nome}
                    </span>
                    <Badge tone={tomEstado(filho.estado)} className="text-[10px]">
                      {rotuloEstado(filho.estado)}
                    </Badge>
                    {podeEditar && (
                      <button
                        type="button"
                        aria-label="Desvincular chamado filho"
                        className="p-1 text-muted hover:text-danger transition-colors"
                        onClick={() => desvincularFilho(filho.id)}
                      >
                        <Unlink size={13} />
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {!solicitacao.incidente_pai_id && filhos.length === 0 && (
          <p className="text-xs text-muted">
            Nenhum chamado correlato agrupado. Se este chamado for decorrente de uma falha massiva,
            você pode vincular chamados filhos para solucionar todos de forma sincronizada.
          </p>
        )}
      </CardContent>

      {/* Modal para Vincular Chamado Filho */}
      <Dialog
        open={modalVincularAberto}
        onClose={() => setModalVincularAberto(false)}
        title="Vincular Chamado Filho (AIOps)"
      >
        <form onSubmit={vincularFilho} className="space-y-4">
          {erroVincular && (
            <div className="p-3 bg-red-100 text-red-700 text-xs rounded border border-red-200">
              {erroVincular}
            </div>
          )}

          <div>
            <label className="text-xs font-semibold text-muted-foreground block mb-1">
              Número do Chamado ou ID (UUID)
            </label>
            <Input
              required
              placeholder="Ex.: 601 ou 00000000-0000-..."
              value={idOuNumeroFilho}
              onChange={(e) => setIdOuNumeroFilho(e.target.value)}
            />
            <p className="text-[11px] text-muted mt-1">
              O chamado filho passará a ser coordenado sob este chamado pai (#{solicitacao.numero}).
            </p>
          </div>

          <div className="flex justify-end gap-2 pt-2 border-t">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setModalVincularAberto(false)}
            >
              Cancelar
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? "Vinculando..." : "Vincular Chamado"}
            </Button>
          </div>
        </form>
      </Dialog>
    </Card>
  );
}

