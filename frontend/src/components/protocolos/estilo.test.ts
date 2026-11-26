import { describe, expect, it } from "vitest";

import {
  bordaDaPrioridade,
  bordaDoPrazo,
  colunaDoKanban,
  COLUNAS_KANBAN,
  corDaEstatistica,
  dataCurta,
  diasUteis,
  gradienteDaPrioridade,
  gradienteDoStatus,
  nomeDaEtapa,
  rotuloDaPrioridade,
  rotuloDoStatus,
  seloDaPrioridade,
  seloDoPrazo,
  seloDoStatus,
  seloPequenoDaPrioridade,
  textoDoPrazo,
} from "./estilo";

describe("estilo do protocolo", () => {
  it("rotula status e prioridade como o original e devolve o valor que não conhece", () => {
    expect(rotuloDoStatus("pending_approval")).toBe("Pendente de Aprovação");
    expect(rotuloDoStatus("em_andamento")).toBe("Em Andamento");
    expect(rotuloDoStatus("outro")).toBe("outro");
    expect(rotuloDaPrioridade("urgente")).toBe("Urgente");
    expect(rotuloDaPrioridade("x")).toBe("x");
  });

  it("pinta cada status e prioridade com a cor do original", () => {
    expect(seloDoStatus("novo")).toBe("bg-blue-500");
    expect(seloDoStatus("em_complementacao")).toContain("bg-violet-500");
    expect(seloDoStatus("x")).toBe("bg-gray-600");
    expect(corDaEstatistica("em_analise")).toBe("bg-cyan-500");
    expect(corDaEstatistica("x")).toBe("bg-gray-600");
    expect(seloDaPrioridade("alta")).toBe("bg-orange-500");
    expect(seloDaPrioridade("x")).toBe("bg-blue-500");
    expect(gradienteDoStatus("novo")).toContain("from-[#4facfe]");
    expect(gradienteDoStatus("em_analise")).toBe("bg-cyan-500");
    expect(gradienteDaPrioridade("urgente")).toContain("from-[#ff9a9e]");
    expect(gradienteDaPrioridade("x")).toBe("bg-blue-500");
  });

  it("marca o prazo na borda e no selo, com no prazo como padrão", () => {
    expect(bordaDoPrazo("atrasado")).toContain("border-l-red-600");
    expect(bordaDoPrazo(undefined)).toContain("border-l-emerald-600");
    expect(bordaDoPrazo("x")).toContain("border-l-emerald-600");
    expect(seloDoPrazo("atencao")).toContain("text-amber-600");
    expect(seloDoPrazo(undefined)).toContain("text-emerald-600");
    expect(seloDoPrazo("x")).toContain("text-emerald-600");
  });

  it("conta dias úteis sem sábado e domingo", () => {
    // 2026-11-20 é sexta; até a sexta seguinte são 5 dias úteis.
    expect(diasUteis(new Date(2026, 10, 20, 9), new Date(2026, 10, 27, 9))).toBe(5);
    expect(diasUteis(new Date(2026, 10, 27), new Date(2026, 10, 20))).toBe(0);
  });

  it("escreve o prazo com os dias corridos e o total", () => {
    const agora = new Date(2026, 10, 25, 12);
    const base = { created_at: new Date(2026, 10, 20, 9).toISOString() };
    expect(
      textoDoPrazo(
        { ...base, situacao_prazo: "atencao", prazo_em: new Date(2026, 10, 27, 9).toISOString() },
        agora,
      ),
    ).toBe("Atenção (3/5d)");
    expect(textoDoPrazo({ ...base, situacao_prazo: "finalizado" }, agora)).toBe("Finalizado");
    expect(textoDoPrazo({ ...base }, agora)).toBe("No Prazo");
    expect(textoDoPrazo({ ...base, situacao_prazo: "novo_tipo", prazo_em: "x" }, agora)).toBe(
      "novo_tipo",
    );
  });

  it("agrupa o kanban como o original e cobre todos os status", () => {
    expect(colunaDoKanban("aprovado")).toBe("concluido");
    expect(colunaDoKanban("cancelado")).toBe("rejeitado");
    expect(colunaDoKanban("em_analise")).toBe("em_analise");
    const colunas = new Set(COLUNAS_KANBAN.map((c) => c.id));
    for (const status of [
      "novo",
      "em_andamento",
      "pendente",
      "em_analise",
      "em_complementacao",
      "pending_approval",
      "aprovado",
      "rejeitado",
      "concluido",
      "cancelado",
    ]) {
      expect(colunas.has(colunaDoKanban(status))).toBe(true);
    }
    expect(bordaDaPrioridade("urgente")).toBe("border-l-red-500");
    expect(bordaDaPrioridade("normal")).toBe("border-l-blue-500");
    expect(seloPequenoDaPrioridade("alta")).toContain("bg-red-100");
    expect(seloPequenoDaPrioridade("baixa")).toContain("bg-slate-200");
  });

  it("formata data e etapa", () => {
    expect(dataCurta("2026-11-21T02:00:00Z")).toBe("20/11/2026");
    expect(dataCurta(undefined)).toBe("Data não disponível");
    expect(dataCurta("x")).toBe("Data inválida");
    expect(nomeDaEtapa("analise_tecnica")).toBe("Analise tecnica");
    expect(nomeDaEtapa("")).toBe("Etapa não definida");
    expect(nomeDaEtapa(undefined)).toBe("Etapa não definida");
  });
});
