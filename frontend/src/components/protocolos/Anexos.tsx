"use client";

import { useState } from "react";

import { Paperclip, X } from "lucide-react";

import { ApiError, apiClient } from "@/lib/api/client";
import type { UploadTicket } from "@/lib/plataforma/types";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";

import { CAMPO, ROTULO } from "./ui";

export interface Anexo {
  nome: string;
  chave: string;
}

export const MAXIMO_ANEXOS = 5;

/** Anexos que a mesa junta ao pedido: sobem direto ao armazenamento e voltam como chave. */
export function CampoAnexos({
  pedidoId,
  valor,
  onChange,
  onEnviando,
}: {
  pedidoId: string;
  valor: Anexo[];
  onChange: (anexos: Anexo[]) => void;
  onEnviando?: (enviando: boolean) => void;
}) {
  const [subindo, setSubindo] = useState(false);
  const [erro, setErro] = useState("");

  async function enviar(arquivos: FileList | null) {
    if (!arquivos || arquivos.length === 0) return;
    const lista = Array.from(arquivos).slice(0, MAXIMO_ANEXOS - valor.length);
    setErro("");
    setSubindo(true);
    onEnviando?.(true);
    const novos: Anexo[] = [];
    try {
      for (const arquivo of lista) {
        const { data: ticket } = await apiClient.post<UploadTicket>(
          `v1/protocolos/pedidos/${pedidoId}/anexo`,
          { nome: arquivo.name, content_type: mimeOf(arquivo) },
        );
        await putToTicket(ticket, arquivo);
        novos.push({ nome: arquivo.name, chave: ticket.object_key });
      }
    } catch (err) {
      setErro(
        err instanceof ApiError || err instanceof Error
          ? err.message
          : "Não foi possível enviar o arquivo.",
      );
    } finally {
      onChange([...valor, ...novos]);
      setSubindo(false);
      onEnviando?.(false);
    }
  }

  return (
    <div>
      <label htmlFor={`anexos-${pedidoId}`} className={ROTULO}>
        Anexos (até {MAXIMO_ANEXOS})
      </label>
      <input
        id={`anexos-${pedidoId}`}
        type="file"
        multiple
        disabled={subindo || valor.length >= MAXIMO_ANEXOS}
        onChange={(e) => {
          void enviar(e.target.files);
          e.target.value = "";
        }}
        className={`${CAMPO} file:mr-3 file:rounded-md file:border-0 file:bg-blue-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-blue-700 dark:file:bg-sky-400/15 dark:file:text-sky-300`}
      />
      {subindo && (
        <p className="mt-1 text-xs text-gray-500 dark:text-slate-400">Enviando arquivo…</p>
      )}
      {erro && (
        <p role="alert" className="mt-1 text-xs text-red-600 dark:text-red-400">
          {erro}
        </p>
      )}
      {valor.length > 0 && (
        <ul className="mt-2 flex flex-col gap-1">
          {valor.map((anexo) => (
            <li
              key={anexo.chave}
              className="flex items-center justify-between gap-2 rounded-md bg-slate-50 px-3 py-1.5 text-xs text-gray-700 dark:bg-slate-900/50 dark:text-slate-200"
            >
              <span className="flex min-w-0 items-center gap-1.5">
                <Paperclip className="h-3.5 w-3.5 shrink-0" aria-hidden />
                <span className="truncate">{anexo.nome}</span>
              </span>
              <button
                type="button"
                aria-label={`Tirar ${anexo.nome}`}
                onClick={() => onChange(valor.filter((a) => a.chave !== anexo.chave))}
                className="rounded p-0.5 text-gray-500 hover:bg-gray-200 dark:text-slate-400 dark:hover:bg-slate-700"
              >
                <X className="h-3.5 w-3.5" aria-hidden />
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
