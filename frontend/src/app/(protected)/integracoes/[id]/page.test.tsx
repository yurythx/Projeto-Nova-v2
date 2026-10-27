import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

import ConexaoPage from "./page";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

const base = "v1/integracoes/conexoes/c1";

const conector = {
  id: "c1",
  tipo: "glpi",
  nome: "GLPI da prefeitura",
  modo: "conector",
  url: "",
  perfil: "",
  fuso: "America/Sao_Paulo",
  tem_credenciais: false,
  tem_token: false,
  created_at: "2026-10-27T12:00:00Z",
  created_by: "admin",
};

const ORG = [
  {
    id: "e1",
    nome: "Prefeitura",
    unidades: [{ id: "u1", entidade_id: "e1", nome: "TI", sigla: "TI", departamentos: [] }],
  },
];

const inventario = {
  itens: { chamado: 3, pessoa: 2, categoria: 1, entidade: 2, sla: 1 },
  secretarias: { SMS: 1, PREFEITURA: 2 },
  grupos: [],
  niveis: [{ id: 2, nome: "Solução N1", minutos: 240, chamados: 1, prioridade: "alta" }],
};

const linha = {
  secretaria: "PREFEITURA",
  categoria: 7,
  categoria_nome: "Serviços > Computadores",
  chamados: 2,
  com_prazo: 1,
  nivel: "Solução N1",
  unidade_id: null,
  oferta: "Computadores",
  tipo: "requisicao",
  prioridade: "alta",
};

function rotas(extra: Record<string, unknown> = {}) {
  return {
    ...identityRoutes(),
    [`GET ${base}`]: { data: conector },
    [`GET ${base}/tarefas`]: { data: [] },
    [`GET ${base}/inventario`]: { data: inventario },
    [`GET ${base}/tabela`]: { data: [] },
    "GET v1/iam/org-tree": { data: ORG },
    [`PUT ${base}`]: { data: conector },
    [`POST ${base}/token`]: { data: { token: "nova_con_abc" } },
    [`DELETE ${base}/token`]: { data: null },
    [`POST ${base}/tarefas`]: { data: {} },
    [`POST ${base}/tabela/sugestao`]: { data: [linha] },
    [`PUT ${base}/tabela`]: { data: [linha] },
    ...extra,
  };
}

beforeEach(() => resetNavigation({ id: "c1" }, "/integracoes/c1"));

