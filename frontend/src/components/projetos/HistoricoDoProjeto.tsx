"use client";

import { useApiQuery } from "@/lib/api/swr";

export interface MarcoProjeto {
  id: string;
  acao: string;
  autor_nome?: string;
  quando: string;
  texto: string;
}

function quando(iso: string) {
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return "";
  return data.toLocaleString("pt-BR", {
    timeZone: "America/Sao_Paulo",
    dateStyle: "short",
    timeStyle: "short",
  });
}

/** A trilha do projeto: abertura, mudanças de estado, responsável, reabertura e cancelamento. */
export function HistoricoDoProjeto({ projetoId }: { projetoId: string }) {
  const historico = useApiQuery<MarcoProjeto[]>(
    projetoId ? `v1/projetos/${projetoId}/historico` : null,
  );
  const itens = historico.data ?? [];
  if (itens.length === 0) return null;
  return (
    <section aria-label="Histórico do projeto" className="flex flex-col gap-2">
      <h2 className="text-sm font-medium text-foreground">Histórico</h2>
      <ol className="flex flex-col gap-2">
        {itens.map((m) => (
          <li key={m.id} className="text-sm text-muted">
            <span className="text-foreground">{m.texto}</span>
            {m.autor_nome ? ` · ${m.autor_nome}` : ""}
            {m.quando ? ` · ${quando(m.quando)}` : ""}
          </li>
        ))}
      </ol>
    </section>
  );
}
