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
  id = "tarefa-responsavel-painel",
}: {
  projetoId: string;
  value: ResponsavelOpcao;
  onChange: (v: ResponsavelOpcao) => void;
  id?: string;
}) {
  const [q, setQ] = useState("");
  const lista = useApiQuery<ResponsavelOpcao[]>(
    projetoId ? `v1/projetos/${projetoId}/responsaveis` : null,
  );
  const termo = q.trim().toLocaleLowerCase("pt-BR");
  const buscando = termo.length >= 2;
  const visiveis = useMemo(() => {
    if (!buscando) return [];
    return (lista.data ?? [])
      .filter((pessoa) => pessoa.nome.toLocaleLowerCase("pt-BR").includes(termo))
      .slice(0, 8);
  }, [buscando, lista.data, termo]);

  return (
    <div className="flex flex-col gap-2">
      <Input
        id={id}
        label="Responsável pela tarefa"
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="Digite o nome"
        autoComplete="off"
      />
      <p className="text-xs text-muted">Os nomes aparecem a partir da segunda letra.</p>
      {buscando && (
        <ul role="listbox" aria-label="Responsáveis" className="flex flex-col rounded-lg border border-surface-border">
          {lista.isLoading && <li className="px-3 py-2 text-sm text-muted">Carregando pessoas…</li>}
          {!lista.isLoading && visiveis.length === 0 && (
            <li className="px-3 py-2 text-sm text-muted">Nenhuma pessoa com esse nome nesta entidade.</li>
          )}
          {visiveis.map((pessoa) => (
            <li key={pessoa.id} role="option" aria-selected={value.id === pessoa.id}>
              <button
                type="button"
                className="w-full px-3 py-2 text-left text-sm hover:bg-surface-hover"
                onClick={() => {
                  onChange(pessoa);
                  setQ("");
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
