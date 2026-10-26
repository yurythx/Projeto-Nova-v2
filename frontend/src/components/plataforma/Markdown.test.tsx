import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Markdown, safeMarkdownHref } from "./Markdown";

describe("Markdown seguro", () => {
  it("renderiza títulos, listas, ênfase e código", () => {
    const { container } = render(
      <Markdown
        source={
          "# Título\n\n- um\n- **dois**\n\n1. primeiro\n\n`codigo` e *itálico*\n\n```\nbloco\n```"
        }
      />,
    );
    expect(screen.getByRole("heading", { name: "Título" })).toBeInTheDocument();
    expect(container.querySelectorAll("ul li")).toHaveLength(2);
    expect(container.querySelector("ol li")).toHaveTextContent("primeiro");
    expect(container.querySelector("strong")).toHaveTextContent("dois");
    expect(container.querySelector("pre code")).toHaveTextContent("bloco");
  });

  it("HTML embutido aparece como texto (sem XSS)", () => {
    const { container } = render(
      <Markdown source={'<img src=x onerror="alert(1)"><script>alert(2)</script>'} />,
    );
    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("script")).toBeNull();
    expect(container).toHaveTextContent("<script>alert(2)</script>");
  });

  it("links: internos e http(s) viram <a>; javascript:/data:/protocol-relative viram texto", () => {
    render(
      <Markdown
        source={
          "[interno](/wiki/manual) [site](https://gov.br) [mau](javascript:alert(1)) [dados](data:text/html,x) [outro](//evil.com)"
        }
      />,
    );
    expect(screen.getByRole("link", { name: "interno" })).toHaveAttribute("href", "/wiki/manual");
    expect(screen.getByRole("link", { name: "interno" })).not.toHaveAttribute("target");
    expect(screen.getByRole("link", { name: "site" })).toHaveAttribute(
      "rel",
      "noopener noreferrer",
    );
    expect(screen.queryByRole("link", { name: "mau" })).toBeNull();
    expect(screen.queryByRole("link", { name: "dados" })).toBeNull();
    expect(screen.queryByRole("link", { name: "outro" })).toBeNull();
  });

  it("safeMarkdownHref", () => {
    expect(safeMarkdownHref("/a/b")).toBe("/a/b");
    expect(safeMarkdownHref("//evil.com")).toBeNull();
    expect(safeMarkdownHref("/\\evil.com")).toBeNull();
    expect(safeMarkdownHref("JAVASCRIPT:alert(1)")).toBeNull();
    expect(safeMarkdownHref("http://x.gov.br/")).toBe("http://x.gov.br/");
  });

  it("renderiza tabelas Markdown GFM com scroll responsivo e alinhamento", () => {
    const tableSource = `
| Ferramenta | Função | Benefício |
| :--- | :---: | ---: |
| Proxmox | Virtualização | Open-source |
| Ansible | Automação | Idempotência |
`;
    const { container } = render(<Markdown source={tableSource} />);
    const table = container.querySelector("table");
    expect(table).not.toBeNull();
    expect(container.querySelectorAll("thead th")).toHaveLength(3);
    expect(container.querySelectorAll("tbody tr")).toHaveLength(2);
    expect(container.querySelector("th")).toHaveTextContent("Ferramenta");
    expect(container.querySelector("td")).toHaveTextContent("Proxmox");
  });
});

describe("Markdown — imagens, tabelas e avisos", () => {
  it("imagens: seguras viram <img> com legenda; inseguras viram texto", () => {
    const { container } = render(
      <Markdown
        source={
          "![Fachada](/img/a.png) ![](https://gov.br/b.png) ![Mau](javascript:x) ![](javascript:y)"
        }
      />,
    );
    const imgs = container.querySelectorAll("img");
    expect(imgs).toHaveLength(2);
    expect(imgs[0]).toHaveAttribute("alt", "Fachada");
    expect(screen.getByText("Fachada")).toBeInTheDocument();
    expect(container).toHaveTextContent("[Imagem: Mau]");
  });

  it("tabela GFM com alinhamentos", () => {
    const { container } = render(
      <Markdown
        source={
          "| Série | Prazo | Destino |\n| :--- | :---: | ---: |\n| **2.0** | 5 anos | Eliminação |\n| 3.0 | 10 |\n\ndepois"
        }
      />,
    );
    const ths = container.querySelectorAll("th");
    expect(ths[0]).toHaveClass("text-left");
    expect(ths[1]).toHaveClass("text-center");
    expect(ths[2]).toHaveClass("text-right");
    expect(container.querySelectorAll("tbody tr")).toHaveLength(2);
    expect(container.querySelector("td strong")).toHaveTextContent("2.0");
    expect(screen.getByText("depois")).toBeInTheDocument();
  });

  it("avisos [!TIPO], tipo desconhecido e citação comum", () => {
    render(
      <Markdown
        source={
          "> [!WARNING] Prazo de 30 dias\n> conta da publicação\n\n> [!TIP]\n> Use o modelo\n\n> [!OUTRO] x\n\n> [!CAUTION] a\n\n> [!IMPORTANT] b\n\n> [!INFO] c\n\n> [!NOTE] d\n\n> citação simples\n> continua"
        }
      />,
    );
    const notas = screen.getAllByRole("note");
    expect(notas).toHaveLength(7);
    expect(notas[0]).toHaveTextContent("AtençãoPrazo de 30 dias conta da publicação");
    expect(notas[1]).toHaveTextContent("DicaUse o modelo");
    expect(notas[2]).toHaveTextContent("Notax");
    expect(screen.getByText("citação simples continua").closest("blockquote")).not.toBeNull();
  });

  it("títulos ganham âncora e listas com * também valem", () => {
    const { container } = render(
      <Markdown
        source={"## Prazos de guarda\n#### Nível 4\n* um\r\n* dois\n\nlinha um\nlinha dois"}
      />,
    );
    expect(screen.getByRole("heading", { level: 3, name: /Prazos de guarda/ })).toHaveAttribute(
      "id",
    );
    expect(screen.getByRole("heading", { level: 5, name: /Nível 4/ })).toBeInTheDocument();
    expect(container.querySelectorAll("ul li")).toHaveLength(2);
    expect(screen.getByText("linha um linha dois")).toBeInTheDocument();
  });
});
