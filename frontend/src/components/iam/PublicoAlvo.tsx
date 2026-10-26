"use client";

import { Users } from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import { useApiQuery } from "@/lib/api/swr";
import type { OrgTree, Publico } from "@/lib/plataforma/types";

export const TODOS: Publico = { entidades: [], unidades: [] };

export function publicoVazio(p?: Publico | null): boolean {
  return !p || ((p.entidades ?? []).length === 0 && (p.unidades ?? []).length === 0);
}

/** Nomes do público-alvo (secretarias e unidades), na ordem da árvore. */
export function nomesDoPublico(
  p: Publico | undefined | null,
  tree: readonly OrgTree[] | undefined,
): string[] {
  if (publicoVazio(p)) return [];
  const nomes: string[] = [];
  for (const e of tree ?? []) {
    if (p!.entidades?.includes(e.id)) nomes.push(e.sigla || e.nome);
    for (const u of e.unidades) if (p!.unidades?.includes(u.id)) nomes.push(u.sigla || u.nome);
  }
  return nomes;
}

/** Público-alvo (ADR 014): a secretaria inteira e/ou unidades específicas
 * (valem as unidades abaixo). Nada marcado = todos. Inicia sempre minimizado por padrão. */
export function PublicoAlvoPicker({
  value,
  onChange,
  idPrefix,
  defaultOpen = false,
}: {
  value: Publico;
  onChange: (p: Publico) => void;
  idPrefix: string;
  defaultOpen?: boolean;
}) {
  const tree = useApiQuery<OrgTree[]>("v1/iam/org-tree");
  const toggle = (lista: "entidades" | "unidades", id: string, on: boolean) => {
    const atual = value[lista] ?? [];
    onChange({ ...TODOS, ...value, [lista]: on ? [...atual, id] : atual.filter((x) => x !== id) });
  };
  const nomes = nomesDoPublico(value, tree.data);
  const temRestricao = !publicoVazio(value);

  return (
    <details
      open={temRestricao || defaultOpen}
      className="group rounded-xl border border-surface-border bg-surface p-3 text-sm transition-all"
    >
      <summary className="flex cursor-pointer list-none items-center justify-between gap-2 font-medium select-none">
        <div className="flex items-center gap-2">
          <Users size={16} className="text-primary" aria-hidden="true" />
          <span>Público-alvo & Visibilidade</span>
          <Badge tone={temRestricao ? "info" : "neutral"}>
            {temRestricao ? `Restrito (${nomes.length})` : "Todos (Padrão)"}
          </Badge>
        </div>
        <span className="text-xs text-primary hover:underline">
          {temRestricao ? "Alterar restrições" : "Restringir público (opcional)"}
        </span>
      </summary>

      <div className="mt-3 pt-3 border-t border-surface-border flex flex-col gap-2">
        <p className="text-xs text-muted" aria-live="polite">
          {nomes.length === 0
            ? "Todos os servidores (sem restrição)."
            : `Só para: ${nomes.join(", ")} — e as unidades abaixo. Não aparece no site público.`}
        </p>

        {temRestricao && (
          <div className="flex justify-end">
            <button
              type="button"
              onClick={() => onChange(TODOS)}
              className="text-xs text-primary hover:underline"
            >
              Restaurar para Todos (sem restrição)
            </button>
          </div>
        )}

        <div className="flex max-h-60 flex-col gap-1 overflow-y-auto pr-1">
          {(tree.data ?? []).map((e) => (
            <details
              key={e.id}
              open={
                value.entidades?.includes(e.id) ||
                e.unidades.some((u) => value.unidades?.includes(u.id))
              }
            >
              <summary className="cursor-pointer text-sm font-medium hover:text-primary">
                {e.sigla ? `${e.sigla} — ${e.nome}` : e.nome}
              </summary>
              <div className="ml-4 mt-1 flex flex-col gap-1 border-l border-surface-border pl-2">
                <label className="flex items-center gap-2 text-sm font-medium">
                  <input
                    id={`${idPrefix}-e-${e.id}`}
                    type="checkbox"
                    className="h-4 w-4 accent-primary"
                    checked={value.entidades?.includes(e.id) ?? false}
                    onChange={(ev) => toggle("entidades", e.id, ev.target.checked)}
                  />
                  Toda a secretaria
                </label>
                {e.unidades.map((u) => (
                  <label key={u.id} className="flex items-center gap-2 text-sm">
                    <input
                      id={`${idPrefix}-u-${u.id}`}
                      type="checkbox"
                      className="h-4 w-4 accent-primary"
                      checked={value.unidades?.includes(u.id) ?? false}
                      onChange={(ev) => toggle("unidades", u.id, ev.target.checked)}
                    />
                    {u.sigla ? `${u.sigla} — ${u.nome}` : u.nome}
                  </label>
                ))}
              </div>
            </details>
          ))}
        </div>
      </div>
    </details>
  );
}

/** Selo "Restrito a …" para conteúdo com público-alvo. */
export function PublicoBadge({ publico }: { publico?: Publico | null }) {
  const tree = useApiQuery<OrgTree[]>(publicoVazio(publico) ? null : "v1/iam/org-tree");
  if (publicoVazio(publico)) return null;
  const nomes = nomesDoPublico(publico, tree.data);
  return (
    <Badge tone="info" title="Público-alvo: só quem é destas secretarias/unidades vê">
      <Users size={12} aria-hidden="true" className="mr-1 inline" />
      {nomes.length > 0 ? `Para: ${nomes.join(", ")}` : "Público restrito"}
    </Badge>
  );
}
