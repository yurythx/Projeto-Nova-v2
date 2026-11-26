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
      { id: "descricao", tipo: "textarea", rotulo: "Descrição" },
      { id: "tem_placa", tipo: "boolean", rotulo: "Tem placa" },
      { id: "contrato", tipo: "file", rotulo: "Contrato social" },
    ],
    respostas: {
      razao_social: "Padaria Pão Dourado Ltda",
      descricao: "Funciona das 6h às 20h.",
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
      created_at: "2026-11-10T15:00:00Z",
    },
  ],
  agendamento: { data: "2026-12-02", hora: "11:30", status: "agendado" },
};

function rotas(permissoes: string[], extra: Record<string, unknown> = {}) {
  return mockBackend({
    ...identityRoutes({ permissions: permissoes }),
    "GET v1/protocolos/pedidos": { data: PEDIDOS },
    "GET v1/protocolos/pedidos/p1": { data: FICHA },
    "GET v1/iam/org-tree": { data: [] },
    "POST v1/protocolos/pedidos/p1/avancar": { data: {} },
    "POST v1/protocolos/pedidos/p1/complemento": { data: {} },
    "POST v1/protocolos/pedidos/p1/concluir": { data: {} },
    "POST v1/protocolos/pedidos/p1/nota": { data: {} },
    "POST v1/protocolos/faltas": { status: 204 },
    ...(extra as Record<string, never>),
  });
}

describe("Mesa do protocolo", () => {
  beforeEach(() => resetNavigation({}, "/protocolos"));

  it("mostra os números da mesa e o quadro por status, com o semáforo do prazo", async () => {
    rotas(["protocolos:tratar"]);
    renderApp(<ProtocolosPage />);
    const quadro = await screen.findByTestId("protocolos-kanban");
    expect(within(quadro).getByText("PREF202690001")).toBeInTheDocument();
    expect(within(quadro).getByText("Em análise")).toBeInTheDocument();
    expect(within(quadro).getByText("Aprovados")).toBeInTheDocument();
    expect(within(quadro).queryByText("Pendentes")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Atrasados\s*1/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Aguardando cidadão\s*1/ })).toBeInTheDocument();
    // Sem gerir nem abrir, a mesa não oferece configuração nem balcão.
    expect(screen.queryByRole("link", { name: "Serviços" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /balcão/ })).not.toBeInTheDocument();
  });

  it("filtra pelos cartões de cima e limpa os filtros", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar", "protocolos:gerir", "protocolos:abrir"]);
    renderApp(<ProtocolosPage />);
    const quadro = await screen.findByTestId("protocolos-kanban");
    expect(screen.getByRole("link", { name: "Serviços" })).toHaveAttribute(
      "href",
      "/protocolos/servicos",
    );
    expect(screen.getByRole("link", { name: /Novo pedido no balcão/ })).toHaveAttribute(
      "href",
      "/protocolos/balcao",
    );

    await user.click(screen.getByRole("button", { name: /Atrasados/ }));
    expect(within(quadro).getByText("PREF202690001")).toBeInTheDocument();
    expect(within(quadro).queryByText("SMS202690031")).not.toBeInTheDocument();
    expect(screen.getByText("1 de 3 pedidos com os filtros atuais.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Atrasados/ }));
    expect(within(quadro).getByText("SMS202690031")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Aguardando cidadão/ }));
    expect(within(quadro).queryByText("PREF202690001")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Limpar filtros" }));
    expect(within(quadro).getByText("PREF202690001")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Buscar"), "nada assim");
    expect(await screen.findByText("Nenhum pedido com esses filtros.")).toBeInTheDocument();
  });

  it("troca de vista pela URL e abre o pedido no painel", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar"]);
    renderApp(<ProtocolosPage />);
    await screen.findByTestId("protocolos-kanban");
    await user.click(screen.getByRole("tab", { name: "Lista" }));
    expect(router.replace).toHaveBeenCalledWith("/protocolos?vista=lista");
    await user.click(screen.getAllByRole("button", { name: "Alvará de funcionamento" })[0]!);
    expect(router.push).toHaveBeenCalledWith("/protocolos?painel=p1", { scroll: false });
    expect(await screen.findByRole("heading", { name: "Detalhe do pedido" })).toBeInTheDocument();
    expect(await screen.findByText("Pedido PREF202690001")).toBeInTheDocument();
  });

  it("mostra a lista agrupada e os indicadores", async () => {
    rotas(["protocolos:tratar"]);
    nav.search = new URLSearchParams("vista=lista");
    const { unmount } = renderApp(<ProtocolosPage />);
    // Um grupo por status com pedido, cada um com o próprio cabeçalho.
    expect(await screen.findAllByRole("columnheader", { name: "Requerente" })).toHaveLength(3);
    expect(screen.getByLabelText("Status")).toBeInTheDocument();
    unmount();

    nav.search = new URLSearchParams("vista=indicadores");
    renderApp(<ProtocolosPage />);
    expect(await screen.findByText("Pedidos por serviço")).toBeInTheDocument();
    expect(screen.getByText("Pedidos abertos por mês")).toBeInTheDocument();
    expect(screen.getAllByText("Vacinação em domicílio").length).toBeGreaterThan(0);
  });

  it("exporta o CSV dos pedidos visíveis", async () => {
    const user = userEvent.setup();
    rotas(["protocolos:tratar"]);
    const criar = vi.fn(() => "blob:csv");
    const revogar = vi.fn();
    vi.stubGlobal("URL", Object.assign(URL, { createObjectURL: criar, revokeObjectURL: revogar }));
    renderApp(<ProtocolosPage />);
    await screen.findByTestId("protocolos-kanban");
    await user.click(screen.getByRole("button", { name: /Exportar CSV/ }));
    expect(criar).toHaveBeenCalledTimes(1);
    expect(revogar).toHaveBeenCalledWith("blob:csv");
  });

  it("diz quando a mesa está vazia", async () => {
    rotas(["protocolos:tratar"], { "GET v1/protocolos/pedidos": { data: [] } });
    renderApp(<ProtocolosPage />);
    expect(await screen.findByText("Nenhum pedido na sua mesa")).toBeInTheDocument();
  });
});

