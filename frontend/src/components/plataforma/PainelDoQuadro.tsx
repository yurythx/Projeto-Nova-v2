"use client";

import { X } from "lucide-react";
import Link from "next/link";
import { useParams, usePathname, useRouter, useSearchParams } from "next/navigation";
import { createContext, useContext, useId, useState, type ReactNode } from "react";

import { ModalShell } from "@/components/ui/ModalShell";

const IdDoPainel = createContext<string | null>(null);
const Embutido = createContext(false);

/** Id do card aberto no painel. Na página de detalhe, cai no parâmetro da rota. */
export function useIdDaRota() {
  const params = useParams();
  const ctx = useContext(IdDoPainel);
  if (ctx) return ctx;
  const id = params?.id;
  return typeof id === "string" ? id : Array.isArray(id) ? (id[0] ?? "") : "";
}

/** Verdadeiro quando o detalhe está no painel, e não na página inteira. */
export function usePainelEmbutido() {
  return useContext(Embutido);
}

/**
 * Abre o detalhe por cima do quadro. A URL ganha `painel` para o voltar
 * do navegador fechar, sem trocar a página.
 */
export function usePainelNaUrl() {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const idUrl = params.get("painel");
  const [escolha, setEscolha] = useState<string | null>(idUrl);
  // Voltar e avançar do navegador trocam a URL: a escolha acompanha.
  const [urlVista, setUrlVista] = useState(idUrl);
  if (urlVista !== idUrl) {
    setUrlVista(idUrl);
    setEscolha(idUrl);
  }

  function href(extra: Record<string, string | null> = {}) {
    const q = new URLSearchParams(params.toString());
    if (escolha) q.set("painel", escolha);
    else q.delete("painel");
    for (const [chave, valor] of Object.entries(extra)) {
      if (valor == null) q.delete(chave);
      else q.set(chave, valor);
    }
    const texto = q.toString();
    return texto ? `${pathname}?${texto}` : pathname;
  }

  function abrir(id: string) {
    setEscolha(id);
    const q = new URLSearchParams(params.toString());
    q.set("painel", id);
    router.push(`${pathname}?${q.toString()}`, { scroll: false });
  }

  function fechar() {
    setEscolha(null);
    const q = new URLSearchParams(params.toString());
    q.delete("painel");
    const texto = q.toString();
    router.push(texto ? `${pathname}?${texto}` : pathname, { scroll: false });
  }

  return { id: escolha, abrir, fechar, href };
}

/** Detalhe do card no centro da tela, por cima do quadro. */
export function PainelDoQuadro({
  aberto,
  id,
  titulo,
  href,
  onClose,
  children,
}: {
  aberto: boolean;
  id: string | null;
  titulo: string;
  href: string;
  onClose: () => void;
  children: ReactNode;
}) {
  const titleId = useId();
  return (
    <ModalShell open={aberto} onClose={onClose} size="full" variant="center" labelledBy={titleId}>
      <IdDoPainel.Provider value={id}>
        <Embutido.Provider value>
          <div className="flex min-h-[min(88dvh,calc(100dvh-2rem))] max-h-[calc(100dvh-2rem)] flex-col">
            <div className="flex shrink-0 items-center justify-between gap-3 border-b border-surface-border px-8 py-5">
              <h2 id={titleId} className="truncate text-xl font-semibold">
                {titulo}
              </h2>
              <div className="flex shrink-0 items-center gap-3">
                <Link href={href} className="text-sm font-medium text-primary hover:underline">
                  Abrir página
                </Link>
                <button
                  type="button"
                  onClick={onClose}
                  aria-label="Fechar painel"
                  className="inline-flex h-10 w-10 items-center justify-center rounded-md text-muted hover:bg-black/5 hover:text-foreground dark:hover:bg-white/10"
                >
                  <X aria-hidden className="h-5 w-5" />
                </button>
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto px-8 py-6">{children}</div>
          </div>
        </Embutido.Provider>
      </IdDoPainel.Provider>
    </ModalShell>
  );
}
