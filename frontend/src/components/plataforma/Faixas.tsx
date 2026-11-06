"use client";

import { ChevronDown, ChevronRight } from "lucide-react";
import { type DragEvent, type ReactNode, useId, useMemo, useState } from "react";

import { gravarArmazenado, useArmazenado } from "@/lib/prefs/armazenado";

/** Cor da faixa. Os mesmos cinco tons do selo, para a lista e o quadro
 * falarem a mesma língua. */
export type TomFaixa = "neutral" | "info" | "warning" | "success" | "danger";

export interface ColunaFaixa {
  id: string;
  titulo: string;
  tom: TomFaixa;
  /** Posição no fluxo, a partir de zero. O nome da coluna não muda isto. */
  ordem: number;
  /** Cor da organização, no formato #rrggbb. Sem isto, vale o tom. */
  cor?: string;
  /** Coluna em que o fluxo começa. Não sai do quadro. */
  entrada?: boolean;
  /** false: retirada do quadro. Ausente vale como no quadro. */
  ativa?: boolean;
  /** Colunas de onde o card entra nesta. Só a coluna criada na tela. */
  de?: string[];
  /** Colunas para onde o card sai desta. Só a coluna criada na tela. */
  para?: string[];
}

/** Rótulo do estado. A coluna do catálogo ganha do nome de fábrica. */
export function seloDoEstado(
  colunas: readonly { id: string; titulo: string; tom: TomFaixa }[],
  id: string,
  conhecido?: { label?: string; tone?: TomFaixa; acao?: string },
) {
  const coluna = colunas.find((item) => item.id === id);
  return {
    label: coluna?.titulo || conhecido?.label || id,
    tone: coluna?.tom || conhecido?.tone || ("neutral" as TomFaixa),
    acao: conhecido?.acao || coluna?.titulo || id,
  };
}

/** O catálogo responde: o nome e a cor da coluna vêm do servidor. */
export interface ControlesColuna {
  oficial?: boolean;
}

/** A posição nasce na ordem da lista: a primeira coluna é 0. */
export function colunasEmOrdem(colunas: Omit<ColunaFaixa, "ordem">[]): ColunaFaixa[] {
  return colunas.map((coluna, ordem) => ({ ...coluna, ordem }));
}

/** A posição numérica manda. O nome fica de fora. */
export function ordenarColunas<T extends { ordem: number }>(colunas: readonly T[]): T[] {
  return [...colunas].sort((a, b) => a.ordem - b.ordem);
}

const ESTILO: Record<TomFaixa, { barra: string; cartao: string; ponto: string }> = {
  neutral: {
    barra: "border-l-foreground/40 bg-black/[0.03] dark:bg-white/[0.04]",
    cartao: "border-l-foreground/40",
    ponto: "bg-foreground/40",
  },
  info: {
    barra: "border-l-accent bg-accent/10",
    cartao: "border-l-accent",
    ponto: "bg-accent",
  },
  warning: {
    barra: "border-l-warning bg-warning/10",
    cartao: "border-l-warning",
    ponto: "bg-warning",
  },
  success: {
    barra: "border-l-success bg-success/10",
    cartao: "border-l-success",
    ponto: "bg-success",
  },
  danger: {
    barra: "border-l-danger bg-danger/10",
    cartao: "border-l-danger",
    ponto: "bg-danger",
  },
};

export function pontoFaixa(tom: TomFaixa): string {
  return ESTILO[tom].ponto;
}

function lerRecolhidas(cru: string | null): string[] {
  if (!cru) return [];
  try {
    const lista = JSON.parse(cru) as unknown;
    return Array.isArray(lista)
      ? lista.filter((item): item is string => typeof item === "string")
      : [];
  } catch {
    return [];
  }
}

/** Lembra quais faixas a pessoa recolheu neste navegador. A primeira
 * pintura sai sem nada recolhido e o gravado entra em seguida. */
export function useFaixasRecolhidas(memoria: string) {
  const cru = useArmazenado(memoria);
  const recolhidas = useMemo(() => lerRecolhidas(cru), [cru]);

  function alternar(id: string) {
    const proxima = recolhidas.includes(id)
      ? recolhidas.filter((item) => item !== id)
      : [...recolhidas, id];
    gravarArmazenado(memoria, JSON.stringify(proxima));
  }

  return { recolhidas, alternar };
}

