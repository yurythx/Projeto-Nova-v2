import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { diaDoPrazo, situacaoDoPedido } from "@/components/protocolos/AcoesDoPedido";
import { identityRoutes, ME, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import FichaPage from "./[id]/page";
import ProtocolosPage from "./page";

const BASE = {
  nome: "Cidadão",
  servico_id: "s1",
  servico_nome: "Alvará de funcionamento",
  unidade_id: "u1",
  unidade_nome: "PREFEITURA",
  etapa_id: "triagem",
  prioridade: "normal",
  situacao_prazo: "no_prazo",
  created_at: "2026-11-20T12:00:00Z",
  decisao: "",
  justificativa: "",
  complemento: "",
};

const PEDIDOS = [
  { ...BASE, id: "a1", numero: "PREF202690101", status: "novo", exige_aceite: true },
  {
    ...BASE,
    id: "a2",
    numero: "PREF202690102",
    status: "em_andamento",
    responsavel_id: ME.id,
    responsavel_nome: "Maria Servidora",
    prazo_em: "2026-12-05T03:59:00Z",
  },
  {
    ...BASE,
    id: "a3",
    numero: "PREF202690103",
    status: "em_analise",
    responsavel_id: "outra",
    responsavel_nome: "Ana",
    encaminhamento: { unidade_id: "u2", unidade_nome: "SMS", justificativa: "é da saúde" },
  },
  {
    ...BASE,
    id: "a4",
    numero: "PREF202690104",
    status: "pending_approval",
    decisao: "favoravel",
    justificativa: "deferido",
    responsavel_id: "outra",
    responsavel_nome: "Ana",
  },
];

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    ativo: true,
    unidades: [
      { id: "u1", nome: "Prefeitura", sigla: "PREF", ativo: true, departamentos: [] },
      { id: "u2", nome: "Saúde", sigla: "SMS", ativo: true, departamentos: [] },
      { id: "u3", nome: "Antiga", sigla: "", ativo: false, departamentos: [] },
    ],
  },
];

function rotas(permissoes: string[]) {
  const ok = { data: {} };
  return mockBackend({
    ...identityRoutes({ permissions: permissoes }),
    "GET v1/protocolos/pedidos": { data: PEDIDOS },
    "GET v1/iam/org-tree": { data: ORG },
    "GET v1/protocolos/pedidos/a1/pessoas": {
      data: [
        { id: "pa", nome: "Ana" },
        { id: "pb", nome: "Bruno" },
      ],
    },
    "GET v1/protocolos/pedidos/a2/pessoas": { data: [{ id: "pa", nome: "Ana" }] },
    "POST v1/protocolos/pedidos/*": ok,
  });
}

async function menu(user: ReturnType<typeof userEvent.setup>, numero: string) {
  await user.click(await screen.findByRole("button", { name: `Ações do protocolo ${numero}` }));
  return screen.getByRole("menu");
}

