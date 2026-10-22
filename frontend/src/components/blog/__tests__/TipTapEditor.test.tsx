import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { TipTapEditor } from "../TipTapEditor";

describe("TipTapEditor", () => {
  it("renderiza a barra de ferramentas e permite alternar para modo markdown", () => {
    const onChange = vi.fn();
    render(<TipTapEditor value="Conteúdo inicial" onChange={onChange} />);

    // Botão de alternar para Markdown
    const toggleBtn = screen.getByText("Ver Markdown");
    expect(toggleBtn).toBeInTheDocument();

    fireEvent.click(toggleBtn);

    // Agora está no modo Markdown puro
    const textarea = screen.getByLabelText("Código Markdown puro");
    expect(textarea).toBeInTheDocument();
    expect(textarea).toHaveValue("Conteúdo inicial");

    // Edita o textarea
    fireEvent.change(textarea, { target: { value: "Novo texto markdown" } });
    expect(onChange).toHaveBeenCalledWith("Novo texto markdown");

    // Alterna de volta para modo visual
    const visualBtn = screen.getByText("Modo Visual");
    fireEvent.click(visualBtn);
    expect(screen.getByText("Ver Markdown")).toBeInTheDocument();
  });

  it("renderiza os botões de formatação da barra de ferramentas", () => {
    const onChange = vi.fn();
    render(<TipTapEditor value="Texto" onChange={onChange} />);

    expect(screen.getByLabelText("Negrito")).toBeInTheDocument();
    expect(screen.getByLabelText("Itálico")).toBeInTheDocument();
    expect(screen.getByLabelText("Título H2")).toBeInTheDocument();
    expect(screen.getByLabelText("Subtítulo H3")).toBeInTheDocument();
    expect(screen.getByLabelText("Lista com marcadores")).toBeInTheDocument();
    expect(screen.getByLabelText("Lista numerada")).toBeInTheDocument();
    expect(screen.getByLabelText("Citação")).toBeInTheDocument();
    expect(screen.getByLabelText("Bloco de código")).toBeInTheDocument();
  });

  it("aciona os comandos ao clicar nos botões de formatação quando o editor está ativo", () => {
    const onChange = vi.fn();
    render(<TipTapEditor value="Parágrafo" onChange={onChange} />);

    // Clica nos botões
    fireEvent.click(screen.getByLabelText("Negrito"));
    fireEvent.click(screen.getByLabelText("Itálico"));
    fireEvent.click(screen.getByLabelText("Título H2"));
    fireEvent.click(screen.getByLabelText("Subtítulo H3"));
    fireEvent.click(screen.getByLabelText("Citação"));
  });
});
