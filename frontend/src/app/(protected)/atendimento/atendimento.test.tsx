import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ME, fail, identityRoutes, mockBackend, page, renderApp } from "@/test/backend";
import { resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtendimentoFilaPage from "./fila/page";
import AtendimentoPortalPage from "./page";
import SolicitacaoPage from "./solicitacoes/[id]/page";

const oferta = {
  id: "o1",
  nome: "Não consigo acessar o e-mail",
  descricao: "senha, bloqueio",
  tipo: "incidente",
  ativa: true,
};

const sol = {
  id: "s1",
  numero: 42,
  tipo: "incidente",
  estado: "aberto",
  titulo: "Senha expirou",
  descricao: "desde hoje",
  oferta_id: "o1",
  oferta_nome: oferta.nome,
  unidade_id: "u1",
  solicitante_id: "outro",
  solicitante_nome: "João",
  created_at: "2026-10-21T12:00:00Z",
  updated_at: "2026-10-21T12:00:00Z",
};

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

beforeEach(() => resetNavigation({ id: "s1" }));

const categoria = {
  id: "c1",
  nome: "Acesso",
  descricao: "Contas e senhas",
  icone: "key-round",
  ordem: 1,
  ativa: true,
};

describe("Atendimento — central do solicitante", () => {
  it("início em blocos, com o que está em andamento", async () => {
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/minhas": page([{ ...sol, estado: "aguardando_servidor" }]),
    });
    renderApp(<AtendimentoPortalPage />);

    const portas = await screen.findByRole("list", { name: "O que você precisa" });
    const links = within(portas).getAllByRole("link");
    expect(links.map((l) => l.getAttribute("href"))).toEqual([
      "/atendimento?ver=pedido",
      "/atendimento?ver=problema",
      "/atendimento?ver=chamados",
      "/wiki",
    ]);
    expect(await within(portas).findByText("1 em andamento")).toBeInTheDocument();
    expect(await screen.findByText("Aguardando resposta do servidor")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "#42 · Senha expirou" })).toHaveAttribute(
      "href",
      "/atendimento/solicitacoes/s1",
    );
    expect(screen.queryByText("Incidente")).not.toBeInTheDocument();
  });

  it("catálogo em blocos: categorias da porta, Outros e busca", async () => {
    resetNavigation({}, "/atendimento", "ver=problema");
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": {
        data: [
          { ...oferta, categoria_id: "c1", icone: "mail" },
          { ...oferta, id: "o2", nome: "Impressora parou" },
          { ...oferta, id: "o3", nome: "Novo computador", tipo: "requisicao" },
        ],
      },
      "GET v1/atendimento/catalogo/categorias": { data: [categoria] },
    });
    const lista = renderApp(<AtendimentoPortalPage />);
    const blocos = await screen.findByRole("list", { name: "Categorias" });
    expect(
      within(blocos)
        .getAllByRole("link")
        .map((l) => l.getAttribute("href")),
    ).toEqual([
      "/atendimento?ver=problema&categoria=c1",
      "/atendimento?ver=problema&categoria=outros",
    ]);
    expect(screen.queryByText("Novo computador")).not.toBeInTheDocument();
    await user.type(screen.getByLabelText("Procurar no catálogo"), "impressora");
    const achados = await screen.findByRole("list", { name: "Atendimentos" });
    expect(within(achados).getByRole("link", { name: /Impressora parou/ })).toHaveAttribute(
      "href",
      "/atendimento?oferta=o2",
    );
    await user.clear(screen.getByLabelText("Procurar no catálogo"));
    await user.type(screen.getByLabelText("Procurar no catálogo"), "xyz");
    expect(await screen.findByText("Nada encontrado com “xyz”.")).toBeInTheDocument();
    lista.unmount();

    resetNavigation({}, "/atendimento", "ver=problema&categoria=c1");
    renderApp(<AtendimentoPortalPage />);
    const ofertas = await screen.findByRole("list", { name: "Atendimentos" });
    expect(within(ofertas).getAllByRole("link")).toHaveLength(1);
    expect(
      within(ofertas).getByRole("link", { name: /Não consigo acessar o e-mail/ }),
    ).toHaveAttribute("href", "/atendimento?oferta=o1");
    const trilha = screen.getByRole("navigation", { name: "Você está em" });
    expect(within(trilha).getByRole("link", { name: "Relatar um problema" })).toHaveAttribute(
      "href",
      "/atendimento?ver=problema",
    );
    expect(within(trilha).getByText("Acesso")).toHaveAttribute("aria-current", "page");
  });

  it("separa o catálogo pelas 3 entidades provedoras (SECITI, SMS, SEMED) e pré-seleciona a lotação", async () => {
    resetNavigation({}, "/atendimento", "ver=problema&categoria=c1");
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({
        permissions: ["atendimento:abrir"],
        scopes: [
          {
            perfil: "servidor",
            unidade_id: "u-sms",
            origem: "ad",
            permissions: ["atendimento:abrir"],
          },
        ],
      }),
      "GET v1/atendimento/catalogo": {
        data: [
          {
            ...oferta,
            id: "o-pref",
            nome: "Impressora parou",
            categoria_id: "c1",
            secretaria_id: "u-seciti",
            secretaria_nome: "Ciência, Tecnologia e Inovação",
            secretaria_sigla: "SECITI",
          },
          {
            ...oferta,
            id: "o-sms",
            nome: "Impressora parou",
            categoria_id: "c1",
            secretaria_id: "u-sms",
            secretaria_nome: "Secretaria Municipal de Saúde",
            secretaria_sigla: "SMS",
          },
          {
            ...oferta,
            id: "o-semed",
            nome: "Impressora parou",
            categoria_id: "c1",
            secretaria_id: "u-semed",
            secretaria_nome: "Secretaria Municipal de Educação",
            secretaria_sigla: "SEMED",
          },
        ],
      },
      "GET v1/atendimento/catalogo/categorias": { data: [categoria] },
    });
    renderApp(<AtendimentoPortalPage />);

    // Servidor lotado na SMS já vê apenas a oferta da SMS pré-selecionada.
    const ofertas = await screen.findByRole("list", { name: "Atendimentos" });
    expect(within(ofertas).getAllByRole("link")).toHaveLength(1);
    expect(within(ofertas).getByRole("link")).toHaveAttribute("href", "/atendimento?oferta=o-sms");

    // Pode alternar para a Educação (SEMED) ou ver todas as 3 centrais identificadas pela sigla.
    const abas = screen.getByRole("tablist", { name: "Entidade provedora do atendimento" });
    await user.click(within(abas).getByRole("tab", { name: /SEMED/ }));
    expect(within(ofertas).getByRole("link")).toHaveAttribute(
      "href",
      "/atendimento?oferta=o-semed",
    );

    await user.click(within(abas).getByRole("tab", { name: "Todas as centrais" }));
    expect(within(ofertas).getAllByRole("link")).toHaveLength(3);
  });

  it("abre pela oferta e vai para o chamado criado", async () => {
    resetNavigation({}, "/atendimento", "oferta=o1");
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": { data: [oferta] },
      "POST v1/atendimento/solicitacoes": { status: 201, data: sol },
    });
    renderApp(<AtendimentoPortalPage />);

    expect(await screen.findByRole("heading", { name: oferta.nome })).toBeInTheDocument();
    expect(screen.getByText("senha, bloqueio")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Resumo *"), "Senha expirou");
    await user.type(screen.getByLabelText("Detalhes"), "desde hoje");
    await user.click(screen.getByRole("button", { name: "Abrir chamado" }));
    await waitFor(() => expect(api.to("POST v1/atendimento/solicitacoes")).toHaveLength(1));
    expect(api.to("POST v1/atendimento/solicitacoes")[0]!.body).toEqual({
      oferta_id: "o1",
      titulo: "Senha expirou",
      descricao: "desde hoje",
    });
    expect(await screen.findByText("Chamado aberto. Acompanhe por aqui.")).toBeInTheDocument();
    expect(router.push).toHaveBeenCalledWith("/atendimento/solicitacoes/s1");
  });

  it("preenche campos complementares e formata descrição estruturada", async () => {
    resetNavigation({}, "/atendimento", "oferta=o1");
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": {
        data: [
          {
            ...oferta,
            campos: [
              { chave: "ramal", rotulo: "Ramal", tipo: "numero", obrigatorio: true },
              {
                chave: "sistema",
                rotulo: "Sistema",
                tipo: "lista",
                obrigatorio: false,
                opcoes: ["SEI", "Folha"],
              },
            ],
          },
        ],
      },
      "POST v1/atendimento/solicitacoes": { status: 201, data: sol },
    });
    renderApp(<AtendimentoPortalPage />);

    await user.type(await screen.findByLabelText("Ramal *"), "3450");
    await user.selectOptions(screen.getByLabelText("Sistema"), "SEI");
    await user.type(screen.getByLabelText("Resumo *"), "Computador não liga");
    await user.type(screen.getByLabelText("Localização / Ramal"), "Bloco B - Sala 102");
    await user.type(screen.getByLabelText("Patrimônio / Equipamento"), "PAT-2024-9912");
    await user.selectOptions(screen.getByLabelText("Impacto percebido"), "setorial");
    await user.type(screen.getByLabelText("Detalhes"), "Houve oscilação de energia");

    await user.click(screen.getByRole("button", { name: "Abrir chamado" }));
    await waitFor(() => expect(api.to("POST v1/atendimento/solicitacoes")).toHaveLength(1));

    const body = api.to("POST v1/atendimento/solicitacoes")[0]!.body as Record<string, unknown>;
    expect(body.oferta_id).toBe("o1");
    expect(body.titulo).toBe("Computador não liga");
    expect(body.respostas).toEqual({ ramal: "3450", sistema: "SEI" });
    expect(body.descricao).toContain("[Informações Complementares]");
    expect(body.descricao).toContain("• Local / Ramal: Bloco B - Sala 102");
    expect(body.descricao).toContain("• Patrimônio / Ativo: PAT-2024-9912");
    expect(body.descricao).toContain("• Impacto: Afeta todo o setor / equipe");
    expect(body.descricao).toContain("Houve oscilação de energia");
  });

  it("meus chamados: em andamento e todos", async () => {
    resetNavigation({}, "/atendimento", "ver=chamados");
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/minhas": (req) =>
        req.query.get("situacao") === "todas" ? page([{ ...sol, estado: "encerrado" }]) : page([]),
    });
    renderApp(<AtendimentoPortalPage />);
    expect(await screen.findByText("Nada em andamento")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Todos" }));
    expect(await screen.findByText("Encerrada")).toBeInTheDocument();
  });

  it("sem ofertas, oferta indisponível, falha ao abrir e perfil sem abertura", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/atendimento", "ver=pedido");
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": { data: [oferta] },
      "GET v1/atendimento/catalogo/categorias": { data: [] },
    });
    const vazio = renderApp(<AtendimentoPortalPage />);
    expect(await screen.findByText("Nenhum atendimento disponível aqui")).toBeInTheDocument();
    vazio.unmount();

    resetNavigation({}, "/atendimento", "oferta=sumiu");
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": { data: [oferta] },
    });
    const sumiu = renderApp(<AtendimentoPortalPage />);
    expect(await screen.findByText("Este atendimento não está disponível")).toBeInTheDocument();
    sumiu.unmount();

    resetNavigation({}, "/atendimento", "oferta=o1");
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": { data: [oferta] },
      "POST v1/atendimento/solicitacoes": fail(422, "esta oferta não está mais disponível"),
    });
    const erro = renderApp(<AtendimentoPortalPage />);
    await user.type(await screen.findByLabelText("Resumo *"), "Pedido");
    await user.click(screen.getByRole("button", { name: "Abrir chamado" }));
    expect(await screen.findByText("esta oferta não está mais disponível")).toBeInTheDocument();
    expect(screen.getByLabelText("Resumo *")).toHaveValue("Pedido");
    expect(router.push).not.toHaveBeenCalled();
    erro.unmount();

    resetNavigation();
    const api = mockBackend(identityRoutes(perms("atendimento:atender")));
    renderApp(<AtendimentoPortalPage />);
    expect(await screen.findByText("Seu perfil não abre chamados")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Ir para a fila da equipe" })).toHaveAttribute(
      "href",
      "/atendimento/fila",
    );
    expect(api.to("GET v1/atendimento/catalogo")).toHaveLength(0);
  });
});

