"use client";

import { Building2, Check, ChevronDown } from "lucide-react";
import { useEffect, useMemo, useRef, useState, useSyncExternalStore } from "react";

import {
  getActiveEntitySnapshot,
  setActiveEntity,
  subscribeActiveEntity,
} from "@/lib/plataforma/activeEntityStore";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import { useApiQuery } from "@/lib/api/swr";
import type { OrgTree } from "@/lib/plataforma/types";

/**
 * Filtro oficial de entidades do município:
 * Apenas as 3 instituições oficiais configuradas:
 * 1. Prefeitura Municipal
 * 2. SMS (Secretaria Municipal de Saúde)
 * 3. SEMECEL (Secretaria Municipal de Educação, Cultura, Esporte e Lazer)
 */
function ehEntidadeOficial(nomeOuSigla: string): boolean {
  const s = nomeOuSigla.toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "");
  return s.includes("prefeitura") || s.includes("sms") || s.includes("saude") || s.includes("semecel") || s.includes("educacao");
}

function formatarRotuloEntidade(nome: string, sigla?: string): { nome: string; sigla: string } {
  const norm = (sigla || nome).toUpperCase();
  if (norm.includes("PREFEITURA")) {
    return { nome: "Prefeitura Municipal", sigla: "Prefeitura" };
  }
  if (norm.includes("SMS") || norm.includes("SAUDE")) {
    return { nome: "Secretaria Municipal de Saúde", sigla: "SMS" };
  }
  if (norm.includes("SEMECEL") || norm.includes("EDUCACAO")) {
    return { nome: "Secretaria de Educação, Cultura, Esporte e Lazer", sigla: "SEMECEL" };
  }
  return { nome, sigla: sigla || nome };
}

/**
 * Seletor de Entidade Ativa no cabeçalho (Conceito de Active Entity do GLPI).
 * Permite que administradores gerais e servidores com acesso a múltiplas secretarias
 * alternem o contexto institucional da aplicação inteira em 1 clique.
 * Restringe-se estritamente às 3 entidades oficiais: Prefeitura, SMS e SEMECEL.
 */
export function EntitySelector() {
  const { me, canGlobal } = usePlataforma();
  const tree = useApiQuery<OrgTree[]>("v1/iam/org-tree");
  const activeEntity = useSyncExternalStore(
    subscribeActiveEntity,
    getActiveEntitySnapshot,
    () => "",
  );
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  const isGlobal = Boolean(canGlobal("*") || canGlobal("atendimento:atender"));

  // Lista estritamente as entidades oficiais alcançáveis pelo usuário (sem unidades filhas)
  const opcoes = useMemo(() => {
    const list: { id: string; nome: string; sigla: string }[] = [];

    for (const ent of tree.data ?? []) {
      // Filtra apenas entidades oficiais necessárias
      if (!ehEntidadeOficial(ent.nome) && !ehEntidadeOficial(ent.sigla || "")) {
        continue;
      }

      const temAcessoAEntidade =
        isGlobal ||
        (me?.scopes ?? []).some(
          (s) =>
            s.entidade_id === ent.id ||
            ent.unidades.some((u) => u.id === s.unidade_id),
        );

      if (temAcessoAEntidade && !list.some((item) => item.id === ent.id)) {
        const fmt = formatarRotuloEntidade(ent.nome, ent.sigla);
        list.push({
          id: ent.id,
          nome: fmt.nome,
          sigla: fmt.sigla,
        });
      }
    }

    // Ordenação fixa de governança: Prefeitura primeiro, depois SMS e SEMECEL
    return list.sort((a, b) => {
      if (a.sigla === "Prefeitura") return -1;
      if (b.sigla === "Prefeitura") return 1;
      return a.sigla.localeCompare(b.sigla, "pt-BR");
    });
  }, [tree.data, me?.scopes, isGlobal]);

  // Fecha ao clicar fora ou pressionar Escape
  useEffect(() => {
    if (!open) return;
    function onPointerDown(e: PointerEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  // Se o usuário tem apenas 1 entidade/secretaria e não é admin global, não exibe o seletor
  if (!isGlobal && opcoes.length <= 1) {
    return null;
  }

  const selecionada = opcoes.find((o) => o.id === activeEntity);
  const rotuloAtual = selecionada ? selecionada.sigla : "Todas as instituições";

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Selecionar instituição ativa"
        className="inline-flex max-w-44 items-center gap-1.5 rounded-md border border-surface-border bg-surface px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-surface-hover hover:border-primary/40 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary sm:max-w-60"
        title={`Instituição ativa: ${selecionada ? selecionada.nome : "Todas as instituições"}`}
      >
        <Building2 size={13} className="shrink-0 text-primary" aria-hidden="true" />
        <span className="truncate">{rotuloAtual}</span>
        <ChevronDown size={12} className="shrink-0 text-muted" aria-hidden="true" />
      </button>

      {open && (
        <div
          role="menu"
          aria-label="Instituições disponíveis"
          className="absolute right-0 top-9 z-50 w-72 max-w-[calc(100vw-2rem)] rounded-lg border border-surface-border bg-surface py-1 shadow-xl animate-in fade-in zoom-in-95 duration-100"
        >
          <div className="border-b border-surface-border px-3 py-2">
            <p className="text-xs font-semibold text-foreground">Entidade / Secretaria Ativa</p>
            <p className="text-[11px] text-muted">Filtra catálogo, chamados e ativos na entidade selecionada.</p>
          </div>

          <div className="max-h-64 overflow-y-auto py-1">
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setActiveEntity("");
                setOpen(false);
              }}
              className="flex w-full items-center justify-between px-3 py-2 text-left text-xs text-foreground transition-colors hover:bg-surface-hover"
            >
              <span className="font-medium">Todas as instituições (visão global)</span>
              {!activeEntity && <Check size={14} className="text-primary" aria-hidden="true" />}
            </button>

            {opcoes.length > 0 && <div className="my-1 border-t border-surface-border" />}

            {opcoes.map((opc) => {
              const ativa = activeEntity === opc.id;
              return (
                <button
                  key={opc.id}
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    setActiveEntity(opc.id);
                    setOpen(false);
                  }}
                  className={`flex w-full items-center justify-between px-3 py-2 text-left text-xs transition-colors hover:bg-surface-hover ${
                    ativa ? "font-semibold text-primary" : "text-foreground"
                  }`}
                >
                  <div className="min-w-0 pr-2">
                    <p className="truncate font-medium">{opc.nome}</p>
                    <p className="text-[10px] text-muted">{opc.sigla}</p>
                  </div>
                  {ativa && <Check size={14} className="shrink-0 text-primary" aria-hidden="true" />}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