const COR_PADRAO: Record<TomFaixa, string> = {
  neutral: "#64748b",
  info: "#2563eb",
  warning: "#d97706",
  success: "#059669",
  danger: "#dc2626",
};

export interface Aparencia {
  titulo?: string;
  cor?: string;
}

interface Gravado {
  colunas: Record<string, Aparencia>;
  cartoes: Record<string, Aparencia>;
}

function vazio(): Gravado {
  return { colunas: {}, cartoes: {} };
}

function lerAparencia(cru: string | null): Gravado {
  if (!cru) return vazio();
  try {
    const dado = JSON.parse(cru) as Partial<Gravado>;
    return {
      colunas: dado.colunas && typeof dado.colunas === "object" ? dado.colunas : {},
      cartoes: dado.cartoes && typeof dado.cartoes === "object" ? dado.cartoes : {},
    };
  } catch {
    return vazio();
  }
}

/** Nome e cor escolhidos neste navegador. Todo quadro aberto com a mesma
 * memória redesenha junto quando um deles grava. */
export function useAparencia(memoria: string) {
  const chave = `${memoria}:aparencia`;
  const cru = useArmazenado(chave);
  const gravado = useMemo(() => lerAparencia(cru), [cru]);

  function salvarColuna(id: string, parcial: Aparencia) {
    const proximo = {
      ...gravado,
      colunas: { ...gravado.colunas, [id]: { ...gravado.colunas[id], ...parcial } },
    };
    gravarArmazenado(chave, JSON.stringify(proximo));
  }

  function salvarCartao(id: string, parcial: Aparencia) {
    const proximo = {
      ...gravado,
      cartoes: { ...gravado.cartoes, [id]: { ...gravado.cartoes[id], ...parcial } },
    };
    gravarArmazenado(chave, JSON.stringify(proximo));
  }

  return { gravado, salvarColuna, salvarCartao };
}

/** Rótulo que a pessoa vê. O id da coluna continua sendo o estado. */
export function rotuloDaColuna(
  colunas: readonly { id: string; titulo: string }[],
  id: string,
  gravado: { colunas: Record<string, { titulo?: string } | undefined> },
): string {
  return gravado.colunas[id]?.titulo || colunas.find((coluna) => coluna.id === id)?.titulo || id;
}

function pintura(
  cor: string | undefined,
  tom: TomFaixa,
  tipo: "barra" | "cartao",
): { className: string; style?: { borderLeftColor: string; backgroundColor: string } } {
  if (!cor) return { className: ESTILO[tom][tipo] };
  return {
    className: "",
    style: { borderLeftColor: cor, backgroundColor: `${cor}22` },
  };
}

const swatch =
  "size-4 shrink-0 cursor-pointer appearance-none rounded-full border border-black/10 bg-transparent p-0 [&::-webkit-color-swatch-wrapper]:p-0 [&::-webkit-color-swatch]:rounded-full [&::-webkit-color-swatch]:border-0";

