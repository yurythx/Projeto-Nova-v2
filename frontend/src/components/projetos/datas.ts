/** Dia civil em America/Sao_Paulo. Data só com o dia não atravessa a meia-noite UTC. */
export function diaBrasil(iso?: string | null): string {
  if (!iso) return "";
  const dia = iso.slice(0, 10);
  if (iso.length === 10 && /^\d{4}-\d{2}-\d{2}$/.test(dia)) {
    const [ano, mes, d] = dia.split("-");
    return `${d}/${mes}/${ano}`;
  }
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return "";
  return data.toLocaleDateString("pt-BR", { timeZone: "America/Sao_Paulo" });
}
