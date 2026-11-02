"use client";

import { X } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { useApiPage, withQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { Person } from "@/lib/plataforma/types";

export interface PessoaEscolhida {
  id: string;
  nome: string;
}

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Uma pessoa ativa. Com unidade, a busca fica nessa lotação. Sem o
 * Diretório, aceita o id do usuário. */
export function PessoaDaUnidade({
  id,
  label,
  unidadeId,
  value,
  onChange,
}: {
  id: string;
  label: string;
  unidadeId?: string;
  value: PessoaEscolhida;
  onChange: (v: PessoaEscolhida) => void;
}) {
  const { enabled } = usePlataforma();
  const [q, setQ] = useState("");
  const directory = enabled("directory");
  const results = useApiPage<Person>(
    directory && q.trim().length >= 2
      ? withQuery("v1/directory/people", {
          q: q.trim(),
          page_size: 8,
          unidade_id: unidadeId || undefined,
        })
      : null,
  );

  function escolher(pessoa: PessoaEscolhida) {
    onChange(pessoa);
    setQ("");
  }

  return (
    <div className="flex flex-col gap-2" data-testid={`pessoa-${id}`}>
      <Input
        id={id}
        label={label}
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder={directory ? "Digite para buscar a pessoa" : "Id (UUID) do usuário"}
        autoComplete="off"
        onKeyDown={(e) => {
          if (!directory && e.key === "Enter" && UUID_RE.test(q.trim())) {
            e.preventDefault();
            escolher({ id: q.trim(), nome: q.trim() });
          }
        }}
      />
      {directory && (results.data?.items ?? []).length > 0 && (
        <ul role="listbox" aria-label="Pessoas" className="flex flex-col rounded-lg border border-surface-border">
          {(results.data?.items ?? []).map((p) => (
            <li key={p.user_id} role="option" aria-selected={value.id === p.user_id}>
              <button
                type="button"
                onClick={() => escolher({ id: p.user_id, nome: p.name || p.username })}
                className="w-full px-3 py-2 text-left text-sm hover:bg-surface-hover"
              >
                {p.name || p.username}{" "}
                <span className="text-xs text-muted">{[p.job_title, p.unidade].filter(Boolean).join(" · ")}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {!directory && (
        <Button
          type="button"
          variant="secondary"
          size="sm"
          className="self-start"
          disabled={!UUID_RE.test(q.trim())}
          onClick={() => escolher({ id: q.trim(), nome: q.trim() })}
        >
          Usar este id
        </Button>
      )}
      {value.nome && (
        <p className="flex items-center justify-between rounded-md bg-surface-hover px-3 py-1.5 text-sm">
          <span>
            {value.id ? value.nome : `Nome já registrado: ${value.nome}`}
          </span>
          <button
            type="button"
            aria-label={`Remover ${value.nome}`}
            onClick={() => onChange({ id: "", nome: "" })}
            className="rounded p-1 text-muted hover:text-danger"
          >
            <X size={14} aria-hidden="true" />
          </button>
        </p>
      )}
    </div>
  );
}
