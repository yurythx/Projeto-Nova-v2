import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { fail, mockBackend } from "@/test/backend";

import { AbrirPedido } from "./[id]/AbrirPedido";
import { CartaPublica } from "./CartaPublica";

const CARTA = [
  {
    id: "s1",
    nome: "Alvará de funcionamento",
    descricao: "Licença para funcionar",
    categoria: "Tributos",
    prazo_dias: 15,
    destaque: false,
    taxa_centavos: 18750,
    unidade_id: "u1",
    unidade_nome: "PREFEITURA",
  },
  {
    id: "s2",
    nome: "Vacinação em domicílio",
    descricao: "",
    categoria: "",
    prazo_dias: 0,
    destaque: true,
    taxa_centavos: 0,
    unidade_id: "u2",
  },
];

const CONSULTA = {
  numero: "SMS202690031",
  status: "em_complementacao",
  status_rotulo: "",
  prioridade: "normal",
  nome: "Edna",
  servico: "Vacinação em domicílio",
  unidade: "SMS",
  criado_em: "2026-11-24T12:00:00Z",
  prazo_em: "2026-12-01T12:00:00Z",
  complemento: "Envie o cartão de vacina",
  agendamento: { data: "2026-12-02", hora: "08:30", status: "agendado" },
  historico: [
    {
      id: "h1",
      acao: "criado",
      descricao: "Pedido aberto",
      status: "novo",
      created_at: "2026-11-24T12:00:00Z",
    },
    {
      id: "h2",
      acao: "avanco",
      descricao: "Em análise",
      status: "em_analise",
      created_at: "2026-11-24T13:00:00Z",
    },
    {
      id: "h3",
      acao: "complemento",
      descricao: "Falta documento",
      status: "em_complementacao",
      created_at: "2026-11-24T14:00:00Z",
    },
    {
      acao: "conclusao",
      descricao: "Concluído",
      status: "concluido",
      created_at: "2026-11-24T15:00:00Z",
    },
    {
      acao: "cancelado",
      descricao: "Cancelado",
      status: "cancelado",
      created_at: "2026-11-24T16:00:00Z",
    },
  ],
};

