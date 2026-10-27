import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { AlertasDoMonitoramentoCard } from "@/components/atendimento/AlertasDoMonitoramentoCard";
import { fail, identityRoutes, mockBackend, moduleStatus, renderApp } from "@/test/backend";

const base = identityRoutes({ permissions: ["atendimento:manage"] }, [moduleStatus("atendimento")]);
const ofertas = [
  {
    id: "o1",
    nome: "Alertas da infraestrutura",
    descricao: "",
    tipo: "incidente" as const,
    ativa: true,
  },
  { id: "o2", nome: "Pedido de acesso", descricao: "", tipo: "requisicao" as const, ativa: true },
  { id: "o3", nome: "Antiga", descricao: "", tipo: "incidente" as const, ativa: false },
];

describe("AlertasDoMonitoramentoCard (ADR 067)", () => {
  it("só oferece incidentes ativos e grava a escolha", async () => {
    const backend = mockBackend({
      ...base,
      "GET v1/atendimento/eventos/configuracao": { data: { oferta_id: null } },
      "PUT v1/atendimento/eventos/configuracao": { data: { oferta_id: "o1" } },
    });
    renderApp(<AlertasDoMonitoramentoCard ofertas={ofertas} />);

    const select = await screen.findByLabelText("Oferta dos alertas");
    expect(screen.getByRole("option", { name: "Alertas da infraestrutura" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Pedido de acesso" })).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Antiga" })).not.toBeInTheDocument();
    await userEvent.selectOptions(select, "o1");
    await userEvent.click(screen.getByRole("button", { name: "Salvar" }));
    expect(backend.to("PUT v1/atendimento/eventos/configuracao")[0]!.body).toEqual({
      oferta_id: "o1",
    });
  });

  it("limpar a escolha desliga a abertura", async () => {
    const backend = mockBackend({
      ...base,
      "GET v1/atendimento/eventos/configuracao": { data: { oferta_id: "o1" } },
      "PUT v1/atendimento/eventos/configuracao": { data: { oferta_id: null } },
    });
    renderApp(<AlertasDoMonitoramentoCard ofertas={ofertas} />);
    await userEvent.selectOptions(await screen.findByLabelText("Oferta dos alertas"), "");
    await userEvent.click(screen.getByRole("button", { name: "Salvar" }));
    expect(backend.to("PUT v1/atendimento/eventos/configuracao")[0]!.body).toEqual({
      oferta_id: null,
    });
  });

  it("não aparece para quem gere só uma unidade", async () => {
    const backend = mockBackend({
      ...base,
      "GET v1/atendimento/eventos/configuracao": fail(
        403,
        "você não gerencia o atendimento desta unidade",
      ),
    });
    renderApp(<AlertasDoMonitoramentoCard ofertas={ofertas} />);
    await waitFor(() =>
      expect(backend.to("GET v1/atendimento/eventos/configuracao")).toHaveLength(1),
    );
    expect(screen.queryByText("Alertas do monitoramento")).not.toBeInTheDocument();
  });
});
