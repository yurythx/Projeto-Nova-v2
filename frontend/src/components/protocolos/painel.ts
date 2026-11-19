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

export interface FiltroMesa {
  prazo: string;
  busca: string;
  servico: string;
  status: string;
  prioridade: string;
}

export interface PedidoFiltravel {
  numero: string;
  nome: string;
  cpf?: string;
  servico_id: string;
  servico_nome: string;
  status: string;
  situacao_prazo?: string;
  prioridade: string;
}

export function filtrarPedidos<T extends PedidoFiltravel>(pedidos: T[], filtro: FiltroMesa): T[] {
  const termo = filtro.busca.trim().toLowerCase();
  const digitos = termo.replace(/\D/g, "");
  return pedidos.filter((pedido) => {
    if (filtro.prazo !== "todos" && (pedido.situacao_prazo || "no_prazo") !== filtro.prazo)
      return false;
    if (filtro.servico && pedido.servico_id !== filtro.servico) return false;
    if (filtro.status && pedido.status !== filtro.status) return false;
    if (filtro.prioridade && pedido.prioridade !== filtro.prioridade) return false;
    if (!termo) return true;
    const cpf = (pedido.cpf ?? "").replace(/\D/g, "");
    return (
      `${pedido.numero} ${pedido.nome} ${pedido.servico_nome}`.toLowerCase().includes(termo) ||
      (digitos.length >= 3 && cpf.includes(digitos))
    );
  });
}

export function paginar<T>(itens: T[], pagina: number, tamanho: number) {
  const paginas = Math.max(1, Math.ceil(itens.length / tamanho));
  const atual = Math.min(Math.max(1, pagina), paginas);
  const inicio = (atual - 1) * tamanho;
  return {
    itens: itens.slice(inicio, inicio + tamanho),
    pagina: atual,
    paginas,
    total: itens.length,
  };
}
