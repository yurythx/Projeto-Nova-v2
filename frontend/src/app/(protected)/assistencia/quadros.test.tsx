import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, page, renderApp } from "@/test/backend";
import { resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtendimentoFilaPage from "../atendimento/fila/page";
import AtivosPage from "../ativos/page";
import MudancasPage from "../mudancas/page";
import ProblemasPage from "../problemas/page";
import ProjetoPage from "../projetos/[id]/page";
import ProjetosPage from "../projetos/page";

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

function limparFaixas() {
  for (let i = localStorage.length - 1; i >= 0; i -= 1) {
    const chave = localStorage.key(i);
    if (chave?.startsWith("nova-faixas:")) localStorage.removeItem(chave);
  }
}

beforeEach(() => {
  resetNavigation();
  limparFaixas();
});

function faixa(titulo: string) {
  return screen.getByRole("heading", { name: new RegExp(`^${titulo}`) }).closest("section") as HTMLElement;
}

function semDizeres(raiz: HTMLElement) {
  expect(within(raiz).queryByRole("button", { name: "Nome" })).not.toBeInTheDocument();
  expect(within(raiz).queryByRole("button", { name: "Cor" })).not.toBeInTheDocument();
  expect(within(raiz).queryByText("Nome")).not.toBeInTheDocument();
  expect(within(raiz).queryByText("Cor")).not.toBeInTheDocument();
}

function abas(nomes: string[]) {
  const tabs = screen.getAllByRole("tab");
  expect(tabs.slice(0, nomes.length).map((t) => t.textContent)).toEqual(nomes);
  expect(tabs[0]).toHaveAttribute("aria-selected", "true");
}

describe("Problemas — regras do quadro", () => {
  const item = {
    id: "p1",
    numero: 5,
    titulo: "Servidor lento",
    descricao: "cai",
    estado: "registrado",
    causa_raiz: "",
    contorno: "",
    unidade_id: null,
    produto_id: null,
    created_at: "2026-10-21T12:00:00Z",
    incidentes: [],
    ativos: [],
    proximos: ["em_analise"],
  };

  it("abre no quadro, com as colunas vazias e sem os dizeres Nome e Cor", async () => {
    resetNavigation({}, "/problemas");
    mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/problemas": { data: [] },
      "GET v1/problemas/sugestoes": { data: [] },
    });
    renderApp(<ProblemasPage />);
    const quadro = await screen.findByRole("list", { name: "Quadro de problemas" });
    abas(["Quadro", "Lista"]);
    expect(within(quadro).getByRole("heading", { name: /^Registrados/ })).toBeInTheDocument();
    expect(within(quadro).getByRole("heading", { name: /^Encerrados/ })).toBeInTheDocument();
    expect(within(quadro).getAllByText("Nenhum item")).toHaveLength(5);
    expect(screen.getByText("Nenhum problema neste recorte")).toBeInTheDocument();
    semDizeres(quadro);
    expect(within(quadro).getByLabelText("Cor de Registrados")).toHaveAttribute("type", "color");
    const campo = screen.getByLabelText("Título *");
    expect(quadro.compareDocumentPosition(campo) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("recolhe a coluna numa faixa vertical e a expande de novo", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/problemas");
    mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/problemas": { data: [item] },
      "GET v1/problemas/sugestoes": { data: [] },
    });
    renderApp(<ProblemasPage />);
    const quadro = await screen.findByRole("list", { name: "Quadro de problemas" });
    const seta = within(quadro).getByRole("button", { name: "Recolher Registrados" });
    expect(seta.textContent?.trim()).toBe("");
    await user.click(seta);
    const secao = faixa("Registrados");
    expect(secao.className).toContain("w-12");
    expect(within(secao).getByRole("heading", { name: /^Registrados/ }).className).toContain(
      "[writing-mode:vertical-lr]",
    );
    expect(within(secao).queryByText("#5")).not.toBeInTheDocument();
    expect(within(secao).queryByRole("button", { name: "Novo problema" })).not.toBeInTheDocument();
    await user.click(within(secao).getByRole("button", { name: "Expandir Registrados" }));
    expect(faixa("Registrados").className).not.toContain("w-12");
    expect(within(faixa("Registrados")).getByText("#5")).toBeInTheDocument();
  });

  it("o corpo abre o problema; o nome grava o título, sem o número", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/problemas");
    const api = mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/problemas": { data: [item] },
      "GET v1/problemas/sugestoes": { data: [] },
      "PUT v1/problemas/p1": { data: { ...item, titulo: "Servidor novo" } },
      "POST v1/problemas": { status: 201, data: item },
    });
    renderApp(<ProblemasPage />);
    await screen.findByRole("list", { name: "Quadro de problemas" });
    await user.click(screen.getByText("#5"));
    expect(router.push).toHaveBeenCalledWith("/problemas/p1");

    router.push.mockClear();
    await user.click(screen.getByRole("link", { name: "Servidor lento" }));
    expect(router.push).not.toHaveBeenCalled();
    const campo = screen.getByLabelText("Nome do card Servidor lento");
    expect(campo).toHaveValue("Servidor lento");
    await user.clear(campo);
    await user.type(campo, "Servidor novo");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    await waitFor(() => expect(api.to("PUT v1/problemas/p1")).toHaveLength(1));
    expect(api.to("PUT v1/problemas/p1")[0]!.body).toMatchObject({
      titulo: "Servidor novo",
      descricao: "cai",
      causa_raiz: "",
      contorno: "",
      produto_id: null,
      unidade_id: null,
    });

    fireEvent.change(screen.getByLabelText("Cor do card Servidor lento"), {
      target: { value: "#dc2626" },
    });
    expect(router.push).not.toHaveBeenCalled();
    const gravado = JSON.parse(localStorage.getItem("nova-faixas:problemas:aparencia") ?? "{}") as {
      cartoes: Record<string, { cor?: string }>;
    };
    expect(gravado.cartoes.p1?.cor).toBe("#dc2626");

    await user.type(screen.getByLabelText("Nome do novo card em Em análise"), "Ramal mudo");
    await user.click(within(faixa("Em análise")).getByRole("button", { name: "Criar card" }));
    await waitFor(() => expect(api.to("POST v1/problemas")).toHaveLength(1));
    expect(api.to("POST v1/problemas")[0]!.body).toEqual({
      titulo: "Ramal mudo",
      descricao: "",
      unidade_id: null,
    });
  });

  it("quem só consulta vê o quadro e não cria card", async () => {
    resetNavigation({}, "/problemas");
    const api = mockBackend({
      ...identityRoutes(perms("problemas:read")),
      "GET v1/problemas": { data: [item] },
    });
    renderApp(<ProblemasPage />);
    const quadro = await screen.findByRole("list", { name: "Quadro de problemas" });
    expect(within(quadro).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();
    expect(api.to("GET v1/problemas/sugestoes")).toHaveLength(0);
  });

  it("na lista, recolher continua na horizontal e o nome editável é só o título", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/problemas", "vista=lista");
    mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/problemas": { data: [item] },
      "GET v1/problemas/sugestoes": { data: [] },
    });
    renderApp(<ProblemasPage />);
    const lista = await screen.findByRole("list", { name: "Lista de problemas" });
    expect(screen.getByRole("tab", { name: "Lista" })).toHaveAttribute("aria-selected", "true");
    await user.click(within(lista).getByRole("button", { name: "Recolher Registrados" }));
    const secao = faixa("Registrados");
    expect(secao.className).toBe("w-full");
    expect(within(secao).getByRole("heading", { name: /^Registrados/ }).className).not.toContain(
      "writing-mode",
    );
    await user.click(within(secao).getByRole("button", { name: "Expandir Registrados" }));
    await user.click(screen.getByRole("link", { name: "#5 · Servidor lento" }));
    expect(router.push).not.toHaveBeenCalled();
    expect(screen.getByLabelText("Nome do card Servidor lento")).toHaveValue("Servidor lento");
  });
});

