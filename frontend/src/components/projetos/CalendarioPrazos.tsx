"use client";

import Link from "next/link";
import { useMemo, useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PessoaComAvatar } from "@/components/plataforma/AvatarPessoa";
import { Button } from "@/components/ui/Button";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import type { PrazoAgenda } from "@/lib/projetos/projetos";

import { diaBrasil, diaISO, deslocarMes, faixaDaSemana, gradeDoMes, mesISO, nomeDoMes } from "./datas";

const SEMANA = ["Seg", "Ter", "Qua", "Qui", "Sex", "Sáb", "Dom"];
const VISIVEIS = 3;

function rotulo(item: PrazoAgenda) {
  return item.pai_id ? `Subtarefa · ${item.titulo}` : item.titulo;
}

function ItemPrazo({ item, hoje }: { item: PrazoAgenda; hoje: string }) {
  const atrasada = item.estado !== "concluida" && diaISO(item.data_limite) < hoje;
  return (
    <Link
      href={`/projetos/${item.projeto_id}`}
      title={`#${item.projeto_numero} ${item.projeto_titulo}`}
      className={`block truncate rounded px-1 py-0.5 text-left text-[11px] hover:underline ${
        atrasada ? "bg-danger/10 text-danger" : "bg-primary/10 text-foreground"
      }`}
    >
      {rotulo(item)}
    </Link>
  );
}

/** Mês civil de São Paulo com o prazo de cada tarefa e subtarefa. */
export function CalendarioPrazos({ usuarioId, agora }: { usuarioId?: string; agora?: Date }) {
  const relogio = agora ?? new Date();
  const hoje = faixaDaSemana(relogio).hoje;
  const [mes, setMes] = useState(() => mesISO(relogio));
  const [minhas, setMinhas] = useState(false);
  const [abertos, setAbertos] = useState<Record<string, boolean>>({});
  const prazos = useApiQuery<PrazoAgenda[]>(withQuery("v1/projetos/prazos", { mes }));
  const lista = useMemo(() => {
    const itens = prazos.data ?? [];
    if (!minhas) return itens;
    return itens.filter((item) => item.responsavel_id === usuarioId);
  }, [minhas, prazos.data, usuarioId]);
  const grade = useMemo(() => gradeDoMes(mes), [mes]);
  const porDia = useMemo(() => {
    const mapa = new Map<string, PrazoAgenda[]>();
    for (const item of lista) {
      const dia = diaISO(item.data_limite);
      if (dia.slice(0, 7) !== mes) continue;
      mapa.set(dia, [...(mapa.get(dia) ?? []), item]);
    }
    return mapa;
  }, [lista, mes]);
  const atrasadas = lista.filter((item) => diaISO(item.data_limite).slice(0, 7) !== mes);

  return (
    <section aria-label={`Prazos de ${nomeDoMes(mes)}`} className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-medium capitalize text-foreground">{nomeDoMes(mes)}</h2>
        <div className="flex flex-wrap gap-2">
          <Button type="button" size="sm" variant="secondary" onClick={() => setMes((atual) => deslocarMes(atual, -1))}>
            Mês anterior
          </Button>
          <Button type="button" size="sm" variant="secondary" onClick={() => setMes(mesISO(relogio))}>
            Mês atual
          </Button>
          <Button type="button" size="sm" variant="secondary" onClick={() => setMes((atual) => deslocarMes(atual, 1))}>
            Próximo mês
          </Button>
          <button
            type="button"
            aria-pressed={minhas}
            onClick={() => setMinhas((v) => !v)}
            className={`rounded-full border px-3 py-1 text-xs ${
              minhas ? "border-primary bg-primary/10 text-primary" : "border-surface-border text-muted"
            }`}
          >
            Só as minhas
          </button>
        </div>
      </div>
      <DataState
        loading={prazos.isLoading}
        error={prazos.error}
        empty={false}
        onRetry={() => void prazos.mutate()}
        rows={4}
      >
        {atrasadas.length > 0 && (
          <div className="mb-3 flex flex-col gap-2">
            <h3 className="text-sm font-medium text-foreground">Atrasadas de outros meses</h3>
            <ul className="flex flex-col gap-1">
              {atrasadas.map((item) => (
                <li key={item.id} className="flex flex-wrap items-center gap-2 text-sm">
                  <ItemPrazo item={item} hoje={hoje} />
                  <span className="text-xs text-muted">{diaBrasil(item.data_limite)}</span>
                  <PessoaComAvatar id={item.responsavel_id} nome={item.responsavel_nome} className="text-xs" />
                </li>
              ))}
            </ul>
          </div>
        )}
        <div className="grid grid-cols-7 gap-px overflow-hidden rounded-lg border border-surface-border bg-surface-border">
          {SEMANA.map((dia) => (
            <div key={dia} className="bg-surface px-2 py-1 text-center text-[11px] font-medium text-muted">
              {dia}
            </div>
          ))}
          {grade.map((celula) => {
            const itens = porDia.get(celula.dia) ?? [];
            const aberto = Boolean(abertos[celula.dia]);
            const visiveis = aberto ? itens : itens.slice(0, VISIVEIS);
            const resto = itens.length - visiveis.length;
            return (
              <div
                key={celula.dia}
                className={`min-h-24 bg-surface p-1 ${celula.nesteMes ? "" : "opacity-45"}`}
              >
                <span
                  className={`inline-flex h-6 w-6 items-center justify-center rounded-full text-xs ${
                    celula.dia === hoje ? "bg-primary text-primary-foreground" : "text-muted"
                  }`}
                >
                  {Number(celula.dia.slice(8))}
                </span>
                <div className="mt-1 flex flex-col gap-0.5">
                  {visiveis.map((item) => (
                    <ItemPrazo key={item.id} item={item} hoje={hoje} />
                  ))}
                  {resto > 0 && (
                    <button
                      type="button"
                      className="px-1 text-left text-[11px] text-primary hover:underline"
                      onClick={() => setAbertos((atual) => ({ ...atual, [celula.dia]: true }))}
                    >
                      Mais {resto}
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
        {lista.filter((item) => diaISO(item.data_limite).slice(0, 7) === mes).length === 0 && (
          <p className="mt-3 text-sm text-muted">Nenhum prazo neste mês.</p>
        )}
      </DataState>
    </section>
  );
}
