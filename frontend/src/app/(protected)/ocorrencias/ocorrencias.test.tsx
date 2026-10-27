import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, moduleStatus, renderApp } from "@/test/backend";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import OcorrenciasPage from "./page";

const base = identityRoutes({ permissions: ["ocorrencias:manage"] }, [moduleStatus("ocorrencias")]);

const aberta = {
  id: "oc-1",
  titulo: "Protocolo fora do ar",
  impacto: "fora",
  estado: "investigando",
  servicos: ["Protocolo"],
  coordenador_nome: "Maria",
  chamado_id: "sol-9",
  iniciada_em: "2020-01-01T10:00:00Z",
  updated_at: "2020-01-01T10:00:00Z",
  atualizacoes: [
    {
      id: "a1",
      estado: "investigando",
      impacto: "fora",
      texto: "Estamos investigando.",
      autor_nome: "Maria",
      created_at: "2020-01-01T10:00:00Z",
    },
  ],
};

describe("Ocorrências (painel)", () => {
  it("declara com os serviços separados e abre a nova", async () => {
    const backend = mockBackend({
      ...base,
      "GET v1/ocorrencias": { data: [] },
      "POST v1/ocorrencias": { status: 201, data: aberta },
      "GET v1/ocorrencias/oc-1": { data: aberta },
    });
    renderApp(<OcorrenciasPage />);

    expect(await screen.findByText("Nenhuma ocorrência registrada")).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText("Título público *"), "Protocolo fora do ar");
    await userEvent.type(
      screen.getByLabelText("Serviços afetados (separe por vírgula) *"),
      "Protocolo, SEI , ",
    );
    await userEvent.type(
      screen.getByLabelText("Primeira mensagem ao público *"),
      "Estamos investigando.",
    );
    await userEvent.click(screen.getByRole("button", { name: "Declarar ocorrência" }));

    expect(backend.to("POST v1/ocorrencias")[0]!.body).toEqual({
      titulo: "Protocolo fora do ar",
      servicos: ["Protocolo", "SEI"],
      impacto: "fora",
      texto: "Estamos investigando.",
    });
    expect(await screen.findByRole("list", { name: "Linha do tempo" })).toHaveTextContent(
      "Estamos investigando.",
    );
    expect(screen.getByRole("link", { name: "chamado principal" })).toHaveAttribute(
      "href",
      "/atendimento/solicitacoes/sol-9",
    );
  });

  it("avisa quem está sem notícia e publica a atualização", async () => {
    const backend = mockBackend({
      ...base,
      "GET v1/ocorrencias": { data: [aberta] },
      "GET v1/ocorrencias/oc-1": { data: aberta },
      "POST v1/ocorrencias/oc-1/atualizacoes": { data: { ...aberta, estado: "identificado" } },
    });
    renderApp(<OcorrenciasPage />);

    const lista = await screen.findByRole("list", { name: "Ocorrências" });
    expect(within(lista).getByText(/Sem notícia ao público há/)).toBeInTheDocument();
    await userEvent.click(within(lista).getByRole("button", { name: /Protocolo fora do ar/ }));

    await userEvent.selectOptions(await screen.findByLabelText("Estado"), "identificado");
    await userEvent.type(screen.getByLabelText("Mensagem ao público *"), "Certificado vencido.");
    await userEvent.click(screen.getByRole("button", { name: "Publicar" }));
    expect(backend.to("POST v1/ocorrencias/oc-1/atualizacoes")[0]!.body).toEqual({
      estado: "identificado",
      impacto: "fora",
      texto: "Certificado vencido.",
    });
  });

  it("depois de resolvida, oferece a revisão interna em vez de atualização", async () => {
    const resolvida = {
      ...aberta,
      estado: "resolvido",
      resolvida_em: "2020-01-01T12:00:00Z",
      revisao: "",
    };
    const backend = mockBackend({
      ...base,
      "GET v1/ocorrencias": { data: [resolvida] },
      "GET v1/ocorrencias/oc-1": { data: resolvida },
      "PUT v1/ocorrencias/oc-1/revisao": { data: { ...resolvida, revisao: "Renovar antes." } },
    });
    renderApp(<OcorrenciasPage />);

    const lista = await screen.findByRole("list", { name: "Ocorrências" });
    expect(within(lista).queryByText(/Sem notícia/)).not.toBeInTheDocument();
    await userEvent.click(within(lista).getByRole("button", { name: /Protocolo fora do ar/ }));
    expect(screen.queryByRole("button", { name: "Publicar" })).not.toBeInTheDocument();
    await userEvent.type(await screen.findByLabelText(/Causa, o que funcionou/), "Renovar antes.");
    await userEvent.click(screen.getByRole("button", { name: "Gravar revisão" }));
    expect(backend.to("PUT v1/ocorrencias/oc-1/revisao")[0]!.body).toEqual({
      texto: "Renovar antes.",
    });
  });
});
