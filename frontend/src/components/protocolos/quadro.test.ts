import { describe, expect, it } from "vitest";

import {
  COLUNAS_PROTOCOLO,
  contarPor,
  csvDosPedidos,
  diaDoPrazo,
  molduraPrazo,
  tomPrioridade,
  tomStatus,
  type PedidoResumo,
} from "./quadro";

const base: PedidoResumo = {
  id: "p1",
  numero: "PREF202690001",
  nome: "Ana; Paula",
  servico_id: "s1",
  servico_nome: 'Alvará "rápido"',
  unidade_nome: "PREFEITURA",
  status: "em_analise",
  situacao_prazo: "atrasado",
  etapa_id: "triagem",
  prioridade: "alta",
  prazo_em: "2026-11-20T15:00:00Z",
  created_at: "2026-11-10T12:00:00Z",
};

describe("quadro do protocolo", () => {
  it("tem uma coluna por status, com pendente e rejeitado fora até haver pedido", () => {
    expect(COLUNAS_PROTOCOLO.map((c) => c.id)).toEqual([
      "novo",
      "em_analise",
      "em_andamento",
      "pendente",
      "em_complementacao",
      "pending_approval",
      "aprovado",
      "concluido",
      "rejeitado",
      "cancelado",
    ]);
    expect(COLUNAS_PROTOCOLO.filter((c) => c.ativa === false).map((c) => c.id)).toEqual([
      "pendente",
      "rejeitado",
    ]);
  });

  it("dá o tom da prioridade e do status, neutro quando não conhece", () => {
    expect(tomPrioridade("urgente")).toBe("danger");
    expect(tomPrioridade("normal")).toBe("info");
    expect(tomPrioridade("outra")).toBe("neutral");
    expect(tomStatus("concluido")).toBe("success");
    expect(tomStatus("em_complementacao")).toBe("danger");
    expect(tomStatus("desconhecido")).toBe("neutral");
  });

  it("pinta a moldura só no atrasado e na atenção", () => {
    expect(molduraPrazo({ situacao_prazo: "atrasado" })).toContain("border-l-danger");
    expect(molduraPrazo({ situacao_prazo: "atencao" })).toContain("border-l-warning");
    expect(molduraPrazo({ situacao_prazo: "no_prazo" })).toBeUndefined();
    expect(molduraPrazo({})).toBeUndefined();
  });

  it("mostra o dia do prazo em Cuiabá e nada sem data válida", () => {
    expect(diaDoPrazo("2026-11-21T02:00:00Z")).toBe("20/11/2026");
    expect(diaDoPrazo(undefined)).toBe("");
    expect(diaDoPrazo("não é data")).toBe("");
  });

  it("conta do maior para o menor, com empate em ordem alfabética", () => {
    expect(contarPor(["b", "a", "b", "", "c", "a"], (x) => x)).toEqual([
      { nome: "a", total: 2 },
      { nome: "b", total: 2 },
      { nome: "—", total: 1 },
      { nome: "c", total: 1 },
    ]);
  });

  it("exporta o CSV com BOM, ponto e vírgula e aspas onde precisa", () => {
    const csv = csvDosPedidos([
      base,
      {
        ...base,
        id: "p2",
        etapa_id: "",
        situacao_prazo: undefined,
        prazo_em: undefined,
        unidade_nome: undefined,
      },
    ]);
    const [cabecalho, primeira, segunda] = csv.slice(1).split("\n");
    expect(csv.charCodeAt(0)).toBe(0xfeff);
    expect(cabecalho).toBe(
      "Numero;Servico;Requerente;Unidade;Status;Etapa;Prioridade;Prazo;SituacaoPrazo;AbertoEm",
    );
    expect(primeira).toBe(
      'PREF202690001;"Alvará ""rápido""";"Ana; Paula";PREFEITURA;Em análise;Triagem;Alta;20/11/2026;Atrasados;10/11/2026',
    );
    expect(segunda).toBe(
      'PREF202690001;"Alvará ""rápido""";"Ana; Paula";;Em análise;;Alta;;;10/11/2026',
    );
  });
});
