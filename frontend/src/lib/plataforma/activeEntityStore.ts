"use client";

import { useSyncExternalStore } from "react";

// Armazena a Entidade/Secretaria Ativa selecionada pelo usuário no cabeçalho
// (conceito de Active Entity do GLPI). Permite que administradores e
// técnicos com múltiplos escopos comutem o contexto globalmente.

const STORAGE_KEY = "projeto_nova_active_entity";

let currentEntityId: string =
  typeof window !== "undefined" ? window.localStorage.getItem(STORAGE_KEY) || "" : "";

const listeners = new Set<() => void>();

export function getActiveEntitySnapshot(): string {
  return currentEntityId;
}

export function setActiveEntity(entityId: string): void {
  currentEntityId = entityId;
  if (typeof window !== "undefined") {
    if (entityId) {
      window.localStorage.setItem(STORAGE_KEY, entityId);
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  }
  listeners.forEach((listener) => listener());
}

export function subscribeActiveEntity(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useActiveEntity(): string {
  return useSyncExternalStore(subscribeActiveEntity, getActiveEntitySnapshot, () => "");
}