describe("Mudanças — regras do quadro", () => {
  const item = {
    id: "m1",
    numero: 3,
    titulo: "Trocar fonte",
    descricao: "fonte nova",
    justificativa: "queimou",
    tipo: "normal",
    risco: "medio",
    risco_score: null,
    estado: "rascunho",
    plano_reversao: "voltar",
    inicio_planejado: null,
    fim_planejado: null,
    produto_id: null,
    unidade_id: null,
    solicitante_nome: "Ana",
    created_at: "2026-10-21T12:00:00Z",
    proximos: ["em_avaliacao"],
  };

  it("abre no quadro, recolhe na vertical e grava o nome e o card novo", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/mudancas");
    const api = mockBackend({
      ...identityRoutes(perms("mudancas:manage")),
      "GET v1/mudancas": { data: [item] },
      "PUT v1/mudancas/m1": { data: { ...item, titulo: "Trocar fonte reserva" } },
      "POST v1/mudancas": { status: 201, data: item },
    });
    renderApp(<MudancasPage />);
    const quadro = await screen.findByRole("list", { name: "Quadro de mudanças" });
    abas(["Quadro", "Lista"]);
    semDizeres(quadro);
    expect(within(quadro).getByRole("heading", { name: /^Rascunho/ })).toBeInTheDocument();
    expect(within(quadro).getByRole("heading", { name: /^Rejeitada \/ Falhou/ })).toBeInTheDocument();
    expect(within(faixa("Aprovadas")).getByText("Nenhum item")).toBeInTheDocument();
    const formulario = screen.getByLabelText("Título *");
    expect(quadro.compareDocumentPosition(formulario) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    await user.click(screen.getByText("#3"));
    expect(router.push).toHaveBeenCalledWith("/mudancas/m1");
    router.push.mockClear();
    await user.click(screen.getByRole("link", { name: "Trocar fonte" }));
    expect(router.push).not.toHaveBeenCalled();
    await user.clear(screen.getByLabelText("Nome do card Trocar fonte"));
    await user.type(screen.getByLabelText("Nome do card Trocar fonte"), "Trocar fonte reserva");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    await waitFor(() => expect(api.to("PUT v1/mudancas/m1")).toHaveLength(1));
    expect(api.to("PUT v1/mudancas/m1")[0]!.body).toMatchObject({
      titulo: "Trocar fonte reserva",
      tipo: "normal",
      risco: "medio",
      risco_score: null,
    });

    await user.click(within(quadro).getByRole("button", { name: "Recolher Rascunho" }));
    expect(faixa("Rascunho").className).toContain("w-12");

    await user.type(within(faixa("Em avaliação")).getByLabelText("Nome do novo card em Em avaliação"), "Janela");
    await user.click(within(faixa("Em avaliação")).getByRole("button", { name: "Criar card" }));
    await waitFor(() => expect(api.to("POST v1/mudancas")).toHaveLength(1));
    expect(api.to("POST v1/mudancas")[0]!.body).toMatchObject({
      titulo: "Janela",
      tipo: "normal",
      risco: "medio",
      risco_score: null,
      descricao: "",
      justificativa: "",
      plano_reversao: "",
      inicio_planejado: null,
      fim_planejado: null,
    });
  });
});

