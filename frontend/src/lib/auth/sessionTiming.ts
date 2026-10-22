// Prazos da sessão por inatividade, compartilhados pelo callback `jwt`
// (lib/auth/options.ts, servidor) e pelo SessionKeepAlive (navegador).

/** Faltando menos que isto para o access token vencer, um update() da
 * sessão renova o token. */
export const RENEW_WINDOW_MS = 10 * 60_000;

/** Sem clique/tecla/rolagem/toque há mais que isto, o usuário está inativo
 * e a sessão não é renovada. */
export const IDLE_MS = 10 * 60_000;

/** Antecedência do aviso "sua sessão vai expirar". */
export const WARN_MS = 2 * 60_000;
