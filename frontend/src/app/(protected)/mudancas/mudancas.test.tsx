import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { fail, identityRoutes, mockBackend, page, renderApp } from "@/test/backend";
import { resetNavigation, router } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import MudancaPage from "./[id]/page";
import MudancasPage from "./page";

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "Tecnologia", sigla: "TI", departamentos: [] }],
  },
];

function mudanca(extra: Record<string, unknown> = {}) {
  return {
    id: "m1",
    numero: 3,
    titulo: "Trocar fonte",
    descricao: "fonte nova",
    justificativa: "",
    tipo: "normal",
    risco: "alto",
    estado: "rascunho",
    plano_reversao: "voltar a antiga",
    inicio_planejado: "2026-10-22T12:00:00Z",
    fim_planejado: "2026-10-22T14:00:00Z",
    resultado: "",
    unidade_id: "u1",
    solicitante_id: "x",
    solicitante_nome: "Ana",
    created_at: "2026-10-21T12:00:00Z",
    aprovacoes: [],
    vinculos: [],
    proximos: ["em_avaliacao", "cancelada"],
    ...extra,
  };
}

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

beforeEach(() => resetNavigation({ id: "m1" }, "/mudancas", "vista=lista"));

describe("Mudanças — lista", () => {
  it("registra o rascunho e filtra pelo estado", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/mudancas": {
        data: [
          mudanca(),
          mudanca({ id: "m2", numero: 4, inicio_planejado: undefined, risco: "baixo" }),
        ],
      },
      "POST v1/mudancas": { status: 201, data: mudanca() },
    });
    renderApp(<MudancasPage />);
    expect(await screen.findByRole("link", { name: "#3 · Trocar fonte" })).toHaveAttribute(
      "href",
      "/mudancas/m1",
    );
    expect(screen.getByText(/Sem janela · pedida por Ana/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Nova mudança" }));
    await user.type(screen.getByLabelText("Título *"), "Atualizar servidor");
    await user.selectOptions(screen.getByLabelText("Tipo"), "padrao");
    await user.type(screen.getByLabelText("Início planejado"), "2026-10-23T10:00");
    await user.click(screen.getByRole("button", { name: "Registrar mudança" }));
    await waitFor(() => expect(router.push).toHaveBeenCalledWith("/mudancas/m1"));
    const corpo = api.to("POST v1/mudancas")[0]!.body as Record<string, unknown>;
    expect(corpo).toMatchObject({
      titulo: "Atualizar servidor",
      tipo: "padrao",
      risco: "medio",
      fim_planejado: null,
      produto_id: null,
    });
    expect(corpo.inicio_planejado).toBe(new Date("2026-10-23T10:00").toISOString());

    await user.selectOptions(screen.getByLabelText("Estado"), "aprovada");
    await waitFor(() =>
      expect(api.to("GET v1/mudancas").at(-1)!.query.get("estado")).toBe("aprovada"),
    );
  });

  it("utiliza a calculadora de risco do CAB para definir score e justificativa", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/mudancas": { data: [] },
      "POST v1/mudancas": { status: 201, data: mudanca({ id: "m-risco", risco: "alto" }) },
    });
    renderApp(<MudancasPage />);

    await user.click(await screen.findByRole("button", { name: "Nova mudança" }));
    await user.type(screen.getByLabelText("Título *"), "Migração do Cluster Kubernetes");
    // Abre a calculadora de risco do CAB
    await user.click(screen.getByRole("button", { name: /Calculadora de Risco ITIL/i }));

    // Seleciona Impacto Alto (3), Complexidade Alta (3) e Rollback Crítico (3)
    await user.selectOptions(screen.getByLabelText("1. Impacto no Serviço"), "3");
    await user.selectOptions(screen.getByLabelText("2. Complexidade / Testes"), "3");
    await user.selectOptions(screen.getByLabelText("3. Plano de Reversão"), "3");

    // Aplica o risco sugerido (Alto)
    await user.click(
      screen.getByRole("button", { name: /Aplicar risco sugerido \(Risco alto\)/i }),
    );

    // O seletor de risco foi automaticamente atualizado para alto
    expect(screen.getByLabelText("Risco")).toHaveValue("alto");

    await user.click(screen.getByRole("button", { name: "Registrar mudança" }));
    await waitFor(() => expect(api.to("POST v1/mudancas")).toHaveLength(1));

    const corpo = api.to("POST v1/mudancas")[0]!.body as Record<string, unknown>;
    expect(corpo.titulo).toBe("Migração do Cluster Kubernetes");
    expect(corpo.risco).toBe("alto");
    expect(corpo.risco_score).toBe(15);
    expect(corpo.justificativa).toContain("[Cálculo de Risco CAB: Score 15/15 - Risco alto]");
  });

  it("aprovador só vê; vazio, falha e sem permissão", async () => {
    const user = userEvent.setup();
    mockBackend({ ...identityRoutes(perms("mudancas:aprovar")), "GET v1/mudancas": { data: [] } });
    const ap = renderApp(<MudancasPage />);
    expect(await screen.findByText("Nenhuma mudança neste recorte")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Nova mudança" })).not.toBeInTheDocument();
    ap.unmount();

    const erro = mockBackend({
      ...identityRoutes(perms("mudancas:aprovar")),
      "GET v1/mudancas": fail(500, "fora"),
    });
    const e = renderApp(<MudancasPage />);
    await user.click(await screen.findByRole("button", { name: "Tentar novamente" }));
    await waitFor(() => expect(erro.to("GET v1/mudancas").length).toBeGreaterThan(1));
    e.unmount();

    mockBackend(identityRoutes({ permissions: [], scopes: [] }));
    renderApp(<MudancasPage />);
    expect(await screen.findByText("Seu perfil não gere nem aprova mudanças")).toBeInTheDocument();
  });

  it("exibe o quadro kanban de mudanças e permite mover entre colunas", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/mudancas", "vista=quadro");
    const api = mockBackend({
      ...identityRoutes(perms("mudancas:manage")),
      "GET v1/mudancas": {
        data: [
          mudanca({
            id: "m1",
            numero: 3,
            titulo: "Trocar fonte",
            estado: "rascunho",
            proximos: ["em_avaliacao"],
          }),
        ],
      },
      "POST v1/mudancas/m1/estado": { data: mudanca({ estado: "em_avaliacao" }) },
    });
    renderApp(<MudancasPage />);
    expect(await screen.findByRole("list", { name: "Quadro de mudanças" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Rascunho/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Em avaliação/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Aprovadas/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Em execução/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Concluídas/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Rejeitada \/ Falhou/ })).toBeInTheDocument();

    const select = screen.getByLabelText("Mover a mudança #3 para");
    await user.selectOptions(select, "em_avaliacao");
    await waitFor(() => {
      const call = api.to("POST v1/mudancas/m1/estado")[0];
      expect(call).toBeDefined();
      expect(call!.body).toEqual({ estado: "em_avaliacao" });
    });
  });
});

