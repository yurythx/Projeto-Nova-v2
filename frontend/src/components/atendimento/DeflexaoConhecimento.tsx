"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { apiClient } from "@/lib/api/client";
import type { SearchResponse, SearchResult } from "@/lib/plataforma/types";

/** Artigos publicados da wiki que casam com o que a pessoa está digitando.
 * A lista inventada saiu: só aparece o que a Busca Global devolve para o
 * módulo wiki, com a mesma permissão de leitura da página. */
export function DeflexaoConhecimento({
  termo,
  onResolvido,
}: {
  termo: string;
  onResolvido?: () => void;
}) {
  const consulta = termo.trim();
  // Cada resultado guarda a consulta de onde veio: digitar outra coisa
  // esconde o anterior sem apagar estado no meio da renderização.
  const [achado, setAchado] = useState<{ consulta: string; artigos: SearchResult[] }>({ consulta: "", artigos: [] });
  const [dispensadoEm, setDispensadoEm] = useState<string | null>(null);
  const artigos = consulta.length >= 3 && achado.consulta === consulta ? achado.artigos : [];
  const dispensado = dispensadoEm === consulta;

  useEffect(() => {
    if (consulta.length < 3) return;
    let vivo = true;
    const espera = setTimeout(() => {
      const params = new URLSearchParams({ q: consulta, module: "wiki", limit: "4" });
      void apiClient
        .get<SearchResponse>(`v1/search?${params}`)
        .then((resposta) => {
          if (!vivo) return;
          const vistos = (resposta.data.results ?? []).filter((item) => item.module === "wiki").slice(0, 4);
          setAchado({ consulta, artigos: vistos });
          if (vistos.length > 0) {
            void apiClient
              .post("v1/atendimento/deflexao", { artigos: vistos.map((item) => item.id), resolveu: false })
              .catch(() => undefined);
          }
        })
        .catch(() => {
          if (vivo) setAchado({ consulta, artigos: [] });
        });
    }, 300);
    return () => {
      vivo = false;
      clearTimeout(espera);
    };
  }, [consulta]);

  if (dispensado) {
    return (
      <p role="status" className="text-sm text-muted">
        Certo. A solicitação não foi enviada.
      </p>
    );
  }
  if (artigos.length === 0) return null;

  return (
    <div
      data-testid="deflexao-conhecimento"
      className="flex flex-col gap-2 rounded-lg border border-surface-border p-3"
    >
      <h3 className="text-sm font-semibold text-foreground">Artigos que podem resolver antes de abrir</h3>
      <ul className="flex flex-col gap-2">
        {artigos.map((artigo) => (
          <li key={artigo.id}>
            <Link href={artigo.url || "/wiki"} className="text-sm font-medium text-primary hover:underline">
              {artigo.title}
            </Link>
            {artigo.snippet ? <p className="text-xs text-muted line-clamp-2">{artigo.snippet}</p> : null}
          </li>
        ))}
      </ul>
      <button
        type="button"
        className="self-start text-sm text-muted underline-offset-2 hover:underline"
        onClick={() => {
          void apiClient
            .post("v1/atendimento/deflexao", { artigos: artigos.map((item) => item.id), resolveu: true })
            .catch(() => undefined);
          setDispensadoEm(consulta);
          onResolvido?.();
        }}
      >
        Não preciso mais abrir
      </button>
    </div>
  );
}
