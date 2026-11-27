import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { fail, mockBackend } from "@/test/backend";

import { AbrirPedido } from "./[id]/AbrirPedido";
import { CartaPublica } from "./CartaPublica";

const ITEM = {
  servico: {
    id: "s9",
    nome: "Vacinação em domicílio",
    descricao: "",
    exige_agendamento: true,
    prazo_dias: 0,
    taxa_centavos: 1500,
    requisitos: [],
    documentos: [],
    detalhes: {
      quem_pode: "Pessoas acamadas",
      passo_a_passo: "1. Peça\n2. Aguarde a visita",
      contato: "66 3411-0000",
      tempo_estimado: "até 5 dias",
      descricao_taxa: "Paga na visita",
    },
  },
  formulario: { titulo: "Vacina", instrucao: "", campos: [] },
};

describe("Detalhes do serviço e horários livres", () => {
  it("mostra os detalhes da carta", async () => {
    mockBackend({ "GET v1/protocolos/carta/s9": { data: ITEM } });
    render(<AbrirPedido id="s9" />);
    expect(await screen.findByText("Pessoas acamadas")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Passo a passo" })).toBeInTheDocument();
    expect(screen.getByText("até 5 dias")).toBeInTheDocument();
    expect(screen.getByText("66 3411-0000")).toBeInTheDocument();
    expect(screen.getByText("Paga na visita")).toBeInTheDocument();
  });

  it("avisa quando o dia não tem horário e quando a leitura falha", async () => {
    const user = userEvent.setup();
    mockBackend({
      "GET v1/protocolos/carta/s9": { data: ITEM },
      "GET v1/protocolos/carta/s9/horarios": (req) =>
        req.query.get("data") === "2026-12-25" ? { data: [] } : fail(503, "Agenda fora do ar"),
    });
    render(<AbrirPedido id="s9" pedirInicial />);
    const hora = await screen.findByLabelText("Horário *");
    expect(hora).toBeDisabled();
    expect(screen.getByRole("option", { name: "Escolha a data primeiro" })).toBeInTheDocument();
    await user.type(screen.getByLabelText("Data do atendimento *"), "2026-12-25");
    expect(await screen.findByText(/Não há horário livre nessa data/)).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Sem horário livre neste dia" })).toBeInTheDocument();
    await user.clear(screen.getByLabelText("Data do atendimento *"));
    await user.type(screen.getByLabelText("Data do atendimento *"), "2026-12-28");
    expect(await screen.findByRole("alert")).toHaveTextContent("Agenda fora do ar");
  });

  it("a consulta mostra o processo SEI e o prazo da complementação", async () => {
    const user = userEvent.setup();
    mockBackend({
      "GET v1/protocolos/carta": { data: [] },
      "GET v1/protocolos/consulta/PREF202690201": {
        data: {
          numero: "PREF202690201",
          status: "em_complementacao",
          status_rotulo: "Em complementação",
          prioridade: "normal",
          nome: "Maria",
          servico: "Alvará",
          unidade: "PREFEITURA",
          criado_em: "2026-11-20T12:00:00Z",
          complemento: "envie o RG",
          complemento_prazo: "2026-12-03T03:59:00Z",
          processo_sei: "12345.000001/2026-11",
          historico: [],
        },
      },
    });
    render(<CartaPublica />);
    await user.click(screen.getByRole("tab", { name: /Consultar protocolo/ }));
    await user.type(screen.getByLabelText("Número do protocolo"), "PREF202690201");
    await user.click(screen.getByRole("button", { name: /Consultar/ }));
    expect(await screen.findByText("12345.000001/2026-11")).toBeInTheDocument();
    expect(screen.getByText(/Responda até/)).toBeInTheDocument();
  });
});
