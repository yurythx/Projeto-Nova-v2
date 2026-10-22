"use client";

import { type PointerEvent, useRef, useState } from "react";

import { Button } from "@/components/ui/Button";

/** Área para quem recebeu o atendimento assinar com o dedo ou o mouse.
 * A assinatura é imagem; quem não puder assinar fica registrado pelo nome
 * no encerramento da visita. */
export function AssinaturaPad({ onPronta }: { onPronta: (png: Blob | null) => void }) {
  const canvas = useRef<HTMLCanvasElement>(null);
  const desenhando = useRef(false);
  const [vazia, setVazia] = useState(true);

  function ponto(e: PointerEvent<HTMLCanvasElement>) {
    const c = canvas.current!;
    const r = c.getBoundingClientRect();
    return {
      x: ((e.clientX - r.left) * c.width) / (r.width || 1),
      y: ((e.clientY - r.top) * c.height) / (r.height || 1),
    };
  }

  function comecar(e: PointerEvent<HTMLCanvasElement>) {
    const ctx = canvas.current?.getContext("2d");
    if (!ctx) return;
    desenhando.current = true;
    const { x, y } = ponto(e);
    ctx.lineWidth = 2.5;
    ctx.lineCap = "round";
    ctx.strokeStyle = "#111827";
    ctx.beginPath();
    ctx.moveTo(x, y);
  }

  function mover(e: PointerEvent<HTMLCanvasElement>) {
    const ctx = canvas.current?.getContext("2d");
    if (!desenhando.current || !ctx) return;
    const { x, y } = ponto(e);
    ctx.lineTo(x, y);
    ctx.stroke();
    if (vazia) setVazia(false);
  }

  function terminar() {
    if (!desenhando.current) return;
    desenhando.current = false;
    canvas.current?.toBlob((png) => onPronta(png), "image/png");
  }

  function limpar() {
    const c = canvas.current!;
    c.getContext("2d")?.clearRect(0, 0, c.width, c.height);
    setVazia(true);
    onPronta(null);
  }

  return (
    <div className="flex flex-col gap-2">
      <canvas
        ref={canvas}
        width={600}
        height={200}
        role="img"
        aria-label={vazia ? "Área de assinatura, vazia" : "Área de assinatura, assinada"}
        onPointerDown={comecar}
        onPointerMove={mover}
        onPointerUp={terminar}
        onPointerLeave={terminar}
        className="h-40 w-full touch-none rounded-lg border border-surface-border bg-white"
      />
      <div>
        <Button type="button" size="sm" variant="secondary" onClick={limpar} disabled={vazia}>
          Limpar assinatura
        </Button>
      </div>
    </div>
  );
}
