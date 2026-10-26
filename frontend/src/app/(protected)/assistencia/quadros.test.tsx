import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, page, renderApp } from "@/test/backend";
import { resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtendimentoFilaPage from "../atendimento/fila/page";
import ColunasDeProblemasPage from "../problemas/colunas/page";
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
    expect(faixa("Registrados")).toHaveAttribute("data-ordem", "0");
    expect(faixa("Encerrados")).toHaveAttribute("data-ordem", "4");
    expect(within(quadro).getAllByText("Nenhum item")).toHaveLength(5);
    expect(screen.getByText("Nenhum problema neste recorte")).toBeInTheDocument();
    semDizeres(quadro);
    expect(within(quadro).getByLabelText("Cor de Registrados")).toHaveAttribute("type", "color");
    expect(within(quadro).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();
    expect(within(quadro).queryByRole("button", { name: "Novo problema" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Novo problema" })).toBeInTheDocument();
    expect(screen.queryByLabelText("Título *")).not.toBeInTheDocument();
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
    expect(await screen.findByRole("dialog", { name: "Detalhe do problema" })).toBeInTheDocument();
    expect(router.push).toHaveBeenCalledWith("/problemas?painel=p1", { scroll: false });
    await user.click(screen.getByRole("button", { name: "Fechar painel" }));

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
  });

  it("o nome do quadro não muda o estado", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/problemas");
    const emAnalise = {
      ...item,
      id: "p2",
      numero: 6,
      titulo: "Ramal",
      estado: "em_analise",
      proximos: ["erro_conhecido", "resolvido"],
    };
    const api = mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/problemas": { data: [item, emAnalise] },
      "GET v1/problemas/sugestoes": { data: [] },
      "POST v1/problemas": { status: 201, data: { ...item, id: "p9", titulo: "Ramal mudo" } },
      "POST v1/problemas/p9/estado": { data: { ...emAnalise, id: "p9", titulo: "Ramal mudo" } },
    });
    renderApp(<ProblemasPage />);
    await screen.findByRole("list", { name: "Quadro de problemas" });
    expect(within(faixa("Em análise")).getByText("#6")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Em análise" }));
    const nome = screen.getByLabelText("Nome de Em análise");
    await user.clear(nome);
    await user.type(nome, "Triagem do João");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));

    const secao = faixa("Triagem do João");
    expect(secao).toHaveAttribute("data-coluna", "em_analise");
    expect(secao).toHaveAttribute("data-ordem", "1");
    expect(faixa("Registrados")).toHaveAttribute("data-ordem", "0");
    expect(within(secao).getByText("#6")).toBeInTheDocument();
    expect(within(secao).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Novo problema" }));
    const quadro = await screen.findByLabelText("Quadro");
    await waitFor(() => {
      expect(within(quadro).getByRole("option", { name: "Triagem do João" })).toHaveValue("em_analise");
    });
    await user.selectOptions(quadro, "em_analise");
    await user.type(screen.getByLabelText("Título *"), "Ramal mudo");
    await user.click(screen.getByRole("button", { name: "Registrar problema" }));

    await waitFor(() => expect(api.to("POST v1/problemas")).toHaveLength(1));
    const corpo = api.to("POST v1/problemas")[0]!.body as Record<string, unknown>;
    expect(corpo).toMatchObject({ titulo: "Ramal mudo", descricao: "", unidade_id: null });
    expect(JSON.stringify(corpo)).not.toContain("Triagem");
    await waitFor(() => expect(api.to("POST v1/problemas/p9/estado")).toHaveLength(1));
    expect(api.to("POST v1/problemas/p9/estado")[0]!.body).toEqual({ estado: "em_analise" });
  });

  it("o catálogo manda na ordem e o nome gravado não troca a chave", async () => {
    resetNavigation({}, "/problemas");
    const api = mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/problemas": { data: [] },
      "GET v1/problemas/sugestoes": { data: [] },
      "GET v1/quadros/problemas": {
        data: [
          { chave: "em_analise", nome: "Triagem", cor: "", tom: "info", ordem: 0, entrada: false, ativa: true },
          { chave: "registrado", nome: "Registrados", cor: "", tom: "info", ordem: 1, entrada: true, ativa: true },
          { chave: "erro_conhecido", nome: "Erros conhecidos", cor: "", tom: "warning", ordem: 2, entrada: false, ativa: true },
          { chave: "resolvido", nome: "Resolvidos", cor: "", tom: "success", ordem: 3, entrada: false, ativa: true },
          { chave: "encerrado", nome: "Encerrados", cor: "", tom: "neutral", ordem: 4, entrada: false, ativa: true },
        ],
      },
      "PUT v1/quadros/problemas/em_analise": { data: { chave: "em_analise", nome: "Triagem", ordem: 0, ativa: true } },
    });
    renderApp(<ProblemasPage />);
    const triagem = await screen.findByRole("heading", { name: /^Triagem/ });
    expect(triagem.closest("section")).toHaveAttribute("data-coluna", "em_analise");
    expect(triagem.closest("section")).toHaveAttribute("data-ordem", "0");
    expect(faixa("Registrados")).toHaveAttribute("data-ordem", "1");
    expect(screen.queryByRole("button", { name: /Retirar/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Subir/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Descer/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Triagem" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Colunas" })).toHaveAttribute("href", "/problemas/colunas");
    expect(api.to("PUT v1/quadros/problemas/em_analise")).toHaveLength(0);
  });

  it("a página de colunas grava o nome e não mexe na ordem se ela não mudou", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/problemas/colunas");
    const api = mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/quadros/problemas": {
        data: [
          { chave: "registrado", nome: "Registrados", cor: "", tom: "info", ordem: 0, entrada: true, ativa: true },
          { chave: "em_analise", nome: "Triagem", cor: "", tom: "info", ordem: 1, entrada: false, ativa: true },
          { chave: "erro_conhecido", nome: "Erros conhecidos", cor: "", tom: "warning", ordem: 2, entrada: false, ativa: true },
          { chave: "resolvido", nome: "Resolvidos", cor: "", tom: "success", ordem: 3, entrada: false, ativa: true },
          { chave: "encerrado", nome: "Encerrados", cor: "", tom: "neutral", ordem: 4, entrada: false, ativa: true },
        ],
      },
      "PUT v1/quadros/problemas/em_analise": {
        data: { chave: "em_analise", nome: "Fila de triagem", cor: "#2563eb", ordem: 1, ativa: true },
      },
    });
    renderApp(<ColunasDeProblemasPage />);
    const nome = await screen.findByLabelText("Nome", { selector: "#nome-problemas-em_analise" });
    await user.clear(nome);
    await user.type(nome, "Fila de triagem");
    await user.click(screen.getByRole("button", { name: "Salvar colunas" }));
    await waitFor(() => expect(api.to("PUT v1/quadros/problemas/em_analise")).toHaveLength(1));
    expect(api.to("PUT v1/quadros/problemas/em_analise")[0]!.body).toEqual({
      nome: "Fila de triagem",
      cor: "#2563eb",
    });
    expect(api.to("PUT v1/quadros/problemas/registrado")).toHaveLength(0);
    expect(screen.getByText("Coluna inicial")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Retirar" }).length).toBeGreaterThan(0);
  });

  it("cria a coluna só neste quadro, com uma entrada e uma saída", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/problemas/colunas");
    const api = mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/quadros/problemas": {
        data: [
          { chave: "registrado", nome: "Registrados", cor: "", tom: "info", ordem: 0, entrada: true, ativa: true },
          { chave: "em_analise", nome: "Triagem", cor: "", tom: "info", ordem: 1, entrada: false, ativa: true },
          { chave: "erro_conhecido", nome: "Erros conhecidos", cor: "", tom: "warning", ordem: 2, entrada: false, ativa: true },
          { chave: "resolvido", nome: "Resolvidos", cor: "", tom: "success", ordem: 3, entrada: false, ativa: true },
          { chave: "encerrado", nome: "Encerrados", cor: "", tom: "neutral", ordem: 4, entrada: false, ativa: true },
        ],
      },
      "POST v1/quadros/problemas": {
        data: { chave: "c01234567", nome: "Aguardando fornecedor", cor: "#2563eb", ordem: 5, ativa: true },
      },
    });
    renderApp(<ColunasDeProblemasPage />);
    await user.click(await screen.findByRole("button", { name: "Nova coluna" }));
    await user.type(screen.getByLabelText("Nome", { selector: "#nova-nome-problemas" }), "Aguardando fornecedor");
    await user.click(screen.getByRole("button", { name: "Criar coluna" }));
    expect(await screen.findByText("Escolha ao menos uma coluna de onde o card entra.")).toBeInTheDocument();
    expect(api.to("POST v1/quadros/problemas")).toHaveLength(0);

    await user.click(within(screen.getByRole("group", { name: "O card entra a partir de" })).getByRole("checkbox", { name: "Triagem" }));
    await user.click(within(screen.getByRole("group", { name: "O card pode seguir para" })).getByRole("checkbox", { name: "Encerrados" }));
    await user.click(screen.getByRole("button", { name: "Criar coluna" }));
    await waitFor(() => expect(api.to("POST v1/quadros/problemas")).toHaveLength(1));
    expect(api.to("POST v1/quadros/problemas")[0]!.body).toEqual({
      nome: "Aguardando fornecedor",
      cor: "#2563eb",
      ordem: 5,
      de: ["em_analise"],
      para: ["encerrado"],
    });
    expect(api.to("POST v1/quadros/mudancas")).toHaveLength(0);
  });

  it("não cria coluna sem nome, com posição inválida ou sem permissão de gestão", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/problemas/colunas");
    const api = mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/quadros/problemas": {
        data: [
          { chave: "registrado", nome: "Registrados", cor: "", tom: "info", ordem: 0, entrada: true, ativa: true },
          { chave: "em_analise", nome: "Triagem", cor: "", tom: "info", ordem: 1, entrada: false, ativa: true },
        ],
      },
    });
    const { unmount } = renderApp(<ColunasDeProblemasPage />);
    await user.click(await screen.findByRole("button", { name: "Nova coluna" }));
    await user.click(screen.getByRole("button", { name: "Criar coluna" }));
    expect(await screen.findByText("Toda coluna precisa de um nome.")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Nome", { selector: "#nova-nome-problemas" }), "Fornecedor");
    await user.clear(screen.getByLabelText("Posição", { selector: "#nova-posicao-problemas" }));
    await user.type(screen.getByLabelText("Posição", { selector: "#nova-posicao-problemas" }), "x");
    await user.click(screen.getByRole("button", { name: "Criar coluna" }));
    expect(await screen.findByText("A posição é um número a partir de zero.")).toBeInTheDocument();
    expect(api.to("POST v1/quadros/problemas")).toHaveLength(0);
    unmount();

    resetNavigation({}, "/problemas/colunas");
    mockBackend({
      ...identityRoutes(perms("problemas:read")),
      "GET v1/quadros/problemas": {
        data: [{ chave: "registrado", nome: "Registrados", cor: "", tom: "info", ordem: 0, entrada: true, ativa: true }],
      },
    });
    renderApp(<ColunasDeProblemasPage />);
    expect(await screen.findByLabelText("Nome", { selector: "#nome-problemas-registrado" })).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Nova coluna" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Salvar colunas" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retirar" })).not.toBeInTheDocument();
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
    expect(screen.queryByRole("button", { name: "Novo problema" })).not.toBeInTheDocument();
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
      "POST v1/mudancas/m1/estado": { data: { ...item, estado: "em_avaliacao" } },
    });
    renderApp(<MudancasPage />);
    const quadro = await screen.findByRole("list", { name: "Quadro de mudanças" });
    abas(["Quadro", "Lista"]);
    semDizeres(quadro);
    expect(within(quadro).getByRole("heading", { name: /^Rascunho/ })).toBeInTheDocument();
    expect(within(quadro).getByRole("heading", { name: /^Rejeitada \/ Falhou/ })).toBeInTheDocument();
    expect(faixa("Rascunho")).toHaveAttribute("data-ordem", "0");
    expect(faixa("Rejeitada / Falhou")).toHaveAttribute("data-ordem", "5");
    expect(within(faixa("Aprovadas")).getByText("Nenhum item")).toBeInTheDocument();
    expect(within(quadro).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Nova mudança" })).toBeInTheDocument();
    expect(screen.queryByLabelText("Título *")).not.toBeInTheDocument();

    await user.click(screen.getByText("#3"));
    expect(await screen.findByRole("dialog", { name: "Detalhe da mudança" })).toBeInTheDocument();
    expect(router.push).toHaveBeenCalledWith("/mudancas?painel=m1", { scroll: false });
    await user.click(screen.getByRole("button", { name: "Fechar painel" }));
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
    expect(within(faixa("Em avaliação")).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Em avaliação" }));
    const nomeQuadro = screen.getByLabelText("Nome de Em avaliação");
    await user.clear(nomeQuadro);
    await user.type(nomeQuadro, "CAB");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    expect(faixa("CAB")).toHaveAttribute("data-coluna", "em_avaliacao");

    await user.click(screen.getByRole("button", { name: "Nova mudança" }));
    const escolha = await screen.findByLabelText("Quadro");
    await waitFor(() => expect(within(escolha).getByRole("option", { name: "CAB" })).toHaveValue("em_avaliacao"));
    await user.selectOptions(escolha, "em_avaliacao");
    await user.type(screen.getByLabelText("Título *"), "Janela");
    await user.click(screen.getByRole("button", { name: "Registrar mudança" }));
    await waitFor(() => expect(api.to("POST v1/mudancas")).toHaveLength(1));
    const corpo = api.to("POST v1/mudancas")[0]!.body as Record<string, unknown>;
    expect(corpo).toMatchObject({
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
    expect(JSON.stringify(corpo)).not.toContain("CAB");
    await waitFor(() => expect(api.to("POST v1/mudancas/m1/estado")).toHaveLength(1));
    expect(api.to("POST v1/mudancas/m1/estado")[0]!.body).toEqual({ estado: "em_avaliacao" });
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
    expect(faixa("Planejados")).toHaveAttribute("data-ordem", "0");
    expect(faixa("Cancelados")).toHaveAttribute("data-ordem", "4");
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
      "POST v1/projetos/proj-1/estado": { data: { ...item, estado: "em_andamento" } },
    });
    renderApp(<ProjetosPage />);
    await screen.findByTestId("projetos-kanban");
    await user.click(screen.getByText("ligar as salas"));
    expect(await screen.findByRole("dialog", { name: "Detalhe do projeto" })).toBeInTheDocument();
    expect(router.push).toHaveBeenCalledWith("/projetos?painel=proj-1", { scroll: false });
    await user.click(screen.getByRole("button", { name: "Fechar painel" }));
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
    expect(within(faixa("Planejados")).queryByRole("button", { name: "Adicionar projeto" })).not.toBeInTheDocument();
    expect(within(faixa("Em andamento")).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Novo projeto" })).toBeInTheDocument();

    await user.click(within(faixa("Em andamento")).getByRole("button", { name: "Em andamento" }));
    const nomeQuadro = screen.getByLabelText("Nome de Em andamento");
    await user.clear(nomeQuadro);
    await user.type(nomeQuadro, "Execução");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    expect(faixa("Execução")).toHaveAttribute("data-coluna", "em_andamento");

    await user.click(screen.getByRole("button", { name: "Novo projeto" }));
    const escolha = await screen.findByLabelText("Quadro");
    await waitFor(() => expect(within(escolha).getByRole("option", { name: "Execução" })).toHaveValue("em_andamento"));
    await user.type(screen.getByLabelText("Título do projeto *"), "Rede");
    await user.selectOptions(escolha, "em_andamento");
    await user.click(screen.getByRole("button", { name: "Criar projeto" }));
    await waitFor(() => expect(api.to("POST v1/projetos")).toHaveLength(1));
    const corpo = api.to("POST v1/projetos")[0]!.body as Record<string, unknown>;
    expect(corpo).toMatchObject({
      titulo: "Rede",
      descricao: "",
      objetivo: "",
      prioridade: "media",
      orcamento: 0,
    });
    expect(JSON.stringify(corpo)).not.toContain("Execução");
    await waitFor(() => expect(api.to("POST v1/projetos/proj-1/estado")).toHaveLength(1));
    expect(api.to("POST v1/projetos/proj-1/estado")[0]!.body).toEqual({ estado: "em_andamento" });
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
    expect(faixa("A fazer")).toHaveAttribute("data-ordem", "0");
    expect(faixa("Bloqueada")).toHaveAttribute("data-ordem", "3");
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
    expect(within(quadro).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Criar tarefa" })).not.toBeInTheDocument();

    await user.click(within(faixa("Em andamento")).getByRole("button", { name: "Em andamento" }));
    const nomeQuadro = screen.getByLabelText("Nome de Em andamento");
    await user.clear(nomeQuadro);
    await user.type(nomeQuadro, "Fazendo");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    expect(faixa("Fazendo")).toHaveAttribute("data-coluna", "em_andamento");

    await user.click(screen.getByTestId("adicionar-tarefa-btn"));
    const escolha = await screen.findByLabelText("Quadro");
    await waitFor(() => expect(within(escolha).getByRole("option", { name: "Fazendo" })).toHaveValue("em_andamento"));
    await user.type(screen.getByLabelText("Título da tarefa *"), "Lançar");
    await user.selectOptions(escolha, "em_andamento");
    await user.click(screen.getByRole("button", { name: "Adicionar tarefa" }));
    await waitFor(() => expect(api.to("POST v1/projetos/proj-1/tarefas")).toHaveLength(1));
    const corpo = api.to("POST v1/projetos/proj-1/tarefas")[0]!.body as Record<string, unknown>;
    expect(corpo).toMatchObject({
      titulo: "Lançar",
      descricao: "",
      responsavel_nome: "",
      estado: "em_andamento",
      ordem: 2,
    });
    expect(JSON.stringify(corpo)).not.toContain("Fazendo");
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
    expect(faixa("Abertas")).toHaveAttribute("data-ordem", "0");
    expect(faixa("Concluídas")).toHaveAttribute("data-ordem", "4");
    expect(within(faixa("Em atendimento")).getByText("Nenhum item")).toBeInTheDocument();
    expect(within(quadro).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();
    expect(within(quadro).queryByRole("link", { name: "Nova solicitação" })).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Nova solicitação" })).toHaveAttribute("href", "/atendimento");

    await user.click(screen.getByText("Correio · sem responsável"));
    expect(await screen.findByRole("dialog", { name: "Detalhe da solicitação" })).toBeInTheDocument();
    expect(router.push).toHaveBeenCalledWith("/atendimento/fila?painel=s1", { scroll: false });
    await user.click(screen.getByRole("button", { name: "Fechar painel" }));
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
    expect(screen.getByRole("link", { name: "Nova solicitação" })).toHaveAttribute("href", "/atendimento");
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
    expect(faixa("Planejado")).toHaveAttribute("data-ordem", "0");
    expect(faixa("Desativado")).toHaveAttribute("data-ordem", "3");
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
    expect(within(lista).queryByRole("button", { name: "Criar card" })).not.toBeInTheDocument();

    await user.click(within(faixa("Planejado")).getByRole("button", { name: "Planejado" }));
    const nomeQuadro = screen.getByLabelText("Nome de Planejado");
    await user.clear(nomeQuadro);
    await user.type(nomeQuadro, "Reserva");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    expect(faixa("Reserva")).toHaveAttribute("data-coluna", "planejado");

    await user.click(screen.getByRole("button", { name: "Novo ativo" }));
    expect(screen.getByRole("heading", { name: "Novo ativo" })).toBeInTheDocument();
    const estado = screen.getByLabelText("Estado");
    await waitFor(() => expect(within(estado).getByRole("option", { name: "Reserva" })).toHaveValue("planejado"));
    await user.clear(screen.getByLabelText("Nome *"));
    await user.type(screen.getByLabelText("Nome *"), "Nobreak");
    await user.selectOptions(estado, "planejado");
    await user.click(screen.getByRole("button", { name: "Cadastrar ativo" }));
    await waitFor(() => expect(api.to("POST v1/ativos")).toHaveLength(1));
    const corpo = api.to("POST v1/ativos")[0]!.body as Record<string, unknown>;
    expect(corpo).toMatchObject({
      nome: "Nobreak",
      tipo: "hardware",
      estado: "planejado",
      identificador: "",
      descricao: "",
    });
    expect(JSON.stringify(corpo)).not.toContain("Reserva");
  });
});
