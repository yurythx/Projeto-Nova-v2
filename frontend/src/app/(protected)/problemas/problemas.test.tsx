import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProblemasDoIncidente } from "@/components/atendimento/ProblemasDoIncidente";
import { fail, identityRoutes, mockBackend, page, renderApp } from "@/test/backend";
import { resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import ProblemaPage from "./[id]/page";
import ProblemasPage from "./page";

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "Tecnologia", sigla: "TI", departamentos: [] }],
  },
];

function problema(extra: Record<string, unknown> = {}) {
  return {
    id: "p1",
    numero: 5,
    titulo: "Quedas de rede",
    descricao: "cai à tarde",
    estado: "em_analise",
    causa_raiz: "",
    contorno: "",
    unidade_id: "u1",
    created_at: "2026-10-21T12:00:00Z",
    incidentes: [
      { solicitacao_id: "s1", numero: 9, titulo: "Caiu", vinculado_em: "2026-10-21T12:00:00Z" },
    ],
    ativos: [{ ativo_id: "a1", nome: "Switch", vinculado_em: "2026-10-21T12:00:00Z" }],
    proximos: ["erro_conhecido", "resolvido"],
    ...extra,
  };
}

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

beforeEach(() => resetNavigation({ id: "p1" }, "/problemas", "vista=lista"));

describe("Problemas — lista", () => {
  it("sugere a partir de incidentes repetidos, registra e filtra", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/problemas": {
        data: [problema({ estado: "erro_conhecido", contorno: "Reiniciar o switch" })],
      },
      "GET v1/problemas/sugestoes": {
        data: [
          {
            oferta_id: "o1",
            oferta_nome: "Sem rede",
            unidade_id: "u1",
            total: 3,
            ultimo_em: "2026-10-21T12:00:00Z",
            solicitacoes: ["s1", "s2", "s3"],
          },
        ],
      },
      "POST v1/problemas": { status: 201, data: problema() },
      "PUT v1/problemas/p1/incidentes/*": { data: problema() },
    });
    renderApp(<ProblemasPage />);
    expect(await screen.findByText("Contorno: Reiniciar o switch")).toBeInTheDocument();
    await user.click(await screen.findByRole("button", { name: "Abrir problema de Sem rede" }));
    await waitFor(() => expect(router.push).toHaveBeenCalledWith("/problemas/p1"));
    expect(api.to("POST v1/problemas")[0]!.body).toEqual({
      titulo: "Incidentes repetidos: Sem rede",
      unidade_id: "u1",
    });
    expect(
      api.calls.filter((c) => c.method === "PUT").map((c) => c.path.split("/").at(-1)),
    ).toEqual(["s1", "s2", "s3"]);

    await user.type(screen.getByLabelText("Título *"), "Lentidão");
    await user.click(screen.getByRole("button", { name: "Registrar problema" }));
    await waitFor(() => expect(api.to("POST v1/problemas")).toHaveLength(2));
    expect(api.to("POST v1/problemas")[1]!.body).toMatchObject({
      titulo: "Lentidão",
      descricao: "",
    });

    await user.selectOptions(screen.getByLabelText("Estado"), "erro_conhecido");
    await user.type(screen.getByLabelText("Buscar no título ou no contorno"), "sw");
    await waitFor(() => {
      const q = api.to("GET v1/problemas").at(-1)!.query;
      expect(q.get("estado")).toBe("erro_conhecido");
      expect(q.get("q")).toBe("sw");
    });
  });

  it("quem só consulta não registra; vazio, falha e sem permissão", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("problemas:read")),
      "GET v1/problemas": { data: [] },
    });
    const leitor = renderApp(<ProblemasPage />);
    expect(await screen.findByText("Nenhum problema neste recorte")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Registrar problema" })).not.toBeInTheDocument();
    expect(api.to("GET v1/problemas/sugestoes")).toHaveLength(0);
    leitor.unmount();

    const erro = mockBackend({
      ...identityRoutes(perms("problemas:read")),
      "GET v1/problemas": fail(500, "fora"),
    });
    const e = renderApp(<ProblemasPage />);
    await user.click(await screen.findByRole("button", { name: "Tentar novamente" }));
    await waitFor(() => expect(erro.to("GET v1/problemas").length).toBeGreaterThan(1));
    e.unmount();

    mockBackend(identityRoutes({ permissions: [], scopes: [] }));
    renderApp(<ProblemasPage />);
    expect(await screen.findByText("Seu perfil não consulta problemas")).toBeInTheDocument();
  });

  it("exibe o quadro kanban e permite transicionar para estados seguintes", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/problemas", "vista=quadro");
    const api = mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/problemas": {
        data: [
          problema({
            id: "p1",
            numero: 5,
            titulo: "Servidor lento",
            estado: "registrado",
            proximos: ["em_analise"],
          }),
        ],
      },
      "POST v1/problemas/p1/estado": { data: problema({ estado: "em_analise" }) },
    });
    renderApp(<ProblemasPage />);
    expect(await screen.findByRole("list", { name: "Quadro de problemas" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Registrados/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Em análise/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Erros conhecidos/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Resolvidos/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Encerrados/ })).toBeInTheDocument();

    const select = screen.getByLabelText("Mover o problema #5 para");
    await user.selectOptions(select, "em_analise");
    await waitFor(() => {
      const call = api.to("POST v1/problemas/p1/estado")[0];
      expect(call).toBeDefined();
      expect(call!.body).toEqual({ estado: "em_analise" });
    });
  });
});