export function BarraFaixa({
  titulo,
  tituloId,
  total,
  tom,
  cor,
  recolhida,
  vertical = false,
  onAlternar,
  onRenomear,
  onCor,
  extra,
  ordem,
  corImediata = true,
}: {
  titulo: string;
  tituloId: string;
  total: number;
  tom: TomFaixa;
  cor?: string;
  recolhida: boolean;
  /** Faixa recolhida no quadro: uma coluna estreita, com o nome de cima para baixo. */
  vertical?: boolean;
  onAlternar: () => void;
  onRenomear?: (nome: string) => void;
  onCor?: (cor: string) => void;
  /** false grava a cor ao soltar o seletor, em vez de a cada mudança. */
  corImediata?: boolean;
  extra?: ReactNode;
  /** Posição no fluxo, a partir de zero. */
  ordem: number;
}) {
  const [editando, setEditando] = useState(false);
  const [rascunho, setRascunho] = useState(titulo);
  const verbo = recolhida ? "Expandir" : "Recolher";
  const faixa = pintura(cor, tom, "barra");
  function comecar() {
    setRascunho(titulo);
    setEditando(true);
  }
  return (
    <div
      className={`relative rounded-lg border border-surface-border border-l-4 ${faixa.className} ${
        vertical
          ? "flex h-full min-h-48 flex-col items-center gap-2 px-1 py-2"
          : "flex flex-wrap items-center gap-2 px-2 py-1.5"
      }`}
      style={faixa.style}
    >
      <button
        type="button"
        className="inline-flex size-6 shrink-0 items-center justify-center rounded-md text-foreground hover:bg-black/5 dark:hover:bg-white/10"
        aria-expanded={!recolhida}
        aria-label={`${verbo} ${titulo}`}
        onClick={onAlternar}
      >
        {recolhida ? (
          <ChevronRight className="size-4" aria-hidden />
        ) : (
          <ChevronDown className="size-4" aria-hidden />
        )}
      </button>
      <span
        className="inline-flex h-5 min-w-5 shrink-0 items-center justify-center rounded bg-black/5 px-1 text-[11px] font-semibold tabular-nums text-muted dark:bg-white/10"
        aria-hidden
      >
        {ordem}
      </span>
      {onCor && (
        <input
          type="color"
          aria-label={`Cor de ${titulo}`}
          value={cor || COR_PADRAO[tom]}
          onChange={(e) => {
            if (corImediata) onCor(e.target.value);
          }}
          onBlur={(e) => {
            if (!corImediata) onCor(e.target.value);
          }}
          className={swatch}
        />
      )}
      {editando ? (
        <>
          <span id={tituloId} className="sr-only">
            {titulo}
          </span>
          <form
            className={
              vertical
                ? "absolute z-20 flex w-48 max-w-[min(16rem,calc(100vw-2rem))] flex-col gap-1 rounded-md border border-surface-border bg-surface p-2 shadow-md max-md:left-0 max-md:top-full max-md:mt-2 md:left-full md:top-0 md:ml-2"
                : "flex min-w-0 flex-wrap items-center gap-1"
            }
            onSubmit={(e) => {
              e.preventDefault();
              const nome = rascunho.trim();
              if (nome) onRenomear?.(nome);
              setEditando(false);
            }}
          >
            <label className="sr-only" htmlFor={`${tituloId}-nome`}>
              Nome de {titulo}
            </label>
            <input
              id={`${tituloId}-nome`}
              value={rascunho}
              onChange={(e) => setRascunho(e.target.value)}
              maxLength={80}
              className="h-8 rounded-md border border-surface-border bg-surface px-2 text-sm"
            />
            <button type="submit" className="text-xs font-medium text-primary hover:underline">
              Salvar nome
            </button>
          </form>
        </>
      ) : (
        <h2
          id={tituloId}
          className={`min-w-0 text-sm font-semibold ${vertical ? "[writing-mode:vertical-lr]" : "break-words"}`}
        >
          {onRenomear ? (
            <button
              type="button"
              className="max-w-full break-words text-left hover:underline"
              onClick={comecar}
            >
              {titulo}
            </button>
          ) : (
            titulo
          )}
        </h2>
      )}
      <span className="text-xs font-normal text-muted">{total}</span>
      {!vertical && extra && (
        <div className="ml-auto flex flex-wrap items-center gap-2">{extra}</div>
      )}
    </div>
  );
}

/** O nome visível do card. O clique edita o nome; o endereço continua no
 * link para quem abre em outra aba. */
export function NomeClicavel({
  nome,
  rotulo,
  href,
  onSalvar,
  className,
}: {
  nome: string;
  /** Texto mostrado. Sem isto, mostra o nome. */
  rotulo?: string;
  href?: string;
  onSalvar: (nome: string) => void;
  className?: string;
}) {
  const campo = useId();
  const [editando, setEditando] = useState(false);
  const [rascunho, setRascunho] = useState(nome);
  const classe =
    className ?? "line-clamp-2 text-left text-sm font-medium text-primary hover:underline";
  if (editando) {
    return (
      <form
        className="flex flex-wrap items-center gap-1"
        onClick={(e) => e.stopPropagation()}
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          const proximo = rascunho.trim();
          if (proximo) onSalvar(proximo);
          setEditando(false);
        }}
      >
        <label className="sr-only" htmlFor={campo}>
          Nome do card {nome}
        </label>
        <input
          id={campo}
          value={rascunho}
          onChange={(e) => setRascunho(e.target.value)}
          maxLength={160}
          className="h-8 min-w-0 flex-1 rounded-md border border-surface-border bg-surface px-2 text-sm"
        />
        <button type="submit" className="text-xs font-medium text-primary hover:underline">
          Salvar nome
        </button>
      </form>
    );
  }
  function abrir(e: { preventDefault: () => void; stopPropagation: () => void }) {
    e.preventDefault();
    e.stopPropagation();
    setRascunho(nome);
    setEditando(true);
  }
  if (href) {
    return (
      <a href={href} className={classe} onClick={abrir}>
        {rotulo || nome}
      </a>
    );
  }
  return (
    <button type="button" className={classe} onClick={abrir}>
      {rotulo || nome}
    </button>
  );
}

