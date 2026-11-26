"use client";

import { useState, type FormEvent } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Dialog } from "@/components/ui/Dialog";
import { apiClient } from "@/lib/api/client";

export type TipoAcao = "complemento" | "resposta" | "nota" | "concluir" | "cancelar";

export interface AcaoAberta {
  tipo: TipoAcao;
  pedidoId: string;
  numero: string;
}

const TITULO: Record<TipoAcao, string> = {
  complemento: "Solicitar Complementação",
  resposta: "Registrar Resposta da Complementação",
  nota: "Adicionar Nota Interna",
  concluir: "Concluir Protocolo",
  cancelar: "Cancelar Protocolo",
};

const CAMPO =
  "w-full rounded-lg border border-gray-300 bg-surface px-4 py-3 text-sm text-foreground transition-colors focus:border-blue-500 focus:outline-none focus:ring-[3px] focus:ring-blue-500/10";
const ROTULO = "mb-2 block text-sm font-medium text-gray-700 dark:text-muted";
export const BOTAO_PRIMARIO =
  "inline-flex items-center justify-center gap-2 rounded-lg bg-blue-500 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-50";
export const BOTAO_SECUNDARIO =
  "inline-flex items-center justify-center gap-2 rounded-lg border border-gray-300 bg-surface px-4 py-2.5 text-sm font-medium text-gray-700 transition-colors hover:border-gray-400 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-foreground dark:hover:bg-white/5";
const BOTAO_PERIGO =
  "inline-flex items-center justify-center gap-2 rounded-lg bg-red-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50";

/** Modal da ação escolhida no menu do pedido. Chama onFeito depois de gravar. */
export function ModalDeAcao({
  acao,
  onFechar,
  onFeito,
}: {
  acao: AcaoAberta | null;
  onFechar: () => void;
  onFeito: () => void;
}) {
  const { run, pending } = useAction();
  const [texto, setTexto] = useState("");
  const [decisao, setDecisao] = useState("");

  function fechar() {
    setTexto("");
    setDecisao("");
    onFechar();
  }

  async function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!acao) return;
    const base = `v1/protocolos/pedidos/${acao.pedidoId}`;
    const pedidos: Record<TipoAcao, [string, unknown, string]> = {
      complemento: [`${base}/complemento`, { mensagem: texto }, "Complementação solicitada"],
      resposta: [`${base}/resposta`, { mensagem: texto }, "Resposta registrada"],
      nota: [`${base}/nota`, { mensagem: texto }, "Nota adicionada"],
      concluir: [`${base}/concluir`, { decisao, justificativa: texto }, "Protocolo concluído"],
      cancelar: [`${base}/cancelar`, {}, "Protocolo cancelado"],
    };
    const [caminho, corpo, aviso] = pedidos[acao.tipo];
    const ok = await run(() => apiClient.post(caminho, corpo), aviso);
    if (ok) {
      fechar();
      onFeito();
    }
  }

  const tipo = acao?.tipo;
  return (
    <Dialog
      open={acao !== null}
      onClose={fechar}
      title={tipo ? TITULO[tipo] : ""}
      description={acao ? `Protocolo #${acao.numero}` : undefined}
    >
      <form onSubmit={(e) => void enviar(e)} className="mt-4 flex flex-col gap-4">
        {tipo === "cancelar" && (
          <p className="text-sm text-gray-600 dark:text-muted">
            O protocolo é encerrado como cancelado e o cidadão vê a mudança na consulta. Esta ação
            não pode ser desfeita.
          </p>
        )}
        {tipo === "concluir" && (
          <div>
            <label htmlFor="acao-decisao" className={ROTULO}>
              Decisão *
            </label>
            <select
              id="acao-decisao"
              required
              value={decisao}
              onChange={(e) => setDecisao(e.target.value)}
              className={CAMPO}
            >
              <option value="">Selecione…</option>
              <option value="favoravel">Favorável (deferido)</option>
              <option value="desfavoravel">Desfavorável (indeferido)</option>
            </select>
          </div>
        )}
        {tipo && tipo !== "cancelar" && (
          <div>
            <label htmlFor="acao-texto" className={ROTULO}>
              {tipo === "concluir"
                ? "Justificativa *"
                : tipo === "nota"
                  ? "Nota (não aparece para o cidadão) *"
                  : tipo === "resposta"
                    ? "Resposta recebida *"
                    : "O que o cidadão precisa enviar *"}
            </label>
            <textarea
              id="acao-texto"
              required
              rows={4}
              maxLength={tipo === "concluir" ? 4000 : 2000}
              value={texto}
              onChange={(e) => setTexto(e.target.value)}
              className={`${CAMPO} min-h-[100px] resize-y`}
            />
          </div>
        )}
        <div className="flex justify-end gap-3">
          <button type="button" onClick={fechar} className={BOTAO_SECUNDARIO}>
            Cancelar
          </button>
          <button
            type="submit"
            disabled={pending}
            className={tipo === "cancelar" ? BOTAO_PERIGO : BOTAO_PRIMARIO}
          >
            {tipo === "cancelar" ? "Cancelar protocolo" : "Confirmar"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
