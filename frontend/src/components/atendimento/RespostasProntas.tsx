"use client";

import { Button } from "@/components/ui/Button";
import { useApiQuery } from "@/lib/api/swr";

interface Resposta {
  publico: string;
  interna: string;
}

/** Oferece o texto pronto. Quem abriu o formulário o coloca na caixa, e
 * quem atende ainda aperta Enviar. */
export function RespostasProntas({
  ofertaId,
  onColocar,
}: {
  ofertaId: string;
  onColocar: (texto: string, interna: boolean) => void;
}) {
  const respostas = useApiQuery<Resposta>(
    ofertaId ? `v1/atendimento/ofertas/${ofertaId}/respostas` : null,
  );
  const item = respostas.data;
  if (!item || (!item.publico && !item.interna)) return null;

  return (
    <div className="flex flex-wrap gap-2">
      {item.publico ? (
        <Button type="button" variant="secondary" onClick={() => onColocar(item.publico, false)}>
          Colocar resposta pronta
        </Button>
      ) : null}
      {item.interna ? (
        <Button type="button" variant="secondary" onClick={() => onColocar(item.interna, true)}>
          Colocar nota interna
        </Button>
      ) : null}
    </div>
  );
}
