"use client";

// Peças visuais comuns às telas do protocolo, no desenho do Protocolo Digital
// original (ServicesManagement, FormsManagement e ProtocolosManagement).
// Cada classe clara tem a sua par escura: fundo slate-900/800, borda
// slate-600/700 e texto slate-100/400, com contraste AA sobre o fundo.

import type { LucideIcon } from "lucide-react";
import { ChevronDown, ChevronUp, FolderPlus, Search } from "lucide-react";
import type { ReactNode } from "react";

export const CAMPO =
  "w-full rounded-lg border border-gray-300 bg-surface px-4 py-3 text-sm text-foreground transition-colors placeholder:text-gray-400 focus:border-blue-500 focus:outline-none focus:ring-[3px] focus:ring-blue-500/10 disabled:opacity-60 dark:border-slate-600 dark:bg-slate-900/60 dark:placeholder:text-slate-500 dark:focus:border-sky-400 dark:focus:ring-sky-400/20";

export const ROTULO = "mb-2 block text-sm font-medium text-gray-700 dark:text-slate-300";

export const BOTAO_PRIMARIO =
  "inline-flex items-center justify-center gap-2 rounded-lg bg-blue-500 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-blue-600 dark:hover:bg-blue-500";

export const BOTAO_SECUNDARIO =
  "inline-flex items-center justify-center gap-2 rounded-lg border border-gray-300 bg-surface px-4 py-2.5 text-sm font-medium text-gray-700 transition-colors hover:border-gray-400 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-slate-600 dark:bg-slate-800 dark:text-slate-200 dark:hover:border-slate-500 dark:hover:bg-slate-700";

export const BOTAO_PERIGO =
  "inline-flex items-center justify-center gap-2 rounded-lg bg-red-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50";

export const CARTAO =
  "rounded-xl border border-gray-200 bg-surface shadow-[0_1px_3px_rgba(0,0,0,0.1)] dark:border-slate-700 dark:shadow-[0_1px_3px_rgba(0,0,0,0.4)]";

export const TEXTO_FORTE = "text-gray-800 dark:text-slate-100";
export const TEXTO_FRACO = "text-gray-500 dark:text-slate-400";

/** Título da tela com ícone azul, como o "Gerenciamento de Serviços" original. */
export function CabecalhoDaTela({
  icone: Icone,
  titulo,
  descricao,
  children,
}: {
  icone: LucideIcon;
  titulo: string;
  descricao?: string;
  children?: ReactNode;
}) {
  return (
    <div className="mb-8 flex flex-wrap items-center justify-between gap-4">
      <div className="flex min-w-0 items-center gap-3">
        <Icone className="h-8 w-8 shrink-0 text-blue-600 dark:text-sky-400" aria-hidden />
        <div className="min-w-0">
          <h1 className={`text-3xl font-bold ${TEXTO_FORTE}`}>{titulo}</h1>
          {descricao && <p className={`mt-1 text-sm ${TEXTO_FRACO}`}>{descricao}</p>}
        </div>
      </div>
      {children && <div className="flex flex-wrap items-center gap-2">{children}</div>}
    </div>
  );
}

