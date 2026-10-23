"use client";

import { CheckCircle2, MessageSquareHeart, Star, ThumbsUp } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import type { Solicitacao } from "@/lib/atendimento/atendimento";

const ROTULOS_NOTA: Record<number, string> = {
  1: "Muito insatisfeito",
  2: "Insatisfeito",
  3: "Regular",
  4: "Satisfeito",
  5: "Excelente / Muito satisfeito",
};

interface AvaliacaoAtendimentoCardProps {
  solicitacao: Solicitacao;
  solicitante: boolean;
  tecnico: boolean;
  onAvaliado?: () => void;
}

export function AvaliacaoAtendimentoCard({
  solicitacao,
  solicitante,
  tecnico,
  onAvaliado,
}: AvaliacaoAtendimentoCardProps) {
  const finalizada = solicitacao.estado === "resolvido" || solicitacao.estado === "encerrado";
  const storageKey = `atendimento:csat:${solicitacao.id}`;

  const [avaliacaoSalva, setAvaliacaoSalva] = useState<{ nota: number; comentario: string } | null>(
    () => {
      if (typeof window !== "undefined") {
        const item = localStorage.getItem(storageKey);
        if (item) {
          try {
            return JSON.parse(item);
          } catch {
            return null;
          }
        }
      }
      return null;
    },
  );

  const [nota, setNota] = useState<number>(5);
  const [hoverNota, setHoverNota] = useState<number>(0);
  const [comentario, setComentario] = useState("");
  const [enviando, setEnviando] = useState(false);

  if (!finalizada) return null;
  // Se for técnico e o usuário não avaliou ainda, não exibe nada
  if (tecnico && !solicitante && !avaliacaoSalva) return null;

  async function enviarAvaliacao(e: React.FormEvent) {
    e.preventDefault();
    setEnviando(true);
    const rotulo = ROTULOS_NOTA[nota] || "";
    const textoFinal = comentario.trim()
      ? `[Pesquisa de Satisfação - CSAT]\nNota: ${nota}/5 estrelas (${rotulo})\nComentário: "${comentario.trim()}"`
      : `[Pesquisa de Satisfação - CSAT]\nNota: ${nota}/5 estrelas (${rotulo})`;

    try {
      await apiClient.post(`v1/atendimento/solicitacoes/${solicitacao.id}/comentarios`, {
        texto: textoFinal,
        interno: false,
      });

      const dados = { nota, comentario: comentario.trim() };
      if (typeof window !== "undefined") {
        localStorage.setItem(storageKey, JSON.stringify(dados));
      }
      setAvaliacaoSalva(dados);
      onAvaliado?.();
    } catch {
      // Ignora erro e salva localmente
      const dados = { nota, comentario: comentario.trim() };
      if (typeof window !== "undefined") {
        localStorage.setItem(storageKey, JSON.stringify(dados));
      }
      setAvaliacaoSalva(dados);
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
        {avaliacaoSalva ? (
          <div className="flex flex-col gap-2 rounded-lg border border-success/30 bg-success/5 p-4 text-sm">
            <div className="flex items-center gap-2 text-success font-semibold">
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
                    star <= avaliacaoSalva.nota
                      ? "fill-warning text-warning"
                      : "text-surface-border fill-surface-border"
                  }`}
                />
              ))}
              <span className="ml-2 font-medium text-foreground">
                {avaliacaoSalva.nota}/5 — {ROTULOS_NOTA[avaliacaoSalva.nota]}
              </span>
            </div>
            {avaliacaoSalva.comentario && (
              <p className="mt-1 text-xs text-muted italic">
                "{avaliacaoSalva.comentario}"
              </p>
            )}
          </div>
        ) : (
          <form onSubmit={enviarAvaliacao} className="flex flex-col gap-3">
            <p className="text-xs text-muted">
              Sua opinião é fundamental para aprimorarmos a qualidade dos nossos serviços. Como você
              avalia a resolução deste chamado?
            </p>

            <div className="flex flex-col gap-1.5">
              <div className="flex items-center gap-1">
                {[1, 2, 3, 4, 5].map((star) => {
                  const ativa = hoverNota > 0 ? star <= hoverNota : star <= nota;
                  return (
                    <button
                      key={star}
                      type="button"
                      onClick={() => setNota(star)}
                      onMouseEnter={() => setHoverNota(star)}
                      onMouseLeave={() => setHoverNota(0)}
                      aria-label={`${star} estrela${star > 1 ? "s" : ""}: ${ROTULOS_NOTA[star]}`}
                      className="p-1 text-foreground transition-transform hover:scale-110 focus:outline-hidden"
                    >
                      <Star
                        className={`h-6 w-6 transition-colors ${
                          ativa ? "fill-warning text-warning" : "text-muted hover:text-warning"
                        }`}
                      />
                    </button>
                  );
                })}
                <span className="ml-2 text-xs font-semibold text-foreground">
                  {ROTULOS_NOTA[hoverNota || nota]}
                </span>
              </div>
            </div>

            <Textarea
              id="csat-comentario"
              label="Comentário adicional (opcional)"
              placeholder="Conte o que achou do atendimento ou deixe um elogio..."
              rows={2}
              maxLength={500}
              value={comentario}
              onChange={(e) => setComentario(e.target.value)}
            />

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
