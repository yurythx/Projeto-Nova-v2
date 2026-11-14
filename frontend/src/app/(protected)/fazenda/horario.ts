/** Horário legal do protocolo: Cuiabá, UTC−4, sem horário de verão. */
export function emCuiaba(iso?: string | null): string {
  if (!iso) return "";
  const data = new Date(iso);
  if (Number.isNaN(data.getTime()) || data.getUTCFullYear() <= 1) return "";
  return data.toLocaleString("pt-BR", {
    timeZone: "America/Cuiaba",
    dateStyle: "short",
    timeStyle: "short",
  });
}