describe("Aceite, encaminhamento e aprovação na lista", () => {
  beforeEach(() => resetNavigation({}, "/protocolos"));

  it("filtra por responsável e mostra os selos", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar"]);
    renderApp(<ProtocolosPage />);
    expect(await screen.findByText("Responsável: Maria Servidora")).toBeInTheDocument();
    expect(screen.getByText(/Encaminhamento para\s+SMS/)).toBeInTheDocument();
    expect(screen.getByText("Aguardando aceite")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Meus protocolos/ }));
    expect(screen.getByText("Protocolo #PREF202690102")).toBeInTheDocument();
    expect(screen.queryByText("Protocolo #PREF202690101")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Sem responsável/ }));
    expect(screen.getByText("Protocolo #PREF202690101")).toBeInTheDocument();
    expect(screen.queryByText("Protocolo #PREF202690102")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Encaminhamentos/ }));
    expect(screen.getByText("Protocolo #PREF202690103")).toBeInTheDocument();
    expect(screen.queryByText("Protocolo #PREF202690104")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Todos\s*4/ }));
    expect(screen.getByText("Protocolo #PREF202690104")).toBeInTheDocument();
  });

  it("aceita indicando alguém, trava o tratamento sem dono e transfere", async () => {
    const user = userEvent.setup();
    const api = rotas(["protocolos:tratar", "protocolos:gerir"]);
    renderApp(<ProtocolosPage />);

    let m = await menu(user, "PREF202690101");
    expect(within(m).getByRole("menuitem", { name: "Adicionar nota" })).toBeDisabled();
    expect(within(m).getByRole("menuitem", { name: "Concluir" })).toHaveAttribute(
      "title",
      "Aceite o protocolo para liberar esta ação",
    );
    expect(
      within(m).getByRole("menuitem", { name: "Encaminhar para a próxima etapa" }),
    ).toBeDisabled();
    await user.click(within(m).getByRole("menuitem", { name: "Aceitar" }));
    let modal = await screen.findByRole("dialog");
    const quem = within(modal).getByLabelText("Quem assume");
    expect(await within(quem).findByRole("option", { name: "Bruno" })).toBeInTheDocument();
    await user.selectOptions(quem, "pb");
    await user.click(within(modal).getByRole("button", { name: "Aceitar responsabilidade" }));
    expect(api.to("POST v1/protocolos/pedidos/a1/aceitar")[0]?.body).toEqual({
      responsavel_id: "pb",
    });

    m = await menu(user, "PREF202690102");
    expect(within(m).queryByRole("menuitem", { name: "Aceitar" })).not.toBeInTheDocument();
    await user.click(within(m).getByRole("menuitem", { name: "Transferir" }));
    modal = await screen.findByRole("dialog");
    const novo = within(modal).getByLabelText("Novo responsável *");
    await within(novo).findByRole("option", { name: "Ana" });
    await user.selectOptions(novo, "pa");
    await user.click(within(modal).getByRole("button", { name: "Transferir" }));
    expect(api.to("POST v1/protocolos/pedidos/a2/transferir")[0]?.body).toEqual({
      responsavel_id: "pa",
    });
  });

  it("encaminha para outra unidade e edita prioridade e prazo", async () => {
    const user = userEvent.setup();
    const api = rotas(["protocolos:tratar"]);
    renderApp(<ProtocolosPage />);

    let m = await menu(user, "PREF202690102");
    await user.click(within(m).getByRole("menuitem", { name: "Encaminhar para outra unidade" }));
    let modal = await screen.findByRole("dialog");
    const destino = within(modal).getByLabelText("Unidade de destino *");
    await within(destino).findByRole("option", { name: "SMS · Saúde" });
    expect(
      within(destino).queryByRole("option", { name: "PREF · Prefeitura" }),
    ).not.toBeInTheDocument();
    expect(within(destino).queryByRole("option", { name: "Antiga" })).not.toBeInTheDocument();
    await user.selectOptions(destino, "u2");
    await user.clear(within(modal).getByLabelText("Prazo no destino (dias úteis)"));
    await user.type(within(modal).getByLabelText("Prazo no destino (dias úteis)"), "5");
    await user.type(within(modal).getByLabelText("Justificativa *"), "é da saúde");
    await user.click(within(modal).getByRole("button", { name: "Encaminhar" }));
    expect(api.to("POST v1/protocolos/pedidos/a2/encaminhar")[0]?.body).toEqual({
      unidade_id: "u2",
      justificativa: "é da saúde",
      prazo_dias: 5,
    });

    m = await menu(user, "PREF202690102");
    await user.click(within(m).getByRole("menuitem", { name: "Editar" }));
    modal = await screen.findByRole("dialog");
    expect(within(modal).getByLabelText("Prazo")).toHaveValue("2026-12-04");
    await user.selectOptions(within(modal).getByLabelText("Prioridade"), "urgente");
    await user.type(within(modal).getByLabelText("Justificativa da edição *"), "cidadão idoso");
    await user.click(within(modal).getByRole("button", { name: "Salvar" }));
    expect(api.to("POST v1/protocolos/pedidos/a2/editar")[0]?.body).toEqual({
      justificativa: "cidadão idoso",
      prioridade: "urgente",
      prazo: "2026-12-04",
    });
  });

  it("decide o encaminhamento e aprova ou devolve a conclusão", async () => {
    const user = userEvent.setup();
    const api = rotas(["protocolos:tratar", "protocolos:gerir"]);
    renderApp(<ProtocolosPage />);

    let m = await menu(user, "PREF202690103");
    expect(within(m).queryByRole("menuitem", { name: "Editar" })).not.toBeInTheDocument();
    await user.click(within(m).getByRole("menuitem", { name: "Aceitar encaminhamento" }));
    expect(api.to("POST v1/protocolos/pedidos/a3/encaminhamento/aceitar")).toHaveLength(1);
    m = await menu(user, "PREF202690103");
    await user.click(within(m).getByRole("menuitem", { name: "Recusar encaminhamento" }));
    let modal = await screen.findByRole("dialog");
    await user.type(within(modal).getByLabelText("Motivo da recusa *"), "não é nosso");
    await user.click(within(modal).getByRole("button", { name: "Recusar encaminhamento" }));
    expect(api.to("POST v1/protocolos/pedidos/a3/encaminhamento/recusar")[0]?.body).toEqual({
      mensagem: "não é nosso",
    });

    m = await menu(user, "PREF202690104");
    await user.click(within(m).getByRole("menuitem", { name: "Aprovar conclusão" }));
    expect(api.to("POST v1/protocolos/pedidos/a4/aprovar")).toHaveLength(1);
    m = await menu(user, "PREF202690104");
    await user.click(within(m).getByRole("menuitem", { name: "Devolver para revisão" }));
    modal = await screen.findByRole("dialog");
    await user.type(within(modal).getByLabelText("O que precisa ser revisto *"), "falta o parecer");
    await user.click(within(modal).getByRole("button", { name: "Devolver" }));
    expect(api.to("POST v1/protocolos/pedidos/a4/devolver")[0]?.body).toEqual({
      mensagem: "falta o parecer",
    });
  });

  it("sem gerir, a conclusão que aguarda não tem ações", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar"]);
    renderApp(<ProtocolosPage />);
    const m = await menu(user, "PREF202690104");
    expect(
      within(m).queryByRole("menuitem", { name: "Aprovar conclusão" }),
    ).not.toBeInTheDocument();
    expect(within(m).getAllByRole("menuitem")).toHaveLength(1);
  });
});