describe("Mudanças — detalhe", () => {
  it("edita o rascunho, envia e liga problema, incidente e ativo", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/mudancas/m1": {
        data: mudanca({
          vinculos: [
            { tipo: "problema", alvo_id: "p1", rotulo: "#5 · Quedas", vinculado_em: "" },
            { tipo: "incidente", alvo_id: "s1", rotulo: "#9 · Caiu", vinculado_em: "" },
            { tipo: "ativo", alvo_id: "a1", rotulo: "Switch", vinculado_em: "" },
          ],
        }),
      },
      "GET v1/problemas": {
        data: [
          { id: "p1", numero: 5, titulo: "Quedas" },
          { id: "p2", numero: 6, titulo: "Lentidão" },
        ],
      },
      "GET v1/atendimento/fila": page([{ id: "s2", numero: 10, titulo: "Sem rede" }]),
      "GET v1/ativos": { data: [{ id: "a2", nome: "Roteador" }] },
      "PUT v1/mudancas/m1": { data: mudanca() },
      "POST v1/mudancas/m1/estado": { data: mudanca() },
      "PUT v1/mudancas/m1/vinculos/*": { data: mudanca() },
      "DELETE v1/mudancas/m1/vinculos/*": { data: mudanca() },
    });
    renderApp(<MudancaPage />);
    expect(await screen.findByRole("heading", { name: "Trocar fonte" })).toBeInTheDocument();
    expect(screen.getByText("Ainda sem decisão.")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "#5 · Quedas" })).toHaveAttribute(
      "href",
      "/problemas/p1",
    );
    expect(screen.getByRole("link", { name: "#9 · Caiu" })).toHaveAttribute(
      "href",
      "/atendimento/solicitacoes/s1",
    );
    expect(screen.getByRole("link", { name: "Switch" })).toHaveAttribute("href", "/ativos/a1");

    const titulo = screen.getByLabelText("Título *");
    await user.clear(titulo);
    await user.type(titulo, "Trocar a fonte");
    await user.click(screen.getByRole("button", { name: "Salvar mudança" }));
    await waitFor(() => expect(api.to("PUT v1/mudancas/m1")).toHaveLength(1));
    expect(api.to("PUT v1/mudancas/m1")[0]!.body).toMatchObject({
      titulo: "Trocar a fonte",
      plano_reversao: "voltar a antiga",
      produto_id: null,
    });

    await user.click(screen.getByRole("button", { name: "Enviar para aprovação" }));
    await waitFor(() =>
      expect(api.to("POST v1/mudancas/m1/estado")[0]!.body).toEqual({
        estado: "em_avaliacao",
        texto: "",
        resolver_vinculos: false,
      }),
    );

    await user.selectOptions(await screen.findByLabelText("Item"), "p2");
    expect(screen.queryByRole("option", { name: "#5 · Quedas" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Ligar" }));
    await waitFor(() => expect(api.to("PUT v1/mudancas/m1/vinculos/problema/p2")).toHaveLength(1));

    await user.selectOptions(
      screen.getByLabelText("Tipo", { selector: "#mudanca-vinculo-tipo" }),
      "incidente",
    );
    await user.selectOptions(await screen.findByRole("combobox", { name: "Item" }), "s2");
    await user.click(screen.getByRole("button", { name: "Ligar" }));
    await waitFor(() => expect(api.to("PUT v1/mudancas/m1/vinculos/incidente/s2")).toHaveLength(1));

    await user.selectOptions(
      screen.getByLabelText("Tipo", { selector: "#mudanca-vinculo-tipo" }),
      "ativo",
    );
    await waitFor(() =>
      expect(screen.getByRole("option", { name: "Roteador" })).toBeInTheDocument(),
    );
    await user.selectOptions(screen.getByRole("combobox", { name: "Item" }), "a2");
    await user.click(screen.getByRole("button", { name: "Ligar" }));
    await waitFor(() => expect(api.to("PUT v1/mudancas/m1/vinculos/ativo/a2")).toHaveLength(1));

    await user.click(screen.getByRole("button", { name: "Desligar Switch" }));
    await waitFor(() => expect(api.to("DELETE v1/mudancas/m1/vinculos/ativo/a1")).toHaveLength(1));
  });

  it("o aprovador decide com parecer; a rejeição o exige", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms("mudancas:aprovar")),
      "GET v1/mudancas/m1": {
        data: mudanca({
          estado: "em_avaliacao",
          justificativa: "fonte queimando",
          resultado: "",
          aprovacoes: [
            {
              id: "d1",
              aprovador: "",
              decisao: "automatica",
              parecer: "",
              em: "2026-10-21T12:00:00Z",
            },
            {
              id: "d2",
              aprovador: "Bia",
              decisao: "rejeitada",
              parecer: "falta janela",
              em: "2026-10-21T13:00:00Z",
            },
          ],
          proximos: ["aprovada", "rejeitada"],
        }),
      },
      "POST v1/mudancas/m1/estado": { data: mudanca() },
    });
    renderApp(<MudancaPage />);
    expect(await screen.findByText("fonte queimando")).toBeInTheDocument();
    expect(screen.getByText("Pré-aprovada (padrão)")).toBeInTheDocument();
    expect(screen.getByText("falta janela")).toBeInTheDocument();
    expect(screen.getByText(/· Bia/)).toBeInTheDocument();
    expect(screen.getByText("Nada ligado ainda.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Ligar" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Rejeitar" }));
    expect(screen.getByLabelText("Parecer *")).toBeRequired();
    await user.click(screen.getByRole("button", { name: "Voltar" }));
    expect(screen.queryByLabelText("Parecer *")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Aprovar" }));
    await user.type(screen.getByLabelText("Parecer"), "ok, na janela");
    await user.click(screen.getByRole("button", { name: "Confirmar: aprovar" }));
    await waitFor(() =>
      expect(api.to("POST v1/mudancas/m1/estado")[0]!.body).toEqual({
        estado: "aprovada",
        texto: "ok, na janela",
        resolver_vinculos: false,
      }),
    );
    await waitFor(() => expect(screen.queryByLabelText("Parecer")).not.toBeInTheDocument());
  });

  it("concluída mostra a execução e o resultado; sem janela; inexistente", async () => {
    mockBackend({
      ...identityRoutes(perms("mudancas:manage")),
      "GET v1/mudancas/m1": {
        data: mudanca({
          estado: "concluida",
          descricao: "",
          plano_reversao: "",
          inicio_planejado: undefined,
          executada_inicio: "2026-10-22T12:00:00Z",
          executada_fim: "2026-10-22T13:00:00Z",
          resultado: "tudo certo",
          proximos: [],
        }),
      },
    });
    const c = renderApp(<MudancaPage />);
    expect(await screen.findByText("tudo certo")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Plano" })).toBeInTheDocument();
    expect(screen.queryByRole("group", { name: "Estado da mudança" })).not.toBeInTheDocument();
    c.unmount();

    mockBackend({
      ...identityRoutes(perms("mudancas:manage")),
      "GET v1/mudancas/m1": {
        data: mudanca({
          estado: "em_execucao",
          executada_inicio: "2026-10-22T12:00:00Z",
          proximos: ["concluida", "falhou"],
        }),
      },
    });
    const ex = renderApp(<MudancaPage />);
    expect(await screen.findByRole("button", { name: "Registrar falha" })).toBeInTheDocument();
    ex.unmount();

    mockBackend({ ...identityRoutes(), "GET v1/mudancas/m1": fail(404, "mudança não encontrada") });
    renderApp(<MudancaPage />);
    expect(await screen.findByText("mudança não encontrada")).toBeInTheDocument();
  });
});
