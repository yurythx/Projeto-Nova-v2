import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  agrupar,
  csv,
  dias,
  diasDeAtendimento,
  noPeriodo,
  resumo,
  type PedidoDoRelatorio,
} from "@/components/protocolos/relatorios";
import { fail, identityRoutes, ME, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import FichaPage from "./[id]/page";
import ProtocolosPage from "./page";
import RelatoriosPage from "./relatorios/page";

const BASE = {
  nome: "Maria Cidadã",
  servico_id: "s1",
  servico_nome: "Alvará",
  unidade_id: "u1",
  unidade_nome: "PREFEITURA",
  etapa_id: "",
  prioridade: "normal",
  situacao_prazo: "no_prazo",
  created_at: "2026-11-20T12:00:00Z",
  updated_at: "2026-11-20T12:00:00Z",
  decisao: "",
  justificativa: "",
  complemento: "",
  responsavel_id: ME.id,
  responsavel_nome: "Maria Servidora",
};

describe("Conversa com o cidadão na mesa", () => {
  beforeEach(() => resetNavigation({}, "/protocolos"));

  it("mostra o selo e filtra as mensagens novas", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"] }),
      "GET v1/protocolos/pedidos": {
        data: [
          {
            ...BASE,
            id: "m1",
            numero: "PREF202600001",
            status: "em_andamento",
            mensagem_nova: true,
          },
          { ...BASE, id: "m2", numero: "PREF202600002", status: "em_andamento" },
        ],
      },
    });
    renderApp(<ProtocolosPage />);
    expect(await screen.findByText("💬 Mensagem nova")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Mensagens novas/ }));
    expect(screen.getAllByText(/PREF202600001/).length).toBeGreaterThan(0);
    expect(screen.queryByText(/PREF202600002/)).not.toBeInTheDocument();
  });
});

function ficha(pedido: Record<string, unknown>, historico: unknown[] = []) {
  return {
    pedido: { ...BASE, id: "m1", numero: "PREF202600001", campos: [], respostas: {}, ...pedido },
    historico,
    etapas: [],
  };
}

describe("Conversa com o cidadão na ficha", () => {
  beforeEach(() => resetNavigation({ id: "m1" }, "/protocolos/m1"));

  it("mostra a conversa, marca como lida e responde", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"] }),
      "GET v1/protocolos/pedidos/m1": {
        data: ficha({ status: "em_andamento", mensagem_nova: true }, [
          {
            id: "h1",
            acao: "mensagem_cidadao",
            descricao: "Quando fica pronto?",
            status: "em_andamento",
            created_at: "2026-11-21T12:00:00Z",
          },
          {
            id: "h2",
            acao: "mensagem_mesa",
            descricao: "Na semana que vem.",
            status: "em_andamento",
            created_at: "2026-11-22T12:00:00Z",
          },
        ]),
      },
      "GET v1/iam/org-tree": { data: [] },
      "POST v1/protocolos/pedidos/*": { data: {} },
    });
    renderApp(<FichaPage />);
    expect(await screen.findByText("O cidadão mandou mensagem nova.")).toBeInTheDocument();
    expect(screen.getAllByText("Quando fica pronto?").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Mensagem do Cidadão").length).toBeGreaterThan(0);

    await user.click(screen.getByRole("button", { name: "Marcar como lida" }));
    expect(api.to("POST v1/protocolos/pedidos/m1/mensagens/lidas")).toHaveLength(1);

    const enviar = screen.getByRole("button", { name: "Enviar mensagem" });
    expect(enviar).toBeDisabled();
    await user.type(screen.getByLabelText("Responder ao cidadão"), "  Já está pronto.  ");
    await user.click(enviar);
    expect(api.to("POST v1/protocolos/pedidos/m1/mensagem")[0]?.body).toEqual({
      mensagem: "Já está pronto.",
    });
    expect(screen.getByLabelText("Responder ao cidadão")).toHaveValue("");
  });

  it("sem conversa mostra a orientação; a falha mantém o texto", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:tratar"] }),
      "GET v1/protocolos/pedidos/m1": { data: ficha({ status: "em_andamento" }) },
      "GET v1/iam/org-tree": { data: [] },
      "POST v1/protocolos/pedidos/m1/mensagem": fail(409, "pedido encerrado"),
    });
    renderApp(<FichaPage />);
    expect(await screen.findByText(/Nenhuma mensagem/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Marcar como lida" })).not.toBeInTheDocument();
    await user.type(screen.getByLabelText("Responder ao cidadão"), "oi");
    await user.click(screen.getByRole("button", { name: "Enviar mensagem" }));
    expect(await screen.findByText("pedido encerrado")).toBeInTheDocument();
    expect(screen.getByLabelText("Responder ao cidadão")).toHaveValue("oi");
  });
});

