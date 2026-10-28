import { ArrowRightLeft, CirclePlus, MessageSquare } from "lucide-react";
import type { ReactNode } from "react";

import { fmtDateTime } from "@/components/plataforma/useAction";
import {
  rotuloEstado,
  type Comentario,
  type Passagem,
  type Solicitacao,
} from "@/lib/atendimento/atendimento";

interface Evento {
  chave: string;
  em: string;
  icone: ReactNode;
  titulo: ReactNode;
  texto?: string;
  meu?: boolean;
}

/**
 * O chamado como quem pediu o vê (ADR 078): a abertura, as mensagens e as
 * mudanças de estado numa só linha, da mais antiga para a mais nova. Nota
 * interna nunca entra, mesmo que chegue.
 */
export function LinhaDoTempo({
  solicitacao: s,
  comentarios,
  passagens,
  eu,
}: {
  solicitacao: Solicitacao;
  comentarios: Comentario[];
  passagens: Passagem[];
  eu?: string;
}) {
  const eventos: Evento[] = [
    {
      chave: "abertura",
      em: s.created_at,
      icone: <CirclePlus size={16} aria-hidden="true" />,
      titulo: <>Chamado aberto por {s.solicitante_nome}</>,
      texto: s.descricao,
      meu: true,
    },
    ...comentarios
      .filter((c) => !c.interno)
      .map((c) => ({
        chave: `c-${c.id}`,
        em: c.created_at,
        icone: <MessageSquare size={16} aria-hidden="true" />,
        titulo: <>{c.autor_nome || "Usuário removido"}</>,
        texto: c.texto,
        meu: !!eu && c.autor_id === eu,
      })),
    // A primeira passagem é a própria abertura.
    ...passagens
      .filter((p) => p.de)
      .map((p) => ({
        chave: `p-${p.id}`,
        em: p.em,
        icone: <ArrowRightLeft size={16} aria-hidden="true" />,
        titulo: (
          <>
            Passou para <strong>{rotuloEstado(p.para)}</strong>
            {p.autor_nome ? ` · ${p.autor_nome}` : ""}
          </>
        ),
      })),
  ].sort((a, b) => a.em.localeCompare(b.em));

  return (
    <ol aria-label="Linha do tempo do chamado" className="relative flex flex-col gap-4">
      {eventos.map((e) => (
        <li key={e.chave} className="flex gap-3">
          <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-primary">
            {e.icone}
          </span>
          <div
            className={`min-w-0 flex-1 ${
              e.texto
                ? `rounded-lg border p-3 ${e.meu ? "border-primary/30 bg-primary/5" : "border-surface-border"}`
                : "pt-1.5"
            }`}
          >
            <p className="flex flex-wrap items-center gap-x-2 text-sm">
              <span className="font-medium">{e.titulo}</span>
              <time dateTime={e.em} className="text-xs text-muted">
                {fmtDateTime(e.em)}
              </time>
            </p>
            {e.texto ? <p className="mt-1 whitespace-pre-wrap text-sm">{e.texto}</p> : null}
          </div>
        </li>
      ))}
    </ol>
  );
}
