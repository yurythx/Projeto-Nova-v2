import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { PrazoBadge } from "@/components/atendimento/PrazoBadge";
import { PrazoCard } from "@/components/atendimento/PrazoCard";
import { PoliticaForm, sugestao } from "@/components/atendimento/PoliticaForm";
import {
  duracao,
  relogioEmFoco,
  type Prazo,
  type Relogio,
  type Solicitacao,
} from "@/lib/atendimento/atendimento";
import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtendimentoPrazosPage from "./prazos/page";
import SolicitacaoPage from "./solicitacoes/[id]/page";

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "Tecnologia", sigla: "TI", departamentos: [] }],
  },
];

function relogio(extra: Partial<Relogio> = {}): Relogio {
  return {
    meta_min: 120,
    usado_min: 30,
    percentual: 25,
    estado: "correndo",
    situacao: "no_prazo",
    ...extra,
  };
}

function prazo(resposta: Partial<Relogio> = {}, solucao: Partial<Relogio> = {}): Prazo {
  return {
    prioridade: "media",
    resposta: relogio(resposta),
    solucao: relogio({ meta_min: 960, ...solucao }),
  };
}

beforeEach(() => resetNavigation({ id: "s1" }, "/atendimento/prazos"));

describe("Atendimento — prazos da unidade", () => {
  it("sem política: mostra a sugestão e salva o que a gestão ajustou", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/atendimento/politica": { data: null },
      "GET v1/atendimento/feriados": { data: [] },
      "PUT v1/atendimento/politica": { data: { id: "p1" } },
    });
    renderApp(<AtendimentoPrazosPage />);

    expect(await screen.findByText("Sem política")).toBeInTheDocument();
    expect(screen.getByText(/Os valores abaixo são uma sugestão/)).toBeInTheDocument();
    expect(api.to("GET v1/atendimento/politica")[0]!.query.get("unidade_id")).toBeNull();
    expect(screen.getByText("Nenhum feriado cadastrado nesta unidade")).toBeInTheDocument();

    // Sábado de manhã entra; a meta baixa sai (em branco).
    fireEvent.change(screen.getByLabelText("Sábado, turno 1, início"), {
      target: { value: "08:00" },
    });
    fireEvent.change(screen.getByLabelText("Sábado, turno 1, fim"), { target: { value: "12:00" } });
    await user.selectOptions(screen.getByLabelText("Impacto alto, urgência alta"), "alta");
    await user.clear(screen.getByLabelText("Baixa, resposta em horas"));
    await user.clear(screen.getByLabelText("Crítica, solução em horas"));
    await user.type(screen.getByLabelText("Crítica, solução em horas"), "3.5");
    await user.click(screen.getByRole("button", { name: "Salvar política" }));

    await waitFor(() => expect(api.to("PUT v1/atendimento/politica")).toHaveLength(1));
    const corpo = api.to("PUT v1/atendimento/politica")[0]!.body as {
      unidade_id: null;
      fuso: string;
      turnos: unknown[];
      matriz: { impacto: number; urgencia: number; prioridade: string }[];
      metas: { prioridade: string; resposta_min: number; solucao_min: number }[];
    };
    expect(corpo.unidade_id).toBeNull();
    expect(corpo.fuso).toBe("America/Sao_Paulo");
    expect(corpo.turnos).toHaveLength(11);
    expect(corpo.turnos).toContainEqual({ dia: 6, inicio: "08:00", fim: "12:00" });
    expect(corpo.matriz).toHaveLength(9);
    expect(corpo.matriz).toContainEqual({ impacto: 3, urgencia: 3, prioridade: "alta" });
    expect(corpo.metas.map((m) => m.prioridade)).toEqual(["critica", "alta", "media"]);
    expect(corpo.metas[0]).toEqual({ prioridade: "critica", resposta_min: 30, solucao_min: 210 });
    expect(await screen.findByText("Política salva")).toBeInTheDocument();
  });

  it("herdada, com fuso fora da lista, feriados e troca de unidade", async () => {
    const user = userEvent.setup();
    const base = sugestao();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/atendimento/politica": {
        data: { ...base, fuso: "Europe/Lisbon", id: "p0", herdada: true },
      },
      "GET v1/atendimento/feriados": {
        data: [{ id: "f1", data: "2026-11-20", nome: "Consciência Negra", tipo: "nacional" }],
      },
      "POST v1/atendimento/feriados": { status: 201, data: { id: "f2" } },
      "DELETE v1/atendimento/feriados/f1": { status: 204 },
    });
    renderApp(<AtendimentoPrazosPage />);

    expect(await screen.findByText("Herdada")).toBeInTheDocument();
    expect(screen.getByText(/usa a política de uma unidade acima/)).toBeInTheDocument();
    expect(screen.getByLabelText("Fuso horário")).toHaveValue("Europe/Lisbon");
    expect(within(screen.getByRole("list")).getByText("Nacional")).toBeInTheDocument();
    expect(screen.getByText(/Os da unidade acima também valem aqui/)).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Data *"), { target: { value: "2026-12-08" } });
    await user.type(screen.getByLabelText("Nome *"), "Padroeira");
    await user.click(screen.getByRole("button", { name: "Adicionar" }));
    await waitFor(() => expect(api.to("POST v1/atendimento/feriados")).toHaveLength(1));
    expect(api.to("POST v1/atendimento/feriados")[0]!.body).toEqual({
      unidade_id: null,
      data: "2026-12-08",
      nome: "Padroeira",
      tipo: "municipal",
    });

    await user.click(screen.getByRole("button", { name: "Excluir Consciência Negra" }));
    await user.click(
      within(await screen.findByRole("dialog")).getByRole("button", { name: "Excluir" }),
    );
    await waitFor(() => expect(api.to("DELETE v1/atendimento/feriados/f1")).toHaveLength(1));

    await user.selectOptions(await screen.findByLabelText("Unidade"), "u1");
    await waitFor(() =>
      expect(api.to("GET v1/atendimento/politica").at(-1)!.query.get("unidade_id")).toBe("u1"),
    );
  });

  it("gestão de unidade escolhe a unidade; quem não gere não entra; falha ao salvar", async () => {
    const user = userEvent.setup();
    const scoped = {
      permissions: ["atendimento:manage"],
      scopes: [
        {
          perfil: "p",
          origem: "manual" as const,
          permissions: ["atendimento:manage"],
          unidade_id: "u1",
        },
      ],
    };
    const api = mockBackend({
      ...identityRoutes(scoped),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/atendimento/politica": { data: { ...sugestao(), id: "p1", herdada: false } },
      "GET v1/atendimento/feriados": { data: [] },
      "PUT v1/atendimento/politica": fail(422, "turnos de segunda se sobrepõem"),
    });
    const escopo = renderApp(<AtendimentoPrazosPage />);
    expect(await screen.findByText("Escolha a unidade")).toBeInTheDocument();
    expect(api.to("GET v1/atendimento/politica")).toHaveLength(0);
    await user.selectOptions(await screen.findByLabelText("Unidade *"), "u1");
    expect(await screen.findByRole("button", { name: "Salvar política" })).toBeInTheDocument();
    expect(screen.queryByText("Herdada")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Salvar política" }));
    expect(await screen.findByText("turnos de segunda se sobrepõem")).toBeInTheDocument();
    expect(api.to("PUT v1/atendimento/politica")[0]!.body).toMatchObject({ unidade_id: "u1" });
    escopo.unmount();

    mockBackend(identityRoutes({ permissions: ["atendimento:atender"], scopes: [] }));
    renderApp(<AtendimentoPrazosPage />);
    expect(await screen.findByText("Seu perfil não gere o atendimento")).toBeInTheDocument();
  });

  it("falha de leitura e nova tentativa", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/atendimento/politica": fail(500, "banco fora"),
      "GET v1/atendimento/feriados": fail(500, "banco fora"),
    });
    renderApp(<AtendimentoPrazosPage />);
    const botoes = await screen.findAllByRole("button", { name: "Tentar novamente" });
    expect(botoes).toHaveLength(2);
    for (const b of botoes) await user.click(b);
    await waitFor(() => expect(api.to("GET v1/atendimento/politica").length).toBeGreaterThan(1));
    await waitFor(() => expect(api.to("GET v1/atendimento/feriados").length).toBeGreaterThan(1));
  });
});