const RELATORIO: PedidoDoRelatorio[] = [
  { ...BASE, id: "r1", numero: "N1", status: "concluido", updated_at: "2026-11-24T12:00:00Z" },
  {
    ...BASE,
    id: "r2",
    numero: "N2",
    status: "em_andamento",
    situacao_prazo: "atrasado",
    prazo_em: "2026-11-25T12:00:00Z",
  },
  {
    ...BASE,
    id: "r3",
    numero: "N3",
    status: "cancelado",
    servico_id: "s2",
    servico_nome: "Poda",
    unidade_id: "u2",
    unidade_nome: "OBRAS",
  },
  { ...BASE, id: "r4", numero: "N4", status: "em_andamento", created_at: "2025-01-10T12:00:00Z" },
];

describe("Relatórios", () => {
  beforeEach(() => resetNavigation({}, "/protocolos/relatorios"));
  afterEach(() => vi.restoreAllMocks());

  it("conta por serviço e unidade, lista os atrasados e baixa o CSV", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:gerir"] }),
      "GET v1/protocolos/pedidos": { data: RELATORIO },
    });
    const criar = vi.fn(() => "blob:x");
    Object.assign(URL, { createObjectURL: criar, revokeObjectURL: vi.fn() });
    const clicar = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

    renderApp(<RelatoriosPage />);
    const porServico = await screen.findByRole("table", { name: "Por serviço" });
    const alvara = within(porServico).getByRole("row", { name: /Alvará/ });
    expect(alvara).toHaveTextContent("2");
    expect(alvara).toHaveTextContent("4,0 d");
    expect(within(porServico).getByRole("row", { name: /Poda/ })).toHaveTextContent("—");
    expect(screen.getByRole("table", { name: "Por unidade" })).toHaveTextContent("OBRAS");
    expect(screen.getByRole("heading", { name: "Atrasados (1)" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "N2" })).toHaveAttribute("href", "/protocolos/r2");

    for (const botao of screen.getAllByRole("button", { name: "Baixar CSV" })) {
      await user.click(botao);
    }
    expect(criar).toHaveBeenCalledTimes(3);
    expect(clicar).toHaveBeenCalledTimes(3);

    // Período sem pedido.
    await user.clear(screen.getByLabelText("até"));
    await user.clear(screen.getByLabelText("Abertos de"));
    await user.type(screen.getByLabelText("Abertos de"), "2030-01-01");
    expect(await screen.findByText("Nenhum pedido aberto nesse período.")).toBeInTheDocument();
  });

  it("sem atraso, avisa e desliga o CSV", async () => {
    mockBackend({
      ...identityRoutes({ permissions: ["protocolos:gerir"] }),
      "GET v1/protocolos/pedidos": { data: [RELATORIO[0]] },
    });
    renderApp(<RelatoriosPage />);
    expect(await screen.findByText(/Nenhum pedido atrasado/)).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Baixar CSV" })[2]).toBeDisabled();
  });
});

describe("contas dos relatórios", () => {
  it("resume, agrupa e escreve o CSV seguro", () => {
    const total = resumo(RELATORIO);
    expect(total).toMatchObject({
      total: 4,
      abertos: 2,
      atendidos: 1,
      cancelados: 1,
      atrasados: 1,
    });
    expect(total.tempoMedio).toBeCloseTo(4);
    expect(resumo([]).tempoMedio).toBeNull();
    expect(noPeriodo(RELATORIO, "2026-11-20", "2026-11-20")).toHaveLength(3);
    expect(noPeriodo(RELATORIO, "", "")).toHaveLength(4);
    expect(agrupar(RELATORIO, "unidade").map((l) => l.nome)).toEqual(["PREFEITURA", "OBRAS"]);
    expect(
      agrupar([{ ...RELATORIO[0]!, servico_nome: "", servico_id: "" }], "servico")[0]?.nome,
    ).toBe("Sem identificação");
    expect(diasDeAtendimento({ ...RELATORIO[0]!, updated_at: undefined })).toBeNull();
    expect(diasDeAtendimento({ ...RELATORIO[0]!, updated_at: "2026-11-01T00:00:00Z" })).toBe(0);
    expect(dias(null)).toBe("—");
    expect(dias(1.25)).toMatch(/^1,[23] d$/);
    expect(csv(["a", "b"], [["=SOMA(1)", 'diz "oi"; tchau']])).toBe(
      '﻿a;b\r\n\'=SOMA(1);"diz ""oi""; tchau"',
    );
  });
});
