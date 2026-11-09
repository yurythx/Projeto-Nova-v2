"use client";

import { useEffect, useState, useRef, type CSSProperties, type ReactNode } from "react";
import { createPortal } from "react-dom";

export type ModalSize = "sm" | "md" | "lg" | "xl" | "full";

const larguraDrawer: Record<"media" | "larga" | "ampla", string> = {
  media: "min(100vw, 30rem)",
  larga: "min(100vw, 56rem)",
  ampla: "min(100vw, max(68vw, 42rem))",
};

const sizeClass: Record<ModalSize, string> = {
  sm: "max-w-sm",
  md: "max-w-md",
  lg: "max-w-2xl",
  xl: "max-w-3xl",
  full: "max-w-[96rem] min-h-[min(88dvh,calc(100dvh-2rem))]",
};

/**
 * Casca de modal sobre o <dialog> nativo. O consumidor controla todo o
 * conteúdo interno (cabeçalho / corpo / rodapé); esta casca só entrega o
 * comportamento correto de graça:
 *
 * - fechar com Esc (evento `cancel` do <dialog>);
 * - fechar clicando no backdrop (o clique no ::backdrop chega como clique
 *   no próprio <dialog>);
 * - captura de foco e fundo `inert` (padrão do <dialog>.showModal());
 * - render no top layer, acima de qualquer z-index;
 * - trava o scroll do body enquanto aberto.
 *
 * Largura = min(100vw-2rem, size) e altura máxima = 100dvh-2rem, com o
 * conteúdo esperado em coluna flex (cabeçalho/rodapé shrink-0, corpo com
 * overflow-y-auto).
 */
export function ModalShell({
  open,
  onClose,
  size = "md",
  labelledBy,
  describedBy,
  className = "",
  variant = "center",
  amplitude = "media",
  children,
}: {
  open: boolean;
  onClose: () => void;
  size?: ModalSize;
  labelledBy?: string;
  describedBy?: string;
  className?: string;
  /** "drawer": painel encostado à direita, na altura toda. */
  variant?: "center" | "drawer";
  /** No drawer: `larga` cabe um card; `ampla` ocupa mais da metade da tela. */
  amplitude?: "media" | "larga" | "ampla";
  children: ReactNode;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const [corpo, setCorpo] = useState<HTMLElement | null>(null);

  useEffect(() => {
    setCorpo(document.body);
  }, []);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  }, [open, corpo]);

  useEffect(() => {
    if (!open) return;
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previous;
    };
  }, [open]);

  const drawer = variant === "drawer";
  const geometria: CSSProperties | undefined = drawer
    ? {
        position: "fixed",
        top: 0,
        right: 0,
        bottom: 0,
        left: "auto",
        margin: 0,
        height: "100dvh",
        maxHeight: "none",
        width: larguraDrawer[amplitude],
        maxWidth: larguraDrawer[amplitude],
      }
    : undefined;

  const dialog = (
    <dialog
      ref={ref}
      onClose={(event) => {
        event.stopPropagation();
        onClose();
      }}
      onCancel={(event) => {
        event.stopPropagation();
        onClose();
      }}
      onClick={(e) => {
        e.stopPropagation();
        if (e.target === ref.current) onClose();
      }}
      aria-labelledby={labelledBy}
      aria-describedby={describedBy}
      style={geometria}
      className={`${
        drawer
          ? "rounded-l-xl max-md:rounded-none"
          : `m-auto w-[calc(100vw-2rem)] ${sizeClass[size]} max-h-[calc(100dvh-2rem)] rounded-xl`
      } overflow-hidden border border-surface-border bg-surface p-0 text-foreground
        shadow-2xl backdrop:bg-black/50 backdrop:backdrop-blur-sm ${className}`}
    >
      <div
        className={`flex flex-col ${drawer ? "h-full" : "max-h-[calc(100dvh-2rem)]"}`}
      >
        {children}
      </div>
    </dialog>
  );

  if (!corpo) return null;
  return createPortal(dialog, corpo);
}