function ficha(pedido: Record<string, unknown>) {
  return {
    pedido: {
      ...BASE,
      id: "a1",
      numero: "PREF202690101",
      status: "novo",
      campos: [],
      respostas: {},
      ...pedido,
    },
    historico: [
      {
        id: "h1",
        acao: "aceite",
        descricao: "assumido por Ana",
        status: "em_andamento",
        created_at: "2026-11-20T13:00:00Z",
      },
      {
        id: "h2",
        acao: "aprovacao",
        descricao: "conclusão aprovada",
        status: "concluido",
        created_at: "2026-11-20T14:00:00Z",
      },
    ],
    etapas: [],
  };
}

function rotasDaFicha(permissoes: string[], pedido: Record<string, unknown>) {
  return mockBackend({
    ...identityRoutes({ permissions: permissoes }),
    "GET v1/protocolos/pedidos/a1": { data: ficha(pedido) },
    "GET v1/iam/org-tree": { data: ORG },
    "POST v1/protocolos/pedidos/*": { data: {} },
  });
}

describe("Aceite, encaminhamento e aprovação na ficha", () => {
  beforeEach(() => resetNavigation({ id: "a1" }, "/protocolos/a1"));

  it("sem dono e com aceite exigido, pede o aceite e trava o resto", async () => {
    const user = userEvent.setup();
    const api = rotasDaFicha(["protocolos:tratar"], { exige_aceite: true });
    renderApp(<FichaPage />);
    expect(
      await screen.findByText(
        "Este serviço exige que alguém aceite o protocolo antes de tratá-lo.",
      ),
    ).toBeInTheDocument();
    expect(screen.getAllByText("Aguardando aceite").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Adicionar nota" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Concluir protocolo" })).toBeDisabled();
    expect(screen.getByText("Responsabilidade Aceita")).toBeInTheDocument();
    expect(screen.getByText("assumido por Ana")).toBeInTheDocument();
    expect(screen.getByText("Conclusão Aprovada")).toBeInTheDocument();
    await user.click(screen.getAllByRole("button", { name: "Aceitar" })[0]!);
    const modal = await screen.findByRole("dialog");
    expect(within(modal).getByText(/Você assumirá diretamente/)).toBeInTheDocument();
    await user.click(within(modal).getByRole("button", { name: "Aceitar responsabilidade" }));
    expect(api.to("POST v1/protocolos/pedidos/a1/aceitar")[0]?.body).toEqual({});
  });

  it("com dono, transfere, encaminha e edita prioridade e prazo", async () => {
    const user = userEvent.setup();
    const api = rotasDaFicha(["protocolos:tratar"], {
      status: "em_andamento",
      responsavel_id: ME.id,
      responsavel_nome: "Maria Servidora",
      prazo_em: "2026-12-05T03:59:00Z",
    });
    renderApp(<FichaPage />);
    expect(await screen.findByText("Maria Servidora")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Transferir" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Encaminhar para outra unidade" }));
    let modal = await screen.findByRole("dialog");
    expect(
      within(modal).getByRole("heading", { name: "Encaminhar para Outra Unidade" }),
    ).toBeInTheDocument();
    await user.click(within(modal).getByRole("button", { name: "Cancelar" }));

    await user.click(screen.getByRole("button", { name: "Editar dados" }));
    modal = await screen.findByRole("dialog");
    expect(within(modal).getByLabelText("Prazo")).toHaveValue("2026-12-04");
    await user.selectOptions(within(modal).getByLabelText("Prioridade"), "alta");
    await user.type(
      within(modal).getByLabelText("Justificativa da edição *"),
      "pedido do gabinete",
    );
    await user.click(within(modal).getByRole("button", { name: "Salvar edição" }));
    expect(api.to("POST v1/protocolos/pedidos/a1/editar")[0]?.body).toEqual({
      justificativa: "pedido do gabinete",
      prioridade: "alta",
      prazo: "2026-12-04",
    });
  });

  it("com encaminhamento pendente, o destino aceita ou recusa", async () => {
    const user = userEvent.setup();
    const api = rotasDaFicha(["protocolos:tratar"], {
      status: "em_analise",
      encaminhamento: {
        unidade_id: "u2",
        unidade_nome: "SMS",
        justificativa: "é da saúde",
        prazo_dias: 3,
      },
    });
    renderApp(<FichaPage />);
    expect(await screen.findByText(/Encaminhamento para SMS/)).toBeInTheDocument();
    expect(screen.getByText(/Prazo no destino: 3 dias úteis/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Adicionar nota" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Aceitar encaminhamento" }));
    expect(api.to("POST v1/protocolos/pedidos/a1/encaminhamento/aceitar")).toHaveLength(1);
    await user.click(screen.getByRole("button", { name: "Recusar" }));
    expect(await screen.findByRole("dialog")).toHaveTextContent("Recusar Encaminhamento");
  });

  it("com conclusão aguardando, quem gere aprova e quem trata só vê o aviso", async () => {
    const user = userEvent.setup();
    const aguardando = {
      status: "pending_approval",
      decisao: "desfavoravel",
      justificativa: "sem documentos",
      responsavel_id: "outra",
      responsavel_nome: "Ana",
    };
    const api = rotasDaFicha(["protocolos:tratar", "protocolos:gerir"], aguardando);
    const { unmount } = renderApp(<FichaPage />);
    expect(
      await screen.findByText(/Conclusão aguardando aprovação · Desfavorável/),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Aprovar conclusão" }));
    expect(api.to("POST v1/protocolos/pedidos/a1/aprovar")).toHaveLength(1);
    await user.click(screen.getByRole("button", { name: "Devolver para revisão" }));
    expect(await screen.findByRole("dialog")).toHaveTextContent("Devolver a Conclusão");
    unmount();

    rotasDaFicha(["protocolos:tratar"], { ...aguardando, decisao: "favoravel" });
    renderApp(<FichaPage />);
    expect(
      await screen.findByText(/Conclusão aguardando aprovação · Favorável/),
    ).toBeInTheDocument();
    expect(screen.getByText("Quem gere a unidade aprova ou devolve.")).toBeInTheDocument();
  });
});

describe("auxiliares das ações", () => {
  it("lê a situação e o dia do prazo", () => {
    expect(situacaoDoPedido({ status: "novo", exige_aceite: true }).travado).toBe(true);
    expect(situacaoDoPedido({ status: "concluido", responsavel_id: "x" })).toMatchObject({
      final: true,
      semDono: false,
      travado: false,
    });
    expect(diaDoPrazo(undefined)).toBe("");
    expect(diaDoPrazo("x")).toBe("");
    expect(diaDoPrazo("2026-12-05T03:59:00Z")).toBe("2026-12-04");
  });
});
