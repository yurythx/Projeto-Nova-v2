"use client";

import { type ChangeEvent, type FormEvent, useState } from "react";

import { AssinaturaPad } from "@/components/atendimento/AssinaturaPad";
import { useFilaCampo } from "@/components/atendimento/useFilaCampo";
import { useToast } from "@/components/notifications/ToastProvider";
import { ConfirmButton } from "@/components/plataforma/ConfirmButton";
import { fmtDateTime } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { filaCampo, lerPosicao, novoId, type Acao } from "@/lib/atendimento/campo";
import {
  AVISO_LOCALIZACAO,
  ESTADO_VISITA,
  distancia,
  type Visita,
} from "@/lib/atendimento/atendimento";

type CheckinPendente = Extract<Acao, { tipo: "checkin" }>;

/** Visitas da solicitação. A equipe vê todas; o técnico registra a sua:
 * chegada, fotos, assinatura e saída — com ou sem rede. */
export function VisitasCard({
  solicitacaoId,
  visitas,
  tecnico,
  meuId,
  finalizada,
  onMudou,
}: {
  solicitacaoId: string;
  visitas: Visita[];
  tecnico: boolean;
  meuId?: string;
  finalizada: boolean;
  onMudou: () => void;
}) {
  const { showToast } = useToast();
  const [assinatura, setAssinatura] = useState<Blob | null>(null);
  const [ocupado, setOcupado] = useState(false);
  const { pendentes, executar, atualizar } = useFilaCampo(onMudou);
  const noAparelho = pendentes.filter(
    (a): a is CheckinPendente => a.tipo === "checkin" && a.solicitacaoId === solicitacaoId,
  );

  async function agir(fn: () => Promise<unknown>) {
    setOcupado(true);
    try {
      await fn();
      await atualizar();
    } catch (err) {
      showToast({
        title: "Não foi possível registrar",
        description: (err as Error).message,
        tone: "danger",
      });
    } finally {
      setOcupado(false);
    }
  }

  // Visita aberta minha: a do servidor ou o check-in que ainda está no aparelho.
  const servidorIds = new Set(visitas.map((v) => v.id));
  const pendentesAqui = noAparelho.filter((a) => !servidorIds.has(a.visitaId));
  const minhaAberta =
    visitas.find((v) => v.estado === "em_andamento" && v.tecnico_id === meuId)?.id ??
    pendentesAqui[0]?.visitaId;

  function chegar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const local = String(new FormData(e.currentTarget).get("local")).trim();
    void agir(async () => {
      const ponto = await lerPosicao();
      await executar({
        id: novoId(),
        tipo: "checkin",
        solicitacaoId,
        visitaId: crypto.randomUUID(),
        local,
        ponto,
        capturadaEm: new Date().toISOString(),
      });
    });
  }

  function foto(e: ChangeEvent<HTMLInputElement>) {
    const arquivo = e.target.files?.[0];
    e.target.value = "";
    if (!arquivo || !minhaAberta) return;
    void agir(() =>
      executar({
        id: novoId(),
        tipo: "evidencia",
        visitaId: minhaAberta,
        evidencia: "foto",
        nome: arquivo.name,
        arquivo,
      }),
    );
  }

  function sair(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const visitaId = minhaAberta!;
    void agir(async () => {
      const ponto = await lerPosicao();
      if (assinatura) {
        await filaCampo().enfileirar({
          id: novoId(),
          tipo: "evidencia",
          visitaId,
          evidencia: "assinatura",
          nome: "assinatura.png",
          arquivo: assinatura,
        });
      }
      await executar({
        id: novoId(),
        tipo: "checkout",
        visitaId,
        ponto,
        capturadaEm: new Date().toISOString(),
        assinanteNome: String(fd.get("assinante")).trim(),
        observacao: String(fd.get("observacao")).trim(),
      });
      setAssinatura(null);
    });
  }

  async function verEvidencia(visitaId: string, evidenciaId: string) {
    try {
      const { data } = await apiClient.get<{ url: string }>(
        `v1/atendimento/visitas/${visitaId}/evidencias/${evidenciaId}`,
      );
      window.open(data.url, "_blank", "noopener,noreferrer");
    } catch (err) {
      showToast({
        title: "Não foi possível abrir",
        description: (err as Error).message,
        tone: "danger",
      });
    }
  }

  if (visitas.length === 0 && pendentesAqui.length === 0 && !(tecnico && !finalizada)) return null;

  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
        <CardTitle as="h2">Visitas</CardTitle>
        {pendentes.length > 0 && (
          <Badge tone="warning" role="status">
            {pendentes.length} registro(s) aguardando conexão
          </Badge>
        )}
      </CardHeader>
      <CardContent className="flex flex-col gap-4 pb-5">
        {visitas.length > 0 && (
          <ul className="flex flex-col gap-3">
            {visitas.map((v) => (
              <li
                key={v.id}
                className="flex flex-col gap-1 rounded-lg border border-surface-border p-3 text-sm"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-medium">
                    {v.tecnico_nome}
                    {v.local ? ` · ${v.local}` : ""}
                  </span>
                  <Badge tone={ESTADO_VISITA[v.estado].tone}>{ESTADO_VISITA[v.estado].label}</Badge>
                </div>
                <p className="text-xs text-muted">
                  Chegada {fmtDateTime(v.inicio_em)}
                  {v.fim_em ? ` · saída ${fmtDateTime(v.fim_em)}` : ""}
                  {v.distancia_m !== undefined ? ` · ${distancia(v.distancia_m)} do destino` : ""}
                </p>
                {v.fora_da_area && <Badge tone="warning">Check-in fora da área do destino</Badge>}
                {v.assinante_nome && <p className="text-xs">Recebido por {v.assinante_nome}</p>}
                {v.observacao && (
                  <p className="whitespace-pre-wrap text-xs text-muted">{v.observacao}</p>
                )}
                {v.evidencias.length > 0 && (
                  <div className="flex flex-wrap gap-2">
                    {v.evidencias.map((e, i) => (
                      <Button
                        key={e.id}
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={() => void verEvidencia(v.id, e.id)}
                      >
                        {e.tipo === "assinatura" ? "Ver assinatura" : `Ver foto ${i + 1}`}
                      </Button>
                    ))}
                  </div>
                )}
              </li>
            ))}
          </ul>
        )}
        {pendentesAqui.map((a) => (
          <p
            key={a.id}
            role="status"
            className="rounded-lg border border-dashed border-warning/60 p-3 text-sm"
          >
            Chegada registrada em {fmtDateTime(a.capturadaEm)}, guardada no aparelho até a conexão
            voltar.
          </p>
        ))}

        {tecnico && !finalizada && !minhaAberta && (
          <form
            onSubmit={chegar}
            className="flex flex-col gap-3 border-t border-surface-border pt-4"
          >
            <p className="text-xs text-muted">{AVISO_LOCALIZACAO}</p>
            <Input
              id="visita-local"
              name="local"
              label="Local da visita"
              maxLength={300}
              autoComplete="off"
            />
            <div>
              <Button type="submit" loading={ocupado}>
                Registrar chegada
              </Button>
            </div>
          </form>
        )}

        {tecnico && minhaAberta && (
          <div className="flex flex-col gap-4 border-t border-surface-border pt-4">
            <label className="inline-flex w-fit cursor-pointer items-center gap-2 rounded-lg border border-surface-border px-4 py-2 text-sm font-medium hover:bg-black/5 dark:hover:bg-white/5">
              Adicionar foto
              <input
                type="file"
                accept="image/*"
                capture="environment"
                onChange={foto}
                className="sr-only"
              />
            </label>
            <form onSubmit={sair} className="flex flex-col gap-3">
              <fieldset className="flex flex-col gap-2">
                <legend className="mb-1 text-sm font-medium">
                  Assinatura de quem recebeu o atendimento
                </legend>
                <AssinaturaPad onPronta={setAssinatura} />
              </fieldset>
              <Input
                id="visita-assinante"
                name="assinante"
                label="Nome de quem recebeu"
                maxLength={160}
                autoComplete="off"
              />
              <Textarea
                id="visita-observacao"
                name="observacao"
                label="Observação"
                maxLength={4000}
                rows={2}
              />
              <p className="text-xs text-muted">{AVISO_LOCALIZACAO}</p>
              <div>
                <Button type="submit" loading={ocupado}>
                  Registrar saída
                </Button>
              </div>
            </form>
            {/* Fora do formulário: o diálogo de confirmação não pode enviar a saída. */}
            <div>
              <ConfirmButton
                title="Cancelar a visita?"
                description="Use quando a chegada foi registrada por engano."
                confirmLabel="Cancelar visita"
                onConfirm={() =>
                  agir(() => executar({ id: novoId(), tipo: "cancelar", visitaId: minhaAberta }))
                }
              >
                Cancelar visita
              </ConfirmButton>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