describe("Atendimento — fila da equipe", () => {
  it("filtra e assume", async () => {
    resetNavigation({ id: "s1" }, "/atendimento/fila", "vista=lista");
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/fila": page([
        sol,
        { ...sol, id: "s2", numero: 43, responsavel_id: "x", responsavel_nome: "Ana" },
      ]),
      "GET v1/iam/org-tree": {
        data: [
          { id: "e1", nome: "PREFEITURA", unidades: [{ id: "u1", nome: "TI", departamentos: [] }] },
        ],
      },
      "GET v1/atendimento/catalogo/categorias": {
        data: [{ id: "cat1", nome: "Rede", descricao: "", icone: "", ordem: 0, ativa: true }],
      },
      "PUT v1/atendimento/solicitacoes/s1/responsavel": { data: {} },
    });
    renderApp(<AtendimentoFilaPage />);

    const tabela = await screen.findByRole("table", { name: "Solicitações da fila" });
    expect(within(tabela).getAllByText(/^Incidente ·/)).toHaveLength(2);
    expect(within(tabela).getByText("Ana")).toBeInTheDocument();
    expect(within(tabela).getAllByRole("button", { name: /Assumir/ })).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Assumir a solicitação #42" }));
    await waitFor(() =>
      expect(api.to("PUT v1/atendimento/solicitacoes/s1/responsavel")).toHaveLength(1),
    );
    expect(api.to("PUT v1/atendimento/solicitacoes/s1/responsavel")[0]!.body).toEqual({
      responsavel_id: ME.id,
    });

    // Como no GLPI: a mais recém-atualizada primeiro, filtros recolhidos.
    const ultima = () => api.to("GET v1/atendimento/fila").at(-1)!.query;
    expect(ultima().get("ordem")).toBe("atualizada");
    expect(ultima().get("sentido")).toBe("desc");
    expect(screen.getByText("Mostrando 1–2 de 2")).toBeInTheDocument();
    const filtros = screen.getByRole("button", { name: /^Filtros/ });
    expect(filtros).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByLabelText("Técnico")).not.toBeVisible();

    // Cada mudança vai para a URL e volta à primeira página.
    const urls = () => router.replace.mock.calls.map((c) => String(c[0]));
    await user.type(screen.getByRole("searchbox", { name: "Buscar chamados" }), "senha");
    await user.click(screen.getByRole("button", { name: "Buscar" }));
    expect(urls().at(-1)).toBe("/atendimento/fila?vista=lista&q=senha");

    await user.click(filtros);
    expect(filtros).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByLabelText("Técnico")).toBeVisible();
    await user.selectOptions(screen.getByLabelText("Técnico"), "nenhum");
    expect(urls().at(-1)).toBe("/atendimento/fila?vista=lista&responsavel=nenhum");
    await user.selectOptions(screen.getByLabelText("Unidade"), "u1");
    expect(urls().at(-1)).toBe("/atendimento/fila?vista=lista&unidade=u1");
    await user.click(screen.getByRole("button", { name: /^ID/ }));
    expect(urls().at(-1)).toBe("/atendimento/fila?vista=lista&ordem=numero&sentido=asc");
    await user.selectOptions(screen.getByLabelText("Por página"), "100");
    expect(urls().at(-1)).toBe("/atendimento/fila?vista=lista&por=100");
  });

  it("os filtros da URL viram a consulta, e limpar tira os do painel", async () => {
    resetNavigation(
      { id: "s1" },
      "/atendimento/fila",
      "vista=lista&q=senha&situacao=todas&estado=aberto&tipo=requisicao&responsavel=eu&unidade=u1" +
        "&categoria=cat1&prioridade=alta&de=2026-10-01&ate=2026-10-31&ordem=numero&sentido=desc" +
        "&pagina=2&por=20",
    );
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/fila": page([sol]),
      "GET v1/iam/org-tree": { data: [] },
      "GET v1/atendimento/catalogo/categorias": { data: [] },
    });
    renderApp(<AtendimentoFilaPage />);
    await screen.findByRole("table", { name: "Solicitações da fila" });
    const q = api.to("GET v1/atendimento/fila").at(-1)!.query;
    expect(Object.fromEntries(q.entries())).toEqual({
      q: "senha",
      situacao: "todas",
      estado: "aberto",
      tipo: "requisicao",
      responsavel: "eu",
      unidade: "u1",
      categoria: "cat1",
      prioridade: "alta",
      aberta_de: new Date(2026, 9, 1).toISOString(),
      aberta_ate: new Date(2026, 10, 1).toISOString(),
      ordem: "numero",
      sentido: "desc",
      page: "2",
      page_size: "20",
    });
    expect(screen.getByRole("searchbox", { name: "Buscar chamados" })).toHaveValue("senha");
    expect(screen.getByRole("columnheader", { name: /^ID/ })).toHaveAttribute(
      "aria-sort",
      "descending",
    );
    expect(screen.getByRole("button", { name: /^Filtros \(9\)/ })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Limpar filtros" }));
    expect(String(router.replace.mock.calls.at(-1)![0])).toBe(
      "/atendimento/fila?vista=lista&q=senha&ordem=numero&sentido=desc&por=20",
    );
  });

  it("fila vazia e perfil sem atendimento", async () => {
    resetNavigation({ id: "s1" }, "/atendimento/fila", "vista=lista");
    mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/fila": page([]),
    });
    const vazia = renderApp(<AtendimentoFilaPage />);
    expect(await screen.findByText("Fila vazia")).toBeInTheDocument();
    vazia.unmount();

    const api = mockBackend(identityRoutes(perms("atendimento:abrir")));
    renderApp(<AtendimentoFilaPage />);
    expect(await screen.findByText("Seu perfil não atende a fila")).toBeInTheDocument();
    expect(api.to("GET v1/atendimento/fila")).toHaveLength(0);
  });
});

