import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AprovacoesCard } from "@/components/atendimento/AprovacoesCard";
import { EtapasDeAprovacao } from "@/components/atendimento/EtapasDeAprovacao";
import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AprovacoesPage from "./page";

// A chefia aprova sem nenhuma permissão do atendimento (ADR 068).
const semPermissao = { permissions: [], scopes: [] };

const pedido = {
  id: "s1",
  numero: 42,
  tipo: "requisicao",
  estado: "aguardando_aprovacao",
  titulo: "Admitir Ana",
  descricao: "Matrícula 123",
  oferta_id: "o1",
  oferta_nome: "Admissão de servidor",
  solicitante_id: "u1",
  solicitante_nome: "Rita",
  created_at: "2026-10-01T12:00:00Z",
  updated_at: "2026-10-01T12:00:00Z",
  impacto: 3,
  urgencia: 3,
};

describe("Aprovações", () => {
  it("aprova o pedido que espera a sua etapa", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(semPermissao),
      "GET v1/atendimento/aprovacoes": { data: [pedido] },
      "POST v1/atendimento/solicitacoes/s1/aprovacao": { data: { ...pedido, estado: "aberto" } },
    });
    renderApp(<AprovacoesPage />);
    expect(await screen.findByRole("heading", { name: "#42 Admitir Ana" })).toBeInTheDocument();
    expect(screen.getByText("Matrícula 123")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Aprovar" }));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/aprovacao")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/aprovacao")[0]!.body).toEqual({
      aprovar: true,
      motivo: "",
    });
  });

  it("recusa só com motivo e pode voltar atrás", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(semPermissao),
      "GET v1/atendimento/aprovacoes": { data: [pedido] },
      "POST v1/atendimento/solicitacoes/s1/aprovacao": {
        data: { ...pedido, estado: "cancelado" },
      },
    });
    renderApp(<AprovacoesPage />);
    await user.click(await screen.findByRole("button", { name: "Recusar" }));
    await user.click(screen.getByRole("button", { name: "Voltar" }));
    await user.click(screen.getByRole("button", { name: "Recusar" }));
    const confirmar = screen.getByRole("button", { name: "Confirmar recusa" });
    expect(confirmar).toBeDisabled();
    await user.type(screen.getByLabelText("Motivo da recusa"), "  Sem vaga no quadro ");
    await user.click(confirmar);
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/aprovacao")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/aprovacao")[0]!.body).toEqual({
      aprovar: false,
      motivo: "Sem vaga no quadro",
    });
    await waitFor(() =>
      expect(screen.queryByLabelText("Motivo da recusa")).not.toBeInTheDocument(),
    );
  });

  it("sem nada pendente, diz que não há o que aprovar", async () => {
    mockBackend({ ...identityRoutes(semPermissao), "GET v1/atendimento/aprovacoes": { data: [] } });
    renderApp(<AprovacoesPage />);
    expect(await screen.findByText("Nada esperando a sua aprovação")).toBeInTheDocument();
  });

  it("falha de leitura oferece nova tentativa", async () => {
    mockBackend({ ...identityRoutes(semPermissao), "GET v1/atendimento/aprovacoes": fail(500) });
    renderApp(<AprovacoesPage />);
    expect(await screen.findByRole("button", { name: /tentar/i })).toBeInTheDocument();
  });
});

describe("AprovacoesCard", () => {
  it("some quando o pedido não tem aprovação", () => {
    const { container } = render(<AprovacoesCard aprovacoes={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("mostra a etapa que falta e as decisões com o motivo", () => {
    render(
      <AprovacoesCard
        etapaAtual="Secretaria"
        aprovacoes={[
          {
            id: "a1",
            etapa: 0,
            etapa_nome: "Chefia imediata",
            aprovador_nome: "Beto",
            decisao: "aprovada",
            created_at: "2026-10-01T12:00:00Z",
          },
          {
            id: "a2",
            etapa: 1,
            etapa_nome: "Secretaria",
            decisao: "recusada",
            motivo: "Sem vaga",
            created_at: "2026-10-02T12:00:00Z",
          },
        ]}
      />,
    );
    expect(screen.getByRole("status")).toHaveTextContent("Aguardando a etapa Secretaria");
    expect(screen.getByText("Aprovada")).toBeInTheDocument();
    expect(screen.getByText(/Chefia imediata · Beto/)).toBeInTheDocument();
    expect(screen.getByText("Recusada")).toBeInTheDocument();
    expect(screen.getByText("Sem vaga")).toBeInTheDocument();
  });
});

describe("EtapasDeAprovacao", () => {
  const oferta = {
    id: "o1",
    nome: "Admissão",
    descricao: "",
    tipo: "requisicao" as const,
    unidade_id: "un1",
    ativa: true,
  };

  it("monta as etapas com os aprovadores e grava", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(semPermissao),
      "GET v1/atendimento/ofertas/o1/etapas": {
        data: [{ ordem: 0, nome: "Chefia", aprovadores: ["p1"] }],
      },
      "GET v1/atendimento/candidatos*": {
        data: [
          { id: "p1", nome: "Beto" },
          { id: "p2", nome: "Carla" },
        ],
      },
      "PUT v1/atendimento/ofertas/o1/etapas": { data: [] },
    });
    const onFechar = vi.fn();
    renderApp(<EtapasDeAprovacao oferta={oferta} onFechar={onFechar} />);
    expect(await screen.findByDisplayValue("Chefia")).toBeInTheDocument();
    await screen.findAllByRole("checkbox", { name: "Carla" });
    await user.click(screen.getByRole("button", { name: /Acrescentar etapa/ }));
    await user.type(screen.getAllByLabelText("Nome da etapa")[1]!, "Secretaria");
    await user.click(screen.getAllByRole("checkbox", { name: "Carla" })[1]!);
    // Desmarca e marca de novo: o aprovador entra uma vez só.
    await user.click(screen.getAllByRole("checkbox", { name: "Beto" })[0]!);
    await user.click(screen.getAllByRole("checkbox", { name: "Beto" })[0]!);
    await user.click(screen.getByRole("button", { name: "Salvar etapas" }));
    await waitFor(() => expect(api.to("PUT v1/atendimento/ofertas/o1/etapas")).toHaveLength(1));
    expect(api.to("PUT v1/atendimento/ofertas/o1/etapas")[0]!.body).toEqual({
      etapas: [
        { nome: "Chefia", aprovadores: ["p1"] },
        { nome: "Secretaria", aprovadores: ["p2"] },
      ],
    });
    await user.click(screen.getByRole("button", { name: "Fechar" }));
    expect(onFechar).toHaveBeenCalled();
  });

  it("tirar todas as etapas deixa a oferta sem aprovação", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(semPermissao),
      "GET v1/atendimento/ofertas/o1/etapas": {
        data: [{ ordem: 0, nome: "Chefia", aprovadores: ["p1"] }],
      },
      "GET v1/atendimento/candidatos*": { data: [] },
      "PUT v1/atendimento/ofertas/o1/etapas": { data: [] },
    });
    renderApp(<EtapasDeAprovacao oferta={oferta} onFechar={() => {}} />);
    expect(await screen.findByText("Ninguém lotado na unidade da oferta.")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Tirar a etapa 1/ }));
    await user.click(screen.getByRole("button", { name: "Salvar etapas" }));
    await waitFor(() => expect(api.to("PUT v1/atendimento/ofertas/o1/etapas")).toHaveLength(1));
    expect(api.to("PUT v1/atendimento/ofertas/o1/etapas")[0]!.body).toEqual({ etapas: [] });
  });
});