describe("Conexão de integração", () => {
  it("emite o token do conector uma vez, mostra o inventário e pede a extração", async () => {
    const user = userEvent.setup();
    const api = mockBackend(rotas());
    renderApp(<ConexaoPage />);

    expect(await screen.findByRole("heading", { name: "GLPI da prefeitura" })).toBeInTheDocument();
    expect(screen.getByText(/3 chamados, 2 pessoas/)).toBeInTheDocument();
    expect(
      screen.getByText(/Solução N1: 4 h · 1 chamados · prioridade sugerida alta/),
    ).toBeInTheDocument();
    expect(screen.getByText("Nenhum token emitido.", { exact: false })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Emitir token do conector" }));
    expect(await screen.findByText(/NOVA_TOKEN=nova_con_abc/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Extrair" }));
    expect(api.to(`POST ${base}/tarefas`)[0]?.body).toEqual({ tipo: "extracao" });

    await user.clear(screen.getByLabelText("Nome *"));
    await user.type(screen.getByLabelText("Nome *"), "GLPI");
    await user.click(screen.getByRole("button", { name: "Salvar conexão" }));
    expect(api.to(`PUT ${base}`)[0]?.body).toMatchObject({ nome: "GLPI" });
  });

  it("guarda as credenciais do modo direto e acompanha as tarefas", async () => {
    const user = userEvent.setup();
    const api = mockBackend(
      rotas({
        [`GET ${base}`]: {
          data: {
            ...conector,
            modo: "direto",
            url: "http://glpi/apirest.php",
            tem_credenciais: true,
          },
        },
        [`PUT ${base}/credenciais`]: { data: null },
        [`GET ${base}/tarefas`]: {
          data: [
            {
              id: "t1",
              conexao_id: "c1",
              tipo: "carga",
              estado: "em_andamento",
              feitos: 5,
              total: 10,
              pedida_por: "admin",
              criada_em: "2026-10-27T12:00:00Z",
            },
            {
              id: "t2",
              conexao_id: "c1",
              tipo: "extracao",
              estado: "falhou",
              feitos: 0,
              total: 0,
              erro: "sessão no GLPI",
              pedida_por: "admin",
              criada_em: "2026-10-27T11:00:00Z",
              relatorio: { chamados: 1 },
            },
          ],
        },
        "POST v1/integracoes/tarefas/t1/cancelar": { data: null },
      }),
    );
    renderApp(<ConexaoPage />);

    expect(await screen.findByText(/Há credenciais guardadas/)).toBeInTheDocument();
    await user.type(screen.getByLabelText("Login"), "yuri");
    await user.type(screen.getByLabelText("Senha"), "segredo");
    await user.click(screen.getByRole("button", { name: "Guardar credenciais" }));
    expect(api.to(`PUT ${base}/credenciais`)[0]?.body).toEqual({
      app_token: "",
      user_token: "",
      login: "yuri",
      senha: "segredo",
    });

    expect(screen.getByText("Carga · Em andamento")).toBeInTheDocument();
    expect(screen.getByLabelText("Andamento: 5 de 10")).toBeInTheDocument();
    expect(screen.getByText("sessão no GLPI")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Extrair" })).toBeDisabled();
    await user.click(screen.getByRole("button", { name: "Cancelar carga" }));
    expect(api.to("POST v1/integracoes/tarefas/t1/cancelar")).toHaveLength(1);
    await user.click(screen.getByRole("button", { name: "Atualizar" }));
  });

  it("sugere a tabela com a unidade da secretaria, edita e salva", async () => {
    const user = userEvent.setup();
    const api = mockBackend(
      rotas({
        [`GET ${base}`]: {
          data: { ...conector, tem_token: true, token_em: "2026-10-27T12:00:00Z" },
        },
      }),
    );
    renderApp(<ConexaoPage />);

    await user.selectOptions(await screen.findByLabelText("Unidade de PREFEITURA"), "u1");
    await user.click(
      screen.getByRole("button", { name: "Sugerir a tabela a partir do inventário" }),
    );
    expect(api.to(`POST ${base}/tabela/sugestao`)[0]?.body).toEqual({
      unidades: { PREFEITURA: "u1" },
    });

    const linhaDaTabela = (await screen.findByText("Serviços > Computadores")).closest("tr")!;
    await user.selectOptions(within(linhaDaTabela).getByLabelText(/Unidade de PREFEITURA/), "u1");
    await user.clear(within(linhaDaTabela).getByLabelText(/Oferta de/));
    await user.type(within(linhaDaTabela).getByLabelText(/Oferta de/), "Equipamentos");
    await user.selectOptions(within(linhaDaTabela).getByLabelText(/Tipo de/), "incidente");
    await user.selectOptions(within(linhaDaTabela).getByLabelText(/Prioridade de/), "media");
    await user.click(screen.getByRole("button", { name: "Salvar tabela" }));
    expect(api.to(`PUT ${base}/tabela`)[0]?.body).toEqual({
      linhas: [
        {
          ...linha,
          unidade_id: "u1",
          oferta: "Equipamentos",
          tipo: "incidente",
          prioridade: "media",
        },
      ],
    });

    await user.click(screen.getByRole("button", { name: "Revogar token" }));
    expect(api.to(`DELETE ${base}/token`)).toHaveLength(1);
  });

  it("mostra a conexão que não existe", async () => {
    mockBackend(rotas({ [`GET ${base}`]: fail(404, "conexão não encontrada") }));
    renderApp(<ConexaoPage />);
    expect(await screen.findByText(/conexão não encontrada|Não foi possível/)).toBeInTheDocument();
  });
});
