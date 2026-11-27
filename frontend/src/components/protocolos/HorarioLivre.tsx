"use client";

import { useEffect, useState } from "react";

import { ApiError } from "@/lib/api/client";
import { publicFetch } from "@/lib/api/publicClient";

import { CAMPO, Rotulado } from "./ui";

/** Hoje (AAAA-MM-DD) em Cuiabá, o mínimo do calendário. */
export function hojeEmCuiaba(agora: Date = new Date()): string {
  return agora.toLocaleDateString("en-CA", { timeZone: "America/Cuiaba" });
}

/**
 * Data e hora do atendimento. Escolhida a data, só aparecem os horários
 * ainda livres do serviço (sem fim de semana fora da grade nem feriado).
 * Os campos se chamam "data" e "hora" no formulário.
 */
export function HorarioLivre({
  servicoId,
  obrigatorio,
}: {
  servicoId: string;
  obrigatorio: boolean;
}) {
  const [data, setData] = useState("");
  const [horas, setHoras] = useState<string[] | null>(null);
  const [erro, setErro] = useState("");

  useEffect(() => {
    if (!data) return;
    let vivo = true;
    publicFetch<string[]>(
      `v1/protocolos/carta/${servicoId}/horarios?data=${encodeURIComponent(data)}`,
    )
      .then((lista) => {
        if (vivo) setHoras(lista ?? []);
      })
      .catch((e: unknown) => {
        if (vivo) {
          setHoras([]);
          setErro(e instanceof ApiError ? e.message : "Não foi possível ler os horários.");
        }
      });
    return () => {
      vivo = false;
    };
  }, [servicoId, data]);

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <Rotulado id="data" rotulo={obrigatorio ? "Data do atendimento *" : "Data do atendimento"}>
        <input
          id="data"
          name="data"
          type="date"
          required={obrigatorio}
          min={hojeEmCuiaba()}
          value={data}
          onChange={(e) => {
            setErro("");
            setHoras(null);
            setData(e.target.value);
          }}
          className={CAMPO}
        />
      </Rotulado>
      <Rotulado id="hora" rotulo={obrigatorio ? "Horário *" : "Horário"}>
        <select
          id="hora"
          name="hora"
          required={obrigatorio || Boolean(data)}
          disabled={!data || horas === null || horas.length === 0}
          defaultValue=""
          key={`${data}-${(horas ?? []).join(",")}`}
          className={CAMPO}
        >
          <option value="">
            {!data
              ? "Escolha a data primeiro"
              : horas === null
                ? "Carregando…"
                : horas.length === 0
                  ? "Sem horário livre neste dia"
                  : "Selecione…"}
          </option>
          {(horas ?? []).map((hora) => (
            <option key={hora} value={hora}>
              {hora}
            </option>
          ))}
        </select>
      </Rotulado>
      {erro && (
        <p role="alert" className="text-xs text-red-600 sm:col-span-2 dark:text-red-400">
          {erro}
        </p>
      )}
      {data && horas !== null && horas.length === 0 && !erro && (
        <p className="text-xs text-gray-500 sm:col-span-2 dark:text-slate-400">
          Não há horário livre nessa data. Tente outro dia útil.
        </p>
      )}
    </div>
  );
}
