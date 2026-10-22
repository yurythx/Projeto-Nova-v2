"use client";

import { useCallback, useEffect } from "react";
import useSWR from "swr";

import { useToast } from "@/components/notifications/ToastProvider";
import { filaCampo, type Acao, type Resultado } from "@/lib/atendimento/campo";

const CHAVE = "atendimento:fila-campo";

/** Liga a fila de campo à tela: envia ao abrir, quando a conexão volta e
 * a cada ação nova. `pendentes` são as ações guardadas no aparelho;
 * `aoEnviar` roda quando algo subiu (para recarregar a tela). */
export function useFilaCampo(aoEnviar: () => void) {
  const { showToast } = useToast();
  const fila = useSWR<Acao[]>(CHAVE, () => filaCampo().pendentes());
  const { mutate } = fila;

  const processar = useCallback(async (): Promise<Resultado> => {
    const r = await filaCampo().processar();
    await mutate();
    if (r.enviadas > 0) aoEnviar();
    for (const { motivo } of r.recusadas) {
      showToast({ title: "Registro de campo recusado", description: motivo, tone: "danger" });
    }
    return r;
  }, [aoEnviar, mutate, showToast]);

  useEffect(() => {
    void processar();
    const online = () => void processar();
    window.addEventListener("online", online);
    return () => window.removeEventListener("online", online);
  }, [processar]);

  /** Guarda a ação no aparelho e tenta enviar já. */
  const executar = useCallback(
    async (acao: Acao): Promise<Resultado> => {
      await filaCampo().enfileirar(acao);
      const r = await processar();
      if (r.pendentes > 0) {
        showToast({
          title: "Sem conexão",
          description: "O registro ficou guardado no aparelho e será enviado quando a rede voltar.",
          tone: "info",
        });
      }
      return r;
    },
    [processar, showToast],
  );

  return { pendentes: fila.data ?? [], executar, atualizar: mutate };
}
