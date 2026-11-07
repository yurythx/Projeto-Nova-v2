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

/** Dia civil YYYY-MM-DD em America/Sao_Paulo, para o campo de data. */
export function diaISO(iso?: string | null): string {
  if (!iso) return "";
  const dia = iso.slice(0, 10);
  if (/^\d{4}-\d{2}-\d{2}$/.test(dia) && (iso.length === 10 || iso[10] === "T")) return dia;
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return "";
  return data.toLocaleDateString("en-CA", { timeZone: "America/Sao_Paulo" });
}

/** Prazo gravado no fim do expediente de São Paulo. */
export function prazoDoDia(dia: string): string | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(dia)) return null;
  return `${dia}T18:00:00-03:00`;
}

/** Semana corrente de segunda a domingo, no calendário de São Paulo. */
export function faixaDaSemana(agora = new Date()): { inicio: string; fim: string; hoje: string } {
  const hoje = agora.toLocaleDateString("en-CA", { timeZone: "America/Sao_Paulo" });
  const partes = hoje.split("-").map(Number);
  const ano = partes[0] ?? 1970;
  const mes = partes[1] ?? 1;
  const dia = partes[2] ?? 1;
  const base = new Date(Date.UTC(ano, mes - 1, dia));
  const semana = base.getUTCDay();
  const ateSegunda = semana === 0 ? -6 : 1 - semana;
  const inicio = new Date(base);
  inicio.setUTCDate(base.getUTCDate() + ateSegunda);
  const fim = new Date(inicio);
  fim.setUTCDate(inicio.getUTCDate() + 6);
  const iso = (data: Date) => data.toISOString().slice(0, 10);
  return { inicio: iso(inicio), fim: iso(fim), hoje };
}

/** Mês civil YYYY-MM em America/Sao_Paulo. */
export function mesISO(agora = new Date()): string {
  return agora.toLocaleDateString("en-CA", { timeZone: "America/Sao_Paulo" }).slice(0, 7);
}

/** Anda meses civis sem depender do fuso da máquina. */
export function deslocarMes(mes: string, delta: number): string {
  const partes = mes.split("-").map(Number);
  const ano = partes[0] ?? 1970;
  const m = partes[1] ?? 1;
  const base = new Date(Date.UTC(ano, m - 1 + delta, 1));
  const mm = String(base.getUTCMonth() + 1).padStart(2, "0");
  return `${base.getUTCFullYear()}-${mm}`;
}

export function nomeDoMes(mes: string): string {
  const partes = mes.split("-").map(Number);
  const ano = partes[0] ?? 1970;
  const m = partes[1] ?? 1;
  return new Date(Date.UTC(ano, m - 1, 1)).toLocaleDateString("pt-BR", {
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  });
}

/** Grade de segunda a domingo, com os dias vizinhos que completam a semana. */
export function gradeDoMes(mes: string): { dia: string; nesteMes: boolean }[] {
  const partes = mes.split("-").map(Number);
  const ano = partes[0] ?? 1970;
  const m = partes[1] ?? 1;
  const primeiro = new Date(Date.UTC(ano, m - 1, 1));
  const semana = primeiro.getUTCDay();
  const ateSegunda = semana === 0 ? -6 : 1 - semana;
  const inicio = new Date(primeiro);
  inicio.setUTCDate(primeiro.getUTCDate() + ateSegunda);
  const dias: { dia: string; nesteMes: boolean }[] = [];
  for (let i = 0; i < 42; i++) {
    const data = new Date(inicio);
    data.setUTCDate(inicio.getUTCDate() + i);
    const dia = data.toISOString().slice(0, 10);
    dias.push({ dia, nesteMes: dia.slice(0, 7) === mes });
  }
  const ultima = dias.slice(35);
  if (ultima.every((celula) => !celula.nesteMes)) return dias.slice(0, 35);
  return dias;
}

export type FaixaPrazo = "atrasada" | "semana" | "depois" | "sem";

export function faixaDoPrazo(iso?: string | null, agora = new Date()): FaixaPrazo {
  const dia = diaISO(iso);
  if (!dia) return "sem";
  const { fim, hoje } = faixaDaSemana(agora);
  if (dia < hoje) return "atrasada";
  if (dia <= fim) return "semana";
  return "depois";
}
