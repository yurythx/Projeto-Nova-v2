"use client";

import { useParams, useRouter } from "next/navigation";
import { BookOpen, ChevronDown, Plus } from "lucide-react";
import { useState, type ReactNode } from "react";

import { WikiEditor } from "@/components/wiki/WikiEditor";
import { WikiTree } from "@/components/wiki/WikiTree";
import { Button } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { useApiQuery } from "@/lib/api/swr";
import type { WikiPage } from "@/lib/plataforma/types";

/** Layout da Wiki: árvore de páginas à esquerda, conteúdo à direita. */
export default function WikiLayout({ children }: { children: ReactNode }) {
  const params = useParams<{ ref?: string }>();
  const router = useRouter();
  const tree = useApiQuery<WikiPage[]>("v1/wiki/tree");
  const [creating, setCreating] = useState(false);
  const [mobileExpanded, setMobileExpanded] = useState(false);

  return (
    <div className="flex flex-col gap-6 lg:grid lg:grid-cols-[16rem_1fr]">
      <aside className="flex flex-col gap-3 rounded-xl border border-surface-border bg-surface p-3 sm:p-4 lg:sticky lg:top-[calc(var(--topbar-h)+1.5rem)] lg:max-h-[calc(100dvh-var(--topbar-h)-3rem)] lg:overflow-y-auto lg:border-none lg:bg-transparent lg:p-0">
        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={() => setMobileExpanded((prev) => !prev)}
            className="flex items-center gap-2 text-left lg:pointer-events-none"
            aria-expanded={mobileExpanded}
          >
            <BookOpen size={16} className="text-primary lg:hidden" aria-hidden="true" />
            <p className="dateline m-0">Base de conhecimento</p>
            <ChevronDown
              size={14}
              className={`text-muted transition-transform lg:hidden ${mobileExpanded ? "rotate-180" : ""}`}
              aria-hidden="true"
            />
          </button>
          <Button size="sm" variant="ghost" onClick={() => setCreating(true)} aria-label="Nova página">
            <Plus size={16} aria-hidden="true" />
          </Button>
        </div>
        <div className={mobileExpanded ? "block" : "hidden lg:block"}>
          <WikiTree pages={tree.data ?? []} current={params.ref} />
        </div>
      </aside>
      <div className="min-w-0">{children}</div>
      <Dialog open={creating} onClose={() => setCreating(false)} title="Nova página" size="xl">
        {creating && (
          <WikiEditor
            pages={tree.data ?? []}
            defaultParent={tree.data?.find((p) => p.slug === params.ref)?.id}
            onSaved={(p) => {
              setCreating(false);
              void tree.mutate();
              router.push(`/wiki/${p.slug}`);
            }}
          />
        )}
      </Dialog>
    </div>
  );
}
