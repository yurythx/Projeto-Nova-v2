"use client";

import { useMemo, useState } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Input } from "@/components/ui/Input";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { PessoaProjeto } from "@/lib/projetos/projetos";

/** Quem pode entrar nas tarefas. O gerente fica, e não sai por aqui. */
export function ParticipantesDoProjeto({
  projetoId,
  mutavel,
}: {
  projetoId: string;
  mutavel: boolean;
}) {
  const [q, setQ] = useState("");
  const membros = useApiQuery<PessoaProjeto[]>(`v1/projetos/${projetoId}/participantes`);
  const unidade = useApiQuery<PessoaProjeto[]>(
    mutavel ? `v1/projetos/${projetoId}/responsaveis` : null,
  );
  const { run, pending } = useAction();
  const termo = q.trim().toLocaleLowerCase("pt-BR");
  const buscando = termo.length >= 2;
  const participantes = (membros.data ?? []).filter((pessoa) => pessoa.papel !== "gerente");
  const visiveis = useMemo(() => {
    if (!buscando) return [];
    const ja = new Set((membros.data ?? []).map((pessoa) => pessoa.id));
    return (unidade.data ?? [])
      .filter((pessoa) => !ja.has(pessoa.id) && pessoa.nome.toLocaleLowerCase("pt-BR").includes(termo))
      .slice(0, 8);
  }, [buscando, membros.data, termo, unidade.data]);

  async function incluir(pessoa: PessoaProjeto) {
    const ok = await run(
      () => apiClient.post(`v1/projetos/${projetoId}/participantes`, { id: pessoa.id }),
      `${pessoa.nome} entrou no projeto.`,
    );
    if (ok) {
      setQ("");
      void membros.mutate();
    }
  }

  async function tirar(pessoa: PessoaProjeto) {
    const ok = await run(
      () => apiClient.delete(`v1/projetos/${projetoId}/participantes/${pessoa.id}`),
      `${pessoa.nome} saiu do projeto e das tarefas.`,
    );
    if (ok) void membros.mutate();
  }

  return (
    <div>
      <span className="text-muted">Pessoas do projeto:</span>
      {membros.error && <p className="mt-1 text-sm text-danger">Não foi possível carregar as pessoas do projeto.</p>}
      {participantes.length === 0 && !membros.error && (
        <p className="mt-1 text-sm text-foreground">Nenhuma pessoa além do gerente.</p>
      )}
      {participantes.length > 0 && (
        <ul className="mt-1 flex flex-col gap-1">
          {participantes.map((pessoa) => (
            <li key={pessoa.id} className="flex items-center justify-between text-sm text-foreground">
              <span>{pessoa.nome}</span>
              {mutavel && (
                <button
                  type="button"
                  className="text-xs text-muted hover:text-danger"
                  disabled={pending}
                  onClick={() => void tirar(pessoa)}
                >
                  Tirar
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
      {mutavel && (
        <div className="mt-2 flex flex-col gap-2">
          <Input
            id="projeto-participantes"
            label="Incluir pessoa"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Digite o nome"
            autoComplete="off"
          />
          <p className="text-xs text-muted">Quem entra aqui pode ser designado nas tarefas. Os nomes aparecem a partir da segunda letra.</p>
          {buscando && (
            <ul role="listbox" aria-label="Incluir no projeto" className="flex flex-col rounded-lg border border-surface-border">
              {visiveis.length === 0 && (
                <li className="px-3 py-2 text-sm text-muted">Nenhuma pessoa com esse nome nesta unidade.</li>
              )}
              {visiveis.map((pessoa) => (
                <li key={pessoa.id}>
                  <button
                    type="button"
                    className="w-full px-3 py-2 text-left text-sm hover:bg-surface-hover"
                    disabled={pending}
                    onClick={() => void incluir(pessoa)}
                  >
                    {pessoa.nome}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
