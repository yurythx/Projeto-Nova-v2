import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import ContratosPage from "./page";

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "Tecnologia", sigla: "TI", departamentos: [] }],
  },
];

beforeEach(() => resetNavigation({}, "/contratos"));

describe("Contratos", () => {
  it("cadastra fornecedor e lista contrato e licença", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/contratos/fornecedores": {
        data: [{ id: "f1", nome: "Nuvem Sul", email: "a@sul.test", telefone: "" }],
      },
      "GET v1/contratos/contratos": {
        data: [
          {
            id: "c1",
            numero: "12/2026",
            titulo: "Link",
            fornecedor_nome: "Nuvem Sul",
            inicio: "2026-01-01T00:00:00Z",
            fim: "2026-12-31T00:00:00Z",
            valor_centavos: 1000,
          },
        ],
      },
      "GET v1/contratos/licencas": {
        data: [{ id: "l1", nome: "Editor", fornecedor_nome: "Nuvem Sul", quantidade: 10 }],
      },
      "POST v1/contratos/fornecedores": {
        status: 201,
        data: { id: "f2", nome: "Oficina", email: "", telefone: "" },
      },
    });
    renderApp(<ContratosPage />);
    expect(await screen.findByText(/Nuvem Sul · a@sul.test/)).toBeInTheDocument();
    expect(screen.getByText(/12\/2026 · Link/)).toBeInTheDocument();
    expect(screen.getByText(/Editor · 10 assentos/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Novo fornecedor" }));
    await user.type(screen.getByLabelText("Nome *"), "Oficina");
    await user.click(screen.getByRole("button", { name: "Cadastrar fornecedor" }));
    await waitFor(() => expect(api.to("POST v1/contratos/fornecedores")).toHaveLength(1));
    expect(api.to("POST v1/contratos/fornecedores")[0]!.body).toMatchObject({ nome: "Oficina" });
  });
});
