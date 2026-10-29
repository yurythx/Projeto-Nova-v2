import type { ReactNode } from "react";
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("server-only", () => ({}));
vi.mock("@/components/layout/PublicShell", () => ({
  PublicShell: ({ children }: { children: ReactNode }) => <main>{children}</main>,
}));

import FaqPage from "../page";

describe("FaqPage", () => {
  it("renderiza o cabeçalho, categorias e perguntas frequentes incluindo chamados, SSO/AD e cofre", () => {
    render(<FaqPage />);

    expect(
      screen.getByRole("heading", { name: "Perguntas Frequentes (FAQ)", level: 1 }),
    ).toBeInTheDocument();
    expect(screen.getByText("O projeto")).toBeInTheDocument();
    expect(screen.getByText("O que é o Projeto Nova?")).toBeInTheDocument();
    expect(
      screen.getByText("Como funciona o login corporativo com o Active Directory (AD)?"),
    ).toBeInTheDocument();
    expect(screen.getByText("Incidente e requisição")).toBeInTheDocument();
    expect(screen.getByText("Como abro e acompanho um chamado?")).toBeInTheDocument();
    expect(
      screen.getByText("Quem tem acesso ao Cofre de Senhas e como ele é protegido?"),
    ).toBeInTheDocument();
    expect(screen.getAllByText(/^Base: /).length).toBeGreaterThan(0);
    expect(screen.getByRole("link", { name: /sobre o projeto nova/i })).toHaveAttribute(
      "href",
      "/sobre",
    );
    expect(screen.getByText("Ainda tem dúvidas?")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /entrar/i })).toBeInTheDocument();
  });
});
