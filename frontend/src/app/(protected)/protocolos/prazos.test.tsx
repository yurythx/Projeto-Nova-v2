import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { podeAprovar } from "@/components/protocolos/AcoesDoPedido";
import { hojeEmCuiaba } from "@/components/protocolos/HorarioLivre";
import { fail, identityRoutes, ME, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import FichaPage from "./[id]/page";
import FeriadosPage from "./feriados/page";
import ProtocolosPage from "./page";
import ServicosPage from "./servicos/page";

const BASE = {
  nome: "Cidadão",
  servico_id: "s1",
  servico_nome: "Alvará de funcionamento",
  unidade_id: "u1",
  unidade_nome: "PREFEITURA",
  etapa_id: "",
  prioridade: "normal",
  situacao_prazo: "no_prazo",
  created_at: "2026-11-20T12:00:00Z",
  decisao: "",
  justificativa: "",
  complemento: "",
  responsavel_id: ME.id,
  responsavel_nome: "Maria Servidora",
};

const PEDIDOS = [
  {
    ...BASE,
    id: "b1",
    numero: "PREF202690201",
    status: "em_complementacao",
    complemento: "envie o RG",
    complemento_prazo: "2026-12-03T03:59:00Z",
  },
  {
    ...BASE,
    id: "b2",
    numero: "PREF202690202",
    status: "em_andamento",
    processo_sei: "12345.000001/2026-11",
  },
  {
    ...BASE,
    id: "b3",
    numero: "PREF202690203",
    status: "pending_approval",
    decisao: "favoravel",
    justificativa: "deferido",
    aprovador_id: "outra-pessoa",
  },
];

function rotas(permissoes: string[], extra: Record<string, unknown> = {}) {
  return mockBackend({
    ...identityRoutes({ permissions: permissoes }),
    "GET v1/protocolos/pedidos": { data: PEDIDOS },
    "POST v1/protocolos/pedidos/b2/anexo": {
      data: {
        upload_url: "https://minio/up",
        method: "PUT",
        object_key: "protocolos/mesa/b2/a.pdf",
        headers: {},
      },
    },
    "PUT minio/up": { data: {} },
    "POST v1/protocolos/pedidos/*": { data: {} },
    ...(extra as Record<string, never>),
  });
}

async function menu(user: ReturnType<typeof userEvent.setup>, numero: string) {
  await user.click(await screen.findByRole("button", { name: `Ações do protocolo ${numero}` }));
  return screen.getByRole("menu");
}

describe("SEI, prazo da complementação e aprovação na lista", () => {
  beforeEach(() => resetNavigation({}, "/protocolos"));

  it("mostra os selos, filtra por SEI e por aprovação", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar", "protocolos:gerir"]);
    renderApp(<ProtocolosPage />);
    expect(await screen.findByText("SEI 12345.000001/2026-11")).toBeInTheDocument();
    expect(screen.getByText("Resposta até 02/12/2026")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Feriados" })).toHaveAttribute(
      "href",
      "/protocolos/feriados",
    );

    await user.selectOptions(screen.getByLabelText("Processo SEI"), "com");
    expect(screen.getByText("Protocolo #PREF202690202")).toBeInTheDocument();
    expect(screen.queryByText("Protocolo #PREF202690201")).not.toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Processo SEI"), "sem");
    expect(screen.queryByText("Protocolo #PREF202690202")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Limpar" }));
    expect(screen.getByText("Protocolo #PREF202690202")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Aguardando aprovação/ }));
    expect(screen.getByText("Protocolo #PREF202690203")).toBeInTheDocument();
    expect(screen.queryByText("Protocolo #PREF202690202")).not.toBeInTheDocument();
    // O aprovador do serviço é outra pessoa: quem gere não aprova.
    const m = await menu(user, "PREF202690203");
    expect(
      within(m).queryByRole("menuitem", { name: "Aprovar conclusão" }),
    ).not.toBeInTheDocument();
  });

  it("pede complementação com prazo e anexo, e conclui com processo SEI", async () => {
    const user = userEvent.setup();
    const api = rotas(["protocolos:tratar"]);
    renderApp(<ProtocolosPage />);

    let m = await menu(user, "PREF202690202");
    await user.click(within(m).getByRole("menuitem", { name: "Solicitar complementação" }));
    let modal = await screen.findByRole("dialog");
    await user.clear(within(modal).getByLabelText(/Prazo para o cidadão responder/));
    await user.type(within(modal).getByLabelText(/Prazo para o cidadão responder/), "5");
    await user.type(within(modal).getByLabelText("O que o cidadão precisa enviar *"), "envie o RG");
    await user.upload(
      within(modal).getByLabelText(/Anexos/),
      new File(["x"], "modelo.pdf", { type: "application/pdf" }),
    );
    expect(await within(modal).findByText("modelo.pdf")).toBeInTheDocument();
    await user.click(within(modal).getByRole("button", { name: "Tirar modelo.pdf" }));
    expect(within(modal).queryByText("modelo.pdf")).not.toBeInTheDocument();
    await user.upload(
      within(modal).getByLabelText(/Anexos/),
      new File(["x"], "modelo.pdf", { type: "application/pdf" }),
    );
    await within(modal).findByText("modelo.pdf");
    await user.click(within(modal).getByRole("button", { name: "Confirmar" }));
    expect(api.to("POST v1/protocolos/pedidos/b2/complemento")[0]?.body).toEqual({
      mensagem: "envie o RG",
      prazo_dias: 5,
      anexos: ["protocolos/mesa/b2/a.pdf"],
    });

    m = await menu(user, "PREF202690202");
    await user.click(within(m).getByRole("menuitem", { name: "Concluir" }));
    modal = await screen.findByRole("dialog");
    expect(within(modal).getByLabelText("Processo SEI")).toHaveValue("12345.000001/2026-11");
    await user.clear(within(modal).getByLabelText("Processo SEI"));
    await user.type(within(modal).getByLabelText("Processo SEI"), "54321.000002/2026-22");
    await user.selectOptions(within(modal).getByLabelText("Decisão *"), "favoravel");
    await user.type(within(modal).getByLabelText("Justificativa *"), "Decisão no SEI, doc. 123");
    await user.click(within(modal).getByRole("button", { name: "Confirmar" }));
    expect(api.to("POST v1/protocolos/pedidos/b2/concluir")[0]?.body).toEqual({
      decisao: "favoravel",
      justificativa: "Decisão no SEI, doc. 123",
      processo_sei: "54321.000002/2026-22",
    });
  });

  it("mostra a falha do envio do anexo", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar"], {
      "POST v1/protocolos/pedidos/b2/anexo": fail(503, "Armazenamento fora do ar"),
    });
    renderApp(<ProtocolosPage />);
    const m = await menu(user, "PREF202690202");
    await user.click(within(m).getByRole("menuitem", { name: "Adicionar nota" }));
    const modal = await screen.findByRole("dialog");
    await user.upload(
      within(modal).getByLabelText(/Anexos/),
      new File(["x"], "a.pdf", { type: "application/pdf" }),
    );
    expect(await within(modal).findByRole("alert")).toHaveTextContent("Armazenamento fora do ar");
  });
});