describe("Projetos — regras do quadro", () => {
  const item = {
    id: "proj-1",
    numero: 101,
    titulo: "Fibra da escola",
    descricao: "ligar as salas",
    objetivo: "",
    estado: "planejado",
    prioridade: "media",
    gerente_nome: "",
    orcamento: 0,
    created_at: "2026-10-22T10:00:00Z",
    updated_at: "2026-10-22T10:00:00Z",
  };

  it("abre no quadro, antes da lista e do cronograma", async () => {
    resetNavigation({}, "/projetos");
    mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos": { data: [] },
    });
    renderApp(<ProjetosPage />);
    const quadro = await screen.findByTestId("projetos-kanban");
    abas(["Quadro", "Lista", "Cronograma"]);
    expect(within(quadro).getByRole("heading", { name: /^Planejados/ })).toBeInTheDocument();
    expect(within(quadro).getByRole("heading", { name: /^Cancelados/ })).toBeInTheDocument();
    expect(within(quadro).getAllByText("Nenhum item")).toHaveLength(5);
    semDizeres(quadro);
  });

  it("o corpo abre o projeto, o nome grava e a coluna recolhida fica vertical", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/projetos");
    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos": { data: [item] },
      "PUT v1/projetos/proj-1": { data: { ...item, titulo: "Fibra nova" } },
      "POST v1/projetos": { status: 201, data: item },
    });
    renderApp(<ProjetosPage />);
    await screen.findByTestId("projetos-kanban");
    await user.click(screen.getByText("ligar as salas"));
    expect(router.push).toHaveBeenCalledWith("/projetos/proj-1");
    router.push.mockClear();
    await user.click(screen.getByRole("link", { name: "Fibra da escola" }));
    expect(router.push).not.toHaveBeenCalled();
    await user.clear(screen.getByLabelText("Nome do card Fibra da escola"));
    await user.type(screen.getByLabelText("Nome do card Fibra da escola"), "Fibra nova");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    await waitFor(() => expect(api.to("PUT v1/projetos/proj-1")).toHaveLength(1));
    expect(api.to("PUT v1/projetos/proj-1")[0]!.body).toMatchObject({
      titulo: "Fibra nova",
      prioridade: "media",
      orcamento: 0,
    });

    await user.click(screen.getByRole("button", { name: "Recolher Planejados" }));
    expect(faixa("Planejados").className).toContain("w-12");
    expect(screen.queryByRole("button", { name: "Adicionar projeto" })).not.toBeInTheDocument();

    await user.type(within(faixa("Em andamento")).getByLabelText("Nome do novo card em Em andamento"), "Rede");
    await user.click(within(faixa("Em andamento")).getByRole("button", { name: "Criar card" }));
    await waitFor(() => expect(api.to("POST v1/projetos")).toHaveLength(1));
    expect(api.to("POST v1/projetos")[0]!.body).toMatchObject({
      titulo: "Rede",
      descricao: "",
      objetivo: "",
      prioridade: "media",
      orcamento: 0,
    });
  });
});

