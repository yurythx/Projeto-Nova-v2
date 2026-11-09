import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import { CalendarioPrazos } from "@/components/projetos/CalendarioPrazos";

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
    expect(screen.getByRole("columnheader", { name: "Responsável" })).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "Prazo" })).toBeInTheDocument();
    expect(screen.getByText("Carlos Silva")).toBeInTheDocument();

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

  it("cria um projeto guiado pelo 5W2H com as etapas do Como", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/projetos": { data: [] },
      "POST v1/projetos": {
        status: 201,
        data: projeto({ id: "proj-5w2h", numero: 103, titulo: "Rede das escolas" }),
      },
    });

    renderApp(<ProjetosPage />);
    await user.click(await screen.findByTestId("novo-projeto-btn"));
    await user.click(screen.getByTestId("modo-5w2h"));
    await user.type(screen.getByLabelText(/Nome do projeto/i), "Rede das escolas");
    for (let i = 0; i < 5; i += 1) {
      await user.click(screen.getByTestId("wizard-proximo"));
    }
    await user.type(screen.getByLabelText(/Etapa macro/i), "Planejamento");
    await user.click(screen.getByRole("button", { name: "Adicionar etapa" }));
    await user.click(screen.getByTestId("wizard-proximo"));
    await user.click(screen.getByTestId("wizard-criar"));

    await waitFor(() => expect(router.push).toHaveBeenCalledWith("/projetos/proj-5w2h?vista=5w2h"));
    expect(api.to("POST v1/projetos")[0]!.body).toMatchObject({
      titulo: "Rede das escolas",
      etapas: ["Planejamento"],
      carta: { onde: "", patrocinador: "", metodologia: "", centro_custo: "" },
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
  it("abre as tarefas do projeto na própria linha da lista", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(perms("projetos:read")),
      "GET v1/projetos": { data: [projeto({ tarefas_total: 4, tarefas_concluidas: 1 })] },
      "GET v1/projetos/proj-1/tarefas": {
        data: [
          tarefa({ responsavel_id: "u-mariana", responsavel_nome: "Mariana Costa" }),
          tarefa({
            id: "tar-2",
            pai_id: "tar-1",
            titulo: "Pedir cotação",
            responsavel_nome: "João Lima",
            estado: "concluida",
          }),
        ],
      },
    });

    renderApp(<ProjetosPage />);

    await user.click(
      await screen.findByRole("button", { name: "Expandir tarefas de Expansão da Fibra Óptica" }),
    );

    expect(await screen.findByText("Comprar cabos de fibra")).toBeInTheDocument();
    expect(screen.getByText("Pedir cotação")).toBeInTheDocument();
    expect(screen.getByText("A fazer")).toBeInTheDocument();
    expect(screen.getByText("Mariana Costa")).toBeInTheDocument();
    expect(screen.getByText("João Lima")).toBeInTheDocument();
    expect(screen.getByRole("meter", { name: "Progresso de Expansão da Fibra Óptica" })).toHaveAttribute(
      "aria-valuenow",
      "25",
    );
    expect(screen.getByRole("meter", { name: "Progresso de Comprar cabos de fibra" })).toHaveAttribute(
      "aria-valuenow",
      "100",
    );
  });

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

  it("mostra o painel 5W2H com progresso físico e financeiro", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "proj-1" }, "/projetos/proj-1");

    mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos/proj-1": { data: projeto() },
      "GET v1/projetos/proj-1/tarefas": { data: [] },
      "GET v1/projetos/proj-1/carta": {
        data: {
          projeto_id: "proj-1",
          titulo: "Expansão da Fibra Óptica",
          descricao: "Conectar escolas municipais",
          objetivo: "100% das escolas conectadas",
          onde: "Escolas da SEMED",
          patrocinador: "Secretário",
          metodologia: "Entregas mensais",
          centro_custo: "CC-9",
          gerente_nome: "Carlos Silva",
          orcamento: 150000,
          existe: true,
          etapas_how: 2,
          etapas_how_concluidas: 1,
          custo_realizado: 30000,
          horas_chamados: 3.5,
        },
      },
    });

    renderApp(<ProjetoPage />);
    await user.click(await screen.findByRole("tab", { name: "5W2H" }));

    expect(await screen.findByTestId("painel-5w2h")).toBeInTheDocument();
    expect(screen.getByTestId("progresso-fisico")).toHaveTextContent("1 de 2 etapas concluídas");
    expect(screen.getByTestId("progresso-financeiro")).toHaveTextContent("20%");
    expect(screen.getByTestId("horas-chamados")).toHaveTextContent("3,5 h");
    expect(screen.getByText("Escolas da SEMED")).toBeInTheDocument();
  });

  it("ao salvar o responsável, confirma e volta para o detalhe do projeto", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "proj-1" }, "/projetos/proj-1");

    mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos/proj-1": { data: projeto() },
      "GET v1/projetos/proj-1/tarefas": { data: [] },
      "GET v1/projetos/proj-1/carta": {
        data: {
          projeto_id: "proj-1",
          titulo: "Expansão da Fibra Óptica",
          descricao: "Conectar escolas municipais",
          objetivo: "100% das escolas conectadas",
          onde: "",
          patrocinador: "",
          metodologia: "",
          centro_custo: "",
          gerente_nome: "Carlos Silva",
          orcamento: 0,
          existe: true,
          etapas_how: 0,
          etapas_how_concluidas: 0,
          custo_realizado: 0,
        },
      },
      "PUT v1/projetos/proj-1/carta": { data: { gerente_nome: "Carlos Silva" } },
    });

    renderApp(<ProjetoPage />);
    await user.click(await screen.findByRole("tab", { name: "5W2H" }));
    await user.click(await screen.findByRole("button", { name: "Editar escopo 5W2H" }));
    await user.click(screen.getByRole("button", { name: "Salvar escopo" }));

    expect(
      await screen.findAllByText("Responsável do projeto salvo: Carlos Silva."),
    ).not.toHaveLength(0);
    expect(screen.getByRole("tab", { name: "Ficha" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("heading", { name: "Tarefas e Marcos" })).toBeInTheDocument();
  });

  it("na ficha, salvar o responsável confirma e permanece no detalhe", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "proj-1" }, "/projetos/proj-1");

    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos/proj-1": { data: projeto() },
      "GET v1/projetos/proj-1/tarefas": { data: [] },
      "GET v1/projetos/proj-1/responsaveis": {
        data: [{ id: "u-mariana", nome: "Mariana Costa" }],
      },
      "PUT v1/projetos/proj-1": {
        data: projeto({ gerente_id: "u-mariana", gerente_nome: "Mariana Costa" }),
      },
    });

    renderApp(<ProjetoPage />);
    await user.click(await screen.findByRole("button", { name: 'Expandir seção "Ficha"' }));
    await user.type(screen.getByLabelText("Responsável pelo projeto"), "Ma");
    await user.click(await screen.findByRole("button", { name: "Mariana Costa" }));
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    expect(
      await screen.findAllByText("Responsável do projeto salvo: Mariana Costa."),
    ).not.toHaveLength(0);
    expect(screen.getByRole("heading", { name: "Tarefas e Marcos" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Ficha" })).toBeInTheDocument();
    await waitFor(() => expect(api.to("PUT v1/projetos/proj-1")).toHaveLength(1));
    expect(api.to("PUT v1/projetos/proj-1")[0]!.body).toMatchObject({
      gerente_id: "u-mariana",
      gerente_nome: "Mariana Costa",
    });
  });

  it("na lista, salvar o responsável da tarefa confirma e volta ao detalhe", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "proj-1" }, "/projetos/proj-1");

    mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos/proj-1": { data: projeto() },
      "GET v1/projetos/proj-1/tarefas": {
        data: [tarefa({ responsavel_id: "", responsavel_nome: "" })],
      },
      "GET v1/projetos/proj-1/responsaveis": {
        data: [{ id: "u-mariana", nome: "Mariana Costa" }],
      },
      "PUT v1/projetos/proj-1/tarefas/tar-1": {
        data: tarefa({ responsavel_id: "u-mariana", responsavel_nome: "Mariana Costa" }),
      },
    });

    renderApp(<ProjetoPage />);
    await user.click(await screen.findByRole("tab", { name: "Lista" }));
    await user.click(await screen.findByRole("button", { name: "Atribuir" }));
    await user.type(document.getElementById("resp-tar-1")!, "Ma");
    await user.click(await screen.findByRole("button", { name: "Mariana Costa" }));
    await user.click(screen.getByRole("button", { name: "Salvar" }));

    expect(
      await screen.findAllByText("Responsável da tarefa salvo: Mariana Costa."),
    ).not.toHaveLength(0);
    expect(screen.getByRole("heading", { name: "Tarefas e Marcos" })).toBeInTheDocument();
    expect(document.getElementById("resp-tar-1")).not.toBeInTheDocument();
  });

  it("renderiza o cronograma gantt na vista cronograma", async () => {
    const user = userEvent.setup();
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
    await user.click(screen.getByRole("button", { name: "365d" }));
    expect(screen.getByText("Cronograma & Linha do Tempo (365 dias)")).toBeInTheDocument();
    expect(screen.getAllByText("Projeto Cronograma Alpha").length).toBeGreaterThanOrEqual(1);
  });

  it("mostra o prazo da tarefa no calendário do mês", async () => {
    const sexta = new Date("2026-11-06T15:00:00-03:00");
    mockBackend({
      ...identityRoutes(perms("projetos:read")),
      "GET v1/projetos/prazos": {
        data: [
          {
            id: "tar-1",
            titulo: "Comprar cabos de fibra",
            estado: "a_fazer",
            data_limite: "2026-11-10T18:00:00-03:00",
            responsavel_id: "u-mariana",
            responsavel_nome: "Mariana Costa",
            projeto_id: "proj-1",
            projeto_numero: 101,
            projeto_titulo: "Expansão da Fibra Óptica",
          },
          {
            id: "tar-velha",
            titulo: "Trocar o switch",
            estado: "a_fazer",
            data_limite: "2026-10-02T18:00:00-03:00",
            responsavel_nome: "João Lima",
            projeto_id: "proj-1",
            projeto_numero: 101,
            projeto_titulo: "Expansão da Fibra Óptica",
          },
        ],
      },
    });

    renderApp(<CalendarioPrazos agora={sexta} />);

    expect(await screen.findByRole("region", { name: /Prazos de novembro de 2026/i })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Comprar cabos de fibra" })).toHaveAttribute(
      "href",
      "/projetos/proj-1",
    );
    expect(screen.getByRole("heading", { name: "Atrasadas de outros meses" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Trocar o switch" })).toBeInTheDocument();
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
    await user.click(screen.getByRole("button", { name: 'Expandir seção "Ficha"' }));
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

    await user.click(screen.getByRole("tab", { name: "Lista" }));
    expect(screen.getByRole("columnheader", { name: "Tarefa" })).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "Responsável" })).toBeInTheDocument();
    expect(router.replace).toHaveBeenCalled();

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

  it("pede o motivo para o superior reabrir um projeto concluído", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "proj-1" }, "/projetos/proj-1");
    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos/proj-1": {
        data: projeto({ estado: "concluido", pode_reabrir: true }),
      },
      "GET v1/projetos/proj-1/tarefas": { data: [] },
      "POST v1/projetos/proj-1/reabertura": {
        data: projeto({ estado: "em_andamento", pode_reabrir: false }),
      },
    });

    renderApp(<ProjetoPage />);
    await user.click(await screen.findByRole("button", { name: "Reabrir projeto" }));
    await user.type(screen.getByLabelText("Motivo"), "O escopo voltou");
    await user.click(screen.getByRole("button", { name: "Aprovar reabertura" }));

    await waitFor(() => expect(api.to("POST v1/projetos/proj-1/reabertura")).toHaveLength(1));
    expect(api.to("POST v1/projetos/proj-1/reabertura")[0]!.body).toEqual({
      motivo: "O escopo voltou",
    });
  });
});