function NovoCard({
  colunaId,
  titulo,
  onCriar,
}: {
  colunaId: string;
  titulo: string;
  onCriar: (nome: string) => void;
}) {
  const [nome, setNome] = useState("");
  return (
    <form
      className="flex flex-col gap-1"
      onSubmit={(e) => {
        e.preventDefault();
        const proximo = nome.trim();
        if (!proximo) return;
        onCriar(proximo);
        setNome("");
      }}
    >
      <label className="sr-only" htmlFor={`novo-card-${colunaId}`}>
        Nome do novo card em {titulo}
      </label>
      <input
        id={`novo-card-${colunaId}`}
        value={nome}
        onChange={(e) => setNome(e.target.value)}
        placeholder="Nome do card"
        maxLength={160}
        className="h-8 w-full min-w-0 rounded-md border border-surface-border bg-surface px-2 text-xs"
      />
      <button
        type="submit"
        className="h-8 rounded-md border border-surface-border text-xs font-medium text-primary"
      >
        Criar card
      </button>
    </form>
  );
}

export interface VistaCard {
  titulo: string;
  cor?: string;
  salvarNome: (nome: string) => void;
}

interface FaixasProps<T> {
  rotulo: string;
  memoria: string;
  /** Onde o nome e a cor ficam gravados. Sem isto, usa a mesma chave do recolher. */
  aparencia?: string;
  colunas: ColunaFaixa[];
  itens: T[];
  colunaDe: (item: T) => string | null;
  idDe: (item: T) => string;
  nomeDe?: (item: T) => string;
  render: (item: T, vista: VistaCard) => ReactNode;
  /** Botão de criar, desenhado no cabeçalho da coluna — também quando ela está vazia. */
  extra?: (colunaId: string) => ReactNode;
  /** Cria um card com o nome digitado na coluna. */
  aoCriar?: (colunaId: string, nome: string) => void;
  /** Grava o nome oficial do card. Sem isto, o nome fica só neste navegador. */
  aoRenomear?: (item: T, nome: string) => void;
  /** Abre o card para edição. O nome e a cor não disparam isto. */
  aoAbrir?: (item: T) => void;
  testeId?: string;
  /** Nome, cor e ordem vêm das colunas. O navegador só guarda o recolher e o card. */
  oficial?: boolean;
}

/** Lista em faixas na largura da tela. Cada estado aparece, mesmo sem item,
 * e pode ser recolhido numa barra colorida. */
