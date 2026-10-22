import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import AtendimentoCadastroPage from "./cadastro/page";
import AtendimentoFilaPage from "./fila/page";
import AtendimentoLayout from "./layout";
import AtendimentoPortalPage from "./page";
import SolicitacaoPage from "./solicitacoes/[id]/page";

function perms(...p: string[]) {
  return { permissions: p, scopes: [{ perfil: "p", origem: "manual" as const, permissions: p }] };
}

describe("Atendimento — navegação", () => {
  it("as abas seguem as permissões", async () => {
    resetNavigation({}, "/atendimento/fila");
    mockBackend(identityRoutes(perms("atendimento:abrir", "atendimento:atender")));
    renderApp(
      <AtendimentoLayout>
        <p>conteúdo</p>
      </AtendimentoLayout>,
    );
    expect(await screen.findByRole("link", { name: "Fila da equipe" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Minhas solicitações" })).toHaveAttribute(
      "href",
      "/atendimento",
    );
    expect(screen.queryByRole("link", { name: "Ofertas e equipes" })).not.toBeInTheDocument();
    expect(screen.getByText("conteúdo")).toBeInTheDocument();
  });
});

describe("Atendimento — falha de leitura e nova tentativa", () => {
  const casos: { nome: string; rota: string; tela: ReactElement; perms: string[] }[] = [
    {
      nome: "catálogo",
      rota: "GET v1/atendimento/catalogo",
      tela: <AtendimentoPortalPage />,
      perms: ["atendimento:abrir"],
    },
    {
      nome: "minhas",
      rota: "GET v1/atendimento/minhas",
      tela: <AtendimentoPortalPage />,
      perms: ["atendimento:abrir"],
    },
    {
      nome: "fila",
      rota: "GET v1/atendimento/fila",
      tela: <AtendimentoFilaPage />,
      perms: ["atendimento:atender"],
    },
    {
      nome: "detalhe",
      rota: "GET v1/atendimento/solicitacoes/s1",
      tela: <SolicitacaoPage />,
      perms: ["atendimento:abrir"],
    },
    {
      nome: "ofertas",
      rota: "GET v1/atendimento/ofertas",
      tela: <AtendimentoCadastroPage />,
      perms: ["atendimento:manage"],
    },
    {
      nome: "equipes",
      rota: "GET v1/atendimento/equipes",
      tela: <AtendimentoCadastroPage />,
      perms: ["atendimento:manage"],
    },
  ];

  it.each(casos)("$nome", async ({ rota, tela, perms: p }) => {
    resetNavigation({ id: "s1" }, "/atendimento");
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms(...p)),
      "GET v1/atendimento/catalogo": { data: [] },
      "GET v1/atendimento/minhas": { data: [] },
      "GET v1/atendimento/fila": { data: [] },
      "GET v1/atendimento/fila/topicos": { data: { topicos: [] } },
      "GET v1/atendimento/ofertas": { data: [] },
      "GET v1/atendimento/equipes": { data: [] },
      [rota]: fail(500, "banco fora"),
    });
    renderApp(tela);
    await user.click(await screen.findByRole("button", { name: "Tentar novamente" }));
    await waitFor(() => expect(api.to(rota).length).toBeGreaterThan(1));
  });
});
