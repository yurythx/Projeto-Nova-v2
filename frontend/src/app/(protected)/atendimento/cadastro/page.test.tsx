import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";

import AtendimentoCadastroPage from "./page";

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "Tecnologia", sigla: "TI", departamentos: [] }],
  },
];

const ofertaAtiva = {
  id: "o1",
  nome: "E-mail",
  descricao: "acesso",
  tipo: "incidente",
  produto_id: "p1",
  unidade_id: "u1",
  ativa: true,
};
const ofertaInativa = {
  id: "o2",
  nome: "Pedido geral",
  descricao: "",
  tipo: "requisicao",
  ativa: false,
};
const equipeComMembro = { id: "eq1", nome: "N1", unidade_id: "u1", membros: ["c1"] };
const equipeVazia = { id: "eq2", nome: "Plantão", membros: [] as string[] };

function rotas(extra: Record<string, unknown> = {}, me: Parameters<typeof identityRoutes>[0] = {}) {
  return {
    ...identityRoutes(me),
    "GET v1/iam/org-tree": { data: ORG },
    "GET v1/produtos": { data: [{ id: "p1", nome: "Correio" }] },
    "GET v1/atendimento/ofertas": { data: [ofertaAtiva, ofertaInativa] },
    "GET v1/atendimento/equipes": { data: [equipeComMembro, equipeVazia] },
    "GET v1/atendimento/candidatos": {
      data: [
        { id: "c1", nome: "Maria" },
        { id: "c2", nome: "João" },
      ],
    },
    "POST v1/atendimento/ofertas": { data: ofertaAtiva },
    "PUT v1/atendimento/ofertas/o2": { data: ofertaInativa },
    "POST v1/atendimento/equipes": { data: equipeComMembro },
    "PUT v1/atendimento/equipes/eq1": { data: equipeComMembro },
    ...extra,
  };
}

function cartao(nome: string) {
  return screen.getByRole("heading", { name: nome }).closest("div.rounded-xl") as HTMLElement;
}

