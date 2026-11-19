export const STATUS = [
  { value: "novo", label: "Novo" },
  { value: "em_andamento", label: "Em andamento" },
  { value: "pendente", label: "Pendente" },
  { value: "em_analise", label: "Em análise" },
  { value: "em_complementacao", label: "Em complementação" },
  { value: "pending_approval", label: "Aguardando aprovação" },
  { value: "aprovado", label: "Aprovado" },
  { value: "rejeitado", label: "Rejeitado" },
  { value: "concluido", label: "Concluído" },
  { value: "cancelado", label: "Cancelado" },
];

export const PRIORIDADES = [
  { value: "baixa", label: "Baixa" },
  { value: "normal", label: "Normal" },
  { value: "alta", label: "Alta" },
  { value: "urgente", label: "Urgente" },
];

export const PRAZOS = [
  { value: "todos", label: "Todos" },
  { value: "no_prazo", label: "No prazo" },
  { value: "atencao", label: "Atenção" },
  { value: "atrasado", label: "Atrasados" },
  { value: "finalizado", label: "Finalizados" },
];

export function rotuloStatus(status: string) {
  return STATUS.find((item) => item.value === status)?.label ?? status;
}

export function rotuloPrioridade(prioridade: string) {
  return PRIORIDADES.find((item) => item.value === prioridade)?.label ?? prioridade;
}

export function rotuloPrazo(situacao: string) {
  return PRAZOS.find((item) => item.value === situacao)?.label ?? situacao;
}

export function tomPrazo(situacao: string): "success" | "warning" | "danger" | "neutral" {
  if (situacao === "no_prazo") return "success";
  if (situacao === "atencao") return "warning";
  if (situacao === "atrasado") return "danger";
  return "neutral";
}

export function etapaLegivel(etapa: string) {
  const texto = etapa.trim().replaceAll("_", " ");
  if (!texto) return "Sem etapa";
  return texto.charAt(0).toUpperCase() + texto.slice(1);
}
