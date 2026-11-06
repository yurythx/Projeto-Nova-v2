"use client";

import { ChevronRight } from "lucide-react";
import { Fragment, useState, type ReactNode } from "react";

import {
  BarraFaixa,
  ordenarColunas,
  useAparencia,
  useFaixasRecolhidas,
  type ColunaFaixa,
  type TomFaixa,
  type VistaCard,
} from "@/components/plataforma/Faixas";
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from "@/components/ui/Table";

const BORDA: Record<TomFaixa, string> = {
  neutral: "border-l-foreground/40",
  info: "border-l-info",
  warning: "border-l-warning",
  success: "border-l-success",
  danger: "border-l-danger",
};

export interface CabecalhoTabela {
  id: string;
  rotulo: string;
  oculta?: boolean;
}

function cliqueAbre(e: { target: EventTarget | null }, aoAbrir?: () => void) {
  if (!aoAbrir) return;
  const alvo = e.target as HTMLElement | null;
  if (alvo?.closest("a, button, input, select, textarea, label")) return;
  aoAbrir();
}

/** Lista agrupada pelo estado, com cabeçalho de verdade em cada grupo. */
export function ListaTabela<T>({
  rotulo,
  memoria,
  aparencia,
  colunas,
  itens,
  colunaDe,
  idDe,
  nomeDe,
  render,
  aoRenomear,
  aoAbrir,
  detalhe,
  oficial = false,
  cabecalhos,
}: {
  rotulo: string;
  memoria: string;
  aparencia?: string;
  colunas: ColunaFaixa[];
  itens: T[];
  colunaDe: (item: T) => string | null;
  idDe: (item: T) => string;
  nomeDe?: (item: T) => string;
  render: (item: T, vista: VistaCard) => ReactNode;
  aoRenomear?: (item: T, nome: string) => void;
  aoAbrir?: (item: T) => void;
  /** Conteúdo que a linha abre embaixo, como as tarefas de um projeto. */
  detalhe?: (item: T) => ReactNode;
  oficial?: boolean;
  cabecalhos: CabecalhoTabela[];
}) {
  const { recolhidas, alternar } = useFaixasRecolhidas(memoria);
  const [abertos, setAbertos] = useState<Record<string, boolean>>({});
  function alternarItem(id: string) {
    setAbertos((atual) => ({ ...atual, [id]: !atual[id] }));
  }
  const look = useAparencia(aparencia ?? memoria);
  const faixas = ordenarColunas(colunas);

  return (
    <div className="flex min-w-0 max-w-full flex-col gap-2" aria-label={rotulo}>
      {faixas.map((coluna) => {
        const cartoes = itens.filter((item) => colunaDe(item) === coluna.id);
        const recolhida = recolhidas.includes(coluna.id);
        const tituloId = `faixa-${memoria}-${coluna.id}`;
        const titulo = oficial
          ? coluna.titulo
          : look.gravado.colunas[coluna.id]?.titulo || coluna.titulo;
        const corColuna = oficial ? coluna.cor : look.gravado.colunas[coluna.id]?.cor;
        return (
          <section key={coluna.id} aria-labelledby={tituloId} data-coluna={coluna.id}>
            <BarraFaixa
              titulo={titulo}
              tituloId={tituloId}
              total={cartoes.length}
              tom={coluna.tom}
              cor={corColuna}
              recolhida={recolhida}
              ordem={coluna.ordem}
              corImediata={!oficial}
              onAlternar={() => alternar(coluna.id)}
              onRenomear={
                oficial ? undefined : (nome) => look.salvarColuna(coluna.id, { titulo: nome })
              }
              onCor={oficial ? undefined : (cor) => look.salvarColuna(coluna.id, { cor })}
            />
            {!recolhida && cartoes.length === 0 && (
              <p className="px-3 py-2 text-sm text-muted">Nenhum item</p>
            )}
            {!recolhida && cartoes.length > 0 && (
              <Table caption={`${rotulo}: ${titulo}`} className="border-separate border-spacing-0">
                <TableHead>
                  <TableRow>
                    {detalhe && (
                      <TableHeaderCell className="w-10">
                        <span className="sr-only">Tarefas</span>
                      </TableHeaderCell>
                    )}
                    {cabecalhos.map((colunaTabela) => (
                      <TableHeaderCell
                        key={colunaTabela.id}
                        className={colunaTabela.oculta ? "sr-only" : undefined}
                      >
                        {colunaTabela.rotulo}
                      </TableHeaderCell>
                    ))}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {cartoes.map((item) => {
                    const id = idDe(item);
                    const nome = look.gravado.cartoes[id]?.titulo || nomeDe?.(item) || "";
                    const cor = look.gravado.cartoes[id]?.cor || corColuna;
                    const aberto = Boolean(abertos[id]);
                    const aoClicar = detalhe
                      ? () => alternarItem(id)
                      : aoAbrir
                        ? () => aoAbrir(item)
                        : undefined;
                    return (
                      <Fragment key={id}>
                        <TableRow
                          onClick={(e) => cliqueAbre(e, aoClicar)}
                          className={`border-l-4 ${aoClicar ? "cursor-pointer hover:bg-black/[0.03] dark:hover:bg-white/[0.04]" : ""} ${
                            cor ? "" : BORDA[coluna.tom]
                          }`}
                          style={cor ? { borderLeftColor: cor } : undefined}
                        >
                          {detalhe && (
                            <TableCell className="w-10 pr-0">
                              <button
                                type="button"
                                aria-expanded={aberto}
                                aria-label={`${aberto ? "Recolher" : "Expandir"} tarefas de ${nome || "projeto"}`}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  alternarItem(id);
                                }}
                                className="inline-flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-black/5 dark:hover:bg-white/10"
                              >
                                <ChevronRight
                                  size={16}
                                  aria-hidden="true"
                                  className={`transition-transform ${aberto ? "rotate-90" : ""}`}
                                />
                              </button>
                            </TableCell>
                          )}
                          {render(item, {
                            titulo: nome,
                            cor,
                            salvarNome: (valor) => {
                              if (aoRenomear) aoRenomear(item, valor);
                              else look.salvarCartao(id, { titulo: valor });
                            },
                          })}
                        </TableRow>
                        {aberto && detalhe && (
                          <TableRow className="bg-black/[0.02] dark:bg-white/[0.03]">
                            <TableCell colSpan={cabecalhos.length + 1} className="p-0">
                              {detalhe(item)}
                            </TableCell>
                          </TableRow>
                        )}
                      </Fragment>
                    );
                  })}
                </TableBody>
              </Table>
            )}
          </section>
        );
      })}
    </div>
  );
}