describe("Carta pública", () => {
  it("lista com o destaque primeiro, filtra por texto e secretaria", async () => {
    const user = userEvent.setup();
    mockBackend({ "GET v1/protocolos/carta": { data: CARTA } });
    render(<CartaPublica />);
    expect(await screen.findByText("2 serviços disponíveis")).toBeInTheDocument();
    const cartoes = screen.getAllByRole("article");
    expect(within(cartoes[0]!).getByText("Vacinação em domicílio")).toBeInTheDocument();
    expect(within(cartoes[0]!).getByText("Destaque")).toBeInTheDocument();
    expect(within(cartoes[0]!).getByText("Gratuito")).toBeInTheDocument();
    expect(within(cartoes[1]!).getByText("R$ 187,50")).toBeInTheDocument();
    expect(within(cartoes[1]!).getByRole("link", { name: "Solicitar" })).toHaveAttribute(
      "href",
      "/carta/s1?pedir=1",
    );

    await user.selectOptions(screen.getByLabelText("Secretaria"), "u1");
    expect(screen.getByText("1 serviço disponível")).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Secretaria"), "");
    await user.type(screen.getByLabelText("Buscar serviço"), "tributos");
    expect(screen.getByText("Alvará de funcionamento")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Buscar serviço"), "zz");
    expect(screen.getByText("Nenhum serviço encontrado")).toBeInTheDocument();
  });

  it("consulta, mostra a linha do tempo e responde a complementação com arquivo", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      "GET v1/protocolos/carta": { data: CARTA },
      "GET v1/protocolos/consulta/NADA": fail(404, "Protocolo não encontrado"),
      "GET v1/protocolos/consulta/SMS202690031": { data: CONSULTA },
      "POST v1/protocolos/consulta/SMS202690031/arquivo": {
        data: { upload_url: "https://minio/up", method: "PUT", object_key: "k1", headers: {} },
      },
      "PUT minio/up": { data: {} },
      "POST v1/protocolos/consulta/SMS202690031/resposta": {
        data: { ...CONSULTA, status: "em_analise", status_rotulo: "Em análise", complemento: "" },
      },
    });
    render(<CartaPublica />);
    await user.click(screen.getByRole("tab", { name: /Consultar protocolo/ }));
    await user.type(screen.getByLabelText("Número do protocolo"), "NADA");
    await user.click(screen.getByRole("button", { name: /Consultar/ }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Protocolo não encontrado");

    await user.clear(screen.getByLabelText("Número do protocolo"));
    await user.type(screen.getByLabelText("Número do protocolo"), "SMS202690031");
    await user.click(screen.getByRole("button", { name: /Consultar/ }));
    expect(await screen.findByText("Em Complementação")).toBeInTheDocument();
    expect(screen.getByText("Envie o cartão de vacina")).toBeInTheDocument();
    expect(screen.getByText(/2026-12-02 às 08:30/)).toBeInTheDocument();
    const itens = within(screen.getByRole("list")).getAllByRole("listitem");
    expect(itens[0]).toHaveTextContent("Cancelado");
    expect(itens[4]).toHaveTextContent("Pedido aberto");

    await user.type(screen.getByLabelText("CPF do pedido *"), "12345678900");
    await user.type(screen.getByLabelText("Resposta *"), "Segue o cartão");
    await user.upload(
      screen.getByLabelText("Arquivos"),
      new File(["x"], "cartao.pdf", { type: "application/pdf" }),
    );
    await user.click(screen.getByRole("button", { name: "Enviar resposta" }));
    expect(await screen.findByText("Em análise", { selector: "span" })).toBeInTheDocument();
    // O FormData do jsdom não devolve o File do upload; o arquivo fica para o e2e.
    expect(api.to("POST v1/protocolos/consulta/SMS202690031/resposta")[0]?.body).toMatchObject({
      cpf: "12345678900",
      mensagem: "Segue o cartão",
    });
    expect(screen.queryByRole("button", { name: "Enviar resposta" })).not.toBeInTheDocument();
  });

  it("mostra a falha da carta e da resposta", async () => {
    const user = userEvent.setup();
    mockBackend({
      "GET v1/protocolos/carta": fail(503, "Carta fora do ar"),
      "GET v1/protocolos/consulta/SMS202690031": { data: { ...CONSULTA, historico: [] } },
      "POST v1/protocolos/consulta/SMS202690031/resposta": fail(400, "CPF não confere"),
    });
    render(<CartaPublica />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Carta fora do ar");
    await user.click(screen.getByRole("tab", { name: /Consultar protocolo/ }));
    await user.type(screen.getByLabelText("Número do protocolo"), "SMS202690031");
    await user.click(screen.getByRole("button", { name: /Consultar/ }));
    await user.type(await screen.findByLabelText("CPF do pedido *"), "1");
    await user.type(screen.getByLabelText("Resposta *"), "x");
    await user.click(screen.getByRole("button", { name: "Enviar resposta" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("CPF não confere");
  });
});

const ITEM = {
  servico: {
    id: "s1",
    nome: "Alvará de funcionamento",
    descricao: "Licença para funcionar",
    categoria: "Tributos",
    unidade_nome: "PREFEITURA",
    exige_agendamento: true,
    prazo_dias: 15,
    taxa_centavos: 18750,
    requisitos: [
      { titulo: "Ser maior de idade", texto: "18 anos", obrigatorio: true },
      { titulo: "Ter CNPJ", texto: "", obrigatorio: false },
    ],
    documentos: [{ titulo: "RG", texto: "", obrigatorio: true }],
  },
  formulario: {
    titulo: "Alvará",
    instrucao: "Tenha o CNPJ em mãos",
    campos: [
      { id: "pagina", tipo: "page", rotulo: "Empresa" },
      { id: "razao", tipo: "text", rotulo: "Razão social", obrigatorio: true },
      { id: "obs", tipo: "textarea", rotulo: "Observação", obrigatorio: false },
      { id: "porte", tipo: "select", rotulo: "Porte", obrigatorio: false, opcoes: ["MEI", "ME"] },
      {
        id: "placa",
        tipo: "boolean",
        rotulo: "Tem placa",
        obrigatorio: false,
        revela: ["tamanho"],
      },
      { id: "tamanho", tipo: "number", rotulo: "Tamanho", obrigatorio: false },
      {
        id: "dias",
        tipo: "checkbox_group",
        rotulo: "Dias",
        obrigatorio: false,
        opcoes: ["Seg", "Ter"],
      },
      { id: "local", tipo: "map_area", rotulo: "Local", obrigatorio: false },
      { id: "contrato", tipo: "file", rotulo: "Contrato", obrigatorio: false },
    ],
  },
};

describe("Pedido pela carta", () => {
  it("mostra os detalhes, preenche todos os tipos de campo e protocola", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      "GET v1/protocolos/carta/s1": { data: ITEM },
      "GET v1/protocolos/carta/s1/horarios": { data: ["09:00", "09:30"] },
      "POST v1/protocolos/carta/s1/arquivo": {
        data: { upload_url: "https://minio/up", method: "PUT", object_key: "k1", headers: {} },
      },
      "PUT minio/up": { data: {} },
      "POST v1/protocolos/carta/s1/pedido": { data: { numero: "PREF202690100" } },
    });
    render(<AbrirPedido id="s1" />);
    expect(await screen.findByText("Licença para funcionar")).toBeInTheDocument();
    expect(screen.getByText("Ser maior de idade")).toBeInTheDocument();
    expect(screen.getByText("18 anos")).toBeInTheDocument();
    expect(screen.getAllByText("obrigatório")).toHaveLength(2);
    expect(screen.getByText("R$ 187,50")).toBeInTheDocument();
    expect(screen.getByText("Com horário marcado")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Solicitar" }));
    expect(screen.getByText(/2 requisito\(s\) e 1 documento\(s\)/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Voltar aos detalhes" }));
    await user.click(screen.getByRole("button", { name: "Solicitar" }));

    await user.type(screen.getByLabelText("Nome completo *"), "Ana");
    await user.type(screen.getByLabelText("CPF *"), "00807124362");
    await user.type(screen.getByLabelText("Data do atendimento *"), "2026-12-02");
    await screen.findByRole("option", { name: "09:30" });
    await user.selectOptions(screen.getByLabelText("Horário *"), "09:00");
    expect(screen.getByRole("heading", { name: "Empresa" })).toBeInTheDocument();
    await user.type(screen.getByLabelText("Razão social *"), "Padaria");
    await user.type(screen.getByLabelText("Observação"), "Esquina");
    await user.selectOptions(screen.getByLabelText("Porte"), "ME");
    expect(screen.queryByLabelText("Tamanho")).not.toBeInTheDocument();
    await user.click(screen.getByRole("checkbox", { name: "Tem placa" }));
    await user.type(screen.getByLabelText("Tamanho"), "3");
    await user.click(screen.getByRole("checkbox", { name: "Seg" }));
    await user.click(screen.getByRole("checkbox", { name: "Ter" }));
    await user.click(screen.getByRole("checkbox", { name: "Seg" }));
    await user.type(screen.getByLabelText("Latitude do primeiro ponto"), "-16.4");
    await user.type(screen.getByLabelText("Longitude do primeiro ponto"), "-54.6");
    await user.type(screen.getByLabelText("Latitude do segundo ponto"), "-16.5");
    await user.type(screen.getByLabelText("Longitude do segundo ponto"), "-54.7");
    expect(screen.getByRole("link", { name: "Ver no mapa" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Usar minha localização" }));
    expect(screen.getByText("Este navegador não informa a localização.")).toBeInTheDocument();
    await user.upload(
      screen.getByLabelText("Contrato"),
      new File(["x"], "c.pdf", { type: "application/pdf" }),
    );
    expect(await screen.findByText("Arquivo recebido.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Protocolar" }));
    expect(await screen.findByText("PREF202690100")).toBeInTheDocument();
    expect(api.to("POST v1/protocolos/carta/s1/pedido")[0]?.body).toMatchObject({
      nome: "Ana",
      data: "2026-12-02",
      hora: "09:00",
      respostas: {
        razao: "Padaria",
        obs: "Esquina",
        porte: "ME",
        placa: true,
        tamanho: 3,
        dias: ["Ter"],
        local: "-16.4,-54.6;-16.5,-54.7",
        contrato: "k1",
      },
    });
    expect(screen.getByRole("link", { name: /Voltar à carta/ })).toHaveAttribute("href", "/carta");
  });

  it("mostra a falha do serviço, do arquivo e do pedido", async () => {
    const user = userEvent.setup();
    mockBackend({ "GET v1/protocolos/carta/x": fail(404, "Serviço fora da carta") });
    const { unmount } = render(<AbrirPedido id="x" />);
    expect(await screen.findByRole("alert")).toHaveTextContent("Serviço fora da carta");
    unmount();

    mockBackend({
      "GET v1/protocolos/carta/s1": {
        data: {
          servico: {
            ...ITEM.servico,
            descricao: "",
            exige_agendamento: false,
            prazo_dias: 0,
            taxa_centavos: 0,
            requisitos: [],
            documentos: [],
          },
          formulario: { titulo: "Alvará", instrucao: "", campos: [ITEM.formulario.campos[8]] },
        },
      },
      "POST v1/protocolos/carta/s1/arquivo": fail(400, "Arquivo grande demais"),
      "POST v1/protocolos/carta/s1/pedido": fail(409, "CPF com pedido aberto"),
    });
    render(<AbrirPedido id="s1" pedirInicial />);
    await user.upload(
      await screen.findByLabelText("Contrato"),
      new File(["x"], "c.pdf", { type: "application/pdf" }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent("Arquivo grande demais");
    await user.type(screen.getByLabelText("Nome completo *"), "Ana");
    await user.type(screen.getByLabelText("CPF *"), "1");
    await user.click(screen.getByRole("button", { name: "Protocolar" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("CPF com pedido aberto");
    expect(screen.getByText("Gratuito")).toBeInTheDocument();
    expect(screen.getByText("Sem prazo fixo")).toBeInTheDocument();
  });
});
