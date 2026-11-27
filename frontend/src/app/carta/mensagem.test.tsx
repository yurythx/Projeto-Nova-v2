import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { fail, mockBackend } from "@/test/backend";

import { CartaPublica } from "./CartaPublica";

const CONSULTA = {
  numero: "PREF202690201",
  status: "em_andamento",
  status_rotulo: "Em andamento",
  prioridade: "normal",
  nome: "Maria",
  servico: "Alvará",
  unidade: "PREFEITURA",
  criado_em: "2026-11-20T12:00:00Z",
  complemento: "",
  historico: [
    {
      id: "h1",
      acao: "mensagem_mesa",
      descricao: "Estamos analisando.",
      status: "em_andamento",
      created_at: "2026-11-21T12:00:00Z",
    },
  ],
};

async function consultar(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("tab", { name: /Consultar protocolo/ }));
  await user.type(screen.getByLabelText("Número do protocolo"), "PREF202690201");
  await user.click(screen.getByRole("button", { name: /Consultar/ }));
}

describe("Mensagem do cidadão à prefeitura", () => {
  it("mostra a mensagem da prefeitura e envia a do cidadão", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      "GET v1/protocolos/carta": { data: [] },
      "GET v1/protocolos/consulta/PREF202690201": { data: CONSULTA },
      "POST v1/protocolos/consulta/PREF202690201/mensagem": {
        data: {
          ...CONSULTA,
          historico: [
            ...CONSULTA.historico,
            {
              id: "h2",
              acao: "mensagem_cidadao",
              descricao: "Obrigado!",
              status: "em_andamento",
              created_at: "2026-11-22T12:00:00Z",
            },
          ],
        },
      },
    });
    render(<CartaPublica />);
    await consultar(user);
    expect(await screen.findByText("Estamos analisando.")).toBeInTheDocument();
    expect(screen.getByText("Mensagem da prefeitura")).toBeInTheDocument();
    const form = screen.getByRole("form", { name: "Mensagem à prefeitura" });
    expect(form).toBeInTheDocument();
    await user.type(screen.getByLabelText("CPF do pedido *"), "123.456.789-00");
    await user.type(screen.getByLabelText("Mensagem *"), " Obrigado! ");
    await user.click(screen.getByRole("button", { name: "Enviar mensagem" }));
    expect(await screen.findByText("Sua mensagem")).toBeInTheDocument();
    expect(api.to("POST v1/protocolos/consulta/PREF202690201/mensagem")[0]?.body).toEqual({
      cpf: "123.456.789-00",
      mensagem: "Obrigado!",
    });
  });

  it("mostra o erro do envio e esconde o formulário no pedido encerrado", async () => {
    const user = userEvent.setup();
    mockBackend({
      "GET v1/protocolos/carta": { data: [] },
      "GET v1/protocolos/consulta/PREF202690201": { data: CONSULTA },
      "POST v1/protocolos/consulta/PREF202690201/mensagem": fail(
        409,
        "aguarde a resposta da prefeitura antes de mandar outra mensagem",
      ),
    });
    render(<CartaPublica />);
    await consultar(user);
    await user.type(await screen.findByLabelText("CPF do pedido *"), "12345678900");
    await user.type(screen.getByLabelText("Mensagem *"), "oi");
    await user.click(screen.getByRole("button", { name: "Enviar mensagem" }));
    expect(await screen.findByText(/aguarde a resposta da prefeitura/)).toBeInTheDocument();
  });

  it("não oferece a mensagem no pedido concluído", async () => {
    const user = userEvent.setup();
    mockBackend({
      "GET v1/protocolos/carta": { data: [] },
      "GET v1/protocolos/consulta/PREF202690201": {
        data: { ...CONSULTA, status: "concluido", status_rotulo: "Concluído" },
      },
    });
    render(<CartaPublica />);
    await consultar(user);
    expect(await screen.findByText("Estamos analisando.")).toBeInTheDocument();
    expect(screen.queryByRole("form", { name: "Mensagem à prefeitura" })).not.toBeInTheDocument();
  });
});
