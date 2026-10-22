"use client";

import { type FormEvent } from "react";

import { Button } from "@/components/ui/Button";
import { Select } from "@/components/ui/Select";
import {
  DIAS_DA_SEMANA,
  NIVEIS,
  PRIORIDADE,
  PRIORIDADES,
  type Celula,
  type Meta,
  type Politica,
  type Prioridade,
  type Turno,
} from "@/lib/atendimento/atendimento";

/** O que a tela envia: a política sem os campos do servidor. */
export type PoliticaCorpo = Pick<Politica, "fuso" | "turnos" | "matriz" | "metas">;

/** Fusos do Brasil (IANA). */
const FUSOS = [
  "America/Sao_Paulo",
  "America/Bahia",
  "America/Fortaleza",
  "America/Recife",
  "America/Belem",
  "America/Araguaina",
  "America/Maceio",
  "America/Cuiaba",
  "America/Campo_Grande",
  "America/Manaus",
  "America/Porto_Velho",
  "America/Boa_Vista",
  "America/Rio_Branco",
  "America/Noronha",
];

const TURNOS_POR_DIA = 2;

/** Ponto de partida quando a unidade ainda não tem política: expediente
 * comercial e a matriz usual. Nada vale até a gestão salvar. */
export function sugestao(): PoliticaCorpo {
  const turnos: Turno[] = [];
  for (let dia = 1; dia <= 5; dia++) {
    turnos.push({ dia, inicio: "08:00", fim: "12:00" }, { dia, inicio: "14:00", fim: "18:00" });
  }
  const matriz: Celula[] = [];
  for (const impacto of [1, 2, 3]) {
    for (const urgencia of [1, 2, 3]) {
      const soma = impacto + urgencia;
      const prioridade: Prioridade =
        soma === 6 ? "critica" : soma >= 5 ? "alta" : soma <= 2 ? "baixa" : "media";
      matriz.push({ impacto, urgencia, prioridade });
    }
  }
  const metas: Meta[] = [
    { prioridade: "critica", resposta_min: 30, solucao_min: 240 },
    { prioridade: "alta", resposta_min: 60, solucao_min: 480 },
    { prioridade: "media", resposta_min: 120, solucao_min: 960 },
    { prioridade: "baixa", resposta_min: 480, solucao_min: 2400 },
  ];
  return { fuso: "America/Sao_Paulo", turnos, matriz, metas };
}

function horas(min?: number): string {
  return min === undefined ? "" : String(min / 60);
}

// Todo campo lido existe no formulário (vazio vem como "").
function texto(fd: FormData, nome: string): string {
  return String(fd.get(nome)).trim();
}

/** Lê o formulário. Turno ou meta em branco não entra; a validação de
 * verdade é do servidor, que explica o que está errado. */
export function lerPolitica(fd: FormData): PoliticaCorpo {
  const turnos: Turno[] = [];
  for (let dia = 0; dia < 7; dia++) {
    for (let n = 0; n < TURNOS_POR_DIA; n++) {
      const inicio = texto(fd, `turno-${dia}-${n}-inicio`);
      const fim = texto(fd, `turno-${dia}-${n}-fim`);
      if (inicio && fim) turnos.push({ dia, inicio, fim });
    }
  }
  const matriz: Celula[] = [];
  for (const impacto of [1, 2, 3]) {
    for (const urgencia of [1, 2, 3]) {
      matriz.push({
        impacto,
        urgencia,
        prioridade: texto(fd, `matriz-${impacto}-${urgencia}`) as Prioridade,
      });
    }
  }
  const metas: Meta[] = [];
  for (const prioridade of PRIORIDADES) {
    const resposta = texto(fd, `meta-${prioridade}-resposta`);
    const solucao = texto(fd, `meta-${prioridade}-solucao`);
    if (resposta && solucao) {
      metas.push({
        prioridade,
        resposta_min: Math.round(Number(resposta) * 60),
        solucao_min: Math.round(Number(solucao) * 60),
      });
    }
  }
  return { fuso: texto(fd, "fuso"), turnos, matriz, metas };
}

const opcoesPrioridade = PRIORIDADES.map((p) => ({ value: p, label: PRIORIDADE[p].label }));

