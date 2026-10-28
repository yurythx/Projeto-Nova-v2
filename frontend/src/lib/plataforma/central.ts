import type { Me } from "@/lib/plataforma/types";

/** Permissão de quem só pede atendimento (ADR 078). */
export const PERM_ABRIR = "atendimento:abrir";

/** Caminhos que a central do solicitante oferece; o resto volta para ela. */
const ROTAS_DA_CENTRAL = ["/atendimento", "/wiki", "/perfil", "/notificacoes"];

/**
 * Quem só pede: abre atendimento e não tem nenhuma outra permissão nem
 * papel de administração. Essa pessoa entra na central, sem o menu do
 * sistema. A API já recusa o resto; isto só não oferece o que ela não usa.
 */
export function soSolicitante(me: Me | undefined): boolean {
  if (!me || me.roles?.includes("admin")) return false;
  const perms = me.permissions ?? [];
  return perms.includes(PERM_ABRIR) && perms.every((p) => p === PERM_ABRIR);
}

/** Na central, o caminho é um dos que ela oferece (menos o trabalho da equipe). */
export function rotaDaCentral(pathname: string): boolean {
  if (/^\/atendimento\/(fila|cadastro|regras|prazos|relatorio|campo)(\/|$)/.test(pathname))
    return false;
  return ROTAS_DA_CENTRAL.some((r) => pathname === r || pathname.startsWith(r + "/"));
}
