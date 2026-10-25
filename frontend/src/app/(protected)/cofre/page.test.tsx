import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import CofrePage from "./page";

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "Tecnologia", sigla: "TI", departamentos: [] }],
  },
];

const SEGREDO = { id: "s1", nome: "Correio", usuario: "ana", versao: 1 };

beforeEach(() => resetNavigation({}, "/cofre"));

describe("Cofre", () => {
  it("lista sem a senha e só mostra o valor depois de revelar", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/cofre/segredos": { data: [SEGREDO] },
      "POST v1/cofre/segredos/s1/revelar": { data: { ...SEGREDO, senha: "senha-super", notas: "pista" } },
    });
    renderApp(<CofrePage />);
    expect(await screen.findByRole("heading", { name: "Correio" })).toBeInTheDocument();
    expect(screen.getByText("ana")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("senha-super")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Revelar Correio" }));
    await user.type(screen.getByLabelText("Sua senha de acesso"), "acesso");
    await user.click(screen.getByRole("button", { name: "Confirmar revelação de Correio" }));
    expect(await screen.findByLabelText("Senha de Correio")).toHaveValue("senha-super");
    expect(api.to("POST v1/cofre/segredos/s1/revelar")).toHaveLength(1);
    expect(api.to("POST v1/cofre/segredos/s1/revelar")[0]!.body).toMatchObject({ senha_de_acesso: "acesso" });
    expect(screen.queryByDisplayValue("acesso")).not.toBeInTheDocument();
    expect(screen.getByText("pista")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Ocultar Correio" }));
    expect(screen.queryByDisplayValue("senha-super")).not.toBeInTheDocument();
  });

  it("guarda um segredo sem devolver a senha na lista", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/cofre/segredos": { data: [] },
      "POST v1/cofre/segredos": { status: 201, data: { id: "s2", nome: "Wifi", usuario: "rede", versao: 1 } },
    });
    renderApp(<CofrePage />);
    expect(await screen.findByText("Nenhum segredo neste recorte")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Nome *"), "Wifi");
    await user.type(screen.getByLabelText("Usuário"), "rede");
    await user.type(screen.getByLabelText("Senha *"), "chave-da-rede");
    await user.click(screen.getByRole("button", { name: "Guardar no cofre" }));
    await waitFor(() => expect(api.to("POST v1/cofre/segredos")).toHaveLength(1));
    expect(api.to("POST v1/cofre/segredos")[0]!.body).toMatchObject({
      nome: "Wifi",
      usuario: "rede",
      senha: "chave-da-rede",
    });
    expect(screen.queryByDisplayValue("chave-da-rede")).not.toBeInTheDocument();
  });

  it("quem não tem permissão não consulta o cofre", async () => {
    const api = mockBackend({
      ...identityRoutes({ permissions: [], scopes: [] }),
      "GET v1/cofre/segredos": { data: [SEGREDO] },
    });
    renderApp(<CofrePage />);
    expect(await screen.findByText("Seu perfil não abre o cofre")).toBeInTheDocument();
    expect(api.to("GET v1/cofre/segredos")).toHaveLength(0);
  });
});