export function ListaFaixas<T>({
  rotulo,
  memoria,
  aparencia,
  colunas,
  itens,
  colunaDe,
  idDe,
  nomeDe,
  render,
  extra,
  aoCriar,
  aoRenomear,
  aoAbrir,
  testeId,
  oficial = false,
}: FaixasProps<T>) {
  const { recolhidas, alternar } = useFaixasRecolhidas(memoria);
  const look = useAparencia(aparencia ?? memoria);
  const faixas = ordenarColunas(colunas);
  return (
    <div
      className="flex min-w-0 max-w-full flex-col gap-2"
      role="list"
      aria-label={rotulo}
      data-testid={testeId}
    >
      {faixas.map((coluna) => {
        const cartoes = itens.filter((item) => colunaDe(item) === coluna.id);
        const recolhida = recolhidas.includes(coluna.id);
        const tituloId = `faixa-${memoria}-${coluna.id}`;
        const titulo = oficial
          ? coluna.titulo
          : look.gravado.colunas[coluna.id]?.titulo || coluna.titulo;
        const corColuna = oficial ? coluna.cor : look.gravado.colunas[coluna.id]?.cor;
        return (
          <section
            key={coluna.id}
            role="listitem"
            aria-labelledby={tituloId}
            data-coluna={coluna.id}
            data-ordem={coluna.ordem}
            className="w-full"
          >
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
              extra={extra?.(coluna.id)}
            />
            {!recolhida && (
              <div className="flex flex-col gap-2 p-2">
                {cartoes.length === 0 && (
                  <p className="px-1 py-2 text-sm text-muted">Nenhum item</p>
                )}
                {cartoes.length > 0 && (
                  <ul className="flex flex-col">
                    {cartoes.map((item) => {
                      const id = idDe(item);
                      const nome = look.gravado.cartoes[id]?.titulo || nomeDe?.(item) || "";
                      const cor = look.gravado.cartoes[id]?.cor || corColuna;
                      const faixa = pintura(cor, coluna.tom, "cartao");
                      return (
                        <li
                          key={id}
                          onClick={(e) => cliqueAbre(e, aoAbrir ? () => aoAbrir(item) : undefined)}
                          className={`min-w-0 border-b border-surface-border border-l-4 bg-surface px-3 py-3 last:border-b-0 ${aoAbrir ? "cursor-pointer" : ""} ${faixa.className}`}
                          style={faixa.style}
                        >
                          <CorDoCard
                            nome={nome || titulo}
                            tom={coluna.tom}
                            cor={cor}
                            onCor={(valor) => look.salvarCartao(id, { cor: valor })}
                          />
                          {render(item, {
                            titulo: nome,
                            cor,
                            salvarNome: (valor) => {
                              if (aoRenomear) aoRenomear(item, valor);
                              else look.salvarCartao(id, { titulo: valor });
                            },
                          })}
                        </li>
                      );
                    })}
                  </ul>
                )}
                {aoCriar && (
                  <NovoCard
                    colunaId={coluna.id}
                    titulo={titulo}
                    onCriar={(nome) => aoCriar(coluna.id, nome)}
                  />
                )}
              </div>
            )}
          </section>
        );
      })}
    </div>
  );
}

function CorDoCard({
  nome,
  tom,
  cor,
  onCor,
}: {
  nome: string;
  tom: TomFaixa;
  cor?: string;
  onCor: (cor: string) => void;
}) {
  return (
    <input
      type="color"
      aria-label={`Cor do card ${nome}`}
      value={cor || COR_PADRAO[tom]}
      onClick={(e) => e.stopPropagation()}
      onChange={(e) => onCor(e.target.value)}
      className={swatch}
    />
  );
}

function cliqueAbre(e: { target: EventTarget | null }, aoAbrir?: () => void) {
  if (!aoAbrir) return;
  const alvo = e.target as HTMLElement | null;
  if (alvo?.closest("a, button, input, select, textarea, label")) return;
  aoAbrir();
}

/** Quadro: as mesmas faixas, lado a lado. No celular o quadro rola para o
 * lado, uma coluna por vez; numa tela larga as colunas dividem a largura.
 * Recolhida, a coluna vira uma barra vertical. A coluna vazia continua no quadro. */
