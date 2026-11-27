import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import BalcaoPage from "./balcao/page";
import FormulariosPage from "./formularios/page";
import ResponsaveisPage from "./responsaveis/page";
import ServicosPage from "./servicos/page";

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
        departamentos: [{ id: "d1", unidade_id: "u1", nome: "Tributos", ativo: true }],
      },
    ],
  },
];

const FORMS = [
  {
    id: "f1",
    unidade_id: "u1",
    titulo: "Alvará",
    descricao: "Pedido de alvará",
    instrucao: "",
    ativo: true,
    campos: [{ id: "razao", tipo: "text", rotulo: "Razão social", obrigatorio: true }],
    updated_at: "2026-11-20T12:00:00Z",
  },
  {
    id: "f2",
    unidade_id: "u1",
    titulo: "Poda",
    descricao: "",
    ativo: false,
    campos: null,
  },
];

const SERVICOS = [
  {
    id: "s1",
    unidade_id: "u1",
    unidade_nome: "PREFEITURA",
    formulario_id: "f1",
    nome: "Alvará de funcionamento",
    descricao: "Licença para funcionar",
    categoria: "Tributos",
    prazo_dias: 15,
    taxa_centavos: 18750,
    destaque: true,
    ativo: true,
    aprova_sozinho: false,
    exige_agendamento: true,
    so_responsavel_tecnico: false,
    impedir_cpf_aberto: true,
    campo_anonimo: "",
    workflow: {
      ligado: true,
      etapas: [{ id: "triagem", ordem: 1, status: "em_analise", prazo_dias: 3 }],
    },
    agenda: { dias: [1, 2], inicio: "08:00", fim: "12:00", duracao_min: 30, vagas_dia: 8 },
    requisitos: [{ titulo: "Ser maior", texto: "", obrigatorio: true }],
    documentos: [],
  },
  {
    id: "s2",
    unidade_id: "u1",
    formulario_id: "f2",
    nome: "Poda de árvore",
    descricao: "",
    categoria: "",
    prazo_dias: 0,
    taxa_centavos: 0,
    destaque: false,
    ativo: false,
    aprova_sozinho: true,
    exige_agendamento: false,
    so_responsavel_tecnico: true,
    impedir_cpf_aberto: false,
    campo_anonimo: "anonimo",
    workflow: { ligado: false, etapas: null },
    requisitos: null,
    documentos: null,
  },
];

beforeEach(() => resetNavigation({}, "/protocolos/servicos"));

