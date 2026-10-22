"use client";

import { useMemo, type ReactNode } from "react";
import { generateQRCode } from "@/lib/qrcode/qrcode";

export interface QRCodeSVGProps {
  value: string;
  size?: number;
  className?: string;
  fgColor?: string;
  bgColor?: string;
  title?: string;
}

/**
 * Renderiza um QR Code estritamente em SVG vetorial.
 * Ideal para exibição em telas, totens, impressão de certidões e folhas de rosto.
 */
export function QRCodeSVG({
  value,
  size = 160,
  className = "",
  fgColor = "currentColor",
  bgColor = "transparent",
  title = "QR Code para verificação de autenticidade",
}: QRCodeSVGProps) {
  const qr = useMemo(() => {
    try {
      return generateQRCode(value, "M");
    } catch {
      return generateQRCode(value, "L");
    }
  }, [value]);

  const margin = 2;
  const viewBoxSize = qr.size + margin * 2;

  // Monta os retângulos dos módulos ativos
  const rects = useMemo(() => {
    const list: ReactNode[] = [];
    for (let r = 0; r < qr.size; r++) {
      const row = qr.modules[r];
      if (!row) continue;
      for (let c = 0; c < qr.size; c++) {
        if (row[c]) {
          list.push(
            <rect
              key={`${r}-${c}`}
              x={c + margin}
              y={r + margin}
              width={1}
              height={1}
              fill={fgColor}
            />,
          );
        }
      }
    }
    return list;
  }, [qr, margin, fgColor]);

  return (
    <svg
      role="img"
      aria-label={title}
      viewBox={`0 0 ${viewBoxSize} ${viewBoxSize}`}
      width={size}
      height={size}
      className={`shape-rendering-crispEdges ${className}`}
      style={{ shapeRendering: "crispEdges" }}
    >
      <title>{title}</title>
      {bgColor !== "transparent" && (
        <rect width={viewBoxSize} height={viewBoxSize} fill={bgColor} />
      )}
      {rects}
    </svg>
  );
}
