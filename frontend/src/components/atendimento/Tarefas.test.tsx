import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import type { Tarefa } from "@/lib/atendimento/atendimento";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import { TarefasCard } from "./TarefasCard";
import { TarefasDaOferta } from "./TarefasDaOferta";

const semPermissao = { permissions: [], scopes: [] };

const oferta = {
  id: "o1",
  nome: "Admissão",
  descricao: "",
  tipo: "requisicao" as const,
  unidade_id: "un1",
  ativa: true,
};

const equipes = [
  { id: "e1", nome: "Suporte" },
  { id: "e2", nome: "Rede" },
];

describe("TarefasDaOferta", () => {
  it("monta o modelo com a equipe de cada tarefa e grava", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(semPermissao),
      "GET v1/atendimento/ofertas/o1/tarefas": {
        data: [{ ordem: 0, titulo: "Criar conta", equipe_id: "e1", equipe_nome: "Suporte" }],
      },
      "PUT v1/atendimento/ofertas/o1/tarefas": { data: [] },
    });
    const onFechar = vi.fn();
    renderApp(<TarefasDaOferta oferta={oferta} equipes={equipes} onFechar={onFechar} />);
    expect(await screen.findByDisplayValue("Criar conta")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Acrescentar tarefa/ }));
    await user.type(screen.getAllByLabelText("O que fazer")[1]!, "Liberar VPN");
    await user.selectOptions(screen.getAllByLabelText("Equipe")[1]!, "e2");
    await user.click(screen.getByRole("button", { name: "Salvar tarefas" }));
    await waitFor(() => expect(api.to("PUT v1/atendimento/ofertas/o1/tarefas")).toHaveLength(1));
    expect(api.to("PUT v1/atendimento/ofertas/o1/tarefas")[0]!.body).toEqual({
      tarefas: [
        { titulo: "Criar conta", equipe_id: "e1" },
        { titulo: "Liberar VPN", equipe_id: "e2" },
      ],
    });
    await user.click(screen.getByRole("button", { name: "Fechar" }));
    expect(onFechar).toHaveBeenCalled();
  });

  it("tirar todas deixa a oferta sem tarefas", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(semPermissao),
      "GET v1/atendimento/ofertas/o1/tarefas": {
        data: [{ ordem: 0, titulo: "Criar conta", equipe_id: "e1" }],
      },
      "PUT v1/atendimento/ofertas/o1/tarefas": { data: [] },
    });
    renderApp(<TarefasDaOferta oferta={oferta} equipes={equipes} onFechar={() => {}} />);
    await user.click(await screen.findByRole("button", { name: /Tirar a tarefa 1/ }));
    await user.click(screen.getByRole("button", { name: "Salvar tarefas" }));
    await waitFor(() => expect(api.to("PUT v1/atendimento/ofertas/o1/tarefas")).toHaveLength(1));
    expect(api.to("PUT v1/atendimento/ofertas/o1/tarefas")[0]!.body).toEqual({ tarefas: [] });
  });
});

const tarefa = (t: Partial<Tarefa>): Tarefa => ({
  id: "t1",
  ordem: 0,
  titulo: "Criar conta",
  equipe_nome: "Suporte",
  estado: "pendente",
  minha: true,
  ...t,
});

describe("TarefasCard", () => {
  it("some sem tarefas", () => {
    mockBackend(identityRoutes(semPermissao));
    const { container } = renderApp(
      <TarefasCard solicitacao="s1" tarefas={[]} podeAgir onMudou={() => {}} />,
    );
    expect(container.querySelector("h2")).toBeNull();
  });

  it("conclui a da minha equipe com observação", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(semPermissao),
      "POST v1/atendimento/solicitacoes/s1/tarefas/t1": { data: {} },
    });
    const onMudou = vi.fn();
    renderApp(
      <TarefasCard
        solicitacao="s1"
        tarefas={[
          tarefa({}),
          tarefa({ id: "t2", titulo: "Liberar VPN", equipe_nome: "Rede", minha: false }),
        ]}
        podeAgir
        onMudou={onMudou}
      />,
    );
    expect(screen.getByText(/2 de 2 pendentes/)).toBeInTheDocument();
    // A da outra equipe só se dispensa.
    expect(screen.queryByRole("button", { name: "Concluir Liberar VPN" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Dispensar Liberar VPN" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Concluir Criar conta" }));
    await user.type(screen.getByLabelText("Observação (opcional)"), " ana.silva ");
    await user.click(screen.getByRole("button", { name: "Confirmar conclusão" }));
    await waitFor(() => expect(onMudou).toHaveBeenCalled());
    expect(api.to("POST v1/atendimento/solicitacoes/s1/tarefas/t1")[0]!.body).toEqual({
      concluir: true,
      observacao: "ana.silva",
    });
  });

  it("dispensa só com motivo e pode voltar atrás", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(semPermissao),
      "POST v1/atendimento/solicitacoes/s1/tarefas/t1": { data: {} },
    });
    renderApp(
      <TarefasCard
        solicitacao="s1"
        tarefas={[tarefa({ minha: false })]}
        podeAgir
        onMudou={() => {}}
      />,
    );
    await user.click(screen.getByRole("button", { name: "Dispensar Criar conta" }));
    await user.click(screen.getByRole("button", { name: "Voltar" }));
    await user.click(screen.getByRole("button", { name: "Dispensar Criar conta" }));
    const confirmar = screen.getByRole("button", { name: "Confirmar dispensa" });
    expect(confirmar).toBeDisabled();
    await user.type(screen.getByLabelText("Motivo da dispensa"), "Não usa VPN");
    await user.click(confirmar);
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/tarefas/t1")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/tarefas/t1")[0]!.body).toEqual({
      concluir: false,
      observacao: "Não usa VPN",
    });
  });

  it("mostra quem fez e não oferece ação sem poder agir", () => {
    mockBackend(identityRoutes(semPermissao));
    renderApp(
      <TarefasCard
        solicitacao="s1"
        tarefas={[
          tarefa({
            estado: "concluida",
            feita_por_nome: "Beto",
            feita_em: "2026-10-01T12:00:00Z",
            observacao: "ok",
          }),
          tarefa({ id: "t2", estado: "dispensada", titulo: "VPN" }),
          tarefa({ id: "t3", titulo: "Notebook" }),
        ]}
        podeAgir={false}
        onMudou={() => {}}
      />,
    );
    expect(screen.getByText(/1 de 3 pendente\./)).toBeInTheDocument();
    expect(screen.getByText(/Beto em .*: ok/)).toBeInTheDocument();
    expect(screen.getByText("Alguém")).toBeInTheDocument();
    expect(screen.getByText("Dispensada")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("todas feitas", () => {
    mockBackend(identityRoutes(semPermissao));
    renderApp(
      <TarefasCard
        solicitacao="s1"
        tarefas={[tarefa({ estado: "concluida" })]}
        podeAgir
        onMudou={() => {}}
      />,
    );
    expect(screen.getByText("Todas as tarefas foram feitas.")).toBeInTheDocument();
  });
});
