"use client";

import type { RefObject } from "react";

import { Button } from "@/components/ui/Button";
import { useApiQuery } from "@/lib/api/swr";

interface Resposta {
  publico: string;
  interna: string;
}

/** Coloca o texto na caixa. Quem atende ainda aperta Enviar. */
export function RespostasProntas({
  ofertaId,
  texto,
  interno,
}: {
  ofertaId: string;
  texto: RefObject<HTMLTextAreaElement | null>;
  interno: RefObject<HTMLInputElement | null>;
}) {
  const respostas = useApiQuery<Resposta>(ofertaId ? `v1/atendimento/ofertas/${ofertaId}/respostas` : null);
  const item = respostas.data;
  if (!item || (!item.publico && !item.interna)) return null;

  function colocar(valor: string, nota: boolean) {
    if (texto.current) texto.current.value = valor;
    if (interno.current) interno.current.checked = nota;
    texto.current?.focus();
  }

  return (
    <div className="flex flex-wrap gap-2">
      {item.publico ? (
        <Button type="button" variant="secondary" onClick={() => colocar(item.publico, false)}>
          Colocar resposta pronta
        </Button>
      ) : null}
      {item.interna ? (
        <Button type="button" variant="secondary" onClick={() => colocar(item.interna, true)}>
          Colocar nota interna
        </Button>
      ) : null}
    </div>
  );
}