describe("Tarefas — regras do quadro", () => {
  const projeto = {
    id: "proj-1",
    numero: 101,
    titulo: "Fibra da escola",
    descricao: "",
    objetivo: "",
    estado: "planejado",
    prioridade: "media",
    gerente_nome: "",
    orcamento: 0,
    created_at: "2026-10-22T10:00:00Z",
    updated_at: "2026-10-22T10:00:00Z",
  };
  const tarefa = {
    id: "tar-1",
    projeto_id: "proj-1",
    titulo: "Comprar cabos",
    descricao: "Lote homologado",
    responsavel_nome: "Mariana",
    estado: "a_fazer",
    ordem: 1,
    created_at: "2026-10-22T10:00:00Z",
    updated_at: "2026-10-22T10:00:00Z",
  };

  it("o nome grava a tarefa, o corpo não abre página e a coluna recolhida fica vertical", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "proj-1" }, "/projetos/proj-1");
    const api = mockBackend({
      ...identityRoutes(perms("projetos:read", "projetos:manage")),
      "GET v1/projetos/proj-1": { data: projeto },
      "GET v1/projetos/proj-1/tarefas": { data: [tarefa] },
      "GET v1/projetos/proj-1/vinculos": { data: [] },
      "GET v1/projetos/proj-1/apontamentos": { data: { items: [], resumo: { total_horas: 0, total_custo: 0, por_usuario: {} } } },
      "PUT v1/projetos/proj-1/tarefas/tar-1": { data: { ...tarefa, titulo: "Cabos novos" } },
      "POST v1/projetos/proj-1/tarefas": { status: 201, data: tarefa },
    });
    renderApp(<ProjetoPage />);
    const quadro = await screen.findByRole("list", { name: "Quadro de tarefas" });
    semDizeres(quadro);
    expect(within(quadro).getByRole("heading", { name: /^A fazer/ })).toBeInTheDocument();
    expect(within(quadro).getByRole("heading", { name: /^Bloqueada/ })).toBeInTheDocument();
    expect(within(faixa("Concluída")).getByText("Nenhum item")).toBeInTheDocument();

    await user.click(screen.getByText("Lote homologado"));
    expect(router.push).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Comprar cabos" }));
    await user.clear(screen.getByLabelText("Nome do card Comprar cabos"));
    await user.type(screen.getByLabelText("Nome do card Comprar cabos"), "Cabos novos");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    await waitFor(() => expect(api.to("PUT v1/projetos/proj-1/tarefas/tar-1")).toHaveLength(1));
    expect(api.to("PUT v1/projetos/proj-1/tarefas/tar-1")[0]!.body).toMatchObject({
      titulo: "Cabos novos",
      descricao: "Lote homologado",
      responsavel_nome: "Mariana",
      estado: "a_fazer",
      ordem: 1,
    });

    await user.click(within(quadro).getByRole("button", { name: "Recolher A fazer" }));
    expect(faixa("A fazer").className).toContain("w-12");
    expect(screen.queryByRole("button", { name: "Criar tarefa" })).not.toBeInTheDocument();

    await user.type(within(faixa("Em andamento")).getByLabelText("Nome do novo card em Em andamento"), "Lançar");
    await user.click(within(faixa("Em andamento")).getByRole("button", { name: "Criar card" }));
    await waitFor(() => expect(api.to("POST v1/projetos/proj-1/tarefas")).toHaveLength(1));
    expect(api.to("POST v1/projetos/proj-1/tarefas")[0]!.body).toMatchObject({
      titulo: "Lançar",
      descricao: "",
      responsavel_nome: "",
      estado: "em_andamento",
      ordem: 2,
    });
  });
});

