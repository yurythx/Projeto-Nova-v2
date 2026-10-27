"use client";

import { Sparkles } from "lucide-react";
import { useState } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { NIVEIS } from "@/lib/atendimento/atendimento";
import type { DecisaoIA, EstadoIA, SugestaoIA } from "@/lib/atendimento/ia";

function nivel(n: number, feminino = false): string {
  const item = NIVEIS.find((x) => x.value === n);
  if (!item) return String(n);
  return feminino ? item.feminino : item.label;
}

const DECIDIDA: Record<DecisaoIA, string> = {
  aceita: "Você aceitou esta sugestão.",
  recusada: "Você descartou esta sugestão.",
};

/** Sugestões da IA para quem atende (ADR 065). A IA não muda o chamado:
 * aplicar a classificação passa pela mesma rota da classificação manual.
 * Sem IA configurada, o cartão não aparece. */
export function AssistenteIA({
  solicitacaoId,
  editavel,
  onAplicarClassificacao,
}: {
  solicitacaoId: string;
  editavel: boolean;
  onAplicarClassificacao: (impacto: number, urgencia: number) => Promise<boolean>;
}) {
  const estado = useApiQuery<EstadoIA>("v1/atendimento/ia");
  const { run, pending } = useAction();
  const [classificacao, setClassificacao] = useState<SugestaoIA | null>(null);
  const [resumo, setResumo] = useState<SugestaoIA | null>(null);
  const [pedindo, setPedindo] = useState<"classificacao" | "resumo" | null>(null);

  if (!estado.data?.disponivel) return null;

  async function pedir(tarefa: "classificacao" | "resumo") {
    setPedindo(tarefa);
    const res = await run(() =>
      apiClient.post<SugestaoIA>(`v1/atendimento/solicitacoes/${solicitacaoId}/ia/${tarefa}`, {}),
    );
    setPedindo(null);
    if (!res) return;
    if (tarefa === "classificacao") setClassificacao(res.data);
    else setResumo(res.data);
  }

  async function decidir(sugestao: SugestaoIA, aceita: boolean, sucesso?: string) {
    const res = await run(
      () =>
        apiClient.post<SugestaoIA>(`v1/atendimento/ia/sugestoes/${sugestao.id}/decisao`, {
          aceita,
        }),
      sucesso,
    );
    if (!res) return;
    if (sugestao.tipo === "classificacao") setClassificacao(res.data);
    else setResumo(res.data);
  }

  async function aplicar(sugestao: SugestaoIA) {
    const c = sugestao.classificacao;
    if (!c) return;
    // Primeiro a classificação de sempre (prazo, evento e auditoria); só
    // então a decisão, para a taxa de aceite contar o que valeu de fato.
    if (await onAplicarClassificacao(c.impacto, c.urgencia)) await decidir(sugestao, true);
  }

  return (
    <Card data-testid="assistente-ia">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Sparkles aria-hidden className="h-4 w-4 text-primary" />
          Assistente de IA
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <p className="text-xs text-muted">
          A IA sugere; você decide. Nada muda no chamado sem a sua confirmação.
        </p>
        <div className="flex flex-wrap gap-2">
          {editavel && (
            <Button
              type="button"
              variant="secondary"
              size="sm"
              loading={pedindo === "classificacao"}
              disabled={pending}
              onClick={() => void pedir("classificacao")}
            >
              Sugerir classificação
            </Button>
          )}
          <Button
            type="button"
            variant="secondary"
            size="sm"
            loading={pedindo === "resumo"}
            disabled={pending}
            onClick={() => void pedir("resumo")}
          >
            Resumir o chamado
          </Button>
        </div>

        <div aria-live="polite" className="flex flex-col gap-4">
          {classificacao?.classificacao && (
            <section
              aria-label="Classificação sugerida"
              className="flex flex-col gap-2 rounded-md border border-surface-border p-3 text-sm"
            >
              <p className="font-medium">
                Impacto {nivel(classificacao.classificacao.impacto).toLowerCase()} · Urgência{" "}
                {nivel(classificacao.classificacao.urgencia, true).toLowerCase()}
              </p>
              <p className="text-muted">{classificacao.classificacao.motivo}</p>
              {classificacao.decisao ? (
                <p role="status" className="text-xs text-muted">
                  {DECIDIDA[classificacao.decisao]}
                </p>
              ) : (
                <div className="flex flex-wrap gap-2">
                  <Button
                    type="button"
                    size="sm"
                    disabled={pending || !editavel}
                    onClick={() => void aplicar(classificacao)}
                  >
                    Aplicar esta classificação
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    disabled={pending}
                    onClick={() => void decidir(classificacao, false, "Sugestão descartada")}
                  >
                    Descartar
                  </Button>
                </div>
              )}
              <p className="text-xs text-muted">Sugerido por: {classificacao.conexao}</p>
            </section>
          )}

          {resumo?.resumo && (
            <section
              aria-label="Resumo do chamado"
              className="flex flex-col gap-2 rounded-md border border-surface-border p-3 text-sm"
            >
              <p className="whitespace-pre-line">{resumo.resumo}</p>
              {resumo.decisao ? (
                <p role="status" className="text-xs text-muted">
                  {resumo.decisao === "aceita"
                    ? "Obrigado: o resumo ajudou."
                    : "Obrigado: o resumo não ajudou."}
                </p>
              ) : (
                <div className="flex flex-wrap gap-2">
                  <Button
                    type="button"
                    size="sm"
                    variant="secondary"
                    disabled={pending}
                    onClick={() => void decidir(resumo, true)}
                  >
                    Ajudou
                  </Button>
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    disabled={pending}
                    onClick={() => void decidir(resumo, false)}
                  >
                    Não ajudou
                  </Button>
                </div>
              )}
              <p className="text-xs text-muted">
                Gerado por: {resumo.conexao}. Confira antes de repassar.
              </p>
            </section>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
