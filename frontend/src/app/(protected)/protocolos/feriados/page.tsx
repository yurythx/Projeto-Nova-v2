"use client";

import Link from "next/link";
import { type FormEvent } from "react";

import { ArrowLeft, CalendarOff, Plus, Trash2 } from "lucide-react";

import { DataState } from "@/components/plataforma/DataState";
import { useAction } from "@/components/plataforma/useAction";
import {
  BOTAO_PRIMARIO,
  BOTAO_SECUNDARIO,
  CabecalhoDaTela,
  CAMPO,
  CARTAO,
  Rotulado,
  TEXTO_FORTE,
  TEXTO_FRACO,
  Vazio,
} from "@/components/protocolos/ui";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Feriado {
  data: string;
  nome: string;
}

const DIA_DA_SEMANA = ["domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"];

function legivel(data: string): string {
  const [ano, mes, dia] = data.split("-");
  const semana = new Date(Number(ano), Number(mes) - 1, Number(dia)).getDay();
  return `${dia}/${mes}/${ano} · ${DIA_DA_SEMANA[semana] ?? ""}`;
}

export default function FeriadosPage() {
  const { can } = usePlataforma();
  const gere = can("protocolos:gerir");
  const lista = useApiQuery<Feriado[]>("v1/protocolos/feriados");
  const { run, pending } = useAction();
  const feriados = [...(lista.data ?? [])].sort((a, b) => a.data.localeCompare(b.data));

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const dados = new FormData(form);
    const ok = await run(
      () =>
        apiClient.post("v1/protocolos/feriados", {
          data: String(dados.get("data") ?? ""),
          nome: String(dados.get("nome") ?? "").trim(),
        }),
      "Feriado salvo",
    );
    if (ok) {
      form.reset();
      void lista.mutate();
    }
  }

  async function remover(f: Feriado) {
    const ok = await run(
      () => apiClient.delete(`v1/protocolos/feriados/${f.data}`),
      "Feriado removido",
    );
    if (ok) void lista.mutate();
  }

  return (
    <div className="mx-auto flex w-full max-w-[1000px] flex-col">
      <CabecalhoDaTela
        icone={CalendarOff}
        titulo="Feriados Municipais"
        descricao="O feriado não conta como dia útil nos prazos do protocolo e não abre horário de atendimento."
      >
        <Link href="/protocolos" className={BOTAO_SECUNDARIO}>
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Painel
        </Link>
      </CabecalhoDaTela>

      {gere && (
        <form
          onSubmit={(e) => void salvar(e)}
          className={`${CARTAO} mb-6 grid items-end gap-4 p-5 sm:grid-cols-[200px_1fr_auto]`}
        >
          <Rotulado id="feriado-data" rotulo="Data *">
            <input id="feriado-data" name="data" type="date" required className={CAMPO} />
          </Rotulado>
          <Rotulado id="feriado-nome" rotulo="Nome *">
            <input
              id="feriado-nome"
              name="nome"
              required
              minLength={2}
              maxLength={120}
              placeholder="Aniversário da cidade"
              className={CAMPO}
            />
          </Rotulado>
          <button type="submit" disabled={pending} className={BOTAO_PRIMARIO}>
            <Plus className="h-4 w-4" aria-hidden />
            Adicionar
          </button>
        </form>
      )}

      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={false}
      >
        {feriados.length === 0 ? (
          <Vazio
            titulo="Nenhum feriado cadastrado"
            texto="Sem feriados, só sábado e domingo ficam fora da conta dos prazos."
          />
        ) : (
          <ul className={`${CARTAO} divide-y divide-gray-100 dark:divide-slate-700`}>
            {feriados.map((f) => (
              <li key={f.data} className="flex items-center justify-between gap-4 px-6 py-4">
                <div>
                  <p className={`font-semibold ${TEXTO_FORTE}`}>{f.nome}</p>
                  <p className={`text-sm ${TEXTO_FRACO}`}>{legivel(f.data)}</p>
                </div>
                {gere && (
                  <button
                    type="button"
                    aria-label={`Remover ${f.nome}`}
                    disabled={pending}
                    onClick={() => void remover(f)}
                    className="flex h-9 w-9 items-center justify-center rounded-lg text-red-600 transition-colors hover:bg-red-50 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-500/10"
                  >
                    <Trash2 className="h-4 w-4" aria-hidden />
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </DataState>
    </div>
  );
}
