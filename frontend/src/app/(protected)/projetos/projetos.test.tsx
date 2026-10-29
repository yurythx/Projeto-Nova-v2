import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import ProjetoPage from "./[id]/page";
import ProjetosPage from "./page";

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "Tecnologia", sigla: "TI", departamentos: [] }],
  },
];

function projeto(extra: Record<string, unknown> = {}) {
  return {
    id: "proj-1",
    numero: 101,
    titulo: "Expansão da Fibra Óptica",
    descricao: "Conectar escolas municipais",
    objetivo: "100% das escolas conectadas",
    estado: "planejado",
    prioridade: "alta",
    gerente_nome: "Carlos Silva",
    orcamento: 150000,
    created_at: "2026-10-22T10:00:00Z",
    updated_at: "2026-10-22T10:00:00Z",
    ...extra,
  };
}

function tarefa(extra: Record<string, unknown> = {}) {
  return {
    id: "tar-1",
    projeto_id: "proj-1",
    titulo: "Comprar cabos de fibra",
    descricao: "Lote homologado",
    responsavel_nome: "Mariana",
    estado: "a_fazer",
    ordem: 1,
    created_at: "2026-10-22T10:00:00Z",
    updated_at: "2026-10-22T10:00:00Z",
    ...extra,
  };
}

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

beforeEach(() => resetNavigation({ id: "proj-1" }, "/projetos", "vista=lista"));

describe("Projetos — Lista e Criação", () => {
  it("lista os projetos e permite criar um novo projeto", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/projetos": {
        data: [projeto()],
      },
      "POST v1/projetos": {
        status: 201,
        data: projeto({ id: "proj-novo", numero: 102, titulo: "Modernização do Data Center" }),
      },
    });

    renderApp(<ProjetosPage />);

    // Encontra o projeto existente
    expect(await screen.findByText("Expansão da Fibra Óptica")).toBeInTheDocument();
    expect(screen.getByText("#101")).toBeInTheDocument();

    // Clica no botão de novo projeto
    await user.click(screen.getByTestId("novo-projeto-btn"));
    expect(await screen.findByRole("heading", { name: "Novo Projeto" })).toBeInTheDocument();

    await user.type(screen.getByLabelText(/Título do projeto/i), "Modernização do Data Center");
    await user.click(screen.getByRole("button", { name: "Criar projeto" }));

    await waitFor(() => expect(router.push).toHaveBeenCalledWith("/projetos/proj-novo"));
    expect(api.to("POST v1/projetos")[0]!.body).toMatchObject({
      titulo: "Modernização do Data Center",
    });
  });

  it("renderiza o quadro kanban e permite transicionar projeto", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/projetos", "vista=quadro");

    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos": {
        data: [
          projeto({ id: "p1", estado: "planejado", titulo: "P1 Planejado" }),
          projeto({ id: "p2", estado: "em_andamento", titulo: "P2 Em Execução" }),
        ],
      },
      "POST v1/projetos/p1/estado": {
        data: projeto({ id: "p1", estado: "em_andamento" }),
      },
    });

    renderApp(<ProjetosPage />);

    expect(await screen.findByTestId("projetos-kanban")).toBeInTheDocument();
    expect(screen.getByText("P1 Planejado")).toBeInTheDocument();
    expect(screen.getByText("P2 Em Execução")).toBeInTheDocument();

    // Move p1 para em_andamento usando o seletor acessível
    const select = screen.getByLabelText("Mudar estado de P1 Planejado");
    await user.selectOptions(select, "em_andamento");

    await waitFor(() => expect(api.to("POST v1/projetos/p1/estado")).toHaveLength(1));
    expect(api.to("POST v1/projetos/p1/estado")[0]!.body).toEqual({
      estado: "em_andamento",
    });
  });
});

