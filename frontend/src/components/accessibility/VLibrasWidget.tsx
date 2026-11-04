"use client";

import { useEffect, useState, type HTMLAttributes } from "react";
import Script from "next/script";

declare global {
  interface Window {
    VLibras?: {
      Widget: new (url: string) => void;
    };
    VLibrasWidget?: {
      open?: () => void;
    };
  }
}

// Estende os atributos HTML para aceitar os data-attributes do VLibras.
interface VLibrasDivProps extends HTMLAttributes<HTMLDivElement> {
  vw?: string;
  "vw-access-button"?: string;
  "vw-plugin-wrapper"?: string;
}

function initVLibras() {
  if (typeof window !== "undefined" && window.VLibras?.Widget) {
    try {
      new window.VLibras.Widget("https://vlibras.gov.br/app");
    } catch {
      // ignora se já montado ou inicializado
    }
  }
}

// Widget oficial VLibras (Governo Federal — https://vlibras.gov.br): tradução
// para a Língua Brasileira de Sinais. O botão fica em toda página. O player
// (script e WebAssembly) só baixa quando a pessoa pede Libras — ver as
// liberações de CSP em src/proxy.ts.
export function VLibrasWidget() {
  const [pedido, setPedido] = useState(false);

  useEffect(() => {
    if (pedido) initVLibras();
  }, [pedido]);

  if (!pedido) {
    return (
      <button
        type="button"
        onClick={() => setPedido(true)}
        className="fixed bottom-4 right-4 z-40 inline-flex h-11 items-center rounded-full bg-primary px-4 text-sm font-semibold text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary"
      >
        Traduzir para Libras
      </button>
    );
  }

  return (
    <>
      <div {...({ vw: "true" } as VLibrasDivProps)} className="enabled">
        <div {...({ "vw-access-button": "true" } as VLibrasDivProps)} className="active" />
        <div {...({ "vw-plugin-wrapper": "true" } as VLibrasDivProps)}>
          <div className="vw-plugin-top-wrapper" />
        </div>
      </div>
      <Script
        src="https://vlibras.gov.br/app/vlibras-plugin.js"
        strategy="afterInteractive"
        onLoad={initVLibras}
      />
    </>
  );
}
