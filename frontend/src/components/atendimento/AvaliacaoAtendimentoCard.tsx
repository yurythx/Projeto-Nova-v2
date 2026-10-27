"use client";

import { CheckCircle2, MessageSquareHeart, Star, ThumbsUp } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import type { Avaliacao, Solicitacao } from "@/lib/atendimento/atendimento";

const ROTULOS_NOTA: Record<number, string> = {
  1: "Muito insatisfeito",
  2: "Insatisfeito",
  3: "Regular",
  4: "Satisfeito",
  5: "Excelente / Muito satisfeito",
};

const ROTULOS_FACILIDADE: Record<number, string> = {
  1: "Muito difícil",
  2: "Difícil",
  3: "Nem fácil nem difícil",
  4: "Fácil",
  5: "Muito fácil",
};

interface AvaliacaoAtendimentoCardProps {
  solicitacao: Solicitacao;
  avaliacao?: Avaliacao | null;
  solicitante: boolean;
  tecnico: boolean;
  onAvaliado?: () => void;
}

export function AvaliacaoAtendimentoCard({
  solicitacao,
  avaliacao,
  solicitante,
  tecnico,
  onAvaliado,
}: AvaliacaoAtendimentoCardProps) {
  const podeAvaliar =
    solicitacao.estado === "resolvido" ||
    solicitacao.estado === "atendido" ||
    solicitacao.estado === "encerrado";

  const [nota, setNota] = useState<number>(5);
  const [hoverNota, setHoverNota] = useState<number>(0);
  const [facilidade, setFacilidade] = useState<number | null>(null);
  const [comentario, setComentario] = useState("");
  const [enviando, setEnviando] = useState(false);
  const [erro, setErro] = useState("");

  if (!podeAvaliar) return null;
  if (tecnico && !solicitante && !avaliacao) return null;

  async function enviarAvaliacao(e: React.FormEvent) {
    e.preventDefault();
    setEnviando(true);
    setErro("");
    try {
      await apiClient.post(`v1/atendimento/solicitacoes/${solicitacao.id}/avaliacao`, {
        nota,
        facilidade,
        comentario: comentario.trim(),
      });
      onAvaliado?.();
    } catch (err) {
      setErro(err instanceof Error ? err.message : "Não foi possível registrar a avaliação.");
    } finally {
      setEnviando(false);
    }
  }

  return (
    <Card data-testid="card-csat-avaliacao">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <div className="flex items-center gap-2">
          <MessageSquareHeart className="h-5 w-5 text-primary" />
          <CardTitle as="h2">Pesquisa de Satisfação (CSAT)</CardTitle>
        </div>
      </CardHeader>
      <CardContent className="pb-5">
        {avaliacao ? (
          <div className="flex flex-col gap-2 rounded-lg border border-success/30 bg-success/5 p-4 text-sm">
            <div className="flex items-center gap-2 font-semibold text-success">
              <CheckCircle2 className="h-4 w-4" />
              <span>
                {solicitante
                  ? "Obrigado por avaliar o atendimento!"
                  : "Avaliação registrada pelo solicitante:"}
              </span>
            </div>
            <div className="flex items-center gap-1.5 pt-1">
              {[1, 2, 3, 4, 5].map((star) => (
                <Star
                  key={star}
                  className={`h-5 w-5 ${
                    star <= avaliacao.nota
                      ? "fill-warning text-warning"
                      : "fill-surface-border text-surface-border"
                  }`}
                  aria-hidden
                />
              ))}
              <span className="ml-2 font-medium text-foreground">
                {avaliacao.nota}/5 — {ROTULOS_NOTA[avaliacao.nota]}
              </span>
            </div>
            {avaliacao.facilidade && (
              <p className="text-xs">
                Facilidade: {avaliacao.facilidade}/5 — {ROTULOS_FACILIDADE[avaliacao.facilidade]}
              </p>
            )}
            {avaliacao.comentario && (
              <p className="mt-1 text-xs text-muted italic">&quot;{avaliacao.comentario}&quot;</p>
            )}
          </div>
        ) : (
          <form onSubmit={enviarAvaliacao} className="flex flex-col gap-3">
            <p className="text-xs text-muted">
              Como você avalia a resolução deste chamado? A nota fica registrada no atendimento.
            </p>

            <div className="flex items-center gap-1" role="group" aria-label="Nota de 1 a 5">
              {[1, 2, 3, 4, 5].map((star) => {
                const ativa = hoverNota > 0 ? star <= hoverNota : star <= nota;
                return (
                  <button
                    key={star}
                    type="button"
                    onClick={() => setNota(star)}
                    onMouseEnter={() => setHoverNota(star)}
                    onMouseLeave={() => setHoverNota(0)}
                    aria-pressed={nota === star}
                    aria-label={`${star} estrela${star > 1 ? "s" : ""}: ${ROTULOS_NOTA[star]}`}
                    className="p-1 text-foreground focus:outline-hidden"
                  >
                    <Star
                      className={`h-6 w-6 ${ativa ? "fill-warning text-warning" : "text-muted"}`}
                    />
                  </button>
                );
              })}
              <span className="ml-2 text-xs font-semibold text-foreground">
                {ROTULOS_NOTA[hoverNota || nota]}
              </span>
            </div>

            <fieldset className="flex flex-col gap-1">
              <legend className="text-xs text-muted">
                Foi fácil conseguir o que você precisava? (opcional)
              </legend>
              <div className="flex flex-wrap gap-1" role="group" aria-label="Facilidade de 1 a 5">
                {[1, 2, 3, 4, 5].map((n) => (
                  <button
                    key={n}
                    type="button"
                    onClick={() => setFacilidade(facilidade === n ? null : n)}
                    aria-pressed={facilidade === n}
                    aria-label={`Facilidade ${n}: ${ROTULOS_FACILIDADE[n]}`}
                    className={`rounded-md border px-2 py-1 text-xs ${
                      facilidade === n
                        ? "border-primary bg-primary text-primary-foreground"
                        : "border-surface-border"
                    }`}
                  >
                    {n}
                  </button>
                ))}
                {facilidade && (
                  <span className="ml-1 self-center text-xs">{ROTULOS_FACILIDADE[facilidade]}</span>
                )}
              </div>
            </fieldset>

            <Textarea
              id="csat-comentario"
              label="Comentário adicional (opcional)"
              placeholder="Conte o que achou do atendimento"
              rows={2}
              maxLength={4000}
              value={comentario}
              onChange={(e) => setComentario(e.target.value)}
            />

            {erro && (
              <p role="alert" className="text-sm text-danger">
                {erro}
              </p>
            )}
            <div className="flex justify-end pt-1">
              <Button
                type="submit"
                variant="secondary"
                size="sm"
                loading={enviando}
                className="inline-flex items-center gap-1.5"
              >
                <ThumbsUp className="h-3.5 w-3.5" />
                Enviar avaliação
              </Button>
            </div>
          </form>
        )}
      </CardContent>
    </Card>
  );
}
