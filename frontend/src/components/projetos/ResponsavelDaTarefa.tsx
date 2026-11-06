"use client";

import { useMemo, useState } from "react";

import { Input } from "@/components/ui/Input";
import { useApiQuery } from "@/lib/api/swr";

export interface ResponsavelOpcao {
  id: string;
  nome: string;
}

/** Pessoas que podem gerir projetos da entidade deste projeto. */
export function ResponsavelDaTarefa({
  projetoId,
  value,
  onChange,
}: {
  projetoId: string;
  value: ResponsavelOpcao;
  onChange: (v: ResponsavelOpcao) => void;
}) {
  const [q, setQ] = useState("");
  const [aberto, setAberto] = useState(false);
  const lista = useApiQuery<ResponsavelOpcao[]>(
    projetoId ? `v1/projetos/${projetoId}/responsaveis` : null,
  );
  const visiveis = useMemo(() => {
    const termo = q.trim().toLocaleLowerCase("pt-BR");
    return (lista.data ?? [])
      .filter((pessoa) => !termo || pessoa.nome.toLocaleLowerCase("pt-BR").includes(termo))
      .slice(0, 12);
  }, [lista.data, q]);

  return (
    <div className="flex flex-col gap-2">
      <Input
        id="tarefa-responsavel-painel"
        label="Responsável pela tarefa"
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          setAberto(true);
        }}
        onFocus={() => setAberto(true)}
        placeholder="Busque quem pode gerir projetos"
        autoComplete="off"
      />
      {aberto && (
        <ul role="listbox" aria-label="Responsáveis" className="flex flex-col rounded-lg border border-surface-border">
          {lista.isLoading && <li className="px-3 py-2 text-sm text-muted">Carregando pessoas…</li>}
          {!lista.isLoading && visiveis.length === 0 && (
            <li className="px-3 py-2 text-sm text-muted">
              Ninguém com permissão de gerir projetos nesta entidade.
            </li>
          )}
          {visiveis.map((pessoa) => (
            <li key={pessoa.id} role="option" aria-selected={value.id === pessoa.id}>
              <button
                type="button"
                className="w-full px-3 py-2 text-left text-sm hover:bg-surface-hover"
                onClick={() => {
                  onChange(pessoa);
                  setQ("");
                  setAberto(false);
                }}
              >
                {pessoa.nome}
              </button>
            </li>
          ))}
        </ul>
      )}
      {value.nome && (
        <p className="flex items-center justify-between rounded-md bg-surface-hover px-3 py-1.5 text-sm">
          <span>{value.nome}</span>
          <button
            type="button"
            className="text-xs text-muted hover:text-danger"
            onClick={() => onChange({ id: "", nome: "" })}
          >
            Tirar
          </button>
        </p>
      )}
    </div>
  );
}