describe("Serviços do protocolo", () => {
  it("filtra, expande, desativa e edita o serviço", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/protocolos/servicos": { data: SERVICOS },
      "GET v1/protocolos/formularios": { data: FORMS },
      "GET v1/iam/org-tree": { data: ORG },
      "POST v1/protocolos/servicos": { data: SERVICOS[0] },
    });
    renderApp(<ServicosPage />);
    expect(await screen.findByText("Alvará de funcionamento")).toBeInTheDocument();
    expect(screen.getByText("Tributos • PREFEITURA")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Inativos/ }));
    expect(screen.queryByText("Alvará de funcionamento")).not.toBeInTheDocument();
    expect(screen.getByText("Poda de árvore")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Ativos/ }));
    expect(screen.queryByText("Poda de árvore")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Todos/ }));
    await user.type(screen.getByLabelText("Pesquisar"), "zzz");
    expect(screen.getByText("Nenhum serviço encontrado")).toBeInTheDocument();
    await user.clear(screen.getByLabelText("Pesquisar"));

    await user.click(screen.getByRole("button", { name: "Expandir Alvará de funcionamento" }));
    expect(screen.getByText("Ativo e Configurado (1 etapa)")).toBeInTheDocument();
    expect(screen.getByText("Destaque na carta · Um pedido aberto por CPF")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Recolher Alvará de funcionamento" }));
    await user.click(screen.getByRole("button", { name: /^Poda de árvore/ }));
    expect(screen.getByText("Gratuito")).toBeInTheDocument();
    expect(screen.getByText("Padrão")).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "Ações do serviço Alvará de funcionamento" }),
    );
    await user.click(screen.getByRole("menuitem", { name: /Desativar/ }));
    expect(api.to("POST v1/protocolos/servicos")[0]?.body).toMatchObject({
      id: "s1",
      ativo: false,
    });

    await user.click(screen.getByRole("button", { name: "Ações do serviço Poda de árvore" }));
    await user.click(screen.getByRole("button", { name: "Fechar menu" }));
    await user.click(
      screen.getByRole("button", { name: "Ações do serviço Alvará de funcionamento" }),
    );
    await user.click(screen.getByRole("menuitem", { name: /Editar Info/ }));
    const modal = await screen.findByRole("dialog");
    expect(within(modal).getByLabelText("Nome do Serviço *")).toHaveValue(
      "Alvará de funcionamento",
    );
    await user.click(within(modal).getByRole("tab", { name: "Acesso e Horário" }));
    expect(within(modal).getByLabelText("Início")).toHaveValue("08:00");
    await user.click(within(modal).getByRole("checkbox", { name: "Seg" }));
    await user.click(within(modal).getByRole("checkbox", { name: "Qua" }));
    await user.click(within(modal).getByRole("tab", { name: "Workflow" }));
    await user.click(within(modal).getByRole("button", { name: "Etapa" }));
    await user.type(
      within(modal).getByLabelText("Identificador *", { selector: "#etapa-id-1" }),
      "vistoria",
    );
    await user.selectOptions(
      within(modal).getByLabelText("Responsável", { selector: "#etapa-tipo-1" }),
      "unidade",
    );
    await user.selectOptions(
      await within(modal).findByLabelText("Unidade", { selector: "#etapa-resp-1" }),
      "u1",
    );
    await user.selectOptions(
      within(modal).getByLabelText("Responsável", { selector: "#etapa-tipo-0" }),
      "departamento",
    );
    await user.selectOptions(await within(modal).findByLabelText("Departamento"), "d1");
    await user.clear(within(modal).getByLabelText("Ordem", { selector: "#etapa-ordem-1" }));
    await user.type(within(modal).getByLabelText("Ordem", { selector: "#etapa-ordem-1" }), "2");
    await user.clear(
      within(modal).getByLabelText("Prazo (dias úteis)", { selector: "#etapa-prazo-1" }),
    );
    await user.type(
      within(modal).getByLabelText("Prazo (dias úteis)", { selector: "#etapa-prazo-1" }),
      "4",
    );
    await user.selectOptions(
      within(modal).getByLabelText("Status", { selector: "#etapa-status-1" }),
      "pendente",
    );
    await user.click(within(modal).getAllByRole("button", { name: "Remover etapa" })[0]!);
    await user.click(within(modal).getByRole("tab", { name: "Requisitos e Documentos" }));
    await user.click(within(modal).getAllByRole("button", { name: "Incluir" })[1]!);
    await user.type(
      within(modal).getByLabelText("Título *", { selector: "#Documentos\\ esperados-titulo-0" }),
      "RG",
    );
    await user.type(
      within(modal).getByLabelText("Texto", { selector: "#Documentos\\ esperados-texto-0" }),
      "Frente e verso",
    );
    await user.click(within(modal).getAllByRole("checkbox", { name: "Obrigatório" })[1]!);
    await user.click(within(modal).getByRole("button", { name: "Remover Ser maior" }));
    await user.click(within(modal).getByRole("button", { name: "Salvar alterações" }));
    const corpo = api.to("POST v1/protocolos/servicos")[1]?.body as Record<string, unknown>;
    expect(corpo).toMatchObject({
      id: "s1",
      nome: "Alvará de funcionamento",
      taxa_centavos: 18750,
      exige_agendamento: true,
      agenda: { dias: [2, 3], inicio: "08:00" },
      workflow: {
        ligado: true,
        etapas: [
          {
            id: "vistoria",
            ordem: 2,
            status: "pendente",
            prazo_dias: 4,
            responsavel_tipo: "unidade",
            responsavel_id: "u1",
          },
        ],
      },
      requisitos: [],
      documentos: [{ titulo: "RG", texto: "Frente e verso", obrigatorio: true }],
    });
  });

  it("cria um serviço novo, mostra a aba do campo que falta e cancela", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/protocolos/servicos": { data: [] },
      "GET v1/protocolos/formularios": { data: FORMS },
      "GET v1/iam/org-tree": { data: ORG },
      "POST v1/protocolos/servicos": { data: {} },
    });
    renderApp(<ServicosPage />);
    expect(await screen.findByText("Nenhum serviço encontrado")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Novo Serviço/ }));
    const modal = await screen.findByRole("dialog");
    await user.click(within(modal).getByRole("tab", { name: "Taxas e Prazos" }));
    await user.click(within(modal).getByRole("button", { name: "Salvar" }));
    // O nome obrigatório fica na primeira aba: ela volta a aparecer.
    expect(within(modal).getByRole("tab", { name: "Informações Gerais" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    await user.selectOptions(within(modal).getByLabelText("Unidade dona *"), "u1");
    await user.selectOptions(within(modal).getByLabelText("Formulário *"), "f1");
    await user.type(within(modal).getByLabelText("Nome do Serviço *"), "Certidão");
    await user.click(within(modal).getByRole("tab", { name: "Taxas e Prazos" }));
    await user.clear(within(modal).getByLabelText("Taxa (R$)"));
    await user.type(within(modal).getByLabelText("Taxa (R$)"), "12,5");
    await user.click(within(modal).getByRole("checkbox", { name: "Destaque na carta" }));
    await user.click(within(modal).getByRole("tab", { name: "Acesso e Horário" }));
    await user.click(within(modal).getByRole("checkbox", { name: "Exige horário" }));
    await user.click(within(modal).getByRole("checkbox", { name: "Exige horário" }));
    await user.click(within(modal).getByRole("tab", { name: "Workflow" }));
    await user.click(within(modal).getByRole("checkbox", { name: "Exige aceite antes de tratar" }));
    await user.click(within(modal).getByRole("checkbox", { name: "Workflow ligado" }));
    await user.selectOptions(within(modal).getByLabelText("Responsável"), "usuario");
    await user.click(within(modal).getByRole("button", { name: "Salvar" }));
    expect(api.to("POST v1/protocolos/servicos")[0]?.body).toMatchObject({
      unidade_id: "u1",
      formulario_id: "f1",
      nome: "Certidão",
      destaque: true,
      exige_aceite: true,
      exige_aprovacao: false,
      exige_agendamento: false,
      agenda: { dias: [] },
      workflow: { ligado: true },
    });
    expect(api.to("POST v1/protocolos/servicos")[0]?.body).not.toHaveProperty("id");
  });

  it("sem gerir não mostra o botão de novo nem o menu", async () => {
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"], scopes: [] }),
      "GET v1/protocolos/servicos": { data: SERVICOS },
      "GET v1/protocolos/formularios": { data: FORMS },
    });
    renderApp(<ServicosPage />);
    await screen.findByText("Alvará de funcionamento");
    expect(screen.queryByRole("button", { name: /Novo Serviço/ })).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Ações do serviço Alvará de funcionamento" }),
    ).not.toBeInTheDocument();
  });

  it("pede a leitura do diretório para escolher usuário na etapa", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:gerir"] }),
      "GET v1/protocolos/servicos": { data: [] },
      "GET v1/protocolos/formularios": { data: FORMS },
      "GET v1/iam/org-tree": { data: ORG },
    });
    renderApp(<ServicosPage />);
    await user.click(await screen.findByRole("button", { name: /Novo Serviço/ }));
    const modal = await screen.findByRole("dialog");
    await user.click(within(modal).getByRole("tab", { name: "Workflow" }));
    await user.click(within(modal).getByRole("checkbox", { name: "Workflow ligado" }));
    await user.selectOptions(within(modal).getByLabelText("Responsável"), "usuario");
    expect(
      within(modal).getByText("A lista de usuários pede a permissão de leitura do diretório."),
    ).toBeInTheDocument();
    await user.click(within(modal).getByRole("button", { name: "Cancelar" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("lista os usuários ativos para a etapa", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(),
      "GET v1/protocolos/servicos": { data: [] },
      "GET v1/protocolos/formularios": { data: FORMS },
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/users": {
        data: [
          { id: "us1", username: "ana", display_name: "Ana", active: true },
          { id: "us2", username: "bia", display_name: "", active: true },
          { id: "us3", username: "caio", display_name: "Caio", active: false },
        ],
        meta: { page: 1, page_size: 50, total_items: 3, total_pages: 1 },
      },
    });
    renderApp(<ServicosPage />);
    await user.click(await screen.findByRole("button", { name: /Novo Serviço/ }));
    const modal = await screen.findByRole("dialog");
    await user.click(within(modal).getByRole("tab", { name: "Workflow" }));
    await user.click(within(modal).getByRole("checkbox", { name: "Workflow ligado" }));
    await user.selectOptions(within(modal).getByLabelText("Responsável"), "usuario");
    const usuario = await within(modal).findByLabelText("Usuário");
    expect(await within(usuario).findByRole("option", { name: "Ana" })).toBeInTheDocument();
    expect(within(usuario).getByRole("option", { name: "bia" })).toBeInTheDocument();
    expect(within(usuario).queryByRole("option", { name: "Caio" })).not.toBeInTheDocument();
  });
});