/** Grupo de pílulas (Todos, Ativos, Inativos). */
export function Pilulas<T extends string>({
  rotulo,
  valor,
  opcoes,
  onChange,
}: {
  rotulo: string;
  valor: T;
  opcoes: { value: T; label: string; total?: number }[];
  onChange: (valor: T) => void;
}) {
  return (
    <div
      className="flex gap-1 rounded-lg bg-slate-100 p-1 dark:bg-slate-900/70"
      role="group"
      aria-label={rotulo}
    >
      {opcoes.map((opcao) => {
        const ativa = opcao.value === valor;
        return (
          <button
            key={opcao.value}
            type="button"
            aria-pressed={ativa}
            onClick={() => onChange(opcao.value)}
            className={`flex items-center gap-1.5 rounded-md px-4 py-2 text-sm font-medium transition-all ${
              ativa
                ? "bg-white text-blue-600 shadow-[0_1px_3px_rgba(0,0,0,0.1)] dark:bg-slate-700 dark:text-sky-300"
                : "text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
            }`}
          >
            {opcao.label}
            {opcao.total !== undefined && (
              <span
                className={`rounded-full px-1.5 text-[0.7rem] font-semibold ${
                  ativa
                    ? "bg-blue-100 text-blue-700 dark:bg-sky-400/20 dark:text-sky-200"
                    : "bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-300"
                }`}
              >
                {opcao.total}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}

/** Campo de busca com a lupa à esquerda. */
export function Busca({
  valor,
  onChange,
  placeholder,
}: {
  valor: string;
  onChange: (valor: string) => void;
  placeholder: string;
}) {
  return (
    <div className="relative min-w-[220px] flex-1">
      <Search
        className="absolute left-3 top-1/2 h-[18px] w-[18px] -translate-y-1/2 text-gray-500 dark:text-slate-400"
        aria-hidden
      />
      <input
        type="search"
        aria-label="Pesquisar"
        placeholder={placeholder}
        value={valor}
        onChange={(e) => onChange(e.target.value)}
        className={`${CAMPO} pl-10`}
      />
    </div>
  );
}

/** Estado vazio com a pasta cinza do original. */
export function Vazio({ titulo, texto }: { titulo: string; texto: string }) {
  return (
    <div className={`${CARTAO} flex flex-col items-center gap-2 px-6 py-14 text-center`}>
      <FolderPlus className="h-12 w-12 text-gray-400 dark:text-slate-500" aria-hidden />
      <h2 className={`text-lg font-semibold ${TEXTO_FORTE}`}>{titulo}</h2>
      <p className={`text-sm ${TEXTO_FRACO}`}>{texto}</p>
    </div>
  );
}

/** Selo Ativo/Inativo. */
export function SeloAtivo({ ativo }: { ativo: boolean }) {
  return (
    <span
      className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${
        ativo
          ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300"
          : "bg-gray-100 text-gray-600 dark:bg-slate-700 dark:text-slate-300"
      }`}
    >
      {ativo ? "Ativo" : "Inativo"}
    </span>
  );
}

/** Linha que expande, como a ServiceRow original. */
export function LinhaExpansivel({
  titulo,
  selo,
  subtitulo,
  aberta,
  onAlternar,
  acoes,
  children,
}: {
  titulo: string;
  selo?: ReactNode;
  subtitulo: string;
  aberta: boolean;
  onAlternar: () => void;
  acoes?: ReactNode;
  children: ReactNode;
}) {
  return (
    <li className={`${CARTAO} transition-shadow hover:shadow-md`}>
      <div className="flex items-center gap-2 px-6 py-4">
        <button
          type="button"
          aria-expanded={aberta}
          onClick={onAlternar}
          className="min-w-0 flex-1 text-left"
        >
          <span className="flex flex-wrap items-center gap-2">
            <span className={`text-base font-semibold ${TEXTO_FORTE}`}>{titulo}</span>
            {selo}
          </span>
          <span className={`mt-0.5 block text-sm ${TEXTO_FRACO}`}>{subtitulo}</span>
        </button>
        {acoes}
        <button
          type="button"
          aria-label={aberta ? `Recolher ${titulo}` : `Expandir ${titulo}`}
          onClick={onAlternar}
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 dark:text-slate-400 dark:hover:bg-slate-700"
        >
          {aberta ? (
            <ChevronUp className="h-5 w-5" aria-hidden />
          ) : (
            <ChevronDown className="h-5 w-5" aria-hidden />
          )}
        </button>
      </div>
      {aberta && (
        <div className="grid gap-4 rounded-b-xl border-t border-gray-100 bg-slate-50/60 px-6 py-5 sm:grid-cols-2 lg:grid-cols-3 dark:border-slate-700 dark:bg-slate-900/40">
          {children}
        </div>
      )}
    </li>
  );
}

/** Bloco rotulado do detalhe expandido. */
export function Detalhe({ rotulo, children }: { rotulo: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-slate-400">
        {rotulo}
      </span>
      <div className="text-sm text-gray-800 dark:text-slate-200">{children}</div>
    </div>
  );
}

/** Abas sublinhadas (as abas do modal de serviço). */
export function Abas<T extends string>({
  rotulo,
  valor,
  opcoes,
  onChange,
}: {
  rotulo: string;
  valor: T;
  opcoes: { value: T; label: string }[];
  onChange: (valor: T) => void;
}) {
  return (
    <div
      className="mb-5 flex gap-5 overflow-x-auto border-b-2 border-gray-100 dark:border-slate-700"
      role="tablist"
      aria-label={rotulo}
    >
      {opcoes.map((opcao) => {
        const ativa = opcao.value === valor;
        return (
          <button
            key={opcao.value}
            type="button"
            role="tab"
            aria-selected={ativa}
            onClick={() => onChange(opcao.value)}
            className={`relative -bottom-0.5 shrink-0 border-b-[3px] px-1 py-2.5 text-sm transition-colors ${
              ativa
                ? "border-blue-600 font-semibold text-blue-600 dark:border-sky-400 dark:text-sky-300"
                : "border-transparent font-medium text-gray-500 hover:text-blue-600 dark:text-slate-400 dark:hover:text-sky-300"
            }`}
          >
            {opcao.label}
          </button>
        );
      })}
    </div>
  );
}

/** Caixa de seleção com o texto ao lado. */
export function Marcar({
  name,
  texto,
  checked,
  defaultChecked,
  onChange,
}: {
  name?: string;
  texto: string;
  checked?: boolean;
  defaultChecked?: boolean;
  onChange?: (marcado: boolean) => void;
}) {
  return (
    <label className="flex cursor-pointer items-center gap-2.5 rounded-lg border border-gray-200 px-3 py-2.5 text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:border-slate-700 dark:text-slate-200 dark:hover:bg-slate-700/50">
      <input
        type="checkbox"
        name={name}
        checked={checked}
        defaultChecked={defaultChecked}
        onChange={onChange ? (e) => onChange(e.target.checked) : undefined}
        className="h-4 w-4 accent-blue-500"
      />
      {texto}
    </label>
  );
}

/** Rótulo em cima do campo, no estilo do FormGroup original. */
export function Rotulado({
  id,
  rotulo,
  className = "",
  children,
}: {
  id: string;
  rotulo: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={className}>
      <label htmlFor={id} className={ROTULO}>
        {rotulo}
      </label>
      {children}
    </div>
  );
}
