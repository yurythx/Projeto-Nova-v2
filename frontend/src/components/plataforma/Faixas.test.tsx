import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ListaFaixas, NomeClicavel, Quadro, type ColunaFaixa } from "./Faixas";

const COLUNAS: ColunaFaixa[] = [
  { id: "aberta", titulo: "Abertas", tom: "info" },
  { id: "feita", titulo: "Concluídas", tom: "success" },
];

interface Card {
  id: string;
  coluna: string;
  titulo: string;
}

function limparMemoria() {
  for (let i = localStorage.length - 1; i >= 0; i -= 1) {
    const chave = localStorage.key(i);
    if (chave?.startsWith("teste-faixas")) localStorage.removeItem(chave);
  }
}

afterEach(limparMemoria);

let seq = 0;
function memoriaNova() {
  seq += 1;
  return `teste-faixas-${seq}`;
}

function faixa(titulo: string) {
  return screen.getByRole("heading", { name: new RegExp(`^${titulo}`) }).closest("section") as HTMLElement;
}

function semDizeres(raiz: HTMLElement) {
  expect(within(raiz).queryByRole("button", { name: "Nome" })).not.toBeInTheDocument();
  expect(within(raiz).queryByRole("button", { name: "Cor" })).not.toBeInTheDocument();
  expect(within(raiz).queryByText("Nome")).not.toBeInTheDocument();
  expect(within(raiz).queryByText("Cor")).not.toBeInTheDocument();
}

function desenho(
  Componente: typeof Quadro,
  opts: {
    memoria?: string;
    itens?: Card[];
    aoAbrir?: (item: Card) => void;
    aoRenomear?: (item: Card, nome: string) => void;
    aoCriar?: (colunaId: string, nome: string) => void;
    extra?: (colunaId: string) => ReactNode;
  } = {},
) {
  const memoria = opts.memoria ?? memoriaNova();
  const view = render(
    <Componente
      rotulo="Quadro de teste"
      memoria={memoria}
      colunas={COLUNAS}
      itens={opts.itens ?? [{ id: "c1", coluna: "aberta", titulo: "Pedido um" }]}
      colunaDe={(item) => item.coluna}
      idDe={(item) => item.id}
      nomeDe={(item) => item.titulo}
      aoAbrir={opts.aoAbrir}
      aoRenomear={opts.aoRenomear}
      aoCriar={opts.aoCriar}
      extra={opts.extra}
      render={(item, vista) => (
        <>
          <NomeClicavel
            nome={vista.titulo || item.titulo}
            href={`/itens/${item.id}`}
            onSalvar={vista.salvarNome}
          />
          <p>corpo {item.titulo}</p>
          <a href={`/itens/${item.id}`}>Editar</a>
        </>
      )}
    />,
  );
  return { memoria, ...view };
}