/** Formulário da política de prazo: expediente, matriz e metas. */
export function PoliticaForm({
  inicial,
  pending,
  onSalvar,
}: {
  inicial: PoliticaCorpo;
  pending: boolean;
  onSalvar: (p: PoliticaCorpo) => void;
}) {
  function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    onSalvar(lerPolitica(new FormData(e.currentTarget)));
  }

  const turnoDe = (dia: number, n: number) => inicial.turnos.filter((t) => t.dia === dia)[n];
  const prioridadeDe = (impacto: number, urgencia: number) =>
    inicial.matriz.find((c) => c.impacto === impacto && c.urgencia === urgencia)?.prioridade ??
    "media";
  const metaDe = (p: Prioridade) => inicial.metas.find((m) => m.prioridade === p);

  return (
    <form onSubmit={enviar} className="flex flex-col gap-6">
      <div className="max-w-sm">
        <Select
          id="politica-fuso"
          name="fuso"
          label="Fuso horário"
          defaultValue={inicial.fuso}
          options={(FUSOS.includes(inicial.fuso) ? FUSOS : [inicial.fuso, ...FUSOS]).map((f) => ({
            value: f,
            label: f.replace("America/", "").replace("_", " "),
          }))}
        />
      </div>

      <fieldset className="flex flex-col gap-2">
        <legend className="mb-1 text-sm font-semibold">Expediente</legend>
        <p className="text-xs text-muted">
          Até dois turnos por dia. Dia sem turno não conta como tempo útil.
        </p>
        {DIAS_DA_SEMANA.map((nome, dia) => (
          <div key={dia} className="flex flex-wrap items-center gap-2 text-sm">
            <span className="w-20 font-medium">{nome}</span>
            {Array.from({ length: TURNOS_POR_DIA }, (_, n) => (
              <span key={n} className="flex items-center gap-1">
                <input
                  type="time"
                  name={`turno-${dia}-${n}-inicio`}
                  aria-label={`${nome}, turno ${n + 1}, início`}
                  defaultValue={turnoDe(dia, n)?.inicio}
                  className="h-9 rounded-md border border-surface-border bg-surface px-2"
                />
                <span aria-hidden="true">–</span>
                <input
                  type="time"
                  name={`turno-${dia}-${n}-fim`}
                  aria-label={`${nome}, turno ${n + 1}, fim`}
                  defaultValue={turnoDe(dia, n)?.fim}
                  className="h-9 rounded-md border border-surface-border bg-surface px-2"
                />
              </span>
            ))}
          </div>
        ))}
      </fieldset>

      <fieldset className="flex flex-col gap-2">
        <legend className="mb-1 text-sm font-semibold">Prioridade por impacto e urgência</legend>
        <table className="w-full max-w-xl text-sm">
          <caption className="sr-only">Matriz de prioridade</caption>
          <thead>
            <tr>
              <th scope="col" className="p-1 text-left text-xs text-muted">
                Impacto \ Urgência
              </th>
              {NIVEIS.map((u) => (
                <th key={u.value} scope="col" className="p-1 text-left text-xs text-muted">
                  {u.feminino}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {[...NIVEIS].reverse().map((i) => (
              <tr key={i.value}>
                <th scope="row" className="p-1 text-left text-xs text-muted">
                  {i.label}
                </th>
                {NIVEIS.map((u) => (
                  <td key={u.value} className="p-1">
                    <select
                      name={`matriz-${i.value}-${u.value}`}
                      aria-label={`Impacto ${i.label.toLowerCase()}, urgência ${u.feminino.toLowerCase()}`}
                      defaultValue={prioridadeDe(i.value, u.value)}
                      className="h-9 w-full rounded-md border border-surface-border bg-surface px-2"
                    >
                      {opcoesPrioridade.map((o) => (
                        <option key={o.value} value={o.value}>
                          {o.label}
                        </option>
                      ))}
                    </select>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </fieldset>

      <fieldset className="flex flex-col gap-2">
        <legend className="mb-1 text-sm font-semibold">Metas em horas úteis</legend>
        <table className="w-full max-w-xl text-sm">
          <caption className="sr-only">Metas de resposta e solução por prioridade</caption>
          <thead>
            <tr>
              <th scope="col" className="p-1 text-left text-xs text-muted">
                Prioridade
              </th>
              <th scope="col" className="p-1 text-left text-xs text-muted">
                Resposta
              </th>
              <th scope="col" className="p-1 text-left text-xs text-muted">
                Solução
              </th>
            </tr>
          </thead>
          <tbody>
            {PRIORIDADES.map((p) => (
              <tr key={p}>
                <th scope="row" className="p-1 text-left font-medium">
                  {PRIORIDADE[p].label}
                </th>
                {(["resposta", "solucao"] as const).map((campo) => (
                  <td key={campo} className="p-1">
                    <input
                      type="number"
                      min={0}
                      step={0.25}
                      name={`meta-${p}-${campo}`}
                      aria-label={`${PRIORIDADE[p].label}, ${campo === "resposta" ? "resposta" : "solução"} em horas`}
                      defaultValue={horas(
                        campo === "resposta" ? metaDe(p)?.resposta_min : metaDe(p)?.solucao_min,
                      )}
                      className="h-9 w-28 rounded-md border border-surface-border bg-surface px-2"
                    />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </fieldset>

      <div>
        <Button type="submit" loading={pending}>
          Salvar política
        </Button>
      </div>
    </form>
  );
}
