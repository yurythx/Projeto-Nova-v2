import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { Solicitacao } from "@/lib/atendimento/atendimento";
import { identityRoutes, mockBackend, moduleStatus, renderApp } from "@/test/backend";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import { ContratoDeApoioCard } from "./ContratoDeApoioCard";

const sol: Solicitacao = {
  id: "s1",
  numero: 7,
  tipo: "incidente",
  estado: "aguardando_terceiro",
  titulo: "Link caiu",
  descricao: "",
  oferta_id: "o1",
  oferta_nome: "Rede",
  solicitante_id: "u1",
  solicitante_nome: "Rita",
  created_at: "2026-10-01T12:00:00Z",
  updated_at: "2026-10-01T12:00:00Z",
  impacto: 2,
  urgencia: 2,
};

const ligado = [moduleStatus("contratos")];

describe("ContratoDeApoioCard", () => {
  it("aponta o contrato com o prazo do fornecedor", async () => {
    const user = userEvent.setup();
    const onMudou = vi.fn();
    const api = mockBackend({
      ...identityRoutes({}, ligado),
      "GET v1/atendimento/contratos-de-apoio": {
        data: [
          { id: "k1", rotulo: "12/2026 — Link (Provedor)", fornecedor: "Provedor", prazo_horas: 8 },
          {
            id: "k2",
            rotulo: "Energia (Concessionária)",
            fornecedor: "Concessionária",
            prazo_horas: null,
          },
        ],
      },
      "PUT v1/atendimento/solicitacoes/s1/contrato": { data: {} },
    });
    renderApp(<ContratoDeApoioCard solicitacao={sol} finalizada={false} onMudou={onMudou} />);
    expect(await screen.findByText(/aponte o contrato/)).toBeInTheDocument();
    expect(
      await screen.findByRole("option", { name: "Energia (Concessionária) · sem prazo" }),
    ).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Contrato"), "k1");
    await user.click(screen.getByRole("button", { name: "Salvar contrato" }));
    await waitFor(() => expect(onMudou).toHaveBeenCalled());
    expect(api.to("PUT v1/atendimento/solicitacoes/s1/contrato")[0]!.body).toEqual({
      contrato_id: "k1",
    });
  });

  it("mostra o vinculado e permite tirar", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes({}, ligado),
      "GET v1/atendimento/contratos-de-apoio": { data: [] },
      "PUT v1/atendimento/solicitacoes/s1/contrato": { data: {} },
    });
    renderApp(
      <ContratoDeApoioCard
        solicitacao={{
          ...sol,
          contrato_apoio_id: "k1",
          contrato_apoio: "Link",
          contrato_prazo_horas: 8,
        }}
        finalizada={false}
        onMudou={() => {}}
      />,
    );
    expect(await screen.findByText("Link · o fornecedor tem 8 h")).toBeInTheDocument();
    await user.selectOptions(screen.getByLabelText("Contrato"), "");
    await user.click(screen.getByRole("button", { name: "Salvar contrato" }));
    await waitFor(() =>
      expect(api.to("PUT v1/atendimento/solicitacoes/s1/contrato")[0]?.body).toEqual({
        contrato_id: null,
      }),
    );
  });

  it("encerrado, só mostra; sem prazo diz", async () => {
    mockBackend(identityRoutes({}, ligado));
    renderApp(
      <ContratoDeApoioCard
        solicitacao={{ ...sol, contrato_apoio_id: "k2", contrato_apoio: "Energia" }}
        finalizada
        onMudou={() => {}}
      />,
    );
    expect(await screen.findByText("Energia · sem prazo no contrato")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Salvar contrato" })).not.toBeInTheDocument();
  });

  it("com o plug-in desligado e sem contrato, some", async () => {
    mockBackend(identityRoutes({}, [moduleStatus("contratos", { enabled: false })]));
    const { container } = renderApp(
      <ContratoDeApoioCard solicitacao={sol} finalizada={false} onMudou={() => {}} />,
    );
    await waitFor(() => expect(container.querySelector("h2")).toBeNull());
  });
});
