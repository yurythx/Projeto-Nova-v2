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

const SEGREDO = { id: "s1", nome: "Correio", usuario: "ana", tipo: "aplicacao", endereco: "https://correio.exemplo", versao: 1 };

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
    expect(await screen.findByText(/ana/)).toBeInTheDocument();
    expect(screen.queryByText("https://correio.exemplo")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Correio" }));
    expect(screen.getByRole("heading", { name: "Correio" })).toBeInTheDocument();
    expect(screen.getByText("Acesso de administrador")).toBeInTheDocument();
    expect(screen.getByText("Usuário administrador")).toBeInTheDocument();
    expect(screen.getByText("https://correio.exemplo")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("senha-super")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Revelar Correio" }));
    expect(await screen.findByLabelText("Senha de administrador")).toHaveValue("senha-super");
    expect(api.to("POST v1/cofre/segredos/s1/revelar")).toHaveLength(1);
    expect(api.to("POST v1/cofre/segredos/s1/revelar")[0]!.body).toEqual({});
    expect(screen.getByText("pista")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Ocultar Correio" }));
    expect(screen.queryByDisplayValue("senha-super")).not.toBeInTheDocument();
  });

  it("abre o painel de uma VM com IP, usuário e senha de SSH", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes(),
      "GET v1/cofre/segredos": {
        data: [{ id: "vm", nome: "web", usuario: "root", tipo: "maquina", endereco: "10.0.0.8", versao: 1 }],
      },
      "POST v1/cofre/segredos/vm/revelar": {
        data: { id: "vm", nome: "web", usuario: "root", tipo: "maquina", endereco: "10.0.0.8", senha: "senha-ssh", notas: "", versao: 1 },
      },
    });
    renderApp(<CofrePage />);
    const item = await screen.findByRole("button", { name: "web" });
    expect(screen.queryByText("10.0.0.8")).not.toBeInTheDocument();
    await user.click(item);
    expect(screen.getByText("Acesso SSH")).toBeInTheDocument();
    expect(screen.getByText("10.0.0.8")).toBeInTheDocument();
    expect(screen.getByText("Usuário SSH")).toBeInTheDocument();
    expect(screen.getByText("root")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Revelar web" }));
    expect(await screen.findByLabelText("Senha SSH")).toHaveValue("senha-ssh");
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
    await user.click(screen.getByRole("button", { name: "Nova senha" }));
    await user.type(screen.getByLabelText("Nome *"), "Wifi");
    await user.type(screen.getByLabelText("Usuário"), "rede");
    await user.type(screen.getByLabelText("Senha *"), "chave-da-rede");
    await user.click(screen.getByRole("button", { name: "Guardar no cofre" }));
    await waitFor(() => expect(api.to("POST v1/cofre/segredos")).toHaveLength(1));
    expect(api.to("POST v1/cofre/segredos")[0]!.body).toMatchObject({
      nome: "Wifi",
      usuario: "rede",
      senha: "chave-da-rede",
      tipo: "outro",
      endereco: "",
      rotaciona: false,
    });
    expect(screen.queryByDisplayValue("chave-da-rede")).not.toBeInTheDocument();
    expect(await screen.findByText("Senha guardada no cofre")).toBeInTheDocument();
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

  it("não revela um segredo retirado por outra pessoa", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/cofre/segredos": {
        data: [{ ...SEGREDO, retirada: { usuario_id: "outra", nome: "Bia", ate: "2026-10-25T18:00:00Z" } }],
      },
      "POST v1/cofre/segredos/s1/devolver": { status: 204, data: null },
    });
    renderApp(<CofrePage />);
    await user.click(await screen.findByRole("button", { name: "Correio" }));
    expect(screen.getByText(/Retirado por Bia/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Revelar Correio" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retirar Correio" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Devolver Correio" }));
    await waitFor(() => expect(api.to("POST v1/cofre/segredos/s1/devolver")).toHaveLength(1));
  });

  it("retira com prazo e senha de acesso", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/cofre/segredos": { data: [SEGREDO] },
      "POST v1/cofre/segredos/s1/retirar": { data: { ...SEGREDO, senha: "senha-super", notas: "" } },
    });
    renderApp(<CofrePage />);
    await user.click(await screen.findByRole("button", { name: "Correio" }));
    await user.click(screen.getByRole("button", { name: "Mais de Correio" }));
    await user.click(screen.getByRole("button", { name: "Retirar Correio" }));
    await user.type(screen.getByLabelText("Sua senha de acesso"), "acesso");
    await user.click(screen.getByRole("button", { name: "Confirmar retirada de Correio" }));
    await waitFor(() => expect(api.to("POST v1/cofre/segredos/s1/retirar")).toHaveLength(1));
    expect(api.to("POST v1/cofre/segredos/s1/retirar")[0]!.body).toMatchObject({
      senha_de_acesso: "acesso",
      minutos: 60,
    });
    expect(await screen.findByLabelText("Senha de administrador")).toHaveValue("senha-super");
  });

  it("avisa que a senha rotacionada ainda precisa ser aplicada", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/cofre/segredos": {
        data: [{ ...SEGREDO, rotaciona: true, rotacao_pendente: true }],
      },
      "POST v1/cofre/segredos/s1/rotacao/aplicada": { status: 204, data: null },
    });
    renderApp(<CofrePage />);
    await user.click(await screen.findByRole("button", { name: "Correio" }));
    expect(screen.getByText(/aplique em https:\/\/correio.exemplo/)).toBeInTheDocument();
    expect(screen.queryByText("senha-super")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Já apliquei Correio" }));
    await waitFor(() => expect(api.to("POST v1/cofre/segredos/s1/rotacao/aplicada")).toHaveLength(1));
  });

  it("coloca o segredo numa pasta sem pedir a senha de novo", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/cofre/segredos": { data: [SEGREDO] },
      "GET v1/cofre/pastas": { data: [{ id: "p1", nome: "Servidores" }] },
      "PUT v1/cofre/segredos/s1": { data: { ...SEGREDO, pasta_id: "p1" } },
    });
    renderApp(<CofrePage />);
    await user.click(await screen.findByRole("button", { name: "Correio" }));
    await user.click(screen.getByRole("button", { name: "Mais de Correio" }));
    await user.selectOptions(screen.getByLabelText("Pasta de Correio"), "p1");
    await user.click(screen.getByRole("button", { name: "Guardar pasta de Correio" }));
    await waitFor(() => expect(api.to("PUT v1/cofre/segredos/s1")).toHaveLength(1));
    expect(api.to("PUT v1/cofre/segredos/s1")[0]!.body).toEqual({
      nome: "Correio",
      usuario: "ana",
      mover_pasta: true,
      pasta_id: "p1",
    });
    expect(screen.queryByDisplayValue("senha-super")).not.toBeInTheDocument();
  });

  it("troca quem entra na pasta e a apaga devolvendo o segredo à unidade", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/cofre/segredos": { data: [] },
      "GET v1/cofre/pastas": {
        data: [{ id: "p1", nome: "Servidores", membros: [{ usuario_id: "u9", pode_revelar: false }] }],
      },
      "GET v1/cofre/pessoas": { data: [{ id: "u9", nome: "Maria" }] },
      "PUT v1/cofre/pastas/p1/membros": { data: { id: "p1", nome: "Servidores" } },
      "DELETE v1/cofre/pastas/p1": { status: 204, data: null },
    });
    renderApp(<CofrePage />);
    await user.click(await screen.findByRole("button", { name: "Servidores" }));
    await user.click(screen.getByRole("button", { name: "Opções da pasta Servidores" }));
    await user.click(screen.getByRole("button", { name: "Quem entra em Servidores (1)" }));
    expect(await screen.findByRole("checkbox", { name: "Maria" })).toBeChecked();
    await user.click(screen.getByRole("checkbox", { name: "pode revelar Maria" }));
    await user.click(screen.getByRole("button", { name: "Guardar quem entra em Servidores" }));
    await waitFor(() => expect(api.to("PUT v1/cofre/pastas/p1/membros")).toHaveLength(1));
    expect(api.to("PUT v1/cofre/pastas/p1/membros")[0]!.body).toEqual({
      membros: [{ usuario_id: "u9", pode_revelar: true }],
    });
    await user.click(screen.getByRole("button", { name: "Opções da pasta Servidores" }));
    await user.click(screen.getByRole("button", { name: "Apagar pasta Servidores" }));
    expect(api.to("DELETE v1/cofre/pastas/p1")).toHaveLength(0);
    expect(screen.getByRole("heading", { name: "Apagar a pasta Servidores?" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Confirmar exclusão da pasta Servidores" }));
    await waitFor(() => expect(api.to("DELETE v1/cofre/pastas/p1")).toHaveLength(1));
  });

  it("não apaga a pasta que ainda tem senha", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/cofre/segredos": { data: [{ ...SEGREDO, pasta_id: "p1" }] },
      "GET v1/cofre/pastas": { data: [{ id: "p1", nome: "Servidores" }] },
      "DELETE v1/cofre/pastas/p1": { status: 204, data: null },
    });
    renderApp(<CofrePage />);
    await user.click(await screen.findByRole("button", { name: "Servidores" }));
    await user.click(screen.getByRole("button", { name: "Opções da pasta Servidores" }));
    await user.click(screen.getByRole("button", { name: "Apagar pasta Servidores" }));
    expect(screen.getByRole("heading", { name: "A pasta Servidores tem senha" })).toBeInTheDocument();
    expect(screen.getByText(/não apaga enquanto houver senha/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Confirmar exclusão da pasta Servidores" })).not.toBeInTheDocument();
    expect(api.to("DELETE v1/cofre/pastas/p1")).toHaveLength(0);
  });

  it("pede confirmação antes de apagar a senha", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/cofre/segredos": { data: [SEGREDO] },
      "DELETE v1/cofre/segredos/s1": { status: 204, data: null },
    });
    renderApp(<CofrePage />);
    await user.click(await screen.findByRole("button", { name: "Correio" }));
    await user.click(screen.getByRole("button", { name: "Mais de Correio" }));
    await user.click(screen.getByRole("button", { name: "Excluir Correio" }));
    expect(api.to("DELETE v1/cofre/segredos/s1")).toHaveLength(0);
    expect(screen.getByRole("heading", { name: "Apagar a senha Correio?" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(api.to("DELETE v1/cofre/segredos/s1")).toHaveLength(0);
    await user.click(screen.getByRole("button", { name: "Excluir Correio" }));
    await user.click(screen.getByRole("button", { name: "Confirmar exclusão de Correio" }));
    await waitFor(() => expect(api.to("DELETE v1/cofre/segredos/s1")).toHaveLength(1));
  });
});
