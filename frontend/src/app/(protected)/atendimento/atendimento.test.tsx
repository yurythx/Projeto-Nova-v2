import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ME, fail, identityRoutes, mockBackend, page, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

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

describe("Atendimento — portal do servidor", () => {
  it("abre pela oferta e acompanha", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": { data: [oferta] },
      "GET v1/atendimento/minhas": (req) =>
        req.query.get("situacao") === "todas"
          ? page([{ ...sol, estado: "encerrado" }])
          : page([{ ...sol, estado: "aguardando_servidor" }]),
      "POST v1/atendimento/solicitacoes": { status: 201, data: sol },
    });
    renderApp(<AtendimentoPortalPage />);

    expect(await screen.findByText("Aguardando resposta do servidor")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "#42 · Senha expirou" })).toHaveAttribute(
      "href",
      "/atendimento/solicitacoes/s1",
    );
    expect(screen.queryByText("Incidente")).not.toBeInTheDocument();

    await user.selectOptions(await screen.findByLabelText("Preciso de ajuda com *"), "o1");
    expect(screen.getByText("senha, bloqueio")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Resumo *"), "Senha expirou");
    await user.type(screen.getByLabelText("Detalhes"), "desde hoje");
    await user.click(screen.getByRole("button", { name: "Enviar solicitação" }));
    await waitFor(() => expect(api.to("POST v1/atendimento/solicitacoes")).toHaveLength(1));
    expect(api.to("POST v1/atendimento/solicitacoes")[0]!.body).toEqual({
      oferta_id: "o1",
      titulo: "Senha expirou",
      descricao: "desde hoje",
    });
    expect(await screen.findByText("Solicitação enviada")).toBeInTheDocument();
    expect(screen.getByLabelText("Resumo *")).toHaveValue("");

    await user.click(screen.getByRole("tab", { name: "Todas" }));
    expect(await screen.findByText("Encerrada")).toBeInTheDocument();
  });

  it("preenche campos complementares e formata descrição estruturada", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": { data: [oferta] },
      "GET v1/atendimento/minhas": page([]),
      "POST v1/atendimento/solicitacoes": { status: 201, data: sol },
    });
    renderApp(<AtendimentoPortalPage />);

    await user.selectOptions(await screen.findByLabelText("Preciso de ajuda com *"), "o1");
    await user.type(screen.getByLabelText("Resumo *"), "Computador não liga");
    await user.type(screen.getByLabelText("Localização / Ramal"), "Bloco B - Sala 102");
    await user.type(screen.getByLabelText("Patrimônio / Equipamento"), "PAT-2024-9912");
    await user.selectOptions(screen.getByLabelText("Impacto percebido"), "setorial");
    await user.type(screen.getByLabelText("Detalhes"), "Houve oscilação de energia");

    await user.click(screen.getByRole("button", { name: "Enviar solicitação" }));
    await waitFor(() => expect(api.to("POST v1/atendimento/solicitacoes")).toHaveLength(1));

    const body = api.to("POST v1/atendimento/solicitacoes")[0]!.body as Record<string, string>;
    expect(body.oferta_id).toBe("o1");
    expect(body.titulo).toBe("Computador não liga");
    expect(body.descricao).toContain("[Informações Complementares]");
    expect(body.descricao).toContain("• Local / Ramal: Bloco B - Sala 102");
    expect(body.descricao).toContain("• Patrimônio / Ativo: PAT-2024-9912");
    expect(body.descricao).toContain("• Impacto: Afeta todo o setor / equipe");
    expect(body.descricao).toContain("Houve oscilação de energia");
  });

  it("sem ofertas, falha ao abrir e perfil sem abertura", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": { data: [] },
      "GET v1/atendimento/minhas": page([]),
    });
    const vazio = renderApp(<AtendimentoPortalPage />);
    expect(await screen.findByText("Nenhum atendimento disponível")).toBeInTheDocument();
    expect(screen.getByText("Nada em andamento")).toBeInTheDocument();
    vazio.unmount();

    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/catalogo": { data: [oferta] },
      "GET v1/atendimento/minhas": page([]),
      "POST v1/atendimento/solicitacoes": fail(422, "esta oferta não está mais disponível"),
    });
    const erro = renderApp(<AtendimentoPortalPage />);
    await user.selectOptions(await screen.findByLabelText("Preciso de ajuda com *"), "o1");
    await user.type(screen.getByLabelText("Resumo *"), "Pedido");
    await user.click(screen.getByRole("button", { name: "Enviar solicitação" }));
    expect(await screen.findByText("esta oferta não está mais disponível")).toBeInTheDocument();
    expect(screen.getByLabelText("Resumo *")).toHaveValue("Pedido");
    erro.unmount();

    const api = mockBackend(identityRoutes(perms("atendimento:atender")));
    renderApp(<AtendimentoPortalPage />);
    expect(await screen.findByText("Seu perfil não abre solicitações")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Ir para a fila da equipe" })).toHaveAttribute(
      "href",
      "/atendimento/fila",
    );
    expect(api.to("GET v1/atendimento/catalogo")).toHaveLength(0);
  });
});

describe("Atendimento — fila da equipe", () => {
  it("filtra e assume", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/fila": page([
        sol,
        { ...sol, id: "s2", numero: 43, responsavel_id: "x", responsavel_nome: "Ana" },
      ]),
      "PUT v1/atendimento/solicitacoes/s1/responsavel": { data: {} },
    });
    renderApp(<AtendimentoFilaPage />);

    const tabela = await screen.findByRole("table", { name: "Solicitações da fila" });
    expect(within(tabela).getAllByText("Incidente")).toHaveLength(2);
    expect(within(tabela).getByText("Ana")).toBeInTheDocument();
    expect(within(tabela).getAllByRole("button", { name: /Assumir/ })).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Assumir a solicitação #42" }));
    await waitFor(() =>
      expect(api.to("PUT v1/atendimento/solicitacoes/s1/responsavel")).toHaveLength(1),
    );
    expect(api.to("PUT v1/atendimento/solicitacoes/s1/responsavel")[0]!.body).toEqual({
      responsavel_id: ME.id,
    });

    await user.selectOptions(screen.getByLabelText("Responsável"), "nenhum");
    await user.selectOptions(screen.getByLabelText("Tipo"), "requisicao");
    await user.selectOptions(screen.getByLabelText("Estado"), "aberto");
    await user.click(screen.getByLabelText("Incluir encerradas e canceladas"));
    await waitFor(() => {
      const q = api.to("GET v1/atendimento/fila").at(-1)!.query;
      expect(q.get("responsavel")).toBe("nenhum");
      expect(q.get("tipo")).toBe("requisicao");
      expect(q.get("estado")).toBe("aberto");
      expect(q.get("situacao")).toBe("todas");
    });
  });

  it("fila vazia e perfil sem atendimento", async () => {
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
        }),
      },
    });
    const fim = renderApp(<SolicitacaoPage />);
    expect(await screen.findByText("Encerrada em")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Enviar" })).not.toBeInTheDocument();
    expect(screen.getByText("Nenhuma mensagem ainda.")).toBeInTheDocument();
    fim.unmount();

    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/solicitacoes/s1": fail(404, "registro de atendimento não encontrado"),
    });
    renderApp(<SolicitacaoPage />);
    expect(await screen.findByText("registro de atendimento não encontrado")).toBeInTheDocument();
  });
});