function ficha(pedido: Record<string, unknown>) {
  return {
    pedido: { ...BASE, id: "b2", numero: "PREF202690202", campos: [], respostas: {}, ...pedido },
    historico: [],
    etapas: [
      { id: "a", ordem: 1, status: "em_analise", prazo_dias: 1 },
      { id: "b", ordem: 2, status: "em_andamento", prazo_dias: 1 },
    ],
  };
}

describe("SEI, prazo e aprovador na ficha", () => {
  beforeEach(() => resetNavigation({ id: "b2" }, "/protocolos/b2"));

  it("mostra o SEI, edita o vínculo e avança com anexo", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"] }),
      "GET v1/protocolos/pedidos/b2": {
        data: ficha({
          status: "em_andamento",
          etapa_id: "a",
          processo_sei: "12345.000001/2026-11",
        }),
      },
      "GET v1/iam/org-tree": { data: [] },
      "POST v1/protocolos/pedidos/b2/anexo": {
        data: {
          upload_url: "https://minio/up",
          method: "PUT",
          object_key: "protocolos/mesa/b2/p.pdf",
          headers: {},
        },
      },
      "PUT minio/up": { data: {} },
      "POST v1/protocolos/pedidos/*": { data: {} },
    });
    renderApp(<FichaPage />);
    expect(await screen.findByText("12345.000001/2026-11")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Encaminhar para a próxima etapa" }));
    let modal = await screen.findByRole("dialog");
    await user.upload(
      within(modal).getByLabelText(/Anexos/),
      new File(["x"], "parecer.pdf", { type: "application/pdf" }),
    );
    await within(modal).findByText("parecer.pdf");
    await user.click(within(modal).getByRole("button", { name: "Avançar" }));
    expect(api.to("POST v1/protocolos/pedidos/b2/avancar")[0]?.body).toEqual({
      anexos: ["protocolos/mesa/b2/p.pdf"],
    });

    await user.click(screen.getByRole("button", { name: "Editar dados" }));
    modal = await screen.findByRole("dialog");
    await user.clear(within(modal).getByLabelText("Processo SEI"));
    await user.type(
      within(modal).getByLabelText("Justificativa da edição *"),
      "o SEI foi arquivado",
    );
    await user.click(within(modal).getByRole("button", { name: "Salvar edição" }));
    expect(api.to("POST v1/protocolos/pedidos/b2/editar")[0]?.body).toMatchObject({
      processo_sei: "",
    });
  });

  it("mostra o prazo da complementação e só o aprovador indicado aprova", async () => {
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"] }),
      "GET v1/protocolos/pedidos/b2": {
        data: ficha({
          status: "em_complementacao",
          complemento: "envie o RG",
          complemento_prazo: "2026-12-03T03:59:00Z",
        }),
      },
      "GET v1/iam/org-tree": { data: [] },
    });
    const { unmount } = renderApp(<FichaPage />);
    expect(
      await screen.findByText(/Prazo para o cidadão responder: até 02\/12\/2026/),
    ).toBeInTheDocument();
    expect(screen.getByText("Não vinculado")).toBeInTheDocument();
    unmount();

    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"] }),
      "GET v1/protocolos/pedidos/b2": {
        data: ficha({
          status: "pending_approval",
          decisao: "favoravel",
          justificativa: "ok",
          aprovador_id: ME.id,
        }),
      },
      "GET v1/iam/org-tree": { data: [] },
    });
    renderApp(<FichaPage />);
    expect(await screen.findByRole("button", { name: "Aprovar conclusão" })).toBeInTheDocument();
  });
});

