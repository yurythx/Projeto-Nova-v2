import "@testing-library/jest-dom/vitest";

// jsdom não implementa window.matchMedia — precisa de um stub mínimo para
// qualquer componente que leia prefers-color-scheme (ver
// lib/theme/usePrefersDark.ts, usado por ThemeToggle.tsx). Sempre reporta
// "não corresponde" (matches: false); testes que precisam simular um SO
// em modo escuro substituem isto pontualmente com vi.stubGlobal.
if (typeof window !== "undefined" && !window.matchMedia) {
  window.matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  });
}

// jsdom não implementa HTMLDialogElement.showModal()/close() (nem
// sequer como no-op — a propriedade não existe, o que quebra qualquer
// teste que abra components/ui/Dialog.tsx, construído sobre o <dialog>
// nativo). Stub mínimo que reflete a semântica real via o atributo
// boolean "open" (que jsdom já reflete normalmente em `.open`, como todo
// atributo IDL padrão) — não um mock vazio, pra Dialog.tsx continuar
// exercitando sua própria lógica real (`if (open && !el.open)
// el.showModal()`) nos testes, não um caminho substituto.
if (typeof HTMLDialogElement !== "undefined") {
  if (!HTMLDialogElement.prototype.showModal) {
    HTMLDialogElement.prototype.showModal = function (this: HTMLDialogElement) {
      this.setAttribute("open", "");
    };
  }
  if (!HTMLDialogElement.prototype.close) {
    HTMLDialogElement.prototype.close = function (this: HTMLDialogElement) {
      this.removeAttribute("open");
      this.dispatchEvent(new Event("close"));
    };
  }
}

// jsdom não implementa Element.scrollIntoView (usado para rolar o chat do
// Mercúrio até a última mensagem).
if (typeof Element !== "undefined" && !Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = function () {};
}

// jsdom não implementa Range.prototype.getClientRects/getBoundingClientRect
// necessários para ProseMirror e TipTap funcionarem em ambiente de teste.
if (typeof Range !== "undefined") {
  if (!Range.prototype.getClientRects) {
    Range.prototype.getClientRects = () =>
      [{
        bottom: 0,
        height: 0,
        left: 0,
        right: 0,
        top: 0,
        width: 0,
        x: 0,
        y: 0,
        toJSON: () => {},
      }] as unknown as DOMRectList;
  }
  if (!Range.prototype.getBoundingClientRect) {
    Range.prototype.getBoundingClientRect = () =>
      ({
        bottom: 0,
        height: 0,
        left: 0,
        right: 0,
        top: 0,
        width: 0,
        x: 0,
        y: 0,
        toJSON: () => {},
      }) as DOMRect;
  }
}

// jsdom não implementa document.elementFromPoint
if (typeof document !== "undefined" && !document.elementFromPoint) {
  document.elementFromPoint = () => null;
}
