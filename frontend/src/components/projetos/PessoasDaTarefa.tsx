"use client";

import { useMemo, useState } from "react";

import { Input } from "@/components/ui/Input";
import { useApiQuery } from "@/lib/api/swr";
import type { PessoaProjeto } from "@/lib/projetos/projetos";

/** Pessoas do projeto. A busca adiciona; tirar remove só aquela pessoa. */
export function PessoasDaTarefa({
  projetoId,
  value,
  onChange,
  id = "tarefa-pessoas",
  rotulo = "Pessoas da tarefa",
}: {
  projetoId: string;
  value: PessoaProjeto[];
  onChange: (v: PessoaProjeto[]) => void;
  id?: string;
  rotulo?: string;
}) {
  const [q, setQ] = useState("");
  const lista = useApiQuery<PessoaProjeto[]>(
    projetoId ? `v1/projetos/${projetoId}/participantes` : null,
  );
  const termo = q.trim().toLocaleLowerCase("pt-BR");
  const buscando = termo.length >= 2;
  const visiveis = useMemo(() => {
    if (!buscando) return [];
    const ja = new Set(value.map((pessoa) => pessoa.id));
    return (lista.data ?? [])
      .filter((pessoa) => !ja.has(pessoa.id) && pessoa.nome.toLocaleLowerCase("pt-BR").includes(termo))
      .slice(0, 8);
  }, [buscando, lista.data, termo, value]);

  return (
    <div className="flex flex-col gap-2">
      <Input
        id={id}
        label={rotulo}
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="Digite o nome"
        autoComplete="off"
      />
      <p className="text-xs text-muted">Só quem já participa do projeto. Os nomes aparecem a partir da segunda letra.</p>
      {buscando && (
        <ul role="listbox" aria-label="Pessoas do projeto" className="flex flex-col rounded-lg border border-surface-border">
          {lista.isLoading && <li className="px-3 py-2 text-sm text-muted">Carregando pessoas…</li>}
          {!lista.isLoading && visiveis.length === 0 && (
            <li className="px-3 py-2 text-sm text-muted">
              Nenhuma pessoa do projeto com esse nome. Inclua a pessoa no projeto antes.
            </li>
          )}
          {visiveis.map((pessoa) => (
            <li key={pessoa.id} role="option" aria-selected={false}>
              <button
                type="button"
                className="w-full px-3 py-2 text-left text-sm hover:bg-surface-hover"
                onClick={() => {
                  onChange([...value, { id: pessoa.id, nome: pessoa.nome }]);
                  setQ("");
                }}
              >
                {pessoa.nome}
                {pessoa.papel === "gerente" ? " · gerente" : ""}
              </button>
            </li>
          ))}
        </ul>
      )}
      {value.length > 0 && (
        <ul className="flex flex-col gap-1">
          {value.map((pessoa) => (
            <li key={pessoa.id || pessoa.nome} className="flex items-center justify-between rounded-md bg-surface-hover px-3 py-1.5 text-sm">
              <span>{pessoa.nome}</span>
              <button
                type="button"
                className="text-xs text-muted hover:text-danger"
                onClick={() => onChange(value.filter((item) => item.id !== pessoa.id))}
              >
                Tirar
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
