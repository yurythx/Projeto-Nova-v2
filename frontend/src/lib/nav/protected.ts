/** Prefixos autenticados — o grupo `(protected)` não aparece na URL.
 * Uma lista só: o proxy redireciona visitas anônimas; o apiClient redireciona
 * 401 de sessão. Módulos desativados entram mesmo assim: o 404 é do Kernel. */
export const PROTECTED_PREFIXES = [
  "/dashboard",
  "/configuracao",
  "/monitoramento",
  "/auditoria",
  "/perfil",
  "/busca",
  "/mercurio",
  "/wiki",
  "/agenda",
  "/diretorio",
  "/arquivos",
  "/signum",
  "/tramite",
  "/gestao",
  "/exemplos",
  "/produtos",
  "/atendimento",
  "/ativos",
  "/problemas",
  "/mudancas",
  "/projetos",
  "/assistencia",
  "/notificacoes",
] as const;

export function isProtectedPath(pathname: string): boolean {
  return PROTECTED_PREFIXES.some(
    (prefix) => pathname === prefix || pathname.startsWith(prefix + "/"),
  );
}
