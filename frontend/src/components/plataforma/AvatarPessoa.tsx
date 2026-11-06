"use client";

import { useEffect, useState } from "react";

const EVENTO = "nova-avatar";

const CORES = [
  "bg-sky-700",
  "bg-emerald-700",
  "bg-amber-700",
  "bg-rose-700",
  "bg-violet-700",
  "bg-teal-700",
] as const;

/** Avisa as listas e os quadros que a foto desta pessoa mudou. */
export function avisarAvatar(id: string) {
  window.dispatchEvent(new CustomEvent(EVENTO, { detail: id }));
}

function iniciaisDe(nome: string): string {
  const partes = nome
    .replace(/@.*/, "")
    .split(/[.\s_-]+/)
    .filter(Boolean);
  const letras =
    partes.length >= 2 ? [partes[0]?.[0] ?? "?", partes[1]?.[0] ?? "?"] : [nome[0] ?? "?"];
  return letras.join("").toUpperCase();
}

function corDe(nome: string): string {
  let n = 0;
  for (const c of nome) n += c.charCodeAt(0);
  return CORES[n % CORES.length] ?? "bg-sky-700";
}

/** Círculo da pessoa: a foto do perfil, ou as iniciais quando ela ainda não enviou. */
export function AvatarPessoa({
  id,
  nome,
  tamanho = "sm",
}: {
  id?: string | null;
  nome?: string;
  tamanho?: "sm" | "md";
}) {
  const rotulo = nome?.trim() || "";
  const [versao, setVersao] = useState(0);
  const [foto, setFoto] = useState(true);
  useEffect(() => {
    function ouvir(e: Event) {
      if ((e as CustomEvent<string>).detail === id) {
        setFoto(true);
        setVersao(Date.now());
      }
    }
    window.addEventListener(EVENTO, ouvir);
    return () => window.removeEventListener(EVENTO, ouvir);
  }, [id]);
  const px = tamanho === "md" ? "h-10 w-10 text-xs" : "h-7 w-7 text-[10px]";
  if (!rotulo && !id) {
    return (
      <span
        className={`inline-flex ${px} shrink-0 rounded-full border border-dashed border-surface-border`}
        aria-hidden="true"
      />
    );
  }
  const visivel = rotulo || "Pessoa";
  return (
    <span
      title={visivel}
      className={`relative inline-flex ${px} shrink-0 items-center justify-center overflow-hidden rounded-full font-semibold text-white ${corDe(visivel)}`}
    >
      <span aria-hidden="true">{iniciaisDe(visivel)}</span>
      {id && foto && (
        <img
          src={`/api/backend/v1/directory/avatars/${id}${versao ? `?v=${versao}` : ""}`}
          alt=""
          className="absolute inset-0 h-full w-full object-cover"
          onError={() => setFoto(false)}
        />
      )}
    </span>
  );
}

/** Nome da pessoa como no quadro: o círculo e o nome ao lado. */
export function PessoaComAvatar({
  id,
  nome,
  vazio = "Sem responsável",
  className = "",
}: {
  id?: string | null;
  nome?: string;
  vazio?: string;
  className?: string;
}) {
  const rotulo = nome?.trim() || vazio;
  return (
    <span className={`inline-flex min-w-0 items-center gap-2 ${className}`}>
      <AvatarPessoa id={nome?.trim() ? id : undefined} nome={nome?.trim() ? rotulo : ""} />
      <span className="truncate">{rotulo}</span>
    </span>
  );
}
