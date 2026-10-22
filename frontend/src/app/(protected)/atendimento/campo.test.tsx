import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ArmazemMemoria, FilaCampo, usarFila } from "@/lib/atendimento/campo";
import { ME, fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtendimentoCampoPage from "./campo/page";
import SolicitacaoPage from "./solicitacoes/[id]/page";

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

const SOL = {
  id: "s1",
  numero: 9,
  tipo: "incidente",
  estado: "em_atendimento",
  titulo: "Sem rede na escola",
  descricao: "",
  oferta_id: "o1",
  oferta_nome: "Rede",
  unidade_id: "u1",
  solicitante_id: "x",
  solicitante_nome: "João",
  responsavel_id: ME.id,
  responsavel_nome: "Maria",
  created_at: "2026-10-21T12:00:00Z",
  updated_at: "2026-10-21T12:00:00Z",
  impacto: 2,
  urgencia: 2,
};

function visita(extra: Record<string, unknown> = {}) {
  return {
    id: "v1",
    solicitacao_id: "s1",
    tecnico_id: ME.id,
    tecnico_nome: "Maria",
    estado: "em_andamento",
    local: "Escola",
    base_legal: "LGPD",
    inicio_em: "2026-10-21T13:00:00Z",
    fora_da_area: false,
    assinante_nome: "",
    observacao: "",
    evidencias: [],
    ...extra,
  };
}

function detalhe(visitas: unknown[], papeis = ["tecnico"]) {
  return {
    data: { solicitacao: SOL, papeis, proximos: [], passagens: [], comentarios: [], visitas },
  };
}

const TICKET = {
  object_key: "atendimento/visitas/v1/a.png",
  upload_url: "https://minio.test/b/a?put",
  method: "PUT",
  headers: {},
  expires_at: "",
};

function gps(resposta: "ok" | "negado" = "ok") {
  Object.defineProperty(navigator, "geolocation", {
    configurable: true,
    value: {
      getCurrentPosition: (ok: PositionCallback, erro: PositionErrorCallback) =>
        resposta === "ok"
          ? ok({
              coords: { latitude: -23.55, longitude: -46.63, accuracy: 9 },
            } as GeolocationPosition)
          : erro({ code: 1 } as GeolocationPositionError),
    },
  });
}

beforeEach(() => {
  resetNavigation({ id: "s1" }, "/atendimento/solicitacoes/s1");
  usarFila(new FilaCampo(new ArmazemMemoria()));
  gps();
});
afterEach(() => usarFila(null));

describe("Atendimento — supervisão em campo", () => {
  it("lista quem está em visita, com distância e link para o mapa", async () => {
    mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/visitas/em-campo": {
        data: [
          {
            ...visita(),
            numero: 9,
            titulo: "Sem rede",
            inicio: { lat: -23.55, lng: -46.63, precisao_m: 9 },
            distancia_m: 1240,
            fora_da_area: true,
          },
          {
            ...visita({ id: "v2", tecnico_nome: "Bia", local: "" }),
            numero: 10,
            titulo: "Impressora",
          },
        ],
      },
    });
    renderApp(<AtendimentoCampoPage />);
    const tabela = await screen.findByRole("table", { name: "Visitas em andamento" });
    expect(within(tabela).getByText(/1,2 km/)).toBeInTheDocument();
    expect(within(tabela).getByText("Fora da área")).toBeInTheDocument();
    expect(within(tabela).getByText("Sem destino")).toBeInTheDocument();
    expect(within(tabela).getByRole("link", { name: "#9 · Sem rede" })).toHaveAttribute(
      "href",
      "/atendimento/solicitacoes/s1",
    );
    const mapa = within(tabela).getByRole("link", { name: /Ver no mapa/ });
    expect(mapa).toHaveAttribute(
      "href",
      "https://www.openstreetmap.org/?mlat=-23.55&mlon=-46.63#map=18/-23.55/-46.63",
    );
    expect(mapa).toHaveAttribute("rel", "noopener noreferrer");
    expect(within(tabela).getAllByRole("link", { name: /Ver no mapa/ })).toHaveLength(1);
  });

  it("vazia, com falha e sem permissão", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/visitas/em-campo": { data: [] },
    });
    const vazia = renderApp(<AtendimentoCampoPage />);
    expect(await screen.findByText("Ninguém em campo agora")).toBeInTheDocument();
    vazia.unmount();

    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/visitas/em-campo": fail(500, "fora"),
    });
    const erro = renderApp(<AtendimentoCampoPage />);
    await user.click(await screen.findByRole("button", { name: "Tentar novamente" }));
    await waitFor(() =>
      expect(api.to("GET v1/atendimento/visitas/em-campo").length).toBeGreaterThan(1),
    );
    erro.unmount();

    mockBackend(identityRoutes(perms("atendimento:abrir")));
    renderApp(<AtendimentoCampoPage />);
    expect(await screen.findByText("Seu perfil não atende a fila")).toBeInTheDocument();
  });
});

