import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PaginaDeStatus } from "@/components/ocorrencias/PaginaDeStatus";
import {
  fmtHorario,
  minutosSemNoticia,
  type Ocorrencia,
  type Situacao,
} from "@/lib/ocorrencias/ocorrencias";

const aberta = {
  id: "oc-1",
  titulo: "Protocolo eletrônico fora do ar",
  impacto: "fora" as const,
  estado: "identificado" as const,
  servicos: ["Protocolo", "SEI"],
  iniciada_em: "2026-10-27T12:00:00Z",
  atualizacoes: [
    {
      estado: "identificado" as const,
      texto: "O certificado venceu; renovando.",
      em: "2026-10-27T13:00:00Z",
    },
    { estado: "investigando" as const, texto: "Estamos investigando.", em: "2026-10-27T12:00:00Z" },
  ],
};

describe("PaginaDeStatus (ADR 066)", () => {
  it("sem ocorrência aberta, diz que está tudo funcionando", () => {
    render(<PaginaDeStatus situacao={{ geral: "operacional", abertas: [], recentes: [] }} />);
    expect(screen.getByRole("status")).toHaveTextContent("Todos os serviços funcionando");
    expect(screen.queryByRole("heading", { name: "Em andamento" })).not.toBeInTheDocument();
    expect(
      screen.getByText("Nenhuma ocorrência resolvida nos últimos 7 dias."),
    ).toBeInTheDocument();
  });

  it("mostra a aberta com a notícia mais nova primeiro, no fuso de Mato Grosso", () => {
    const resolvida = {
      ...aberta,
      id: "oc-2",
      titulo: "Rede da Saúde",
      estado: "resolvido" as const,
      resolvida_em: "2026-10-26T15:30:00Z",
    };
    const s: Situacao = { geral: "fora", abertas: [aberta], recentes: [resolvida] };
    render(<PaginaDeStatus situacao={s} />);

    expect(screen.getByRole("status")).toHaveTextContent("Serviços fora do ar");
    const card = screen.getByRole("article", { name: "Protocolo eletrônico fora do ar" });
    expect(within(card).getByText("Fora do ar")).toBeInTheDocument();
    expect(
      within(card).getByText(/Afeta: Protocolo, SEI · desde 27\/10\/2026, 08:00/),
    ).toBeInTheDocument();
    const itens = within(card).getAllByRole("listitem");
    expect(itens[0]).toHaveTextContent("Causa identificada · 27/10/2026, 09:00");
    expect(itens[0]).toHaveTextContent("O certificado venceu; renovando.");

    const recente = screen.getByRole("article", { name: "Rede da Saúde" });
    expect(within(recente).getByText("Resolvido")).toBeInTheDocument();
    expect(within(recente).getByText(/resolvido em 26\/10\/2026, 11:30/)).toBeInTheDocument();
  });
});

describe("regras da tela", () => {
  it("formata no fuso da instituição, não no do servidor", () => {
    expect(fmtHorario("2026-10-27T03:00:00Z")).toBe("26/10/2026, 23:00");
  });

  it("conta os minutos desde a última notícia", () => {
    const o = { iniciada_em: "2026-10-27T12:00:00Z", atualizacoes: [] } as unknown as Ocorrencia;
    expect(minutosSemNoticia(o, Date.parse("2026-10-27T13:30:00Z"))).toBe(90);
    const comNoticia = {
      ...o,
      atualizacoes: [{ created_at: "2026-10-27T13:20:00Z" }],
    } as unknown as Ocorrencia;
    expect(minutosSemNoticia(comNoticia, Date.parse("2026-10-27T13:30:00Z"))).toBe(10);
    expect(minutosSemNoticia(comNoticia, Date.parse("2026-10-27T13:00:00Z"))).toBe(0);
  });
});
