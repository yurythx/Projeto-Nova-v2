"use client";

import { porcentagemDoProjeto } from "@/lib/projetos/projetos";

/** Barra da lista: tarefas concluídas sobre o total do projeto. */
export function ProgressoProjeto({
  titulo,
  total,
  concluidas,
  unidade = "tarefas",
}: {
  titulo: string;
  total?: number;
  concluidas?: number;
  unidade?: string;
}) {
  const pct = porcentagemDoProjeto(total, concluidas);
  const todas = total ?? 0;
  const feitas = concluidas ?? 0;
  const conta = todas === 0 ? `Sem ${unidade}` : `${feitas} de ${todas} ${unidade} concluídas`;
  return (
    <div className="flex min-w-[6.5rem] items-center gap-2" title={conta}>
      <div
        role="meter"
        aria-label={`Progresso de ${titulo}`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct}
        aria-valuetext={`${pct}%. ${conta}`}
        className="h-2 min-w-16 flex-1 overflow-hidden rounded-full bg-black/10 dark:bg-white/10"
      >
        <div className="h-full rounded-full bg-primary" style={{ width: `${pct}%` }} />
      </div>
      <span className="w-9 shrink-0 text-right text-xs tabular-nums text-muted">{pct}%</span>
    </div>
  );
}