describe("Formulários do protocolo", () => {
  it("lista com os serviços vinculados, alterna o status, duplica e edita", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/protocolos/formularios": { data: FORMS },
      "GET v1/protocolos/servicos": { data: SERVICOS },
      "GET v1/iam/org-tree": { data: ORG },
      "POST v1/protocolos/formularios": { data: {} },
    });
    renderApp(<FormulariosPage />);
    expect(await screen.findByText("Pedido de alvará")).toBeInTheDocument();
    expect(screen.getByText("Sem descrição")).toBeInTheDocument();
    expect(screen.getByText("1 campos")).toBeInTheDocument();
    expect(screen.getByText("0 campos")).toBeInTheDocument();
    expect(screen.getByText("Alvará de funcionamento")).toBeInTheDocument();
    expect(screen.getByText("20/11/2026")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Ativar Poda" }));
    expect(api.to("POST v1/protocolos/formularios")[0]?.body).toMatchObject({
      id: "f2",
      ativo: true,
      campos: [],
    });

    await user.type(screen.getByLabelText("Pesquisar"), "nada");
    expect(screen.getByText("Nenhum formulário encontrado")).toBeInTheDocument();
    await user.clear(screen.getByLabelText("Pesquisar"));

    await user.click(screen.getByRole("button", { name: "Duplicar Alvará" }));
    let modal = await screen.findByRole("dialog");
    expect(within(modal).getByLabelText("Título *")).toHaveValue("Alvará (cópia)");
    await user.click(within(modal).getByRole("button", { name: "Salvar" }));
    expect(api.to("POST v1/protocolos/formularios")[1]?.body).not.toHaveProperty("id");

    await user.click(await screen.findByRole("button", { name: "Editar Alvará" }));
    modal = await screen.findByRole("dialog");
    await user.click(within(modal).getByRole("button", { name: "Campo" }));
    await user.type(
      within(modal).getByLabelText("Rótulo *", { selector: "#campo-rotulo-1" }),
      "Tem placa",
    );
    await user.type(
      within(modal).getByLabelText("Identificador *", { selector: "#campo-id-1" }),
      "placa",
    );
    await user.selectOptions(
      within(modal).getByLabelText("Tipo", { selector: "#campo-tipo-1" }),
      "boolean",
    );
    await user.type(within(modal).getByLabelText(/revela/), "razao, ");
    await user.click(within(modal).getAllByRole("checkbox", { name: "Obrigatório" })[1]!);
    await user.click(within(modal).getByRole("button", { name: "Subir Tem placa" }));
    await user.click(within(modal).getByRole("button", { name: "Subir Tem placa" }));
    await user.click(within(modal).getByRole("button", { name: "Descer Razão social" }));
    await user.click(within(modal).getByRole("button", { name: "Campo" }));
    await user.type(
      within(modal).getByLabelText("Rótulo *", { selector: "#campo-rotulo-2" }),
      "Porte",
    );
    await user.type(
      within(modal).getByLabelText("Identificador *", { selector: "#campo-id-2" }),
      "porte",
    );
    await user.selectOptions(
      within(modal).getByLabelText("Tipo", { selector: "#campo-tipo-2" }),
      "select",
    );
    await user.type(within(modal).getByLabelText(/Opções/), "MEI, ME");
    await user.click(within(modal).getByRole("button", { name: "Campo" }));
    await user.click(within(modal).getByRole("button", { name: "Remover campo" }));
    await user.click(within(modal).getByRole("button", { name: "Salvar alterações" }));
    expect(api.to("POST v1/protocolos/formularios")[2]?.body).toMatchObject({
      id: "f1",
      ativo: true,
      campos: [
        { id: "placa", tipo: "boolean", rotulo: "Tem placa", obrigatorio: true, revela: ["razao"] },
        { id: "razao", rotulo: "Razão social" },
        { id: "porte", tipo: "select", opcoes: ["MEI", "ME"] },
      ],
    });
    const campos = (api.to("POST v1/protocolos/formularios")[2]?.body as { campos: object[] })
      .campos;
    expect(campos[0]).not.toHaveProperty("_k");
  });

  it("cria o primeiro formulário e cancela", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/protocolos/formularios": { data: [] },
      "GET v1/protocolos/servicos": { data: [] },
      "GET v1/iam/org-tree": { data: ORG },
      "POST v1/protocolos/formularios": { data: {} },
    });
    renderApp(<FormulariosPage />);
    expect(await screen.findByText("Nenhum formulário cadastrado")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Novo Formulário/ }));
    let modal = await screen.findByRole("dialog");
    await user.click(within(modal).getByRole("button", { name: "Cancelar" }));
    await user.click(screen.getByRole("button", { name: /Novo Formulário/ }));
    modal = await screen.findByRole("dialog");
    await user.selectOptions(within(modal).getByLabelText("Unidade dona *"), "u1");
    await user.type(within(modal).getByLabelText("Título *"), "Ouvidoria");
    await user.type(within(modal).getByLabelText("Instrução ao cidadão"), "Descreva");
    await user.click(within(modal).getByRole("button", { name: "Salvar" }));
    expect(api.to("POST v1/protocolos/formularios")[0]?.body).toMatchObject({
      unidade_id: "u1",
      titulo: "Ouvidoria",
      instrucao: "Descreva",
      ativo: true,
      campos: [{ id: "nome_social" }],
    });
  });

  it("sem gerir só lê", async () => {
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"], scopes: [] }),
      "GET v1/protocolos/formularios": { data: FORMS },
      "GET v1/protocolos/servicos": fail(403),
    });
    renderApp(<FormulariosPage />);
    await screen.findByText("Pedido de alvará");
    expect(screen.queryByRole("button", { name: /Novo Formulário/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Editar Alvará" })).not.toBeInTheDocument();
    expect(screen.getAllByText("Nenhum serviço").length).toBe(2);
  });
});

