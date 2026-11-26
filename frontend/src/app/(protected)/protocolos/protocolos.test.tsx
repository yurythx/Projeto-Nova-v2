import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { nav, resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import FichaPage from "./[id]/page";
import ProtocolosPage from "./page";

const PEDIDOS = [
  {
    id: "p1",
    numero: "PREF202690001",
    nome: "Ana Paula Ribeiro",
    cpf: "***.807.124-36",
    servico_id: "s1",
    servico_nome: "Alvará de funcionamento",
    unidade_id: "u1",
    unidade_nome: "PREFEITURA",
    status: "em_analise",
    situacao_prazo: "atrasado",
    etapa_id: "triagem",
    prioridade: "alta",
    prazo_em: "2026-11-20T15:00:00Z",
    created_at: "2026-11-10T12:00:00Z",
  },
  {
    id: "p2",
    numero: "SMS202690031",
    nome: "Edna Marques Queiroz",
    servico_id: "s2",
    servico_nome: "Vacinação em domicílio",
    unidade_id: "u2",
    unidade_nome: "SMS",
    status: "em_complementacao",
    situacao_prazo: "atencao",
    etapa_id: "",
    prioridade: "normal",
    created_at: "2026-11-24T12:00:00Z",
  },
  {
    id: "p3",
    numero: "PREF202690020",
    nome: "Yasmin Correia Pacheco",
    servico_id: "s3",
    servico_nome: "Certidão negativa de débitos municipais",
    unidade_id: "u1",
    unidade_nome: "PREFEITURA",
    status: "aprovado",
    situacao_prazo: "finalizado",
    etapa_id: "",
    prioridade: "normal",
    created_at: "2026-11-26T12:00:00Z",
  },
];

const FICHA = {
  pedido: {
    ...PEDIDOS[0],
    cpf: "00807124362",
    email: "ana.ribeiro@example.com",
    telefone: "(66) 97919-4729",
    complemento: "",
    decisao: "",
    justificativa: "",
    campos: [
      { id: "pagina", tipo: "page", rotulo: "Dados" },
      { id: "razao_social", tipo: "text", rotulo: "Razão social" },
      { id: "tem_placa", tipo: "boolean", rotulo: "Tem placa" },
      { id: "contrato", tipo: "file", rotulo: "Contrato social" },
    ],
    respostas: {
      razao_social: "Padaria Pão Dourado Ltda",
      tem_placa: true,
      contrato: "protocolos/s1/contrato.pdf",
    },
  },
  historico: [
    {
      id: "h1",
      acao: "criado",
      descricao: "Pedido aberto",
      status: "em_analise",
      created_at: "2026-11-10T12:00:00Z",
    },
    {
      id: "h2",
      acao: "nota",
      descricao: "Conferido com o cadastro.",
      status: "em_analise",
      anexos: ["protocolos/s1/anexo.pdf"],
      created_at: "2026-11-10T15:00:00Z",
    },
  ],
  agendamento: { data: "2026-12-02", hora: "11:30", status: "agendado" },
  etapas: [
    { id: "triagem", ordem: 1, status: "em_analise", prazo_dias: 3 },
    { id: "vistoria", ordem: 2, status: "em_andamento", prazo_dias: 7 },
    { id: "parecer", ordem: 3, status: "pending_approval", prazo_dias: 5 },
  ],
};

function rotas(permissoes: string[], extra: Record<string, unknown> = {}) {
  return mockBackend({
    ...identityRoutes({ permissions: permissoes }),
    "GET v1/protocolos/pedidos": { data: PEDIDOS },
    "GET v1/protocolos/pedidos/p1": { data: FICHA },
    "GET v1/protocolos/pedidos/p1/arquivo": { data: { url: "https://arquivos/x.pdf" } },
    "GET v1/iam/org-tree": { data: [] },
    "POST v1/protocolos/pedidos/p1/avancar": { data: {} },
    "POST v1/protocolos/pedidos/p1/complemento": { data: {} },
    "POST v1/protocolos/pedidos/p1/concluir": { data: {} },
    "POST v1/protocolos/pedidos/p1/cancelar": { data: {} },
    "POST v1/protocolos/pedidos/p1/nota": { data: {} },
    "POST v1/protocolos/pedidos/p1/editar": { data: {} },
    "POST v1/protocolos/pedidos/p2/resposta": { data: {} },
    "POST v1/protocolos/faltas": { status: 204 },
    ...(extra as Record<string, never>),
  });
}

describe("Painel de Protocolos", () => {
  beforeEach(() => resetNavigation({}, "/protocolos"));

  it("mostra as estatísticas, a faixa de complementação, as abas de prazo e a lista", async () => {
    rotas(["protocolos:tratar"]);
    renderApp(<ProtocolosPage />);
    expect(await screen.findByText("Protocolo #PREF202690001")).toBeInTheDocument();
    expect(screen.getByText("Total de Protocolos")).toBeInTheDocument();
    expect(screen.getByText("1 protocolo(s) aguardando complementação")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /Atrasados\s*1/ })).toBeInTheDocument();
    expect(screen.getByText(/Atrasado \(\d+\/\d+d\)/)).toBeInTheDocument();
    expect(screen.getAllByText("Em Análise").length).toBeGreaterThan(0);
    // Sem gerir nem abrir, não há configuração nem balcão.
    expect(screen.queryByRole("link", { name: "Serviços" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Balcão" })).not.toBeInTheDocument();
    // Pedido final não tem menu de ações.
    expect(
      screen.queryByRole("button", { name: "Ações do protocolo PREF202690020" }),
    ).not.toBeInTheDocument();
  });

  it("filtra pelas abas, pela faixa, pela busca e limpa", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar", "protocolos:gerir", "protocolos:abrir"]);
    renderApp(<ProtocolosPage />);
    await screen.findByText("Protocolo #PREF202690001");
    expect(screen.getByRole("link", { name: "Serviços" })).toHaveAttribute(
      "href",
      "/protocolos/servicos",
    );
    expect(screen.getByRole("link", { name: "Balcão" })).toHaveAttribute(
      "href",
      "/protocolos/balcao",
    );

    await user.click(screen.getByRole("tab", { name: /Finalizados/ }));
    expect(screen.getByText("Protocolo #PREF202690020")).toBeInTheDocument();
    expect(screen.queryByText("Protocolo #PREF202690001")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Limpar" }));
    await user.click(screen.getByRole("button", { name: "Ver Protocolos" }));
    expect(screen.getByText("Protocolo #SMS202690031")).toBeInTheDocument();
    expect(screen.queryByText("Protocolo #PREF202690001")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Limpar" }));
    await user.selectOptions(screen.getByLabelText("Prioridade"), "alta");
    expect(screen.queryByText("Protocolo #SMS202690031")).not.toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Prioridade"), "");
    await user.selectOptions(screen.getByLabelText("Serviço"), "s2");
    expect(screen.getByText("Protocolo #SMS202690031")).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Serviço"), "");
    await user.selectOptions(screen.getByLabelText("Status"), "aprovado");
    expect(screen.getByText("Protocolo #PREF202690020")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Limpar" }));

    await user.type(screen.getByLabelText("Buscar"), "nada assim");
    expect(
      await screen.findByText("Nenhum protocolo encontrado com os filtros aplicados."),
    ).toBeInTheDocument();
  });

  it("abre a ficha, troca para o kanban e navega entre as colunas", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar"]);
    const { unmount } = renderApp(<ProtocolosPage />);
    await screen.findByText("Protocolo #PREF202690001");
    await user.click(screen.getAllByRole("button", { name: /Visualizar/ })[0]!);
    expect(router.push).toHaveBeenCalledWith("/protocolos/p1");
    await user.click(screen.getByRole("button", { name: "Kanban" }));
    expect(router.replace).toHaveBeenCalledWith("/protocolos?vista=kanban");
    await user.click(screen.getByRole("button", { name: "Lista" }));
    expect(router.replace).toHaveBeenCalledWith("/protocolos");
    unmount();

    nav.search = new URLSearchParams("vista=kanban");
    renderApp(<ProtocolosPage />);
    const kanban = await screen.findByTestId("protocolos-kanban");
    expect(within(kanban).getByRole("region", { name: "Novos" })).toBeInTheDocument();
    expect(within(kanban).getByText("PREF202690001")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Colunas anteriores" })).not.toBeInTheDocument();
    for (let i = 0; i < 5; i++) {
      await user.click(screen.getByRole("button", { name: "Próximas colunas" }));
    }
    expect(
      within(kanban).getByRole("region", { name: "Cancelados/Rejeitados" }),
    ).toBeInTheDocument();
    expect(within(kanban).getByText("PREF202690020")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Próximas colunas" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Colunas anteriores" }));
    await user.click(within(kanban).getByRole("button", { name: /Vacinação em domicílio/ }));
    expect(router.push).toHaveBeenCalledWith("/protocolos/p2");
  });

  it("trata pelo menu: encaminha, pede complementação, responde, conclui e cancela", async () => {
    const user = userEvent.setup();
    const api = rotas(["protocolos:tratar"]);
    renderApp(<ProtocolosPage />);
    await screen.findByText("Protocolo #PREF202690001");

    const menu = () => screen.getByRole("button", { name: "Ações do protocolo PREF202690001" });
    await user.click(menu());
    await user.click(screen.getByRole("menuitem", { name: "Encaminhar para a próxima etapa" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/avancar")).toHaveLength(1);

    await user.click(menu());
    await user.click(screen.getByRole("menuitem", { name: "Solicitar complementação" }));
    await user.type(screen.getByLabelText(/O que o cidadão precisa enviar/), "Envie o contrato.");
    await user.click(screen.getByRole("button", { name: "Confirmar" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/complemento")[0]?.body).toEqual({
      mensagem: "Envie o contrato.",
    });

    await user.click(menu());
    await user.click(screen.getByRole("menuitem", { name: "Concluir" }));
    await user.selectOptions(screen.getByLabelText("Decisão *"), "favoravel");
    await user.type(screen.getByLabelText("Justificativa *"), "Vistoria sem pendências.");
    await user.click(screen.getByRole("button", { name: "Confirmar" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/concluir")[0]?.body).toEqual({
      decisao: "favoravel",
      justificativa: "Vistoria sem pendências.",
    });

    await user.click(menu());
    await user.click(screen.getByRole("menuitem", { name: "Cancelar" }));
    await user.click(screen.getByRole("button", { name: "Cancelar protocolo" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/cancelar")).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Ações do protocolo SMS202690031" }));
    expect(screen.queryByRole("menuitem", { name: "Concluir" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("menuitem", { name: "Registrar resposta" }));
    await user.type(screen.getByLabelText(/Resposta recebida/), "Chegou por e-mail.");
    await user.click(screen.getByRole("button", { name: "Confirmar" }));
    expect(api.to("POST v1/protocolos/pedidos/p2/resposta")).toHaveLength(1);

    // O menu fecha pelo fundo e o modal pelo "Cancelar".
    await user.click(menu());
    await user.click(screen.getByRole("button", { name: "Fechar menu" }));
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    await user.click(menu());
    await user.click(screen.getByRole("menuitem", { name: "Adicionar nota" }));
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/nota")).toHaveLength(0);
    await user.click(menu());
    await user.click(screen.getByRole("menuitem", { name: "Visualizar" }));
    expect(router.push).toHaveBeenCalledWith("/protocolos/p1");
  });

  it("pagina de dez em dez", async () => {
    const user = userEvent.setup();
    const muitos = Array.from({ length: 12 }, (_, i) => ({
      ...PEDIDOS[2]!,
      id: `m${i}`,
      numero: `PREF2026900${String(i + 10)}`,
    }));
    rotas(["protocolos:tratar"], { "GET v1/protocolos/pedidos": { data: muitos } });
    renderApp(<ProtocolosPage />);
    expect(await screen.findByText("Protocolo #PREF202690010")).toBeInTheDocument();
    expect(screen.queryByText("Protocolo #PREF202690021")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Anterior" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Próxima" }));
    expect(screen.getByText("Protocolo #PREF202690021")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "1" }));
    expect(screen.getByText("Protocolo #PREF202690010")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "2" }));
    await user.click(screen.getByRole("button", { name: "Anterior" }));
    expect(screen.getByText("Protocolo #PREF202690010")).toBeInTheDocument();
  });
});

describe("Ficha do protocolo", () => {
  beforeEach(() => resetNavigation({ id: "p1" }, "/protocolos/p1"));

  it("mostra o fluxo, as informações, o cidadão, o formulário e o histórico", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar"]);
    const abrir = vi.spyOn(window, "open").mockImplementation(() => null);
    renderApp(<FichaPage />);
    expect(
      await screen.findByRole("heading", { name: "Protocolo #PREF202690001" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Fluxo do Processo")).toBeInTheDocument();
    expect(screen.getByText("Vistoria")).toBeInTheDocument();
    expect(screen.getByText("00807124362")).toBeInTheDocument();
    expect(screen.getByText("Padaria Pão Dourado Ltda")).toBeInTheDocument();
    expect(screen.getByText("Sim")).toBeInTheDocument();
    expect(screen.getByText("02/12/2026")).toBeInTheDocument();
    expect(screen.getByText("Protocolo Criado")).toBeInTheDocument();
    // A descrição padrão não repete o título do evento.
    expect(screen.queryByText("Pedido aberto")).not.toBeInTheDocument();
    expect(screen.getByText("Conferido com o cadastro.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Baixar" }));
    expect(abrir).toHaveBeenCalledWith("https://arquivos/x.pdf", "_blank", "noopener,noreferrer");
    await user.click(screen.getByRole("button", { name: /Anexo desta etapa/ }));
    await user.click(screen.getByRole("button", { name: "Voltar" }));
    expect(router.push).toHaveBeenCalledWith("/protocolos");
    abrir.mockRestore();
  });

  it("trata pela barra lateral: encaminha, anota, registra falta, edita e conclui", async () => {
    const user = userEvent.setup();
    const api = rotas(["protocolos:tratar"]);
    renderApp(<FichaPage />);
    await screen.findByRole("heading", { name: "Protocolo #PREF202690001" });

    await user.click(screen.getByRole("button", { name: "Encaminhar para a próxima etapa" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/avancar")).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Adicionar nota" }));
    await user.type(screen.getByLabelText(/Nota \(não aparece/), "Liguei para a requerente.");
    await user.click(screen.getByRole("button", { name: "Confirmar" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/nota")).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Registrar falta" }));
    expect(api.to("POST v1/protocolos/faltas")[0]?.body).toEqual({ pedido_id: "p1" });

    await user.click(screen.getByRole("button", { name: /Editar dados/ }));
    const razao = screen.getByLabelText("Razão social");
    await user.clear(razao);
    await user.type(razao, "Padaria Nova Ltda");
    await user.type(screen.getByLabelText("Justificativa da edição *"), "Erro de digitação.");
    await user.click(screen.getByRole("button", { name: "Salvar edição" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/editar")[0]?.body).toMatchObject({
      justificativa: "Erro de digitação.",
      respostas: { razao_social: "Padaria Nova Ltda" },
    });

    await user.click(screen.getByRole("button", { name: "Solicitar complementação" }));
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    await user.click(screen.getByRole("button", { name: "Concluir protocolo" }));
    expect(screen.getByRole("heading", { name: "Concluir Protocolo" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    await user.click(screen.getByRole("button", { name: "Cancelar protocolo" }));
    expect(screen.getByRole("heading", { name: "Cancelar Protocolo" })).toBeInTheDocument();
  });

  it("no protocolo concluído mostra a decisão e esconde as ações", async () => {
    rotas(["protocolos:tratar"], {
      "GET v1/protocolos/pedidos/p1": {
        data: {
          ...FICHA,
          etapas: [],
          agendamento: undefined,
          historico: [],
          pedido: {
            ...FICHA.pedido,
            status: "concluido",
            situacao_prazo: "finalizado",
            decisao: "desfavoravel",
            justificativa: "Zona residencial exclusiva.",
          },
        },
      },
    });
    renderApp(<FichaPage />);
    expect(await screen.findByText("Decisão Desfavorável (indeferido)")).toBeInTheDocument();
    expect(screen.getByText("Zona residencial exclusiva.")).toBeInTheDocument();
    expect(screen.queryByText("Fluxo do Processo")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /Ações/ })).not.toBeInTheDocument();
    expect(screen.getByText("Nenhuma movimentação registrada")).toBeInTheDocument();
  });

  it("em complementação mostra o aviso e registra a resposta", async () => {
    const user = userEvent.setup();
    const api = rotas(["protocolos:tratar"], {
      "GET v1/protocolos/pedidos/p1": {
        data: {
          ...FICHA,
          pedido: {
            ...FICHA.pedido,
            status: "em_complementacao",
            complemento: "Envie o contrato.",
            decisao: "favoravel",
            justificativa: "",
          },
        },
      },
      "POST v1/protocolos/pedidos/p1/resposta": { data: {} },
    });
    renderApp(<FichaPage />);
    expect(await screen.findByText("Complementação Solicitada")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Encaminhar para a próxima etapa" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Concluir protocolo" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Registrar resposta" }));
    await user.type(screen.getByLabelText(/Resposta recebida/), "Chegou por e-mail.");
    await user.click(screen.getByRole("button", { name: "Confirmar" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/resposta")).toHaveLength(1);
  });
});