describe("Fila — regras do quadro", () => {
  const item = {
    id: "s1",
    numero: 7,
    tipo: "incidente",
    estado: "aberto",
    titulo: "Caixa muda",
    descricao: "",
    oferta_id: "o1",
    oferta_nome: "Correio",
    unidade_id: "u1",
    solicitante_nome: "João",
    created_at: "2026-10-21T12:00:00Z",
    updated_at: "2026-10-21T12:00:00Z",
    proximos: ["em_atendimento"],
  };

  it("abre no quadro, sem criar card, e o nome fica só neste navegador", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/atendimento/fila");
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender", "atendimento:abrir")),
      "GET v1/atendimento/fila": page([item]),
      "GET v1/atendimento/fila/topicos": { data: { topicos: [] } },
    });
    renderApp(<AtendimentoFilaPage />);
    const quadro = await screen.findByRole("list", { name: "Quadro da fila" });
    abas(["Quadro", "Lista"]);
    semDizeres(quadro);
    expect(within(quadro).getByRole("heading", { name: /^Abertas/ })).toBeInTheDocument();
    expect(within(quadro).getByRole("heading", { name: /^Concluídas/ })).toBeInTheDocument();
    expect(within(faixa("Em atendimento")).getByText("Nenhum item")).toBeInTheDocument();
    expect(within(quadro).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();
    expect(within(faixa("Abertas")).getByRole("link", { name: "Nova solicitação" })).toBeInTheDocument();

    await user.click(screen.getByText("Correio · sem responsável"));
    expect(router.push).toHaveBeenCalledWith("/atendimento/solicitacoes/s1");
    router.push.mockClear();
    await user.click(screen.getByRole("link", { name: "Caixa muda" }));
    expect(router.push).not.toHaveBeenCalled();
    await user.clear(screen.getByLabelText("Nome do card Caixa muda"));
    await user.type(screen.getByLabelText("Nome do card Caixa muda"), "Caixa nova");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    expect(screen.getByRole("link", { name: "Caixa nova" })).toBeInTheDocument();
    expect(api.calls.filter((c) => c.method === "PUT")).toEqual([]);
    const gravado = JSON.parse(localStorage.getItem("nova-faixas:fila:aparencia") ?? "{}") as {
      cartoes: Record<string, { titulo?: string }>;
    };
    expect(gravado.cartoes.s1?.titulo).toBe("Caixa nova");

    await user.click(within(quadro).getByRole("button", { name: "Recolher Abertas" }));
    expect(faixa("Abertas").className).toContain("w-12");
    expect(screen.queryByRole("link", { name: "Nova solicitação" })).not.toBeInTheDocument();
  });
});

