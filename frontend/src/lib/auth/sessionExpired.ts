// Leva o usuário de volta ao login quando a sessão acaba, voltando depois
// para a mesma tela (callbackUrl). Navegação DURA de propósito
// (location.replace, não router.push): descarta todo o estado de cliente
// da tela expirada e reexecuta o middleware; replace() não deixa a tela
// morta no histórico do back. reason=session_expired só é carimbado para
// quem tinha sessão (ver o aviso em LoginCard.tsx).
export function redirectToLogin(): void {
  const target = new URL("/login", window.location.origin);
  target.searchParams.set("callbackUrl", window.location.pathname + window.location.search);
  target.searchParams.set("reason", "session_expired");
  window.location.replace(target.toString());
}
