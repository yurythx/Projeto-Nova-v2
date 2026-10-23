import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { ProjetoGanttChart } from "@/components/projetos/ProjetoGanttChart";
import type { Projeto, Tarefa, VinculoProjeto } from "@/lib/projetos/projetos";

const projetoMock: Projeto = {
  id: "proj-1",
  numero: 101,
  titulo: "Modernização da Infraestrutura de Redes",
  descricao: "Substituição de switches e cabeamento estruturado",
  objetivo: "Garantir 10 Gbps em todos os pontos de rede",
  estado: "em_andamento",
  prioridade: "alta",
  data_inicio_planejada: "2026-10-01T00:00:00Z",
  data_fim_planejada: "2026-11-30T00:00:00Z",
  orcamento: 150000,
  gerente_nome: "Carlos Silva",
  created_at: "2026-10-01T08:00:00Z",
  updated_at: "2026-10-15T10:00:00Z",
};

const tarefasMock: Tarefa[] = [
  {
    id: "tar-1",
    projeto_id: "proj-1",
    titulo: "Levantamento topológico dos racks",
    descricao: "Mapear portas e patch panels",
    responsavel_nome: "Mariana Costa",
    estado: "concluida",
    ordem: 1,
    data_limite: "2026-10-10T18:00:00Z",
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-10T17:00:00Z",
  },
  {
    id: "tar-2",
    projeto_id: "proj-1",
    titulo: "Instalação dos novos switches core",
    descricao: "Configuração de VLANs e roteamento",
    responsavel_nome: "Lucas Pereira",
    estado: "em_andamento",
    ordem: 2,
    data_limite: "2026-10-28T18:00:00Z",
    created_at: "2026-10-11T09:00:00Z",
    updated_at: "2026-10-15T11:00:00Z",
  },
  {
    id: "tar-3",
    projeto_id: "proj-1",
    titulo: "Migração dos servidores de produção",
    descricao: "Janela noturna aprovada pelo CAB",
    responsavel_nome: "Carlos Silva",
    estado: "a_fazer",
    ordem: 3,
    data_limite: "2026-11-15T23:59:00Z",
    created_at: "2026-10-11T09:00:00Z",
    updated_at: "2026-10-11T09:00:00Z",
  },
];

const vinculosMock: VinculoProjeto[] = [
  {
    id: "vinc-1",
    projeto_id: "proj-1",
    tipo: "mudanca",
    alvo_id: "mud-42",
    rotulo: "Mudança #42: Janela do Core Switch",
    vinculado_por: "Carlos Silva",
    created_at: "2026-10-12T14:00:00Z",
  },
];

describe("ProjetoGanttChart (Cronograma Visual Interativo)", () => {
  it("renderiza o diagrama de Gantt com progresso geral e tarefas escalonadas", () => {
    render(
      <ProjetoGanttChart
        projeto={projetoMock}
        tarefas={tarefasMock}
        vinculos={vinculosMock}
        podeEditar={true}
      />,
    );

    expect(screen.getByTestId("projeto-gantt-chart")).toBeInTheDocument();
    expect(screen.getByText("Cronograma & Diagrama de Gantt")).toBeInTheDocument();
    expect(screen.getByText("3 atividades")).toBeInTheDocument();
    // 1 de 3 concluída = 33%
    expect(screen.getByText("33% entregue")).toBeInTheDocument();

    // Tarefas
    expect(screen.getAllByText("Levantamento topológico dos racks").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Instalação dos novos switches core").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Migração dos servidores de produção").length).toBeGreaterThanOrEqual(1);

    // Responsáveis
    expect(screen.getByText("Mariana Costa")).toBeInTheDocument();
    expect(screen.getByText("Lucas Pereira")).toBeInTheDocument();

    // Marcos Vinculados da Assistência
    expect(screen.getByText("Marcos Vinculados da Assistência")).toBeInTheDocument();
    expect(screen.getByText(/Marco: Mudança #42: Janela do Core Switch/)).toBeInTheDocument();
  });

  it("permite filtrar por estado da tarefa no diagrama", async () => {
    const user = userEvent.setup();
    render(
      <ProjetoGanttChart
        projeto={projetoMock}
        tarefas={tarefasMock}
        vinculos={vinculosMock}
      />,
    );

    // Clica no filtro "Concluída"
    await user.click(screen.getByRole("button", { name: "Concluída (1)" }));

    // Apenas a concluída deve permanecer
    expect(screen.getAllByText("Levantamento topológico dos racks").length).toBeGreaterThanOrEqual(1);
    expect(screen.queryByText("Instalação dos novos switches core")).not.toBeInTheDocument();
    expect(screen.queryByText("Migração dos servidores de produção")).not.toBeInTheDocument();

    // Volta para todas
    await user.click(screen.getByRole("button", { name: "Todas (3)" }));
    expect(screen.getAllByText("Instalação dos novos switches core").length).toBeGreaterThanOrEqual(1);
  });

  it("permite alternar janelas de tempo e navegar", async () => {
    const user = userEvent.setup();
    render(
      <ProjetoGanttChart
        projeto={projetoMock}
        tarefas={tarefasMock}
      />,
    );

    // Zoom 30d
    await user.click(screen.getByRole("button", { name: "30d" }));
    // Navegar duas semanas
    await user.click(screen.getByRole("button", { name: "Duas semanas depois" }));
    // Voltar para hoje
    await user.click(screen.getByRole("button", { name: "Hoje" }));
  });
});
