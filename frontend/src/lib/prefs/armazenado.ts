"use client";

import { useCallback, useSyncExternalStore } from "react";

// Preferências deste navegador em localStorage, como "external store"
// (useSyncExternalStore). No servidor e na hidratação o valor é null, e o
// gravado aparece logo depois, sem o setState em efeito que
// react-hooks/set-state-in-effect recusa — mesmo motivo de
// lib/layout/sidebarCollapsedStore.ts.
//
// O que se grava fica também na memória da página: em modo privado ou sem
// cota, a escolha vale até recarregar em vez de sumir na hora. Outra aba
// que grava a mesma chave avisa pelo evento "storage".

type Ouvinte = () => void;

const ouvintes = new Map<string, Set<Ouvinte>>();
const emMemoria = new Map<string, string | null>();

function avisar(chave: string) {
  ouvintes.get(chave)?.forEach((ouvinte) => ouvinte());
}

export function lerArmazenado(chave: string): string | null {
  if (emMemoria.has(chave)) return emMemoria.get(chave) ?? null;
  try {
    return localStorage.getItem(chave);
  } catch {
    return null;
  }
}

/** Grava (ou apaga, com null) e redesenha quem usa a mesma chave. */
export function gravarArmazenado(chave: string, valor: string | null): void {
  emMemoria.set(chave, valor);
  try {
    if (valor === null) localStorage.removeItem(chave);
    else localStorage.setItem(chave, valor);
  } catch {
    // Cota ou modo privado: fica só na memória da página.
  }
  avisar(chave);
}

/** Valor cru da chave; null no servidor, na hidratação ou sem nada gravado. */
export function useArmazenado(chave: string): string | null {
  const assinar = useCallback(
    (ouvinte: Ouvinte) => {
      const lista = ouvintes.get(chave) ?? new Set<Ouvinte>();
      lista.add(ouvinte);
      ouvintes.set(chave, lista);
      function outraAba(e: StorageEvent) {
        if (e.key !== chave) return;
        emMemoria.delete(chave);
        ouvinte();
      }
      window.addEventListener("storage", outraAba);
      return () => {
        lista.delete(ouvinte);
        window.removeEventListener("storage", outraAba);
      };
    },
    [chave],
  );
  return useSyncExternalStore(
    assinar,
    () => lerArmazenado(chave),
    () => null,
  );
}