describe("Feriados", () => {
  beforeEach(() => resetNavigation({}, "/protocolos/feriados"));

  it("lista em ordem, adiciona e remove", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes({ permissions: ["protocolos:gerir"] }),
      "GET v1/protocolos/feriados": {
        data: [
          { data: "2026-12-25", nome: "Natal" },
          { data: "2026-12-10", nome: "Aniversário da cidade" },
        ],
      },
      "POST v1/protocolos/feriados": {
        status: 201,
        data: { data: "2026-11-20", nome: "Consciência Negra" },
      },
      "DELETE v1/protocolos/feriados/2026-12-25": { status: 204 },
    });
    renderApp(<FeriadosPage />);
    const itens = await screen.findAllByRole("listitem");
    expect(itens[0]).toHaveTextContent("Aniversário da cidade");
    expect(itens[0]).toHaveTextContent("10/12/2026 · quinta");
    await user.type(screen.getByLabelText("Data *"), "2026-11-20");
    await user.type(screen.getByLabelText("Nome *"), "Consciência Negra");
    await user.click(screen.getByRole("button", { name: /Adicionar/ }));
    expect(api.to("POST v1/protocolos/feriados")[0]?.body).toEqual({
      data: "2026-11-20",
      nome: "Consciência Negra",
    });
    await user.click(screen.getByRole("button", { name: "Remover Natal" }));
    expect(api.to("DELETE v1/protocolos/feriados/2026-12-25")).toHaveLength(1);
  });

  it("sem gerir só lê; sem feriado mostra o vazio", async () => {
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"], scopes: [] }),
      "GET v1/protocolos/feriados": { data: [] },
    });
    renderApp(<FeriadosPage />);
    expect(await screen.findByText("Nenhum feriado cadastrado")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Adicionar/ })).not.toBeInTheDocument();
  });
});