describe("Projetos — Detalhes e Tarefas", () => {
  it("exibe informações do projeto e permite adicionar tarefas", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "proj-1" }, "/projetos/proj-1");

    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos/proj-1": {
        data: projeto(),
      },
      "GET v1/projetos/proj-1/tarefas": {
        data: [tarefa()],
      },
      "POST v1/projetos/proj-1/tarefas": {
        status: 201,
        data: tarefa({ id: "tar-2", titulo: "Lançar cabos nos postes" }),
      },
      "POST v1/projetos/proj-1/estado": {
        data: projeto({ estado: "em_andamento" }),
      },
    });

    renderApp(<ProjetoPage />);

    expect(await screen.findByText(/Expansão da Fibra Óptica/)).toBeInTheDocument();
    expect(screen.getAllByText("Comprar cabos de fibra").length).toBeGreaterThanOrEqual(1);

    // Transição de estado: Iniciar Projeto
    await user.click(screen.getByRole("button", { name: "Iniciar Projeto" }));
    await waitFor(() => expect(api.to("POST v1/projetos/proj-1/estado")).toHaveLength(1));

    // Nova tarefa
    await user.click(screen.getByTestId("adicionar-tarefa-btn"));
    expect(await screen.findByRole("heading", { name: "Nova Tarefa" })).toBeInTheDocument();

    await user.type(screen.getByLabelText(/Título da tarefa/i), "Lançar cabos nos postes");
    await user.click(screen.getByRole("button", { name: "Adicionar tarefa" }));

    await waitFor(() => expect(api.to("POST v1/projetos/proj-1/tarefas")).toHaveLength(1));
    expect(api.to("POST v1/projetos/proj-1/tarefas")[0]!.body).toMatchObject({
      titulo: "Lançar cabos nos postes",
    });
  });

  it("renderiza o cronograma gantt na vista cronograma", async () => {
    resetNavigation({}, "/projetos", "vista=cronograma");

    mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos": {
        data: [projeto({ id: "p1", estado: "em_andamento", titulo: "Projeto Cronograma Alpha" })],
      },
    });

    renderApp(<ProjetosPage />);

    expect(await screen.findByTestId("projetos-cronograma")).toBeInTheDocument();
    expect(screen.getByText("Cronograma & Linha do Tempo (90 dias)")).toBeInTheDocument();
    expect(screen.getAllByText("Projeto Cronograma Alpha").length).toBeGreaterThanOrEqual(1);
  });

  it("permite vincular itens da assistência ao projeto", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "proj-1" }, "/projetos/proj-1");

    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage", "problemas:read")),
      "GET v1/projetos/proj-1": {
        data: projeto({ produto_id: "prod-1" }),
      },
      "GET v1/projetos/proj-1/tarefas": {
        data: [],
      },
      "GET v1/problemas": {
        data: [{ id: "00000000-0000-0000-0000-000000000002", numero: 8, titulo: "Queda de link" }],
      },
      "GET v1/projetos/proj-1/vinculos": {
        data: [
          {
            id: "vinc-1",
            projeto_id: "proj-1",
            tipo: "mudanca",
            alvo_id: "00000000-0000-0000-0000-000000000001",
            rotulo: "Mudança #5: Troca de Switch",
            vinculado_por: "admin@gov.br",
            created_at: "2026-10-22T10:00:00Z",
          },
          {
            id: "vinc-ativo",
            projeto_id: "proj-1",
            tipo: "ativo",
            alvo_id: "00000000-0000-0000-0000-000000000099",
            rotulo: "Ativo · Firewall Central",
            vinculado_por: "admin@gov.br",
            created_at: "2026-10-22T10:00:00Z",
          },
        ],
      },
      "POST v1/projetos/proj-1/vinculos": {
        status: 201,
        data: {
          id: "vinc-2",
          projeto_id: "proj-1",
          tipo: "problema",
          alvo_id: "00000000-0000-0000-0000-000000000002",
          rotulo: "Problema #8: Queda de link",
          vinculado_por: "admin@gov.br",
          created_at: "2026-10-22T10:00:00Z",
        },
      },
    });

    renderApp(<ProjetoPage />);

    expect(await screen.findByText("Mudança #5: Troca de Switch")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Ativo · Firewall Central" })).toHaveAttribute(
      "href",
      "/ativos/00000000-0000-0000-0000-000000000099",
    );
    expect(screen.getByRole("link", { name: "Ver no catálogo de produtos" })).toHaveAttribute(
      "href",
      "/produtos",
    );

    await user.click(screen.getByTestId("adicionar-vinculo-btn"));
    expect(
      await screen.findByRole("heading", { name: "Vincular Item da Assistência" }),
    ).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText(/Tipo de vínculo/i), "problema");
    await user.selectOptions(
      await screen.findByLabelText(/Selecionar item existente/i),
      "00000000-0000-0000-0000-000000000002",
    );
    await user.clear(screen.getByLabelText(/Rótulo descritivo/i));
    await user.type(screen.getByLabelText(/Rótulo descritivo/i), "Problema #8: Queda de link");

    await user.click(screen.getByRole("button", { name: "Vincular" }));

    await waitFor(() => expect(api.to("POST v1/projetos/proj-1/vinculos")).toHaveLength(1));
    expect(api.to("POST v1/projetos/proj-1/vinculos")[0]!.body).toMatchObject({
      tipo: "problema",
      alvo_id: "00000000-0000-0000-0000-000000000002",
      rotulo: "Problema #8: Queda de link",
    });
  });

  it("permite lançar e visualizar apontamentos de horas (timesheet)", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "proj-1" }, "/projetos/proj-1");

    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos/proj-1": {
        data: projeto(),
      },
      "GET v1/projetos/proj-1/tarefas": {
        data: [tarefa()],
      },
      "GET v1/projetos/proj-1/vinculos": {
        data: [],
      },
      "GET v1/projetos/proj-1/apontamentos": {
        data: {
          items: [
            {
              id: "ap-1",
              projeto_id: "proj-1",
              tarefa_id: "tar-1",
              tarefa_titulo: "Comprar cabos de fibra",
              usuario_id: "user-1",
              usuario_nome: "Mariana Engenheira",
              data_apontamento: "2026-10-22",
              horas: 4,
              descricao: "Elaboração do termo de referência para compra",
              custo_hora: 100,
              custo_total: 400,
              created_at: "2026-10-22T10:00:00Z",
            },
          ],
          resumo: {
            total_horas: 4,
            total_custo: 400,
            por_usuario: { "Mariana Engenheira": 4 },
          },
        },
      },
      "POST v1/projetos/proj-1/apontamentos": {
        status: 201,
        data: {
          id: "ap-2",
          projeto_id: "proj-1",
          horas: 3,
          descricao: "Configuração dos switches",
          custo_hora: 100,
          custo_total: 300,
        },
      },
    });

    renderApp(<ProjetoPage />);

    expect(await screen.findByTestId("card-timesheet")).toBeInTheDocument();
    expect(screen.getByText("Apontamento de Horas & Timesheet")).toBeInTheDocument();
    expect(screen.getByText("Mariana Engenheira")).toBeInTheDocument();
    expect(screen.getByText("Elaboração do termo de referência para compra")).toBeInTheDocument();

    await user.click(screen.getByTestId("lancar-horas-btn"));
    expect(
      await screen.findByRole("heading", { name: "Lançar Horas no Projeto" }),
    ).toBeInTheDocument();

    await user.clear(screen.getByLabelText(/Horas trabalhadas/i));
    await user.type(screen.getByLabelText(/Horas trabalhadas/i), "3");
    await user.type(
      screen.getByLabelText(/Descrição das atividades/i),
      "Configuração dos switches",
    );

    await user.click(screen.getByRole("button", { name: "Registrar horas" }));

    await waitFor(() => expect(api.to("POST v1/projetos/proj-1/apontamentos")).toHaveLength(1));
    expect(api.to("POST v1/projetos/proj-1/apontamentos")[0]!.body).toMatchObject({
      horas: 3,
      descricao: "Configuração dos switches",
    });
  });
});