describe("Responsáveis técnicos", () => {
  it("filtra por situação, nome e CPF e cadastra", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/protocolos/responsaveis": {
        data: [
          { id: "r1", nome: "Carla Engenheira", cpf: "***.111.222-33", ativo: true },
          { id: "r2", nome: "Davi Arquiteto", cpf: "***.444.555-66", ativo: false },
        ],
      },
      "GET v1/iam/org-tree": { data: ORG },
      "POST v1/protocolos/responsaveis": {
        data: { id: "r3", nome: "Eva", cpf: "***.777.888-99", ativo: true },
      },
    });
    renderApp(<ResponsaveisPage />);
    expect(await screen.findByText("Carla Engenheira")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Inativos/ }));
    expect(screen.queryByText("Carla Engenheira")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Ativos/ }));
    expect(screen.queryByText("Davi Arquiteto")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Todos/ }));
    await user.type(screen.getByLabelText("Pesquisar"), "444");
    expect(screen.getByText("Davi Arquiteto")).toBeInTheDocument();
    expect(screen.queryByText("Carla Engenheira")).not.toBeInTheDocument();
    await user.clear(screen.getByLabelText("Pesquisar"));
    await user.type(screen.getByLabelText("Pesquisar"), "carla");
    expect(screen.getByText("Carla Engenheira")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Pesquisar"), "x");
    expect(screen.getByText("Nenhum responsável técnico")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Novo Responsável/ }));
    let modal = await screen.findByRole("dialog");
    await user.click(within(modal).getByRole("button", { name: "Cancelar" }));
    await user.click(screen.getByRole("button", { name: /Novo Responsável/ }));
    modal = await screen.findByRole("dialog");
    await user.selectOptions(within(modal).getByLabelText("Unidade dona *"), "u1");
    await user.type(within(modal).getByLabelText("Nome *"), "Eva");
    await user.type(within(modal).getByLabelText("CPF *"), "77788899900");
    await user.click(within(modal).getByRole("button", { name: "Cadastrar" }));
    expect(await screen.findByText(/ficou ativo/)).toHaveTextContent("Eva ficou ativo");
    expect(api.to("POST v1/protocolos/responsaveis")[0]?.body).toEqual({
      unidade_id: "u1",
      nome: "Eva",
      cpf: "77788899900",
    });
  });

  it("sem gerir avisa que não há acesso", async () => {
    mockBackend({ ...identityRoutes({ permissions: ["protocolos:tratar"], scopes: [] }) });
    renderApp(<ResponsaveisPage />);
    expect(await screen.findByText("Sem acesso ao cadastro")).toBeInTheDocument();
  });
});