function detalhe(extra: Record<string, unknown> = {}) {
  return {
    solicitacao: sol,
    papeis: ["tecnico"],
    proximos: ["cancelado"],
    passagens: [{ id: 1, para: "aberto", autor_nome: "João", em: sol.created_at }],
    comentarios: [],
    ...extra,
  };
}

describe("Atendimento — solicitação", () => {
  it("a equipe assume, passa adiante, comenta com nota interna e muda o estado", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/solicitacoes/s1": {
        data: detalhe({
          solicitacao: {
            ...sol,
            estado: "em_atendimento",
            responsavel_id: "ana",
            responsavel_nome: "Ana",
          },
          proximos: ["aguardando_servidor", "aguardando_terceiro", "resolvido", "cancelado"],
          comentarios: [
            {
              id: "c1",
              autor_nome: "Ana",
              texto: "Conta bloqueada",
              interno: true,
              created_at: sol.created_at,
            },
          ],
        }),
      },
      "GET v1/atendimento/solicitacoes/s1/responsaveis": {
        data: [
          { id: "ana", nome: "Ana" },
          { id: "bia", nome: "Bia" },
        ],
      },
      "PUT v1/atendimento/solicitacoes/s1/responsavel": { data: detalhe() },
      "POST v1/atendimento/solicitacoes/s1/estado": { data: detalhe() },
      "POST v1/atendimento/solicitacoes/s1/comentarios": { status: 201, data: {} },
    });
    renderApp(<SolicitacaoPage />);

    expect(await screen.findByRole("heading", { name: "Senha expirou" })).toBeInTheDocument();
    expect(screen.getByText("Solicitação #42")).toBeInTheDocument();
    expect(screen.getByText("Incidente")).toBeInTheDocument();
    expect(screen.getByText("Nota interna")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Voltar para a fila" })).toHaveAttribute(
      "href",
      "/atendimento/fila",
    );

    await user.click(screen.getByRole("button", { name: "Assumir" }));
    await waitFor(() =>
      expect(api.to("PUT v1/atendimento/solicitacoes/s1/responsavel")).toHaveLength(1),
    );
    expect(api.to("PUT v1/atendimento/solicitacoes/s1/responsavel")[0]!.body).toEqual({
      responsavel_id: ME.id,
    });

    expect(await screen.findByRole("option", { name: "Bia" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Ana" })).not.toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Passar para"), "bia");
    await user.click(screen.getByRole("button", { name: "Definir responsável" }));
    await waitFor(() =>
      expect(api.to("PUT v1/atendimento/solicitacoes/s1/responsavel")).toHaveLength(2),
    );
    expect(api.to("PUT v1/atendimento/solicitacoes/s1/responsavel")[1]!.body).toEqual({
      responsavel_id: "bia",
    });

    await user.click(screen.getByRole("button", { name: "Pedir informação ao servidor" }));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/estado")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/estado")[0]!.body).toEqual({
      estado: "aguardando_servidor",
    });
    expect(screen.getByRole("button", { name: "Marcar como resolvida" })).toBeInTheDocument();

    await user.type(screen.getByLabelText("Mensagem"), "Desbloqueei a conta");
    await user.click(screen.getByLabelText("Nota interna (quem abriu não vê)"));
    await user.click(screen.getByRole("button", { name: "Enviar" }));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/comentarios")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/comentarios")[0]!.body).toEqual({
      texto: "Desbloqueei a conta",
      interno: true,
    });

    await user.click(screen.getByRole("button", { name: "Cancelar solicitação" }));
    const dialogo = await screen.findByRole("dialog");
    await user.click(within(dialogo).getByRole("button", { name: "Cancelar solicitação" }));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/estado")).toHaveLength(2),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/estado")[1]!.body).toEqual({
      estado: "cancelado",
    });
  });

  it("quem abriu responde, confirma ou reabre — sem nota interna", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/solicitacoes/s1": {
        data: detalhe({
          solicitacao: {
            ...sol,
            estado: "resolvido",
            solicitante_id: ME.id,
            resolvida_em: sol.updated_at,
          },
          papeis: ["solicitante"],
          proximos: ["encerrado", "em_atendimento"],
        }),
      },
      "POST v1/atendimento/solicitacoes/s1/estado": { data: detalhe() },
      "POST v1/atendimento/solicitacoes/s1/comentarios": { status: 201, data: {} },
    });
    renderApp(<SolicitacaoPage />);

    expect(await screen.findByText(/A equipe concluiu o atendimento/)).toBeInTheDocument();
    expect(screen.queryByText("Incidente")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Nota interna (quem abriu não vê)")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Assumir" })).not.toBeInTheDocument();
    expect(api.to("GET v1/atendimento/solicitacoes/s1/responsaveis")).toHaveLength(0);
    expect(screen.getByRole("button", { name: "Não resolveu, reabrir" })).toBeInTheDocument();

    await user.type(screen.getByLabelText("Sua resposta"), "Funcionou");
    await user.click(screen.getByRole("button", { name: "Enviar" }));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/comentarios")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/comentarios")[0]!.body).toEqual({
      texto: "Funcionou",
      interno: false,
    });

    await user.click(screen.getByRole("button", { name: "Confirmar e encerrar" }));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/estado")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/estado")[0]!.body).toEqual({
      estado: "encerrado",
    });
  });

  it("o espelho do GLPI é só para consulta", async () => {
    mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/solicitacoes/s1": {
        data: detalhe({
          solicitacao: {
            ...sol,
            estado: "em_atendimento",
            responsavel_id: "ana",
            responsavel_nome: "Ana",
          },
          proximos: [],
          espelho: true,
        }),
      },
    });
    renderApp(<SolicitacaoPage />);
    expect(await screen.findByText(/continua sendo atendido lá/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Assumir" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Enviar" })).not.toBeInTheDocument();
  });

  it("aguardando o servidor, encerrada e inexistente", async () => {
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/solicitacoes/s1": {
        data: detalhe({
          solicitacao: { ...sol, estado: "aguardando_servidor" },
          papeis: ["solicitante"],
          proximos: [],
          passagens: [{ id: 1, para: "aberto", autor_nome: "", em: sol.created_at }],
          comentarios: [
            {
              id: "c1",
              autor_nome: "",
              texto: "Qual o seu ramal?",
              interno: false,
              created_at: sol.created_at,
            },
          ],
        }),
      },
    });
    const esperando = renderApp(<SolicitacaoPage />);
    expect(await screen.findByText(/A equipe precisa de uma informação sua/)).toBeInTheDocument();
    expect(screen.getByText("Usuário removido")).toBeInTheDocument();
    expect(screen.getByText("Qual o seu ramal?")).toBeInTheDocument();
    expect(screen.queryByText("Nota interna")).not.toBeInTheDocument();
    esperando.unmount();

    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/solicitacoes/s1": {
        data: detalhe({
          solicitacao: { ...sol, estado: "encerrado", encerrada_em: sol.updated_at },
          papeis: ["solicitante"],
          proximos: [],
          passagens: [
            { id: 1, para: "aberto", autor_nome: "João", em: sol.created_at },
            {
              id: 2,
              de: "aberto",
              para: "encerrado",
              autor_nome: "Ana",
              em: "2026-10-21T14:00:00Z",
            },
          ],
          comentarios: [
            {
              id: "c1",
              autor_id: "ana",
              autor_nome: "Ana",
              texto: "Pode testar agora",
              interno: false,
              created_at: "2026-10-21T13:00:00Z",
            },
            {
              id: "c2",
              autor_nome: "Ana",
              texto: "segredo da equipe",
              interno: true,
              created_at: "2026-10-21T13:30:00Z",
            },
          ],
        }),
      },
    });
    const fim = renderApp(<SolicitacaoPage />);
    expect(await screen.findByText("Encerrada em")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Enviar" })).not.toBeInTheDocument();
    // Quem pediu vê a linha do tempo: a abertura, as mensagens públicas e as
    // passagens, em ordem; a nota interna não entra nem se chegar.
    const linha = screen.getByRole("list", { name: "Linha do tempo do chamado" });
    const itens = within(linha).getAllByRole("listitem");
    expect(itens).toHaveLength(3);
    expect(itens[0]).toHaveTextContent("Chamado aberto por João");
    expect(itens[0]).toHaveTextContent("desde hoje");
    expect(itens[1]).toHaveTextContent("Ana");
    expect(itens[1]).toHaveTextContent("Pode testar agora");
    expect(itens[2]).toHaveTextContent("Passou para Encerrada · Ana");
    expect(screen.queryByText("segredo da equipe")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Histórico" })).not.toBeInTheDocument();
    fim.unmount();

    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/solicitacoes/s1": fail(404, "registro de atendimento não encontrado"),
    });
    renderApp(<SolicitacaoPage />);
    expect(await screen.findByText("registro de atendimento não encontrado")).toBeInTheDocument();
  });
});