describe("Serviço: carta e aprovador", () => {
  beforeEach(() => resetNavigation({}, "/protocolos/servicos"));

  it("grava os detalhes da carta e quem aprova", async () => {
    const user = userEvent.setup();
    const ORG = [
      {
        id: "e1",
        nome: "Prefeitura",
        ativo: true,
        unidades: [
          {
            id: "u1",
            entidade_id: "e1",
            nome: "Prefeitura",
            sigla: "PREF",
            ativo: true,
            departamentos: [],
          },
        ],
      },
    ];
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/protocolos/servicos": { data: [] },
      "GET v1/protocolos/formularios": { data: [{ id: "f1", unidade_id: "u1", titulo: "Alvará" }] },
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/protocolos/servicos/pessoas": { data: [{ id: "pa", nome: "Ana Chefe" }] },
      "POST v1/protocolos/servicos": { data: {} },
    });
    renderApp(<ServicosPage />);
    await user.click(await screen.findByRole("button", { name: /Novo Serviço/ }));
    const modal = await screen.findByRole("dialog");
    await user.selectOptions(within(modal).getByLabelText("Unidade dona *"), "u1");
    await user.selectOptions(within(modal).getByLabelText("Formulário *"), "f1");
    await user.type(within(modal).getByLabelText("Nome do Serviço *"), "Alvará");
    await user.click(within(modal).getByRole("tab", { name: "Carta ao Cidadão" }));
    await user.type(within(modal).getByLabelText("Quem pode solicitar"), "Empresas com CNPJ");
    await user.type(within(modal).getByLabelText("Passo a passo"), "1. Abra o pedido");
    await user.type(within(modal).getByLabelText("Tempo estimado de atendimento"), "10 dias");
    await user.type(within(modal).getByLabelText("Contato"), "66 3411-0000");
    await user.type(within(modal).getByLabelText("Sobre a taxa"), "Paga no DAM");
    await user.click(within(modal).getByRole("tab", { name: "Workflow" }));
    await user.click(
      within(modal).getByRole("checkbox", { name: "Conclusão passa por aprovação" }),
    );
    const quem = within(modal).getByLabelText("Quem aprova");
    await within(quem).findByRole("option", { name: "Ana Chefe" });
    await user.selectOptions(within(modal).getByLabelText("Quem aprova"), "pa");
    await user.click(within(modal).getByRole("button", { name: "Salvar" }));
    expect(api.to("POST v1/protocolos/servicos")[0]?.body).toMatchObject({
      exige_aprovacao: true,
      aprovador_id: "pa",
      detalhes: {
        quem_pode: "Empresas com CNPJ",
        passo_a_passo: "1. Abra o pedido",
        tempo_estimado: "10 dias",
        contato: "66 3411-0000",
        descricao_taxa: "Paga no DAM",
      },
    });
  });
});

describe("auxiliares de prazo", () => {
  it("decide quem aprova e lê o dia em Cuiabá", () => {
    expect(podeAprovar({ status: "pending_approval" }, "eu", true)).toBe(true);
    expect(podeAprovar({ status: "pending_approval", aprovador_id: "eu" }, "eu", false)).toBe(true);
    expect(podeAprovar({ status: "pending_approval", aprovador_id: "eu" }, undefined, true)).toBe(
      false,
    );
    expect(hojeEmCuiaba(new Date("2026-11-28T02:00:00Z"))).toBe("2026-11-27");
  });
});
