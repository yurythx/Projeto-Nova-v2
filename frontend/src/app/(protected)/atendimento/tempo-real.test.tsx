import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  ME,
  fail,
  identityRoutes,
  mockBackend,
  mockWebSocket,
  page,
  renderApp,
  renderRealtime,
  wsTicketRoute,
} from "@/test/backend";
import { resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtendimentoFilaPage from "./fila/page";
import AtendimentoPortalPage from "./page";
import SolicitacaoPage from "./solicitacoes/[id]/page";

function sol(id: string, extra: Record<string, unknown> = {}) {
  return {
    id,
    numero: Number(id.replace(/\D/g, "")) || 1,
    tipo: "incidente",
    estado: "aberto",
    titulo: `Pedido ${id}`,
    descricao: "",
    oferta_id: "o1",
    oferta_nome: "E-mail",
    unidade_id: "u1",
    solicitante_id: "outro",
    solicitante_nome: "João",
    created_at: "2026-10-21T12:00:00Z",
    updated_at: "2026-10-21T12:00:00Z",
    proximos: [],
    ...extra,
  };
}

const QUADRO = [
  sol("s1", { proximos: ["em_atendimento", "cancelado"] }),
  sol("s2", {
    estado: "em_atendimento",
    responsavel_id: "ana",
    responsavel_nome: "Ana",
    proximos: ["aguardando_servidor", "aguardando_terceiro", "resolvido", "cancelado"],
  }),
  sol("s3", { tipo: "requisicao", estado: "atendido", proximos: ["encerrado", "em_atendimento"] }),
  sol("s4", { estado: "aguardando_terceiro", proximos: undefined }),
  sol("s5", { tipo: "requisicao", estado: "em_atendimento", proximos: ["atendido"] }),
  sol("s6", { estado: "encerrado" }),
];

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

function coluna(nome: string) {
  return screen
    .getByRole("heading", { name: new RegExp(`^${nome}`) })
    .closest("section") as HTMLElement;
}

describe("Atendimento — quadro", () => {
  beforeEach(() => resetNavigation({}, "/atendimento/fila", "vista=quadro"));

  it("mostra as colunas e move pelo seletor e arrastando, só para onde a máquina deixa", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/fila": page(QUADRO),
      "GET v1/atendimento/fila/topicos": { data: { topicos: ["atendimento:unidade:u1"] } },
      "POST v1/atendimento/solicitacoes/s1/estado": { data: {} },
      "POST v1/atendimento/solicitacoes/s2/estado": { data: {} },
      "POST v1/atendimento/solicitacoes/s5/estado": { data: {} },
    });
    renderApp(<AtendimentoFilaPage />);

    expect(await screen.findByRole("link", { name: "Pedido s1" })).toBeInTheDocument();
    expect(within(coluna("Abertas")).getByText("Pedido s1")).toBeInTheDocument();
    expect(within(coluna("Em atendimento")).getByText("Pedido s2")).toBeInTheDocument();
    expect(within(coluna("Concluídas")).getByText("Pedido s3")).toBeInTheDocument();
    expect(screen.queryByLabelText("Estado")).not.toBeInTheDocument();
    const q = api.to("GET v1/atendimento/fila").at(-1)!.query;
    expect(q.get("page_size")).toBe("100");
    expect(q.get("estado")).toBeNull();

    // Sem passagens, o cartão não tem seletor nem arrasta.
    const parado = screen.getByRole("link", { name: "Pedido s4" }).closest("li")!;
    expect(parado).toHaveAttribute("draggable", "false");
    expect(screen.queryByLabelText("Mover a solicitação #4 para")).not.toBeInTheDocument();
    fireEvent.dragStart(parado, { dataTransfer: { effectAllowed: "" } });
    fireEvent.dragOver(coluna("Em atendimento"));
    fireEvent.drop(coluna("Em atendimento"));

    // Teclado / toque: o seletor lista só as passagens permitidas, sem cancelar.
    const seletor = screen.getByLabelText("Mover a solicitação #1 para");
    expect(within(seletor).queryByRole("option", { name: "Cancelada" })).not.toBeInTheDocument();
    fireEvent.change(seletor, { target: { value: "" } });
    expect(api.to("POST v1/atendimento/solicitacoes/s1/estado")).toHaveLength(0);
    await user.selectOptions(seletor, "em_atendimento");
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/estado")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/estado")[0]!.body).toEqual({
      estado: "em_atendimento",
    });

    // Mouse: soltar numa coluna proibida não chama a API; na permitida, chama.
    const cartao = screen.getByRole("link", { name: "Pedido s2" }).closest("li")!;
    fireEvent.dragStart(cartao, { dataTransfer: { effectAllowed: "" } });
    fireEvent.dragOver(coluna("Abertas"));
    fireEvent.drop(coluna("Abertas"));
    expect(api.to("POST v1/atendimento/solicitacoes/s2/estado")).toHaveLength(0);
    fireEvent.dragStart(cartao, { dataTransfer: { effectAllowed: "" } });
    fireEvent.dragOver(coluna("Concluídas"));
    fireEvent.dragLeave(coluna("Abertas"));
    fireEvent.dragLeave(coluna("Concluídas"));
    fireEvent.dragOver(coluna("Concluídas"));
    fireEvent.drop(coluna("Concluídas"));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s2/estado")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s2/estado")[0]!.body).toEqual({
      estado: "resolvido",
    });
    // A requisição concluída vai para "atendido", não "resolvido".
    const pedido = screen.getByRole("link", { name: "Pedido s5" }).closest("li")!;
    fireEvent.dragStart(pedido, { dataTransfer: { effectAllowed: "" } });
    fireEvent.dragOver(coluna("Concluídas"));
    fireEvent.drop(coluna("Concluídas"));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s5/estado")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s5/estado")[0]!.body).toEqual({
      estado: "atendido",
    });
    expect(screen.queryByText("Pedido s6")).not.toBeInTheDocument();
    fireEvent.dragStart(cartao, { dataTransfer: { effectAllowed: "" } });
    fireEvent.dragEnd(cartao);
    expect(api.to("POST v1/atendimento/solicitacoes/s2/estado")).toHaveLength(1);

    await user.click(screen.getByRole("tab", { name: "Lista" }));
    expect(router.replace).toHaveBeenCalledWith("/atendimento/fila?vista=lista");
  });

  it("recusa do domínio: o cartão fica e a fila recarrega", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/fila": {
        ...page(QUADRO),
        meta: { page: 1, page_size: 100, total_items: 140, total_pages: 2 },
      },
      "GET v1/atendimento/fila/topicos": { data: { topicos: [] } },
      "POST v1/atendimento/solicitacoes/s1/estado": fail(
        409,
        "assuma a solicitação ou escolha um responsável antes de atender",
      ),
    });
    renderApp(<AtendimentoFilaPage />);
    expect(await screen.findByText(/O quadro mostra as 6 mais antigas de 140/)).toBeInTheDocument();
    const antes = api.to("GET v1/atendimento/fila").length;
    await user.selectOptions(
      screen.getByLabelText("Mover a solicitação #1 para"),
      "em_atendimento",
    );
    expect(
      await screen.findByText("assuma a solicitação ou escolha um responsável antes de atender"),
    ).toBeInTheDocument();
    await waitFor(() => expect(api.to("GET v1/atendimento/fila").length).toBeGreaterThan(antes));
    expect(within(coluna("Abertas")).getByText("Pedido s1")).toBeInTheDocument();
  });

  it("da lista para o quadro", async () => {
    resetNavigation({}, "/atendimento/fila", "vista=lista");
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/fila": page(QUADRO),
      "GET v1/atendimento/fila/topicos": { data: { topicos: [] } },
    });
    renderApp(<AtendimentoFilaPage />);
    expect(await screen.findByRole("table", { name: "Solicitações da fila" })).toBeInTheDocument();
    expect(screen.getByText("Atualização automática indisponível")).toBeInTheDocument();
    await user.click(screen.getByRole("tab", { name: "Quadro" }));
    expect(router.replace).toHaveBeenCalledWith("/atendimento/fila?vista=quadro");
  });
});

