"use client";

import Link from "next/link";
import { useState } from "react";

import { PageHeader } from "@/components/plataforma/PageHeader";
import { useQuadro } from "@/components/plataforma/useQuadro";
import { ordenarColunas, type ColunaFaixa, type TomFaixa } from "@/components/plataforma/Faixas";
import { Button, buttonClass } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

const COR_DO_TOM: Record<TomFaixa, string> = {
  neutral: "#64748b",
  info: "#2563eb",
  warning: "#d97706",
  success: "#059669",
  danger: "#dc2626",
};

interface Linha {
  id: string;
  nome: string;
  cor: string;
  ordem: string;
  entrada: boolean;
  ativa: boolean;
  tom: TomFaixa;
}

function paraLinhas(colunas: readonly ColunaFaixa[]): Linha[] {
  return ordenarColunas(colunas).map((coluna) => ({
    id: coluna.id,
    nome: coluna.titulo,
    cor: coluna.cor || COR_DO_TOM[coluna.tom],
    ordem: String(coluna.ordem),
    entrada: coluna.entrada === true,
    ativa: coluna.ativa !== false,
    tom: coluna.tom,
  }));
}

/** A mesma tela de colunas em todos os quadros: posição a partir de zero, nome e cor. */
export function ConfiguracaoDeColunas({
  quadro,
  fabrica,
  titulo,
  voltar,
  pode,
}: {
  quadro: string;
  fabrica: readonly ColunaFaixa[];
  titulo: string;
  voltar: string;
  pode: boolean;
}) {
  const catalogo = useQuadro(quadro, fabrica);
  const origem = catalogo.colunas.map((coluna) => `${coluna.id}:${coluna.titulo}:${coluna.cor ?? ""}:${coluna.ordem}:${coluna.ativa !== false}`).join("|");
  const [vista, setVista] = useState({ origem, linhas: paraLinhas(catalogo.colunas), sujo: false });
  const [retirar, setRetirar] = useState<string | null>(null);
  const [nova, setNova] = useState<{ nome: string; cor: string; ordem: string; de: string[]; para: string[] } | null>(null);
  const [erro, setErro] = useState("");
  const [gravando, setGravando] = useState(false);

  if (!vista.sujo && !gravando && vista.origem !== origem) {
    setVista({ origem, linhas: paraLinhas(catalogo.colunas), sujo: false });
  }

  function alterar(id: string, parcial: Partial<Linha>) {
    setErro("");
    setVista((atual) => ({
      ...atual,
      sujo: true,
      linhas: atual.linhas.map((linha) => (linha.id === id ? { ...linha, ...parcial } : linha)),
    }));
  }

  async function salvar() {
    if (!pode || !catalogo.oficial) return;
    const ativas = vista.linhas.filter((linha) => linha.ativa);
    for (const linha of ativas) {
      if (!linha.nome.trim()) {
        setErro("Toda coluna precisa de um nome.");
        return;
      }
      if (!/^\d+$/.test(linha.ordem.trim())) {
        setErro("A posição é um número a partir de zero.");
        return;
      }
    }
    setGravando(true);
    setErro("");
    const porId = new Map(catalogo.colunas.map((coluna) => [coluna.id, coluna]));
    for (const linha of ativas) {
      const atual = porId.get(linha.id);
      const nome = linha.nome.trim();
      const cor = linha.cor;
      if (nome !== atual?.titulo || cor !== (atual?.cor || COR_DO_TOM[linha.tom])) {
        const ok = await catalogo.gravar(linha.id, { nome, cor });
        if (!ok) {
          setGravando(false);
          return;
        }
      }
    }
    const desejada = [...ativas].sort((a, b) => Number(a.ordem) - Number(b.ordem) || a.id.localeCompare(b.id));
    const atual = ordenarColunas(catalogo.colunas.filter((coluna) => coluna.ativa !== false)).map((coluna) => coluna.id);
    const mudouOrdem = atual.length !== desejada.length || atual.some((id, i) => id !== desejada[i]?.id);
    if (mudouOrdem) {
      for (let i = 0; i < desejada.length; i += 1) {
        const ok = await catalogo.gravar(desejada[i]!.id, { ordem: i });
        if (!ok) {
          setGravando(false);
          return;
        }
      }
    }
    setGravando(false);
    setVista((atual) => ({ ...atual, sujo: false }));
  }

  function marcar(lista: "de" | "para", id: string) {
    setNova((atual) => {
      if (!atual) return atual;
      const tem = atual[lista].includes(id);
      return { ...atual, [lista]: tem ? atual[lista].filter((item) => item !== id) : [...atual[lista], id] };
    });
  }

  async function criar() {
    if (!pode || !catalogo.oficial || !nova) return;
    if (!nova.nome.trim()) {
      setErro("Toda coluna precisa de um nome.");
      return;
    }
    if (!/^\d+$/.test(nova.ordem.trim())) {
      setErro("A posição é um número a partir de zero.");
      return;
    }
    if (nova.de.length === 0) {
      setErro("Escolha ao menos uma coluna de onde o card entra.");
      return;
    }
    setGravando(true);
    setErro("");
    const ok = await catalogo.criar({
      nome: nova.nome.trim(),
      cor: nova.cor,
      ordem: Number(nova.ordem),
      de: nova.de,
      para: nova.para,
    });
    setGravando(false);
    if (ok) setNova(null);
  }

  async function confirmarRetirada(id: string) {
    setGravando(true);
    const ok = await catalogo.gravar(id, { ativa: false });
    setGravando(false);
    if (ok) setRetirar(null);
  }

  async function devolver(id: string) {
    setGravando(true);
    await catalogo.gravar(id, { ativa: true });
    setGravando(false);
  }

  const ativas = ordenarColunas(vista.linhas.filter((linha) => linha.ativa).map((linha) => ({ ...linha, ordem: Number(linha.ordem) || 0 })));
  const retiradas = vista.linhas.filter((linha) => !linha.ativa);

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Quadro"
        title={titulo}
        description="A posição começa em zero e define o lugar da coluna neste quadro. Uma coluna nova fica só aqui, com uma chave própria e as passagens que você marcar."
        actions={
          <Link href={voltar} className={buttonClass("secondary")}>
            Voltar ao quadro
          </Link>
        }
      />

      {!catalogo.oficial && (
        <p className="text-sm text-muted">O catálogo não respondeu. O quadro segue o padrão de fábrica até a configuração voltar.</p>
      )}

      {pode && catalogo.oficial && !nova && (
        <div>
          <Button type="button" variant="secondary" onClick={() => setNova({ nome: "", cor: "#2563eb", ordem: String(ativas.length), de: [], para: [] })}>
            Nova coluna
          </Button>
        </div>
      )}

      {nova && (
        <form
          className="grid gap-4 rounded-xl border border-surface-border bg-surface p-4"
          onSubmit={(e) => {
            e.preventDefault();
            void criar();
          }}
        >
          <h2 className="text-base font-semibold">Nova coluna deste quadro</h2>
          <div className="grid items-end gap-3 md:grid-cols-[6rem_auto_1fr]">
            <Input id={`nova-posicao-${quadro}`} label="Posição" type="number" min={0} inputMode="numeric" value={nova.ordem} onChange={(e) => setNova({ ...nova, ordem: e.target.value })} />
            <div className="flex flex-col gap-1">
              <label htmlFor={`nova-cor-${quadro}`} className="text-sm font-medium">Cor</label>
              <input id={`nova-cor-${quadro}`} type="color" aria-label="Cor da coluna nova" value={nova.cor} onChange={(e) => setNova({ ...nova, cor: e.target.value })} className="h-10 w-14 cursor-pointer rounded-md border border-surface-border bg-surface" />
            </div>
            <Input id={`nova-nome-${quadro}`} label="Nome" value={nova.nome} maxLength={80} onChange={(e) => setNova({ ...nova, nome: e.target.value })} />
          </div>
          <fieldset className="grid gap-2">
            <legend className="text-sm font-medium">O card entra a partir de</legend>
            <div className="flex flex-wrap gap-3">
              {ativas.map((linha) => (
                <label key={`de-${linha.id}`} className="flex items-center gap-2 text-sm">
                  <input type="checkbox" checked={nova.de.includes(linha.id)} onChange={() => marcar("de", linha.id)} />
                  {linha.nome}
                </label>
              ))}
            </div>
          </fieldset>
          <fieldset className="grid gap-2">
            <legend className="text-sm font-medium">O card pode seguir para</legend>
            <div className="flex flex-wrap gap-3">
              {ativas.map((linha) => (
                <label key={`para-${linha.id}`} className="flex items-center gap-2 text-sm">
                  <input type="checkbox" checked={nova.para.includes(linha.id)} onChange={() => marcar("para", linha.id)} />
                  {linha.nome}
                </label>
              ))}
            </div>
          </fieldset>
          <div className="flex gap-2">
            <Button type="submit" loading={gravando}>Criar coluna</Button>
            <Button type="button" variant="secondary" onClick={() => setNova(null)}>Cancelar</Button>
          </div>
        </form>
      )}

      <ol className="flex flex-col gap-3">
        {ativas.map((linha) => (
          <li key={linha.id} className="grid items-end gap-3 rounded-xl border border-surface-border bg-surface p-4 md:grid-cols-[6rem_auto_1fr_auto]">
            <Input
              id={`posicao-${quadro}-${linha.id}`}
              label="Posição"
              type="number"
              min={0}
              inputMode="numeric"
              value={vista.linhas.find((item) => item.id === linha.id)?.ordem ?? "0"}
              disabled={!pode || !catalogo.oficial}
              onChange={(e) => alterar(linha.id, { ordem: e.target.value })}
            />
            <div className="flex flex-col gap-1">
              <label htmlFor={`cor-${quadro}-${linha.id}`} className="text-sm font-medium">
                Cor
              </label>
              <input
                id={`cor-${quadro}-${linha.id}`}
                type="color"
                aria-label={`Cor de ${linha.nome}`}
                value={linha.cor}
                disabled={!pode || !catalogo.oficial}
                onChange={(e) => alterar(linha.id, { cor: e.target.value })}
                className="h-10 w-14 cursor-pointer rounded-md border border-surface-border bg-surface disabled:opacity-50"
              />
            </div>
            <Input
              id={`nome-${quadro}-${linha.id}`}
              label="Nome"
              value={linha.nome}
              maxLength={80}
              disabled={!pode || !catalogo.oficial}
              onChange={(e) => alterar(linha.id, { nome: e.target.value })}
            />
            <div className="flex flex-col gap-1 pb-1">
              <span className="text-xs text-muted">Estado {linha.id}</span>
              {linha.entrada ? (
                <span className="text-xs text-muted">Coluna inicial</span>
              ) : pode && catalogo.oficial ? (
                retirar === linha.id ? (
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="text-xs text-muted">Retirar esta coluna?</span>
                    <button type="button" className="text-xs font-medium text-primary hover:underline" onClick={() => void confirmarRetirada(linha.id)}>
                      Confirmar retirada
                    </button>
                    <button type="button" className="text-xs font-medium text-muted hover:underline" onClick={() => setRetirar(null)}>
                      Cancelar
                    </button>
                  </span>
                ) : (
                  <button type="button" className="text-left text-xs font-medium text-primary hover:underline" onClick={() => setRetirar(linha.id)}>
                    Retirar
                  </button>
                )
              ) : null}
            </div>
          </li>
        ))}
      </ol>

      {retiradas.length > 0 && (
        <div className="flex flex-col gap-2">
          <h2 className="text-sm font-semibold">Colunas retiradas</h2>
          <ul className="flex flex-col gap-2">
            {retiradas.map((linha) => (
              <li key={linha.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-surface-border px-4 py-3 text-sm">
                <span>
                  {linha.nome} <span className="text-muted">· {linha.id}</span>
                </span>
                {pode && catalogo.oficial && (
                  <button type="button" className="text-xs font-medium text-primary hover:underline" onClick={() => void devolver(linha.id)}>
                    Devolver ao fim
                  </button>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      {erro && <p className="text-sm text-danger">{erro}</p>}

      {pode && (
        <div>
          <Button type="button" onClick={() => void salvar()} loading={gravando} disabled={!catalogo.oficial}>
            Salvar colunas
          </Button>
        </div>
      )}
    </div>
  );
}