export function Quadro<T>({
  rotulo,
  memoria,
  aparencia,
  colunas,
  itens,
  colunaDe,
  idDe,
  nomeDe,
  render,
  extra,
  aoCriar,
  aoRenomear,
  aoAbrir,
  testeId,
  oficial = false,
  pending = false,
  arrastavel,
  podeSoltar,
  aoSoltar,
  moldura,
}: FaixasProps<T> & {
  pending?: boolean;
  arrastavel?: (item: T) => boolean;
  podeSoltar?: (item: T, colunaId: string) => boolean;
  aoSoltar?: (item: T, colunaId: string) => void;
  /** Classes extras do cartão (o semáforo do prazo, por exemplo). */
  moldura?: (item: T) => string | undefined;
}) {
  const { recolhidas, alternar } = useFaixasRecolhidas(memoria);
  const look = useAparencia(aparencia ?? memoria);
  const faixas = ordenarColunas(colunas);
  const [arrastado, setArrastado] = useState<T | null>(null);
  const [sobre, setSobre] = useState<string | null>(null);
  const move = arrastavel !== undefined && aoSoltar !== undefined;

  function soltar(e: DragEvent, colunaId: string) {
    e.preventDefault();
    setSobre(null);
    if (arrastado && podeSoltar?.(arrastado, colunaId)) aoSoltar?.(arrastado, colunaId);
    setArrastado(null);
  }

  return (
    <div
      className="flex max-w-full gap-3 overflow-x-auto pb-2 max-md:snap-x max-md:snap-mandatory md:flex-wrap md:overflow-visible"
      role="list"
      aria-label={rotulo}
      data-testid={testeId}
    >
      {faixas.map((coluna) => {
        const cartoes = itens.filter((item) => colunaDe(item) === coluna.id);
        const recolhida = recolhidas.includes(coluna.id);
        const alvo = arrastado !== null && (podeSoltar?.(arrastado, coluna.id) ?? false);
        const tituloId = `coluna-${memoria}-${coluna.id}`;
        const titulo = oficial
          ? coluna.titulo
          : look.gravado.colunas[coluna.id]?.titulo || coluna.titulo;
        const corColuna = oficial ? coluna.cor : look.gravado.colunas[coluna.id]?.cor;
        return (
          <section
            key={coluna.id}
            role="listitem"
            aria-labelledby={tituloId}
            data-coluna={coluna.id}
            data-ordem={coluna.ordem}
            onDragOver={(e) => {
              if (!alvo) return;
              e.preventDefault();
              setSobre(coluna.id);
            }}
            onDragLeave={() => setSobre((atual) => (atual === coluna.id ? null : atual))}
            onDrop={(e) => soltar(e, coluna.id)}
            className={`relative flex flex-col gap-2 rounded-xl border p-2 ${
              recolhida
                ? "w-12 shrink-0 snap-start self-stretch"
                : "min-h-40 shrink-0 snap-start max-md:w-[min(85vw,20rem)] md:min-w-56 md:flex-1 md:basis-56"
            } ${
              sobre === coluna.id
                ? "border-primary bg-primary/5"
                : alvo
                  ? "border-dashed border-primary/60 bg-surface"
                  : "border-surface-border bg-black/[0.02] dark:bg-white/[0.03]"
            }`}
          >
            <BarraFaixa
              titulo={titulo}
              tituloId={tituloId}
              total={cartoes.length}
              tom={coluna.tom}
              cor={corColuna}
              recolhida={recolhida}
              vertical={recolhida}
              ordem={coluna.ordem}
              corImediata={!oficial}
              onAlternar={() => alternar(coluna.id)}
              onRenomear={
                oficial ? undefined : (nome) => look.salvarColuna(coluna.id, { titulo: nome })
              }
              onCor={oficial ? undefined : (cor) => look.salvarColuna(coluna.id, { cor })}
              extra={!recolhida ? extra?.(coluna.id) : undefined}
            />
            {!recolhida && (
              <>
                {cartoes.length === 0 && (
                  <p className="px-2 py-4 text-center text-xs text-muted">Nenhum item</p>
                )}
                {cartoes.length > 0 && (
                  <ul className="flex flex-col gap-2">
                    {cartoes.map((item) => {
                      const id = idDe(item);
                      const nome = look.gravado.cartoes[id]?.titulo || nomeDe?.(item) || "";
                      const cor = look.gravado.cartoes[id]?.cor || corColuna;
                      const faixa = pintura(cor, coluna.tom, "cartao");
                      return (
                        <li
                          key={id}
                          draggable={move && !pending && !!arrastavel?.(item)}
                          onDragStart={(e) => {
                            e.dataTransfer.effectAllowed = "move";
                            setArrastado(item);
                          }}
                          onDragEnd={() => {
                            setArrastado(null);
                            setSobre(null);
                          }}
                          onClick={(e) => cliqueAbre(e, aoAbrir ? () => aoAbrir(item) : undefined)}
                          className={`relative flex min-w-0 flex-col gap-1.5 rounded-lg border border-surface-border border-l-4 bg-surface p-3 shadow-sm ${
                            aoAbrir ? "cursor-pointer" : ""
                          } ${cor ? faixa.className : (moldura?.(item) ?? faixa.className)}`}
                          style={cor ? faixa.style : undefined}
                        >
                          <CorDoCard
                            nome={nome || titulo}
                            tom={coluna.tom}
                            cor={cor}
                            onCor={(valor) => look.salvarCartao(id, { cor: valor })}
                          />
                          {render(item, {
                            titulo: nome,
                            cor,
                            salvarNome: (valor) => {
                              if (aoRenomear) aoRenomear(item, valor);
                              else look.salvarCartao(id, { titulo: valor });
                            },
                          })}
                        </li>
                      );
                    })}
                  </ul>
                )}
                {aoCriar && (
                  <NovoCard
                    colunaId={coluna.id}
                    titulo={titulo}
                    onCriar={(nome) => aoCriar(coluna.id, nome)}
                  />
                )}
              </>
            )}
          </section>
        );
      })}
    </div>
  );
}