describe("Atendimento — prazo na tela", () => {
  it("o selo mostra o relógio em foco, a pausa e o vencimento", () => {
    const { rerender } = render(<PrazoBadge />);
    expect(screen.getByText("Sem prazo")).toBeInTheDocument();
    rerender(
      <PrazoBadge prazo={prazo({ situacao: "atencao", vence_em: "2026-10-21T15:00:00Z" })} />,
    );
    expect(screen.getByText("Resposta: Atenção")).toBeInTheDocument();
    expect(screen.getByText(/^vence /)).toBeInTheDocument();
    rerender(
      <PrazoBadge
        prazo={prazo({ estado: "parado", situacao: "cumprido" }, { estado: "pausado" })}
      />,
    );
    expect(screen.getByText("Solução: No prazo (pausado)")).toBeInTheDocument();
  });

  it("o cartão do detalhe mostra os dois relógios e reclassifica", async () => {
    const user = userEvent.setup();
    resetNavigation({ id: "s1" });
    const sol = {
      id: "s1",
      numero: 7,
      tipo: "incidente",
      estado: "em_atendimento",
      titulo: "Sem rede",
      descricao: "",
      oferta_id: "o1",
      oferta_nome: "Rede",
      unidade_id: "u1",
      solicitante_id: "x",
      solicitante_nome: "João",
      responsavel_id: "y",
      responsavel_nome: "Ana",
      created_at: "2026-10-21T12:00:00Z",
      updated_at: "2026-10-21T12:00:00Z",
      impacto: 2,
      urgencia: 2,
      prioridade: "media",
      prazo: prazo(
        { estado: "parado", situacao: "descumprido", usado_min: 150, percentual: 125 },
        { situacao: "critico", usado_min: 870, percentual: 90, vence_em: "2026-10-22T13:00:00Z" },
      ),
    };
    const api = mockBackend({
      ...identityRoutes({
        permissions: ["atendimento:atender"],
        scopes: [{ perfil: "p", origem: "manual", permissions: ["atendimento:atender"] }],
      }),
      "GET v1/atendimento/solicitacoes/s1": {
        data: {
          solicitacao: sol,
          papeis: ["tecnico"],
          proximos: [],
          passagens: [],
          comentarios: [],
        },
      },
      "GET v1/atendimento/solicitacoes/s1/responsaveis": { data: [] },
      "PUT v1/atendimento/solicitacoes/s1/classificacao": { data: {} },
    });
    renderApp(<SolicitacaoPage />);

    expect(await screen.findByText("Prioridade média")).toBeInTheDocument();
    expect(screen.getByRole("meter", { name: "Prazo de resposta" })).toHaveAttribute(
      "aria-valuenow",
      "100",
    );
    expect(screen.getByRole("meter", { name: "Prazo de solução" })).toHaveAttribute(
      "aria-valuetext",
      "90% da meta",
    );
    expect(screen.getByText("Descumprido")).toBeInTheDocument();
    expect(screen.getByText(/2h30 de 2h úteis · parado/)).toBeInTheDocument();
    expect(screen.getByText(/14h30 de 16h úteis · correndo · vence/)).toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("Impacto"), "3");
    await user.selectOptions(screen.getByLabelText("Urgência"), "3");
    await user.click(screen.getByRole("button", { name: "Reclassificar" }));
    await waitFor(() =>
      expect(api.to("PUT v1/atendimento/solicitacoes/s1/classificacao")).toHaveLength(1),
    );
    expect(api.to("PUT v1/atendimento/solicitacoes/s1/classificacao")[0]!.body).toEqual({
      impacto: 3,
      urgencia: 3,
    });
  });

  it("política parcial: meta e casa da matriz ausentes ficam em branco e média", async () => {
    const user = userEvent.setup();
    const salvar = vi.fn();
    const base = sugestao();
    render(
      <PoliticaForm
        inicial={{ ...base, matriz: base.matriz.slice(1), metas: base.metas.slice(0, 3) }}
        pending={false}
        onSalvar={salvar}
      />,
    );
    expect(screen.getByLabelText("Baixa, resposta em horas")).toHaveValue(null);
    expect(screen.getByLabelText("Impacto baixo, urgência baixa")).toHaveValue("media");
    await user.click(screen.getByRole("button", { name: "Salvar política" }));
    expect(salvar.mock.calls[0]![0].metas).toHaveLength(3);
  });

  it("relógio em atenção pinta a barra de alerta", () => {
    const s = {
      impacto: 2,
      urgencia: 2,
      prioridade: "alta",
      prazo: prazo({ situacao: "atencao", percentual: 80 }),
    } as Solicitacao;
    render(<PrazoCard solicitacao={s} editavel={false} pending={false} onClassificar={() => {}} />);
    expect(screen.getByRole("meter", { name: "Prazo de resposta" }).firstElementChild).toHaveClass(
      "bg-warning",
    );
    expect(screen.getByText("Prioridade alta")).toBeInTheDocument();
  });

  it("sem política e sem edição", () => {
    const s = { impacto: 2, urgencia: 2 } as Solicitacao;
    render(<PrazoCard solicitacao={s} editavel={false} pending={false} onClassificar={() => {}} />);
    expect(screen.getByText("Sem prioridade")).toBeInTheDocument();
    expect(screen.getByText(/ainda não tem política de prazo/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reclassificar" })).not.toBeInTheDocument();
  });

  it("vocabulário do relógio", () => {
    expect(duracao(45)).toBe("45 min");
    expect(duracao(120)).toBe("2h");
    expect(duracao(90)).toBe("1h30");
    expect(relogioEmFoco(prazo()).nome).toBe("Resposta");
    expect(relogioEmFoco(prazo({ estado: "parado" })).nome).toBe("Solução");
  });
});