describe("Ativos — regras das faixas", () => {
  const item = {
    id: "a1",
    nome: "Switch",
    tipo: "rede",
    estado: "em_uso",
    identificador: "PAT-1",
    descricao: "",
    produto_id: null,
    unidade_id: null,
  };

  it("mostra as colunas vazias, abre ao clicar e grava o nome na própria faixa", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/ativos");
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/ativos": { data: [item] },
      "GET v1/produtos": { data: [] },
      "GET v1/iam/org-tree": { data: [] },
      "PUT v1/ativos/a1": { data: { ...item, nome: "Switch novo" } },
      "POST v1/ativos": { status: 201, data: item },
    });
    renderApp(<AtivosPage />);
    const lista = await screen.findByRole("list", { name: "Ativos cadastrados" });
    semDizeres(lista);
    expect(within(lista).getByRole("heading", { name: /^Planejado/ })).toBeInTheDocument();
    expect(within(lista).getByRole("heading", { name: /^Desativado/ })).toBeInTheDocument();
    expect(within(faixa("Planejado")).getByText("Nenhum item")).toBeInTheDocument();
    expect(screen.queryByText("Nenhum ativo cadastrado")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Switch" }));
    expect(screen.getByRole("heading", { name: "Novo ativo" })).toBeInTheDocument();
    expect(screen.getByLabelText("Nome do card Switch")).toHaveValue("Switch");

    await user.click(screen.getByText("Rede · PAT-1"));
    expect(screen.getByRole("heading", { name: "Editar ativo" })).toBeInTheDocument();
    expect(screen.getByLabelText("Nome *")).toHaveValue("Switch");

    await user.clear(screen.getByLabelText("Nome do card Switch"));
    await user.type(screen.getByLabelText("Nome do card Switch"), "Switch novo");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    await waitFor(() => expect(api.to("PUT v1/ativos/a1")).toHaveLength(1));
    expect(api.to("PUT v1/ativos/a1")[0]!.body).toMatchObject({
      nome: "Switch novo",
      tipo: "rede",
      estado: "em_uso",
      identificador: "PAT-1",
    });

    await user.click(within(lista).getByRole("button", { name: "Recolher Em uso" }));
    const secao = faixa("Em uso");
    expect(secao.className).toBe("w-full");
    expect(within(secao).getByRole("heading", { name: /^Em uso/ }).className).not.toContain("writing-mode");

    await user.type(within(faixa("Planejado")).getByLabelText("Nome do novo card em Planejado"), "Nobreak");
    await user.click(within(faixa("Planejado")).getByRole("button", { name: "Criar card" }));
    await waitFor(() => expect(api.to("POST v1/ativos")).toHaveLength(1));
    expect(api.to("POST v1/ativos")[0]!.body).toMatchObject({
      nome: "Nobreak",
      tipo: "outro",
      estado: "planejado",
      identificador: "",
      descricao: "",
    });
  });
});