describe("Ficha do pedido", () => {
  beforeEach(() => resetNavigation({ id: "p1" }, "/protocolos/p1"));

  it("mostra requerente, prazo, agendamento, dados e a linha do tempo", async () => {
    rotas(["protocolos:tratar"]);
    renderApp(<FichaPage />);
    expect(await screen.findByText("Pedido PREF202690001")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Voltar para a mesa/ })).toHaveAttribute(
      "href",
      "/protocolos",
    );
    expect(screen.getByText("CPF 00807124362")).toBeInTheDocument();
    expect(screen.getByText("Padaria Pão Dourado Ltda")).toBeInTheDocument();
    expect(screen.getByText("sim")).toBeInTheDocument();
    expect(screen.getByText("02/12/2026 às 11:30")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Baixar arquivo/ })).toBeInTheDocument();
    // "Pedido aberto" é o título do evento; a descrição padrão não se repete.
    expect(screen.getAllByText("Pedido aberto")).toHaveLength(1);
    expect(screen.getByText("Conferido com o cadastro.")).toBeInTheDocument();
  });

  it("trata o pedido: avança, pede complementação, anota, registra falta e conclui", async () => {
    const user = userEvent.setup();
    const api = rotas(["protocolos:tratar"]);
    renderApp(<FichaPage />);
    await screen.findByText("Pedido PREF202690001");

    await user.click(screen.getByRole("button", { name: "Avançar etapa" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/avancar")).toHaveLength(1);

    await user.type(screen.getByLabelText("Pedir complementação ao cidadão"), "Envie o contrato.");
    await user.click(screen.getByRole("button", { name: "Pedir complementação" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/complemento")[0]?.body).toEqual({
      mensagem: "Envie o contrato.",
    });

    await user.type(screen.getByLabelText(/Nota interna/), "Liguei para a requerente.");
    await user.click(screen.getByRole("button", { name: "Anotar" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/nota")).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Registrar falta" }));
    expect(api.to("POST v1/protocolos/faltas")[0]?.body).toEqual({ pedido_id: "p1" });

    await user.selectOptions(screen.getByLabelText("Decisão *"), "favoravel");
    await user.type(screen.getByLabelText("Justificativa *"), "Vistoria sem pendências.");
    await user.click(screen.getByRole("button", { name: "Concluir" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/concluir")[0]?.body).toEqual({
      decisao: "favoravel",
      justificativa: "Vistoria sem pendências.",
    });
  });

  it("no pedido concluído mostra a decisão e esconde o tratamento", async () => {
    rotas(["protocolos:tratar"], {
      "GET v1/protocolos/pedidos/p1": {
        data: {
          ...FICHA,
          agendamento: undefined,
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
    expect(await screen.findByText("Decisão desfavorável")).toBeInTheDocument();
    expect(screen.getByText("Zona residencial exclusiva.")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Tratamento" })).not.toBeInTheDocument();
  });

  it("em complementação mostra o aviso e pede a resposta da mesa", async () => {
    const user = userEvent.setup();
    const api = rotas(["protocolos:tratar"], {
      "GET v1/protocolos/pedidos/p1": {
        data: {
          ...FICHA,
          pedido: {
            ...FICHA.pedido,
            status: "em_complementacao",
            complemento: "Envie o contrato.",
          },
        },
      },
      "POST v1/protocolos/pedidos/p1/resposta": { data: {} },
    });
    renderApp(<FichaPage />);
    expect(await screen.findByText("Aguardando o cidadão")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Avançar etapa" })).not.toBeInTheDocument();
    await user.type(screen.getByLabelText("Resposta recebida na mesa"), "Chegou por e-mail.");
    await user.click(screen.getByRole("button", { name: "Registrar resposta" }));
    expect(api.to("POST v1/protocolos/pedidos/p1/resposta")).toHaveLength(1);
  });
});