describe("Problemas — detalhe", () => {
  it("investiga, muda o estado e liga incidentes e ativos", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/problemas/p1": {
        data: problema({ contorno: "Reiniciar", resolvido_em: "2026-10-22T12:00:00Z" }),
      },
      "GET v1/atendimento/fila": page([
        { id: "s1", numero: 9, titulo: "Caiu" },
        { id: "s2", numero: 10, titulo: "Caiu de novo" },
      ]),
      "GET v1/ativos": {
        data: [
          { id: "a1", nome: "Switch" },
          { id: "a2", nome: "Roteador" },
        ],
      },
      "GET v1/mudancas": {
        data: [{ id: "m1", numero: 3, titulo: "Trocar fonte", estado: "aprovada" }],
      },
      "PUT v1/problemas/p1": { data: problema() },
      "POST v1/problemas/p1/estado": { data: problema() },
      "PUT v1/problemas/p1/*": { data: problema() },
      "DELETE v1/problemas/p1/*": { data: problema() },
    });
    renderApp(<ProblemaPage />);
    expect(await screen.findByRole("heading", { name: "Quedas de rede" })).toBeInTheDocument();
    expect(screen.getByText(/Solução de contorno:/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "#9 · Caiu" })).toHaveAttribute(
      "href",
      "/atendimento/solicitacoes/s1",
    );
    expect(await screen.findByRole("link", { name: "#3 · Trocar fonte" })).toHaveAttribute(
      "href",
      "/mudancas/m1",
    );
    expect(screen.getByText(/resolvido em/)).toBeInTheDocument();

    await user.type(screen.getByLabelText("Causa raiz (exigida para resolver)"), "Fonte");
    await user.click(screen.getByRole("button", { name: "Salvar investigação" }));
    await waitFor(() => expect(api.to("PUT v1/problemas/p1")).toHaveLength(1));
    expect(api.to("PUT v1/problemas/p1")[0]!.body).toMatchObject({
      causa_raiz: "Fonte",
      contorno: "Reiniciar",
      unidade_id: "u1",
    });

    await user.click(screen.getByRole("button", { name: "Publicar como erro conhecido" }));
    await waitFor(() =>
      expect(api.to("POST v1/problemas/p1/estado")[0]!.body).toEqual({ estado: "erro_conhecido" }),
    );

    const escolher = await screen.findByLabelText("Ligar incidente");
    expect(within(escolher).queryByRole("option", { name: "#9 · Caiu" })).not.toBeInTheDocument();
    await user.selectOptions(escolher, "s2");
    await user.click(screen.getByRole("button", { name: "Ligar" }));
    await waitFor(() => expect(api.to("PUT v1/problemas/p1/incidentes/s2")).toHaveLength(1));
    await user.click(screen.getByRole("button", { name: "Desligar #9" }));
    await waitFor(() => expect(api.to("DELETE v1/problemas/p1/incidentes/s1")).toHaveLength(1));

    await user.selectOptions(await screen.findByLabelText("Ligar ativo"), "a2");
    await user.click(screen.getByRole("button", { name: "Ligar ativo" }));
    await waitFor(() => expect(api.to("PUT v1/problemas/p1/ativos/a2")).toHaveLength(1));
    await user.click(screen.getByRole("button", { name: "Desligar Switch" }));
    await waitFor(() => expect(api.to("DELETE v1/problemas/p1/ativos/a1")).toHaveLength(1));
  });

  it("encerrado, para quem só consulta, sem vínculos", async () => {
    const api = mockBackend({
      ...identityRoutes(perms("problemas:read")),
      "GET v1/problemas/p1": {
        data: problema({
          estado: "encerrado",
          causa_raiz: "Fonte",
          descricao: "",
          incidentes: [],
          ativos: [{ ativo_id: "a9", nome: "", vinculado_em: "" }],
          proximos: [],
        }),
      },
    });
    renderApp(<ProblemaPage />);
    expect(await screen.findByText("Fonte")).toBeInTheDocument();
    expect(screen.getByText("Nenhum incidente ligado.")).toBeInTheDocument();
    expect(screen.getByText("Ativo indisponível")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Salvar investigação" })).not.toBeInTheDocument();
    expect(screen.queryByRole("group", { name: "Estado do problema" })).not.toBeInTheDocument();
    expect(api.to("GET v1/mudancas")).toHaveLength(0);
  });

  it("sem ativos ligados e problema inexistente", async () => {
    mockBackend({
      ...identityRoutes(perms("problemas:manage")),
      "GET v1/problemas/p1": { data: problema({ ativos: [], incidentes: [] }) },
    });
    const vazio = renderApp(<ProblemaPage />);
    expect(await screen.findByText("Nenhum ativo ligado.")).toBeInTheDocument();
    vazio.unmount();
    mockBackend({
      ...identityRoutes(),
      "GET v1/problemas/p1": fail(404, "problema não encontrado"),
    });
    renderApp(<ProblemaPage />);
    expect(await screen.findByText("problema não encontrado")).toBeInTheDocument();
  });
});

describe("Problema no incidente", () => {
  it("mostra o problema e o contorno; sem permissão, nada", async () => {
    mockBackend({
      ...identityRoutes(perms("problemas:read")),
      "GET v1/problemas": {
        data: [
          problema({ estado: "erro_conhecido", contorno: "Reiniciar o switch" }),
          problema({ id: "p2", numero: 6 }),
        ],
      },
    });
    const { unmount } = renderApp(<ProblemasDoIncidente solicitacaoId="s1" />);
    expect(
      await screen.findByRole("link", { name: "Problema #5 · Quedas de rede" }),
    ).toHaveAttribute("href", "/problemas/p1");
    expect(screen.getByText("Reiniciar o switch")).toBeInTheDocument();
    unmount();
    const api = mockBackend(identityRoutes({ permissions: [], scopes: [] }));
    const { container } = render(<ProblemasDoIncidente solicitacaoId="s1" />);
    expect(container).toBeEmptyDOMElement();
    expect(api.to("GET v1/problemas")).toHaveLength(0);
  });
});