describe("Atendimento — ofertas e equipes", () => {
  it("cadastra oferta e equipe e corrige as duas", async () => {
    const user = userEvent.setup();
    const api = mockBackend(rotas());
    renderApp(<AtendimentoCadastroPage />);

    expect(await screen.findByText("acesso")).toBeInTheDocument();
    expect(screen.getByText("Ativa")).toBeInTheDocument();
    expect(screen.getByText("Inativa")).toBeInTheDocument();
    expect(screen.getByText(/Incidente · Vinculada a um produto/)).toBeInTheDocument();
    expect(screen.getByText(/Requisição · Sem produto/)).toBeInTheDocument();
    expect(screen.getByText("1 membro(s)")).toBeInTheDocument();
    expect(screen.getByText("Sem membros")).toBeInTheDocument();
    expect(
      screen.getByText("Escolha a unidade para ver quem pode entrar na equipe."),
    ).toBeInTheDocument();

    const nova = cartao("Nova oferta");
    await user.type(within(nova).getByLabelText("Nome *"), "Ramal");
    await user.click(within(nova).getByLabelText("Oferta ativa"));
    await user.click(within(nova).getByRole("button", { name: "Cadastrar oferta" }));
    await waitFor(() => expect(api.to("POST v1/atendimento/ofertas")).toHaveLength(1));
    expect(api.to("POST v1/atendimento/ofertas")[0]!.body).toEqual({
      nome: "Ramal",
      descricao: "",
      tipo: "requisicao",
      produto_id: null,
      unidade_id: null,
      ativa: false,
    });

    await user.click(screen.getByRole("button", { name: "Editar Pedido geral" }));
    const edicao = cartao("Editar oferta");
    expect(within(edicao).getByLabelText("Oferta ativa")).not.toBeChecked();
    await user.click(within(edicao).getByLabelText("Oferta ativa"));
    expect(within(edicao).getByLabelText("Tipo")).toHaveValue("requisicao");
    await user.selectOptions(within(edicao).getByLabelText("Tipo"), "incidente");
    expect(await within(edicao).findByRole("option", { name: "Correio" })).toBeInTheDocument();
    await user.selectOptions(within(edicao).getByLabelText("Produto"), "p1");
    expect(
      await within(edicao).findByRole("option", { name: "TI — Tecnologia" }),
    ).toBeInTheDocument();
    await user.selectOptions(within(edicao).getByLabelText("Unidade da oferta"), "u1");
    await user.click(within(edicao).getByRole("button", { name: "Salvar oferta" }));
    await waitFor(() => expect(api.to("PUT v1/atendimento/ofertas/o2")).toHaveLength(1));
    expect(api.to("PUT v1/atendimento/ofertas/o2")[0]!.body).toMatchObject({
      tipo: "incidente",
      produto_id: "p1",
      unidade_id: "u1",
      ativa: true,
    });

    await user.click(screen.getByRole("button", { name: "Editar E-mail" }));
    await user.click(screen.getByRole("button", { name: "Cancelar edição da oferta" }));
    expect(screen.getByRole("heading", { name: "Nova oferta" })).toBeInTheDocument();

    const equipe = cartao("Nova equipe");
    await user.type(within(equipe).getByLabelText("Nome *"), "N2");
    expect(
      await within(equipe).findByRole("option", { name: "TI — Tecnologia" }),
    ).toBeInTheDocument();
    await user.selectOptions(within(equipe).getByLabelText("Unidade da equipe"), "u1");
    const maria = await within(equipe).findByRole("checkbox", { name: "Maria" });
    expect(within(equipe).getByRole("checkbox", { name: "João" })).not.toBeChecked();
    await user.click(maria);
    await user.click(within(equipe).getByRole("button", { name: "Cadastrar equipe" }));
    await waitFor(() => expect(api.to("POST v1/atendimento/equipes")).toHaveLength(1));
    expect(api.to("POST v1/atendimento/equipes")[0]!.body).toEqual({
      nome: "N2",
      unidade_id: "u1",
      membros: ["c1"],
    });

    await user.click(screen.getByRole("button", { name: "Editar N1" }));
    const equipeEdit = cartao("Editar equipe");
    expect(await within(equipeEdit).findByRole("checkbox", { name: "Maria" })).toBeChecked();
    await user.click(within(equipeEdit).getByRole("button", { name: "Salvar equipe" }));
    await waitFor(() => expect(api.to("PUT v1/atendimento/equipes/eq1")).toHaveLength(1));
    expect(api.to("PUT v1/atendimento/equipes/eq1")[0]!.body).toMatchObject({
      membros: ["c1"],
      unidade_id: "u1",
    });

    await user.click(screen.getByRole("button", { name: "Editar Plantão" }));
    await user.click(screen.getByRole("button", { name: "Cancelar edição da equipe" }));
    expect(
      screen.getByText("Escolha a unidade para ver quem pode entrar na equipe."),
    ).toBeInTheDocument();
  });

  it("unidade sem servidores lotados", async () => {
    const user = userEvent.setup();
    mockBackend(rotas({ "GET v1/atendimento/candidatos": { data: [] } }));
    renderApp(<AtendimentoCadastroPage />);
    const equipe = await screen.findByRole("heading", { name: "Nova equipe" });
    const card = equipe.closest("div.rounded-xl") as HTMLElement;
    expect(
      await within(card).findByRole("option", { name: "TI — Tecnologia" }),
    ).toBeInTheDocument();
    await user.selectOptions(within(card).getByLabelText("Unidade da equipe"), "u1");
    expect(
      await within(card).findByText("Nenhum servidor lotado nesta unidade."),
    ).toBeInTheDocument();
  });

  it("sem o cadastro de produtos a oferta segue sem vínculo", async () => {
    const api = mockBackend(
      rotas(
        {},
        {
          permissions: ["atendimento:manage"],
          scopes: [{ perfil: "p", origem: "manual", permissions: ["atendimento:manage"] }],
        },
      ),
    );
    renderApp(<AtendimentoCadastroPage />);
    expect(
      await screen.findByRole("option", { name: "Sem produto vinculado" }),
    ).toBeInTheDocument();
    expect(api.to("GET v1/produtos")).toHaveLength(0);
  });

  it("recusa gravar e mantém o nome", async () => {
    const user = userEvent.setup();
    mockBackend(
      rotas({
        "POST v1/atendimento/ofertas": fail(422, "oferta inválida"),
        "POST v1/atendimento/equipes": fail(422, "equipe inválida"),
      }),
    );
    renderApp(<AtendimentoCadastroPage />);
    const oferta = (await screen.findByRole("heading", { name: "Nova oferta" })).closest(
      "div.rounded-xl",
    ) as HTMLElement;
    await user.type(within(oferta).getByLabelText("Nome *"), "Ramal");
    await user.click(within(oferta).getByRole("button", { name: "Cadastrar oferta" }));
    expect(await screen.findByText("oferta inválida")).toBeInTheDocument();
    expect(within(oferta).getByLabelText("Nome *")).toHaveValue("Ramal");

    const equipe = cartao("Nova equipe");
    await user.type(within(equipe).getByLabelText("Nome *"), "N2");
    await user.click(within(equipe).getByRole("button", { name: "Cadastrar equipe" }));
    expect(await screen.findByText("equipe inválida")).toBeInTheDocument();
    expect(within(equipe).getByLabelText("Nome *")).toHaveValue("N2");
  });

  it("listas vazias, falha de leitura e tela sem gestão", async () => {
    mockBackend(
      rotas({
        "GET v1/atendimento/ofertas": { data: [] },
        "GET v1/atendimento/equipes": { data: [] },
      }),
    );
    const vazio = renderApp(<AtendimentoCadastroPage />);
    expect(await screen.findByText("Nenhuma oferta cadastrada")).toBeInTheDocument();
    expect(screen.getByText("Nenhuma equipe cadastrada")).toBeInTheDocument();
    vazio.unmount();

    mockBackend(rotas({ "GET v1/atendimento/ofertas": fail(500, "banco fora") }));
    const erro = renderApp(<AtendimentoCadastroPage />);
    expect(await screen.findByText("banco fora")).toBeInTheDocument();
    erro.unmount();

    mockBackend(rotas({}, { permissions: [], scopes: [] }));
    renderApp(<AtendimentoCadastroPage />);
    expect(await screen.findByText("E-mail")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cadastrar oferta" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Editar N1" })).not.toBeInTheDocument();
  });
});
