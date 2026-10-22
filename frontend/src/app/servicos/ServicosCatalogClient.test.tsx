import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import type { CatalogService } from "@/lib/plataforma/types";

import { ServicosCatalogClient } from "./ServicosCatalogClient";

function servico(extra: Partial<CatalogService>): CatalogService {
  return {
    id: "s",
    slug: "s",
    title: "",
    summary: "",
    description: "",
    category: "",
    audience: "",
    requirements: [],
    steps: [],
    channels: [],
    sla: "",
    cost: "",
    icon: "",
    ...extra,
  } as CatalogService;
}

const SERVICOS = [
  servico({
    id: "1",
    slug: "iptu",
    title: "Segunda via do IPTU",
    summary: "boleto",
    category: "Tributos",
  }),
  servico({
    id: "2",
    slug: "alvara",
    title: "Alvará",
    summary: "licença de funcionamento",
    category: "Tributos",
  }),
  servico({ id: "3", slug: "matricula", title: "Matrícula escolar", category: "" }),
];

const titulos = () => screen.getAllByRole("heading", { level: 2 }).map((h) => h.textContent);

describe("Carta de serviços (cliente)", () => {
  it("conta por categoria, filtra pela aba e pela busca, e limpa", async () => {
    const user = userEvent.setup();
    render(<ServicosCatalogClient initialServices={SERVICOS} />);
    const abas = screen.getAllByRole("tab").map((t) => t.textContent);
    expect(abas).toEqual(["Todas3", "Geral1", "Tributos2"]);
    expect(screen.getByRole("link", { name: /Segunda via do IPTU/ })).toHaveAttribute(
      "href",
      "/servicos/iptu",
    );

    await user.click(screen.getByRole("tab", { name: /Geral/ }));
    expect(screen.getByRole("tab", { name: /Geral/ })).toHaveAttribute("aria-selected", "true");
    expect(titulos()).toEqual(["Matrícula escolar"]);

    await user.click(screen.getByRole("tab", { name: /Tributos/ }));
    const busca = screen.getByLabelText("Buscar serviços");
    for (const [termo, esperado] of [
      ["licença", ["Alvará"]],
      ["IPTU", ["Segunda via do IPTU"]],
      ["tribut", ["Segunda via do IPTU", "Alvará"]],
    ] as const) {
      await user.clear(busca);
      await user.type(busca, termo);
      expect(titulos()).toEqual(esperado);
    }

    await user.clear(busca);
    await user.type(busca, "zzz");
    expect(
      screen.getByText("Nenhum serviço encontrado para o filtro aplicado."),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Limpar filtros de busca" }));
    expect(titulos()).toHaveLength(3);
    expect(busca).toHaveValue("");

    await user.type(busca, "x");
    await user.click(screen.getByRole("button", { name: "Limpar busca" }));
    expect(busca).toHaveValue("");
  });

  it("sem serviços publicados", () => {
    render(<ServicosCatalogClient initialServices={[]} />);
    expect(screen.getByText("Nenhum serviço publicado ainda")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Limpar filtros de busca" }),
    ).not.toBeInTheDocument();
  });
});