describe("Atendimento — tempo real", () => {
  it("a fila assina os tópicos das unidades e recarrega com a mudança de um colega", async () => {
    resetNavigation({}, "/atendimento/fila");
    const ws = mockWebSocket();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      ...wsTicketRoute,
      "GET v1/atendimento/fila": page(QUADRO),
      "GET v1/atendimento/fila/topicos": {
        data: { topicos: ["atendimento:unidade:u1", "atendimento:unidade:u2"] },
      },
    });
    renderRealtime(<AtendimentoFilaPage />);
    await waitFor(() =>
      expect(ws.socket().sent).toEqual(
        expect.arrayContaining([
          { type: "subscribe", topic: "atendimento:unidade:u1" },
          { type: "subscribe", topic: "atendimento:unidade:u2" },
        ]),
      ),
    );
    expect(await screen.findByText("Atualização automática ligada")).toBeInTheDocument();
    const antes = api.to("GET v1/atendimento/fila").length;
    act(() =>
      ws
        .socket()
        .receive({ type: "atendimento.solicitacao", topic: "atendimento:outra", data: {} }),
    );
    act(() =>
      ws.socket().receive({
        type: "atendimento.solicitacao",
        topic: "atendimento:unidade:u2",
        data: { id: "s9" },
      }),
    );
    await waitFor(() => expect(api.to("GET v1/atendimento/fila").length).toBe(antes + 1));
  });

  it("o detalhe ouve o tópico do papel: a equipe, inclusive notas; quem abriu, só o seu", async () => {
    resetNavigation({ id: "s1" });
    const ws = mockWebSocket();
    const detalhe = (papeis: string[]) => ({
      data: { solicitacao: sol("s1"), papeis, proximos: [], passagens: [], comentarios: [] },
    });
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      ...wsTicketRoute,
      "GET v1/atendimento/solicitacoes/s1": detalhe(["tecnico"]),
      "GET v1/atendimento/solicitacoes/s1/responsaveis": { data: [] },
    });
    const equipe = renderRealtime(<SolicitacaoPage />);
    await waitFor(() =>
      expect(ws.socket().sent).toContainEqual({
        type: "subscribe",
        topic: "atendimento:equipe:s1",
      }),
    );
    const antes = api.to("GET v1/atendimento/solicitacoes/s1").length;
    act(() =>
      ws
        .socket()
        .receive({ type: "atendimento.solicitacao", topic: "atendimento:equipe:s1", data: {} }),
    );
    await waitFor(() =>
      expect(api.to("GET v1/atendimento/solicitacoes/s1").length).toBe(antes + 1),
    );
    equipe.unmount();

    const ws2 = mockWebSocket();
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      ...wsTicketRoute,
      "GET v1/atendimento/solicitacoes/s1": detalhe(["solicitante"]),
    });
    renderRealtime(<SolicitacaoPage />);
    await waitFor(() =>
      expect(ws2.socket().sent).toContainEqual({
        type: "subscribe",
        topic: "atendimento:solicitacao:s1",
      }),
    );
    expect(ws2.socket().sent).not.toContainEqual({
      type: "subscribe",
      topic: "atendimento:equipe:s1",
    });
  });

  it("a lista de quem abriu recarrega pelo tópico pessoal", async () => {
    resetNavigation({}, "/atendimento");
    const ws = mockWebSocket();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      ...wsTicketRoute,
      "GET v1/atendimento/catalogo": { data: [] },
      "GET v1/atendimento/minhas": page([sol("s1", { solicitante_id: ME.id })]),
    });
    renderRealtime(<AtendimentoPortalPage />);
    expect(await screen.findByRole("link", { name: "#1 · Pedido s1" })).toBeInTheDocument();
    await waitFor(() => expect(ws.socket().readyState).toBe(1));
    const antes = api.to("GET v1/atendimento/minhas").length;
    act(() => ws.socket().receive({ type: "outro.frame", topic: `user:${ME.id}`, data: {} }));
    act(() =>
      ws.socket().receive({ type: "atendimento.solicitacao", topic: `user:${ME.id}`, data: {} }),
    );
    await waitFor(() => expect(api.to("GET v1/atendimento/minhas").length).toBe(antes + 1));
  });
});
