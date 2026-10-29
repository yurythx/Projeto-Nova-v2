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
    nome: "PREFEITURA",
    sigla: "PREFEITURA",
    unidades: [
      { id: "u-sinfra", nome: "SINFRA", sigla: "SINFRA", departamentos: [] },
      { id: "u-ti", nome: "TI", sigla: "TI", departamentos: [] },
    ],
  },
  {
    id: "ent-sms",
    nome: "SMS",
    sigla: "SMS",
    unidades: [{ id: "u-sms", nome: "SMS", sigla: "SMS", departamentos: [] }],
  },
  {
    id: "ent-semecel",
    nome: "SEMECEL",
    sigla: "SEMECEL",
    unidades: [{ id: "u-semecel", nome: "SEMECEL", sigla: "SEMECEL", departamentos: [] }],
  },
  {
    id: "ent-obsoleta",
    nome: "Root Entity GLPI Descartavel",
    sigla: "ROOT",
    unidades: [],
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
        scopes: [{ perfil: "servidor", unidade_id: "u-sms", permissions: ["atendimento:abrir"] }],
      },
      canGlobal: () => false,
    });
    mockUseApiQuery.mockReturnValue({ data: mockTree });

    const { container } = render(<EntitySelector />);
    expect(container.firstChild).toBeNull();
  });

  it("renderiza para administrador e filtra estritamente as 3 entidades oficiais (Prefeitura, SMS, SEMECEL)", async () => {
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
    expect(screen.getByText("Prefeitura Municipal")).toBeInTheDocument();
    expect(screen.getByText("Secretaria Municipal de Saúde")).toBeInTheDocument();
    expect(
      screen.getByText("Secretaria de Educação, Cultura, Esporte e Lazer"),
    ).toBeInTheDocument();

    // Entidades obsoletas ou unidades filhas operacionais NÃO aparecem
    expect(screen.queryByText("Root Entity GLPI Descartavel")).not.toBeInTheDocument();
    expect(screen.queryByText("SINFRA")).not.toBeInTheDocument();
    expect(screen.queryByText("TI")).not.toBeInTheDocument();

    // Seleciona SMS
    await user.click(screen.getByText("Secretaria Municipal de Saúde"));
    expect(getActiveEntitySnapshot()).toBe("ent-sms");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("permite comutar para SEMECEL e voltar para visão global", async () => {
    const user = userEvent.setup();
    setActiveEntity("ent-semecel");

    mockUsePlataforma.mockReturnValue({
      me: {
        permissions: ["*"],
        scopes: [{ perfil: "admin", permissions: ["*"] }],
      },
      canGlobal: () => true,
    });
    mockUseApiQuery.mockReturnValue({ data: mockTree });

    render(<EntitySelector />);

    expect(screen.getByRole("button", { name: /Selecionar instituição ativa/i })).toHaveTextContent(
      "SEMECEL",
    );

    await user.click(screen.getByRole("button", { name: /Selecionar instituição ativa/i }));
    await user.click(screen.getByText("Todas as instituições (visão global)"));

    expect(getActiveEntitySnapshot()).toBe("");
  });
});
