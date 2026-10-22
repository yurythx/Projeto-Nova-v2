import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";

import ProdutosPage from "./page";

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "Tecnologia", sigla: "TI", departamentos: [] }],
  },
];

const protocolo = {
  id: "p1",
  nome: "Protocolo",
  descricao: "registro",
  estagio: "descoberta",
  utilidade: "registrar",
  garantia: "30 dias",
  unidade_id: "u1",
};

const institucional = {
  id: "p2",
  nome: "E-mail",
  descricao: "",
  estagio: "legado",
  utilidade: "",
  garantia: "",
};

function rotas(extra: Record<string, unknown> = {}) {
  return {
    ...identityRoutes(),
    "GET v1/iam/org-tree": { data: ORG },
    "GET v1/produtos": { data: [protocolo, institucional] },
    "POST v1/produtos": { data: protocolo },
    "PUT v1/produtos/p1": { data: protocolo },
    "POST v1/produtos/p1/estagio": { data: { ...protocolo, estagio: "operacao" } },
    "POST v1/produtos/p2/estagio": fail(422, "estágio recusado"),
    ...extra,
  };
}

describe("Produtos digitais", () => {
  it("cadastra, corrige o texto e muda o estágio", async () => {
    const user = userEvent.setup();
    const api = mockBackend(rotas());
    renderApp(<ProdutosPage />);

    expect(await screen.findByText("registro")).toBeInTheDocument();
    expect(screen.getByText("legado")).toBeInTheDocument();
    expect(screen.getByText("Da unidade")).toBeInTheDocument();
    expect(screen.getByText("Institucional")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Ver ativos de Protocolo" })).toHaveAttribute(
      "href",
      "/ativos?produto=p1",
    );

    const form = screen
      .getByRole("heading", { name: "Novo produto" })
      .closest("div.rounded-xl") as HTMLElement;
    await user.type(within(form).getByLabelText("Nome *"), "Rede");
    await user.type(within(form).getByLabelText("Descrição"), "link");
    expect(
      await within(form).findByRole("option", { name: "TI — Tecnologia" }),
    ).toBeInTheDocument();
    await user.selectOptions(within(form).getByLabelText("Unidade dona"), "u1");
    await user.click(within(form).getByRole("button", { name: "Cadastrar produto" }));
    await waitFor(() => expect(api.to("POST v1/produtos")).toHaveLength(1));
    expect(api.to("POST v1/produtos")[0]!.body).toEqual({
      nome: "Rede",
      descricao: "link",
      utilidade: "",
      garantia: "",
      unidade_id: "u1",
      estagio: "descoberta",
    });

    await user.click(screen.getByRole("button", { name: "Editar Protocolo" }));
    const edicao = screen
      .getByRole("heading", { name: "Editar produto" })
      .closest("div.rounded-xl") as HTMLElement;
    expect(within(edicao).queryByLabelText("Estágio inicial *")).not.toBeInTheDocument();
    expect(within(edicao).getByLabelText("Nome *")).toHaveValue("Protocolo");
    await user.clear(within(edicao).getByLabelText("Nome *"));
    await user.type(within(edicao).getByLabelText("Nome *"), "Protocolo web");
    await user.selectOptions(within(edicao).getByLabelText("Unidade dona"), "");
    await user.click(within(edicao).getByRole("button", { name: "Salvar alterações" }));
    await waitFor(() => expect(api.to("PUT v1/produtos/p1")).toHaveLength(1));
    expect(api.to("PUT v1/produtos/p1")[0]!.body).toMatchObject({
      nome: "Protocolo web",
      unidade_id: null,
    });
    expect(api.to("PUT v1/produtos/p1")[0]!.body).not.toHaveProperty("estagio");

    await user.click(screen.getByRole("button", { name: "Editar E-mail" }));
    await user.click(screen.getByRole("button", { name: "Cancelar edição" }));
    expect(screen.getByRole("heading", { name: "Novo produto" })).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("Estágio de Protocolo"), "operacao");
    await user.click(screen.getByRole("button", { name: "Aplicar estágio de Protocolo" }));
    await waitFor(() =>
      expect(api.to("POST v1/produtos/p1/estagio")[0]!.body).toEqual({ estagio: "operacao" }),
    );

    await user.selectOptions(screen.getByLabelText("Estágio de E-mail"), "suporte");
    await user.click(screen.getByRole("button", { name: "Aplicar estágio de E-mail" }));
    expect(await screen.findByText("estágio recusado")).toBeInTheDocument();
  });

  it("recusa o cadastro e mantém o que foi digitado", async () => {
    const user = userEvent.setup();
    mockBackend(rotas({ "POST v1/produtos": fail(422, "nome inválido") }));
    renderApp(<ProdutosPage />);
    const nome = await screen.findByLabelText("Nome *");
    await user.type(nome, "Rede");
    await user.click(screen.getByRole("button", { name: "Cadastrar produto" }));
    expect(await screen.findByText("nome inválido")).toBeInTheDocument();
    expect(nome).toHaveValue("Rede");
  });

  it("lista vazia e falha de leitura", async () => {
    mockBackend(rotas({ "GET v1/produtos": { data: [] } }));
    const vazio = renderApp(<ProdutosPage />);
    expect(await screen.findByText("Nenhum produto cadastrado")).toBeInTheDocument();
    vazio.unmount();

    mockBackend(rotas({ "GET v1/produtos": fail(500, "banco fora") }));
    renderApp(<ProdutosPage />);
    expect(await screen.findByText("banco fora")).toBeInTheDocument();
  });

  it("sem a permissão de gestão não mostra o formulário", async () => {
    mockBackend({
      ...rotas(),
      ...identityRoutes({ permissions: [], scopes: [] }),
    });
    renderApp(<ProdutosPage />);
    expect(await screen.findByText("Protocolo")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cadastrar produto" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Editar Protocolo" })).not.toBeInTheDocument();
  });
});