describe("Quadro da assistência", () => {
  it("mostra a coluna vazia e não escreve Nome nem Cor", () => {
    desenho(Quadro, { itens: [] });
    const quadro = screen.getByRole("list", { name: "Quadro de teste" });
    expect(within(quadro).getByRole("heading", { name: /^Abertas/ })).toBeInTheDocument();
    expect(within(quadro).getByRole("heading", { name: /^Concluídas/ })).toBeInTheDocument();
    expect(within(quadro).getAllByText("Nenhum item")).toHaveLength(2);
    expect(quadro.className).toContain("overflow-x-auto");
    expect(quadro.className).toContain("md:flex-wrap");
    expect(faixa("Abertas").className).toContain("max-md:w-[min(85vw,20rem)]");
    expect(faixa("Abertas").className).toContain("md:min-w-56");
    semDizeres(quadro);
    expect(within(quadro).getByLabelText("Cor de Abertas")).toHaveAttribute("type", "color");
  });

  it("renomeia a coluna pelo próprio título e lembra neste navegador", async () => {
    const user = userEvent.setup();
    const { memoria, unmount } = desenho(Quadro);
    await user.click(screen.getByRole("button", { name: "Abertas" }));
    const campo = screen.getByLabelText("Nome de Abertas");
    await user.clear(campo);
    await user.type(campo, "Entrada");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    expect(screen.getByRole("heading", { name: /^Entrada/ })).toBeInTheDocument();
    expect(faixa("Entrada")).toHaveAttribute("aria-labelledby");

    unmount();
    desenho(Quadro, { memoria, itens: [] });
    expect(await screen.findByRole("heading", { name: /^Entrada/ })).toBeInTheDocument();
  });

  it("não grava um nome de coluna vazio", async () => {
    const user = userEvent.setup();
    desenho(Quadro);
    await user.click(screen.getByRole("button", { name: "Abertas" }));
    await user.clear(screen.getByLabelText("Nome de Abertas"));
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    expect(screen.getByRole("heading", { name: /^Abertas/ })).toBeInTheDocument();
  });

  it("o clique no nome do card grava o título, e sem API fica só neste navegador", async () => {
    const user = userEvent.setup();
    const aoRenomear = vi.fn();
    const oficial = desenho(Quadro, { aoRenomear });
    await user.click(screen.getByRole("link", { name: "Pedido um" }));
    const campo = screen.getByLabelText("Nome do card Pedido um");
    expect(campo).toHaveValue("Pedido um");
    await user.clear(campo);
    await user.type(campo, "Pedido dois");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    expect(aoRenomear).toHaveBeenCalledWith(expect.objectContaining({ id: "c1" }), "Pedido dois");
    expect(screen.getByRole("link", { name: "Pedido um" })).toBeInTheDocument();
    oficial.unmount();

    const { memoria } = desenho(Quadro);
    await user.click(screen.getByRole("link", { name: "Pedido um" }));
    await user.clear(screen.getByLabelText("Nome do card Pedido um"));
    await user.type(screen.getByLabelText("Nome do card Pedido um"), "Apelido");
    await user.click(screen.getByRole("button", { name: "Salvar nome" }));
    expect(screen.getByRole("link", { name: "Apelido" })).toHaveAttribute("href", "/itens/c1");
    const gravado = JSON.parse(localStorage.getItem(`${memoria}:aparencia`) ?? "{}") as {
      cartoes: Record<string, { titulo?: string }>;
    };
    expect(gravado.cartoes.c1?.titulo).toBe("Apelido");
  });

  it("o corpo do card abre; o nome, a cor e o link de editar não", async () => {
    const user = userEvent.setup();
    const aoAbrir = vi.fn();
    desenho(Quadro, { aoAbrir });
    await user.click(screen.getByText("corpo Pedido um"));
    expect(aoAbrir).toHaveBeenCalledTimes(1);

    aoAbrir.mockClear();
    await user.click(screen.getByRole("link", { name: "Pedido um" }));
    fireEvent.click(screen.getByLabelText("Cor do card Pedido um"));
    await user.click(screen.getByRole("link", { name: "Editar" }));
    expect(aoAbrir).not.toHaveBeenCalled();
    expect(screen.getByLabelText("Nome do card Pedido um")).toBeInTheDocument();
  });

  it("recolhida, a coluna vira uma faixa vertical e esconde o botão do cabeçalho", async () => {
    const user = userEvent.setup();
    const { memoria, unmount } = desenho(Quadro, {
      extra: (id) => (id === "aberta" ? <button type="button">Novo item</button> : null),
    });
    const seta = screen.getByRole("button", { name: "Recolher Abertas" });
    expect(seta).toHaveAttribute("aria-expanded", "true");
    expect(seta.textContent?.trim()).toBe("");
    expect(screen.getByRole("button", { name: "Novo item" })).toBeInTheDocument();

    await user.click(seta);
    const secao = faixa("Abertas");
    expect(secao.className).toContain("w-12");
    expect(secao.className).not.toContain("w-full");
    expect(within(secao).getByRole("heading", { name: /^Abertas/ }).className).toContain(
      "[writing-mode:vertical-lr]",
    );
    expect(screen.queryByText("corpo Pedido um")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Novo item" })).not.toBeInTheDocument();
    expect(within(secao).getByLabelText("Cor de Abertas")).toBeInTheDocument();

    await user.click(within(secao).getByRole("button", { name: "Abertas" }));
    expect(screen.getByLabelText("Nome de Abertas").closest("form")).toHaveClass("absolute");

    unmount();
    desenho(Quadro, { memoria });
    expect(await screen.findByRole("button", { name: "Expandir Abertas" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
  });

  it("cria o card na coluna em que o nome foi digitado", async () => {
    const user = userEvent.setup();
    const aoCriar = vi.fn();
    desenho(Quadro, { itens: [], aoCriar });
    const feita = faixa("Concluídas");
    await user.click(within(feita).getByRole("button", { name: "Criar card" }));
    expect(aoCriar).not.toHaveBeenCalled();
    await user.type(within(feita).getByLabelText("Nome do novo card em Concluídas"), "  Cabo  ");
    await user.click(within(feita).getByRole("button", { name: "Criar card" }));
    expect(aoCriar).toHaveBeenCalledWith("feita", "Cabo");
  });

  it("a cor da coluna e a cor do card ficam neste navegador", () => {
    const { memoria } = desenho(Quadro);
    fireEvent.change(screen.getByLabelText("Cor de Abertas"), { target: { value: "#dc2626" } });
    fireEvent.change(screen.getByLabelText("Cor do card Pedido um"), { target: { value: "#059669" } });
    const gravado = JSON.parse(localStorage.getItem(`${memoria}:aparencia`) ?? "{}") as {
      colunas: Record<string, { cor?: string }>;
      cartoes: Record<string, { cor?: string }>;
    };
    expect(gravado.colunas.aberta?.cor).toBe("#dc2626");
    expect(gravado.cartoes.c1?.cor).toBe("#059669");
    expect(screen.getByLabelText("Cor do card Pedido um")).toHaveValue("#059669");
  });
});

describe("Lista da assistência", () => {
  it("recolhe na largura da tela, com o nome na horizontal", async () => {
    const user = userEvent.setup();
    desenho(ListaFaixas);
    const quadro = screen.getByRole("list", { name: "Quadro de teste" });
    semDizeres(quadro);
    await user.click(screen.getByRole("button", { name: "Recolher Abertas" }));
    const secao = faixa("Abertas");
    expect(secao.className).toBe("w-full");
    expect(within(secao).getByRole("heading", { name: /^Abertas/ }).className).not.toContain(
      "writing-mode",
    );
    expect(within(secao).queryByText("corpo Pedido um")).not.toBeInTheDocument();
  });
});
