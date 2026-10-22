import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { deCampoData, paraCampoData } from "@/lib/servicos/servicos";
import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtivosPage from "./page";

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "Tecnologia", sigla: "TI", departamentos: [] }],
  },
];
const switchA = {
  id: "a1",
  nome: "Switch",
  tipo: "rede",
  estado: "em_uso",
  identificador: "PAT-1",
  descricao: "",
  produto_id: "p1",
  unidade_id: "u1",
};
const licenca = {
  id: "a2",
  nome: "Licença",
  tipo: "software",
  estado: "manutencao",
  identificador: "",
  descricao: "",
  produto_id: "p9",
};

beforeEach(() => resetNavigation({}, "/ativos"));

describe("Ativos", () => {
  it("lista, cadastra e edita", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/produtos": { data: [{ id: "p1", nome: "Rede da escola" }] },
      "GET v1/ativos": { data: [switchA, licenca] },
      "POST v1/ativos": { status: 201, data: switchA },
      "PUT v1/ativos/a1": { data: switchA },
    });
    renderApp(<AtivosPage />);
    expect(await screen.findByText(/Rede · PAT-1 · Rede da escola/)).toBeInTheDocument();
    expect(screen.getByText(/Sistema ou licença · produto vinculado/)).toBeInTheDocument();
    expect(within(screen.getByRole("list")).getByText("Em manutenção")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Nome *"), "Nobreak");
    await user.selectOptions(screen.getByLabelText("Tipo"), "hardware");
    await user.selectOptions(await screen.findByLabelText("Produto que sustenta"), "p1");
    await user.click(screen.getByRole("button", { name: "Cadastrar ativo" }));
    await waitFor(() => expect(api.to("POST v1/ativos")).toHaveLength(1));
    expect(api.to("POST v1/ativos")[0]!.body).toMatchObject({
      nome: "Nobreak",
      tipo: "hardware",
      estado: "em_uso",
      produto_id: "p1",
    });

    await user.click(screen.getByRole("button", { name: "Editar Switch" }));
    expect(screen.getByRole("heading", { name: "Editar ativo" })).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Estado"), "desativado");
    await user.click(screen.getByRole("button", { name: "Salvar ativo" }));
    await waitFor(() => expect(api.to("PUT v1/ativos/a1")).toHaveLength(1));
    expect(api.to("PUT v1/ativos/a1")[0]!.body).toMatchObject({
      estado: "desativado",
      identificador: "PAT-1",
    });
    await user.click(screen.getByRole("button", { name: "Editar Licença" }));
    await user.click(screen.getByRole("button", { name: "Cancelar edição" }));
    expect(screen.getByRole("heading", { name: "Novo ativo" })).toBeInTheDocument();
  });

  it("filtro por produto, vazio, falha e sem permissão", async () => {
    const user = userEvent.setup();
    resetNavigation({}, "/ativos", "produto=p1");
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/produtos": { data: [{ id: "p1", nome: "Rede da escola" }] },
      "GET v1/ativos": { data: [] },
    });
    const filtro = renderApp(<AtivosPage />);
    expect(await screen.findByText("Nenhum ativo cadastrado")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText(/Mostrando os ativos de/)).toHaveTextContent("Rede da escola"));
    expect(api.to("GET v1/ativos")[0]!.query.get("produto_id")).toBe("p1");
    expect(screen.getByRole("link", { name: "Ver todos" })).toHaveAttribute("href", "/ativos");
    filtro.unmount();

    resetNavigation({}, "/ativos", "produto=p9");
    mockBackend({
      ...identityRoutes({
        permissions: ["ativos:manage"],
        scopes: [{ perfil: "p", origem: "manual", permissions: ["ativos:manage"] }],
      }),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/ativos": fail(500, "fora"),
    });
    const erro = renderApp(<AtivosPage />);
    expect(await screen.findByText(/Mostrando os ativos de/)).toHaveTextContent("um produto");
    const retry = mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/ativos": fail(500, "fora"),
      "GET v1/produtos": { data: [] },
    });
    await user.click(await screen.findByRole("button", { name: "Tentar novamente" }));
    await waitFor(() => expect(retry.to("GET v1/ativos").length).toBeGreaterThan(0));
    erro.unmount();

    mockBackend(identityRoutes({ permissions: [], scopes: [] }));
    renderApp(<AtivosPage />);
    expect(await screen.findByText("Seu perfil não gere ativos")).toBeInTheDocument();
  });

  it("recusa ao gravar mantém o formulário", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(),
      "GET v1/iam/org-tree": { data: ORG },
      "GET v1/produtos": { data: [] },
      "GET v1/ativos": { data: [] },
      "POST v1/ativos": fail(409, "já há ativo com este identificador"),
    });
    renderApp(<AtivosPage />);
    await user.type(await screen.findByLabelText("Nome *"), "Dup");
    await user.click(screen.getByRole("button", { name: "Cadastrar ativo" }));
    expect(await screen.findByText("já há ativo com este identificador")).toBeInTheDocument();
    expect(
      within(
        screen
          .getByRole("heading", { name: "Novo ativo" })
          .closest("div.rounded-xl") as HTMLElement,
      ).getByLabelText("Nome *"),
    ).toHaveValue("Dup");
  });
});

describe("datas da janela", () => {
  it("vai e volta entre ISO e o campo local", () => {
    expect(paraCampoData(undefined)).toBe("");
    expect(deCampoData("")).toBeNull();
    const iso = deCampoData("2026-10-21T14:30")!;
    expect(paraCampoData(iso)).toBe("2026-10-21T14:30");
  });
});