describe("Balcão", () => {
  const CARTA = [
    { id: "s1", nome: "Alvará de funcionamento", categoria: "Tributos", unidade_nome: "PREF" },
    { id: "s2", nome: "Poda de árvore", categoria: "" },
  ];
  const ITEM = {
    servico: {
      id: "s1",
      nome: "Alvará de funcionamento",
      descricao: "Licença",
      exige_agendamento: true,
      prazo_dias: 15,
      taxa_centavos: 18750,
    },
    formulario: {
      titulo: "Alvará",
      instrucao: "Tenha o CNPJ",
      campos: [
        { id: "razao", tipo: "text", rotulo: "Razão social", obrigatorio: true },
        { id: "contrato", tipo: "file", rotulo: "Contrato", obrigatorio: false },
      ],
    },
  };

  it("escolhe o serviço, preenche, envia o arquivo e protocola", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/protocolos/carta": { data: CARTA },
      "GET v1/protocolos/carta/s1": { data: ITEM },
      "GET v1/protocolos/carta/s2": {
        data: {
          servico: { id: "s2", nome: "Poda de árvore", exige_agendamento: false },
          formulario: { titulo: "", campos: [] },
        },
      },
      "POST v1/protocolos/carta/s1/arquivo": {
        data: { upload_url: "https://minio/up", method: "PUT", object_key: "k1", headers: {} },
      },
      "PUT minio/up": { status: 200, data: {} },
      "POST v1/protocolos/balcao": { data: { id: "p9", numero: "PREF202690099" } },
    });
    renderApp(<BalcaoPage />);
    expect(await screen.findByText("Selecione um serviço")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Pesquisar"), "nada");
    expect(screen.getByText("Nenhum serviço encontrado.")).toBeInTheDocument();
    await user.clear(screen.getByLabelText("Pesquisar"));

    await user.click(screen.getByRole("button", { name: /Poda de árvore/ }));
    expect(await screen.findByText("Gratuito")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Trocar serviço" }));
    await user.click(screen.getByRole("button", { name: /Alvará de funcionamento/ }));
    expect(await screen.findByText("Taxa R$ 187,50")).toBeInTheDocument();
    expect(screen.getByText("Prazo de 15 dias úteis")).toBeInTheDocument();
    expect(screen.getByText("Tenha o CNPJ")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Nome completo *"), "Ana Paula");
    await user.type(screen.getByLabelText("CPF *"), "00807124362");
    await user.selectOptions(screen.getByLabelText("Prioridade"), "alta");
    await user.type(screen.getByLabelText("Razão social *"), "Padaria");
    await user.upload(
      screen.getByLabelText("Contrato"),
      new File(["x"], "contrato.pdf", { type: "application/pdf" }),
    );
    expect(await screen.findByText("Arquivo recebido.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Protocolar" }));
    expect(await screen.findByText("PREF202690099")).toBeInTheDocument();
    expect(api.to("POST v1/protocolos/balcao")[0]?.body).toMatchObject({
      servico_id: "s1",
      nome: "Ana Paula",
      prioridade: "alta",
      respostas: { razao: "Padaria", contrato: "k1" },
    });
    expect(screen.getByRole("link", { name: "Abrir a ficha" })).toHaveAttribute(
      "href",
      "/protocolos/p9",
    );
    await user.click(screen.getByRole("button", { name: "Novo atendimento" }));
    expect(screen.getByText("Selecione um serviço")).toBeInTheDocument();
  });

  it("mostra a falha da carta, do serviço e do arquivo", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(),
      "GET v1/protocolos/carta": { data: CARTA },
      "GET v1/protocolos/carta/s1": { data: ITEM },
      "GET v1/protocolos/carta/s2": fail(404, "Serviço fora da carta"),
      "POST v1/protocolos/carta/s1/arquivo": fail(400, "Arquivo grande demais"),
    });
    renderApp(<BalcaoPage />);
    await user.click(await screen.findByRole("button", { name: /Poda de árvore/ }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Serviço fora da carta");
    await user.click(screen.getByRole("button", { name: /Alvará de funcionamento/ }));
    await user.upload(
      await screen.findByLabelText("Contrato"),
      new File(["x"], "contrato.pdf", { type: "application/pdf" }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent("Arquivo grande demais");
  });

  it("sem abrir avisa que não há acesso e mostra a falha da carta", async () => {
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"], scopes: [] }),
      "GET v1/protocolos/carta": fail(503, "Carta fora do ar"),
    });
    renderApp(<BalcaoPage />);
    expect(await screen.findByText("Sem acesso ao balcão")).toBeInTheDocument();
    expect(await screen.findByRole("alert")).toHaveTextContent("Carta fora do ar");
  });
});
