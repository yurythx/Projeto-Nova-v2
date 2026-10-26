import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import RegrasPage from "./page";

const perms = {
  permissions: ["atendimento:manage"],
  scopes: [{ perfil: "p", origem: "manual" as const, permissions: ["atendimento:manage"] }],
};

describe("Triagem", () => {
  it("grava a regra com a palavra e a prioridade", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms),
      "GET v1/atendimento/regras": { data: [] },
      "GET v1/atendimento/recorrencias": { data: [] },
      "GET v1/atendimento/ofertas": {
        data: [{ id: "o1", nome: "E-mail", tipo: "incidente", ativa: true }],
      },
      "GET v1/atendimento/equipes": { data: [{ id: "e1", nome: "N1" }] },
      "GET v1/iam/org-tree": { data: [] },
      "POST v1/atendimento/regras": { status: 201, data: { id: "r1", nome: "Senha", ordem: 10 } },
    });
    renderApp(<RegrasPage />);
    expect(await screen.findByText("Nenhuma regra")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Nome *"), "Senha");
    await user.type(screen.getByLabelText("Palavra no título"), "senha");
    await user.selectOptions(screen.getByLabelText("Prioridade"), "critica");
    await user.click(screen.getByRole("button", { name: "Gravar regra" }));
    await waitFor(() => expect(api.to("POST v1/atendimento/regras")).toHaveLength(1));
    expect(api.to("POST v1/atendimento/regras")[0]!.body).toMatchObject({
      nome: "Senha",
      palavra: "senha",
      prioridade: "critica",
      ordem: 10,
    });
  });

  it("mostra a resposta já gravada da oferta e grava de novo", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(perms),
      "GET v1/atendimento/regras": { data: [] },
      "GET v1/atendimento/recorrencias": { data: [] },
      "GET v1/atendimento/ofertas": {
        data: [{ id: "o1", nome: "E-mail", tipo: "incidente", ativa: true }],
      },
      "GET v1/atendimento/equipes": { data: [] },
      "GET v1/iam/org-tree": { data: [] },
      "GET v1/atendimento/ofertas/o1/respostas": {
        data: { publico: "Reinicie o cliente.", interna: "Verificar VPN." },
      },
      "PUT v1/atendimento/ofertas/o1/respostas": {
        data: { publico: "Reinicie o cliente.", interna: "Verificar VPN." },
      },
    });
    renderApp(<RegrasPage />);
    await user.selectOptions(await screen.findByLabelText("Oferta da resposta *"), "o1");
    expect(await screen.findByLabelText("Resposta pública")).toHaveValue("Reinicie o cliente.");
    expect(screen.getByLabelText("Nota interna")).toHaveValue("Verificar VPN.");
    await user.click(screen.getByRole("button", { name: "Gravar resposta" }));
    await waitFor(() => expect(api.to("PUT v1/atendimento/ofertas/o1/respostas")).toHaveLength(1));
    expect(api.to("PUT v1/atendimento/ofertas/o1/respostas")[0]!.body).toEqual({
      publico: "Reinicie o cliente.",
      interna: "Verificar VPN.",
    });
  });
});
