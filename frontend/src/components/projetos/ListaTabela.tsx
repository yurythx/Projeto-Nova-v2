"use client";

import type { ReactNode } from "react";

import {
  BarraFaixa,
  ordenarColunas,
  useAparencia,
  useFaixasRecolhidas,
  type ColunaFaixa,
  type TomFaixa,
  type VistaCard,
} from "@/components/plataforma/Faixas";
import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from "@/components/ui/Table";

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
  oficial?: boolean;
  cabecalhos: CabecalhoTabela[];
}) {
  const { recolhidas, alternar } = useFaixasRecolhidas(memoria);
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
                    return (
                      <TableRow
                        key={id}
                        onClick={(e) => cliqueAbre(e, aoAbrir ? () => aoAbrir(item) : undefined)}
                        className={`border-l-4 ${aoAbrir ? "cursor-pointer hover:bg-black/[0.03] dark:hover:bg-white/[0.04]" : ""} ${
                          cor ? "" : BORDA[coluna.tom]
                        }`}
                        style={cor ? { borderLeftColor: cor } : undefined}
                      >
                        {render(item, {
                          titulo: nome,
                          cor,
                          salvarNome: (valor) => {
                            if (aoRenomear) aoRenomear(item, valor);
                            else look.salvarCartao(id, { titulo: valor });
                          },
                        })}
                      </TableRow>
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