describe("Atendimento — visita do técnico", () => {
  it("chegada com a localização, foto, assinatura e saída", async () => {
    const user = userEvent.setup();
    let visitas: unknown[] = [];
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/solicitacoes/s1": () => detalhe(visitas),
      "GET v1/atendimento/solicitacoes/s1/responsaveis": { data: [] },
      "POST v1/atendimento/solicitacoes/s1/visitas": (req) => {
        visitas = [visita({ id: (req.body as { id: string }).id })];
        return { status: 201, data: visitas[0] };
      },
      "POST v1/atendimento/visitas/*": { data: TICKET },
      "PUT minio.test/b/a": { status: 200 },
    });
    renderApp(<SolicitacaoPage />);

    expect(await screen.findByText(/a sua localização é lida uma única vez/)).toBeInTheDocument();
    await user.type(screen.getByLabelText("Local da visita"), "Escola Municipal");
    await user.click(screen.getByRole("button", { name: "Registrar chegada" }));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/visitas")).toHaveLength(1),
    );
    const corpo = api.to("POST v1/atendimento/solicitacoes/s1/visitas")[0]!.body as Record<
      string,
      unknown
    >;
    expect(corpo).toMatchObject({
      local: "Escola Municipal",
      ponto: { lat: -23.55, lng: -46.63, precisao_m: 9 },
      destino: null,
    });
    expect(corpo.id).toMatch(/^[0-9a-f-]{36}$/);

    const visitaId = corpo.id as string;
    expect(await screen.findByText("Adicionar foto")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Registrar chegada" })).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Adicionar foto"), {
      target: { files: [new File(["jpeg"], "porta.jpg", { type: "image/jpeg" })] },
    });
    await waitFor(() =>
      expect(api.to(`POST v1/atendimento/visitas/${visitaId}/evidencias`)).toHaveLength(1),
    );
    expect(api.to(`POST v1/atendimento/visitas/${visitaId}/evidencias/upload`)[0]!.body).toEqual({
      nome: "porta.jpg",
      content_type: "image/jpeg",
    });
    fireEvent.change(screen.getByLabelText("Adicionar foto"), { target: { files: [] } });

    // Assinatura: o jsdom não desenha; simulamos o canvas.
    const traco = {
      beginPath: vi.fn(),
      moveTo: vi.fn(),
      lineTo: vi.fn(),
      stroke: vi.fn(),
      clearRect: vi.fn(),
    };
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(
      traco as unknown as CanvasRenderingContext2D,
    );
    vi.spyOn(HTMLCanvasElement.prototype, "toBlob").mockImplementation((cb) =>
      cb(new Blob(["png"], { type: "image/png" })),
    );
    const area = screen.getByRole("img", { name: "Área de assinatura, vazia" });
    fireEvent.pointerMove(area, { clientX: 5, clientY: 5 });
    fireEvent.pointerUp(area);
    fireEvent.pointerDown(area, { clientX: 10, clientY: 10 });
    fireEvent.pointerMove(area, { clientX: 40, clientY: 20 });
    fireEvent.pointerMove(area, { clientX: 60, clientY: 25 });
    fireEvent.pointerUp(area);
    expect(screen.getByRole("img", { name: "Área de assinatura, assinada" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Limpar assinatura" }));
    expect(traco.clearRect).toHaveBeenCalled();
    fireEvent.pointerDown(area, { clientX: 10, clientY: 10 });
    fireEvent.pointerMove(area, { clientX: 40, clientY: 20 });
    fireEvent.pointerLeave(area);

    await user.type(screen.getByLabelText("Nome de quem recebeu"), "Diretora Ana");
    await user.type(screen.getByLabelText("Observação"), "Cabo trocado");
    await user.click(screen.getByRole("button", { name: "Registrar saída" }));
    await waitFor(() =>
      expect(api.to(`POST v1/atendimento/visitas/${visitaId}/checkout`)).toHaveLength(1),
    );
    expect(api.to(`POST v1/atendimento/visitas/${visitaId}/checkout`)[0]!.body).toMatchObject({
      assinante_nome: "Diretora Ana",
      observacao: "Cabo trocado",
      ponto: { lat: -23.55 },
    });
    const evidencias = api
      .to(`POST v1/atendimento/visitas/${visitaId}/evidencias`)
      .map((c) => (c.body as { tipo: string }).tipo);
    expect(evidencias).toEqual(["foto", "assinatura"]);
    vi.restoreAllMocks();
  });

  it("sem rede, guarda a chegada no aparelho e envia quando a conexão volta", async () => {
    const user = userEvent.setup();
    let online = false;
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/solicitacoes/s1": detalhe([]),
      "GET v1/atendimento/solicitacoes/s1/responsaveis": { data: [] },
      "POST v1/atendimento/solicitacoes/s1/visitas": () => {
        if (!online) throw new TypeError("Failed to fetch");
        return { status: 201, data: {} };
      },
      "POST v1/atendimento/visitas/*": { data: {} },
    });
    renderApp(<SolicitacaoPage />);
    await user.click(await screen.findByRole("button", { name: "Registrar chegada" }));
    expect(await screen.findByText("Sem conexão")).toBeInTheDocument();
    expect(
      await screen.findByText(/guardada no aparelho até a conexão voltar/),
    ).toBeInTheDocument();
    expect(screen.getByText("1 registro(s) aguardando conexão")).toBeInTheDocument();
    // Com a chegada no aparelho, o técnico já pode encerrar ou cancelar.
    expect(screen.getByRole("button", { name: "Registrar saída" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancelar visita" }));
    await user.click(
      within(await screen.findByRole("dialog")).getByRole("button", { name: "Cancelar visita" }),
    );
    expect(await screen.findByText("2 registro(s) aguardando conexão")).toBeInTheDocument();

    online = true;
    act(() => {
      window.dispatchEvent(new Event("online"));
    });
    await waitFor(() => expect(screen.queryByText(/aguardando conexão/)).not.toBeInTheDocument());
    // Duas tentativas sem rede (chegada; chegada de novo ao cancelar) e o envio em ordem.
    expect(
      api.calls.filter((c) => c.method === "POST").map((c) => c.path.split("/").at(-1)),
    ).toEqual(["visitas", "visitas", "visitas", "cancelar"]);
  });

  it("localização negada e recusa do servidor", async () => {
    const user = userEvent.setup();
    gps("negado");
    mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/solicitacoes/s1": detalhe([]),
      "GET v1/atendimento/solicitacoes/s1/responsaveis": { data: [] },
      "POST v1/atendimento/solicitacoes/s1/visitas": fail(
        409,
        "você já tem uma visita em andamento nesta solicitação",
      ),
    });
    const negado = renderApp(<SolicitacaoPage />);
    await user.click(await screen.findByRole("button", { name: "Registrar chegada" }));
    expect(await screen.findByText(/A localização foi negada/)).toBeInTheDocument();
    negado.unmount();

    gps();
    renderApp(<SolicitacaoPage />);
    await user.click(await screen.findByRole("button", { name: "Registrar chegada" }));
    expect(await screen.findByText("Registro de campo recusado")).toBeInTheDocument();
    expect(
      screen.getByText("você já tem uma visita em andamento nesta solicitação"),
    ).toBeInTheDocument();
  });

  it("quem abriu vê as visitas e abre a assinatura", async () => {
    const user = userEvent.setup();
    const abrir = vi.spyOn(window, "open").mockReturnValue(null);
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/solicitacoes/s1": detalhe(
        [
          visita({
            estado: "concluida",
            fim_em: "2026-10-21T14:00:00Z",
            distancia_m: 80,
            fora_da_area: true,
            assinante_nome: "Diretora Ana",
            observacao: "ok",
            tecnico_id: "outro",
            evidencias: [
              {
                id: "e1",
                visita_id: "v1",
                tipo: "foto",
                content_type: "image/jpeg",
                tamanho: 1,
                created_at: "",
              },
              {
                id: "e2",
                visita_id: "v1",
                tipo: "assinatura",
                content_type: "image/png",
                tamanho: 1,
                created_at: "",
              },
            ],
          }),
        ],
        ["solicitante"],
      ),
      "GET v1/atendimento/visitas/v1/evidencias/e2": {
        data: { url: "https://minio.test/b/e2?get" },
      },
      "GET v1/atendimento/visitas/v1/evidencias/e1": fail(503, "armazenamento indisponível"),
    });
    renderApp(<SolicitacaoPage />);
    expect(await screen.findByText("Recebido por Diretora Ana")).toBeInTheDocument();
    expect(screen.getByText(/80 m do destino/)).toBeInTheDocument();
    expect(screen.getByText("Check-in fora da área do destino")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Registrar chegada" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Ver assinatura" }));
    await waitFor(() =>
      expect(abrir).toHaveBeenCalledWith(
        "https://minio.test/b/e2?get",
        "_blank",
        "noopener,noreferrer",
      ),
    );
    await user.click(screen.getByRole("button", { name: "Ver foto 1" }));
    expect(await screen.findByText("armazenamento indisponível")).toBeInTheDocument();
    expect(api.to("GET v1/atendimento/visitas/v1/evidencias/e1")).toHaveLength(1);
    abrir.mockRestore();
  });

  it("saída sem assinatura, de visita sem local", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("atendimento:atender")),
      "GET v1/atendimento/solicitacoes/s1": detalhe([visita({ local: "" })]),
      "GET v1/atendimento/solicitacoes/s1/responsaveis": { data: [] },
      "POST v1/atendimento/visitas/v1/checkout": { data: {} },
    });
    renderApp(<SolicitacaoPage />);
    await user.click(await screen.findByRole("button", { name: "Registrar saída" }));
    await waitFor(() => expect(api.to("POST v1/atendimento/visitas/v1/checkout")).toHaveLength(1));
    expect(api.calls.some((c) => c.path.includes("evidencias"))).toBe(false);
  });

  it("sem visitas e sem ação possível, o cartão não aparece", async () => {
    mockBackend({
      ...identityRoutes(perms("atendimento:abrir")),
      "GET v1/atendimento/solicitacoes/s1": detalhe([], ["solicitante"]),
    });
    renderApp(<SolicitacaoPage />);
    expect(await screen.findByRole("heading", { name: "Sem rede na escola" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Visitas" })).not.toBeInTheDocument();
  });
});
