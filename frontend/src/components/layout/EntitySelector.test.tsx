import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EntitySelector } from "./EntitySelector";
import { getActiveEntitySnapshot, setActiveEntity } from "@/lib/plataforma/activeEntityStore";

const mockUsePlataforma = vi.fn();
const mockUseApiQuery = vi.fn();

vi.mock("@/lib/plataforma/PlataformaProvider", () => ({
  usePlataforma: () => mockUsePlataforma(),
}));

vi.mock("@/lib/api/swr", () => ({
  useApiQuery: (...args: unknown[]) => mockUseApiQuery(...args),
}));

const mockTree = [
  {
    id: "ent-pref",
    nome: "Prefeitura Municipal",
    sigla: "PREF",
    unidades: [
      { id: "u-saude", nome: "Secretaria de Saúde", sigla: "SMS", departamentos: [] },
      { id: "u-educacao", nome: "Secretaria de Educação", sigla: "SEMED", departamentos: [] },
    ],
  },
];

describe("EntitySelector — Seletor de Entidade Ativa (GLPI)", () => {
  beforeEach(() => {
    setActiveEntity("");
    vi.clearAllMocks();
  });

  it("não renderiza para usuário comum com apenas uma lotação", () => {
    mockUsePlataforma.mockReturnValue({
      me: {
        permissions: ["atendimento:abrir"],
        scopes: [{ perfil: "servidor", unidade_id: "u-saude", permissions: ["atendimento:abrir"] }],
      },
      canGlobal: () => false,
    });
    mockUseApiQuery.mockReturnValue({ data: mockTree });

    const { container } = render(<EntitySelector />);
    expect(container.firstChild).toBeNull();
  });

  it("renderiza para administrador global e permite selecionar uma secretaria", async () => {
    const user = userEvent.setup();
    mockUsePlataforma.mockReturnValue({
      me: {
        permissions: ["*"],
        scopes: [{ perfil: "admin", permissions: ["*"] }],
      },
      canGlobal: () => true,
    });
    mockUseApiQuery.mockReturnValue({ data: mockTree });

    render(<EntitySelector />);

    const botao = screen.getByRole("button", { name: /Selecionar instituição ativa/i });
    expect(botao).toHaveTextContent("Todas as instituições");

    await user.click(botao);
    expect(screen.getByRole("menu", { name: "Instituições disponíveis" })).toBeInTheDocument();
    expect(screen.getByText("Todas as instituições (visão global)")).toBeInTheDocument();
    expect(screen.getByText("Secretaria de Saúde")).toBeInTheDocument();
    expect(screen.getByText("Secretaria de Educação")).toBeInTheDocument();

    // Seleciona a Secretaria de Educação
    await user.click(screen.getByText("Secretaria de Educação"));
    expect(getActiveEntitySnapshot()).toBe("u-educacao");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("permite voltar para visão global de todas as instituições", async () => {
    const user = userEvent.setup();
    setActiveEntity("u-saude");

    mockUsePlataforma.mockReturnValue({
      me: {
        permissions: ["*"],
        scopes: [{ perfil: "admin", permissions: ["*"] }],
      },
      canGlobal: () => true,
    });
    mockUseApiQuery.mockReturnValue({ data: mockTree });

    render(<EntitySelector />);

    expect(screen.getByRole("button", { name: /Selecionar instituição ativa/i })).toHaveTextContent("SMS");

    await user.click(screen.getByRole("button", { name: /Selecionar instituição ativa/i }));
    await user.click(screen.getByText("Todas as instituições (visão global)"));

    expect(getActiveEntitySnapshot()).toBe("");
  });
});
