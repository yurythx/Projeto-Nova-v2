import { describe, expect, it } from "vitest";

import {
  etapaLegivel,
  filtrarPedidos,
  paginar,
  rotuloPrazo,
  rotuloPrioridade,
  rotuloStatus,
  tomPrazo,
} from "./painel";

const pedidos = [
  {
    numero: "SEMA202680851",
    nome: "Maria Silva",
    cpf: "39053344705",
    servico_id: "poda",
    servico_nome: "Poda de árvore",
    status: "em_andamento",
    situacao_prazo: "no_prazo",
    prioridade: "normal",
  },
  {
    numero: "SEMA202680852",
    nome: "João Lima",
    cpf: "11122233344",
    servico_id: "alvara",
    servico_nome: "Alvará",
    status: "em_complementacao",
    situacao_prazo: "atrasado",
    prioridade: "alta",
  },
];

const aberto = { prazo: "todos", busca: "", servico: "", status: "", prioridade: "" };

describe("painel da mesa", () => {
  it("traduz status, prioridade, prazo e etapa", () => {
    expect(rotuloStatus("em_complementacao")).toBe("Em complementação");
    expect(rotuloStatus("outro")).toBe("outro");
    expect(rotuloPrioridade("urgente")).toBe("Urgente");
    expect(rotuloPrazo("atrasado")).toBe("Atrasados");
    expect(tomPrazo("no_prazo")).toBe("success");
    expect(tomPrazo("atencao")).toBe("warning");
    expect(tomPrazo("atrasado")).toBe("danger");
    expect(tomPrazo("finalizado")).toBe("neutral");
    expect(etapaLegivel("")).toBe("Sem etapa");
    expect(etapaLegivel("em_analise")).toBe("Em analise");
  });

  it("filtra número, nome, CPF, serviço, status, prioridade e prazo", () => {
    expect(filtrarPedidos(pedidos, { ...aberto, busca: "sema202680851" })).toHaveLength(1);
    expect(filtrarPedidos(pedidos, { ...aberto, busca: "390.533.447-05" })).toEqual([pedidos[0]]);
    expect(filtrarPedidos(pedidos, { ...aberto, busca: "39" })).toHaveLength(0);
    expect(filtrarPedidos(pedidos, { ...aberto, busca: "390" })).toEqual([pedidos[0]]);
    expect(
      filtrarPedidos(pedidos, {
        ...aberto,
        servico: "alvara",
        status: "em_complementacao",
        prioridade: "alta",
        prazo: "atrasado",
      }),
    ).toEqual([pedidos[1]]);
    expect(filtrarPedidos(pedidos, { ...aberto, prazo: "no_prazo" })).toEqual([pedidos[0]]);
  });

  it("pagina a lista e segura a página dentro do intervalo", () => {
    const pagina = paginar(pedidos, 2, 1);
    expect(pagina.itens).toEqual([pedidos[1]]);
    expect(pagina.paginas).toBe(2);
    expect(paginar(pedidos, 9, 1).pagina).toBe(2);
    expect(paginar([], 1, 20).paginas).toBe(1);
  });
});
