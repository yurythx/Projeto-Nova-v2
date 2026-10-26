"use client";

import { getSession, useSession } from "next-auth/react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/Button";
import { redirectToLogin } from "@/lib/auth/sessionExpired";
import { IDLE_MS, RENEW_WINDOW_MS, WARN_MS } from "@/lib/auth/sessionTiming";

// Mantém a sessão viva ENQUANTO o usuário usa o sistema e só a deixa
// expirar por inatividade de verdade (antes, o token do login local
// vencia 1h depois do login, em uso ou não; o do Keycloak, em 15 min se
// nada lesse /api/auth/session).
//
// - Uso = clique, tecla, rolagem ou toque (não basta mover o mouse).
// - Faltando menos de RENEW_WINDOW_MS para o token vencer e com uso nos
//   últimos IDLE_MS, chama update() do next-auth: o callback `jwt`
//   (lib/auth/options.ts) renova o token e o cookie é regravado.
// - Sem uso, a WARN_MS do fim aparece um aviso com contagem regressiva e
//   o botão "Continuar conectado" (WCAG 2.2.1 — tempo ajustável).
// - No fim, confere a sessão de novo (outra aba pode ter renovado) e só
//   então volta ao login, já com o retorno para esta tela.
const RETRY_MS = 30_000;
const ACTIVITY_EVENTS = ["pointerdown", "keydown", "wheel", "touchstart"] as const;

function mmss(ms: number): string {
  const s = Math.max(0, Math.ceil(ms / 1000));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

export function SessionKeepAlive() {
  const { data: session, update } = useSession();
  const expiresAt = session?.expiresAt;
  const lastActivity = useRef(0);
  const lastAttempt = useRef(0);
  const renewing = useRef(false);
  const [now, setNow] = useState(() => Date.now());

  // Abrir/recarregar a tela conta como uso. Declarado antes dos outros
  // efeitos, roda antes deles na montagem.
  useEffect(() => {
    lastActivity.current = Date.now();
  }, []);

  const renew = useCallback(
    async (force = false) => {
      if (!expiresAt || renewing.current) return;
      const t = Date.now();
      const left = expiresAt - t;
      if (left <= 0 || left > RENEW_WINDOW_MS) return;
      if (!force && (t - lastActivity.current > IDLE_MS || t - lastAttempt.current < RETRY_MS))
        return;
      renewing.current = true;
      lastAttempt.current = t;
      try {
        await update();
      } finally {
        renewing.current = false;
      }
    },
    [expiresAt, update],
  );

  useEffect(() => {
    const onActivity = () => {
      lastActivity.current = Date.now();
      void renew();
    };
    for (const e of ACTIVITY_EVENTS)
      window.addEventListener(e, onActivity, { passive: true, capture: true });
    return () => {
      for (const e of ACTIVITY_EVENTS) window.removeEventListener(e, onActivity, { capture: true });
    };
  }, [renew]);

  // Relógio: a cada 15 s normalmente; a cada 1 s durante o aviso.
  const left = expiresAt ? expiresAt - now : Infinity;
  const warning = left <= WARN_MS;
  const expired = left <= 0;
  useEffect(() => {
    if (!expiresAt || expired) return;
    const id = window.setInterval(() => setNow(Date.now()), warning ? 1000 : 15_000);
    return () => window.clearInterval(id);
  }, [expiresAt, warning, expired]);

  useEffect(() => {
    if (!expiresAt) return;
    if (left > 0) {
      void renew();
      return;
    }
    let cancelled = false;
    void getSession().then((s) => {
      if (!cancelled && !(s?.expiresAt && s.expiresAt > Date.now())) redirectToLogin();
    });
    return () => {
      cancelled = true;
    };
  }, [expiresAt, left, renew]);

  if (!expiresAt || !warning || left <= 0) return null;
  return (
    <div
      role="alertdialog"
      aria-labelledby="sessao-expirando-titulo"
      aria-describedby="sessao-expirando-texto"
      className="fixed bottom-4 left-1/2 z-50 w-[min(28rem,calc(100%-2rem))] -translate-x-1/2 rounded-lg border border-warning/30 bg-surface p-4 shadow-lg"
    >
      <p id="sessao-expirando-titulo" className="text-sm font-semibold text-foreground">
        Sua sessão vai expirar por inatividade
      </p>
      <p id="sessao-expirando-texto" className="mt-1 text-sm text-muted">
        Faltam <span className="font-mono tabular-nums">{mmss(left)}</span>. Depois disso será
        preciso entrar de novo.
      </p>
      <div className="mt-3 flex justify-end">
        <Button
          size="sm"
          onClick={() => {
            lastActivity.current = Date.now();
            void renew(true);
          }}
        >
          Continuar conectado
        </Button>
      </div>
    </div>
  );
}
