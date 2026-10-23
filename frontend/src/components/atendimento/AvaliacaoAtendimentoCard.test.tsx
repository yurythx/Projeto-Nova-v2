import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AvaliacaoAtendimentoCard } from "@/components/atendimento/AvaliacaoAtendimentoCard";
import type { Solicitacao } from "@/lib/atendimento/atendimento";
import { mockBackend } from "@/test/backend";

const solicitacaoResolvida: Solicitacao = {
  id: "sol-csat-1",
  numero: 88,
  tipo: "requisicao",
  estado: "resolvido",
  titulo: "Acesso à VPN corporativa",
  descricao: "Solicito permissão de acesso à rede interna",
  oferta_id: "of-1",
  oferta_nome: "Acesso e Redes",
  unidade_id: "un-1",
  solicitante_id: "user-1",
  solicitante_nome: "Maria Oliveira",
  impacto: 1,
  urgencia: 1,
  created_at: "2026-10-20T10:00:00Z",
  updated_at: "2026-10-21T15:00:00Z",
  resolvida_em: "2026-10-21T15:00:00Z",
};

beforeEach(() => {
  localStorage.clear();
});

describe("AvaliacaoAtendimentoCard (Pesquisa de Satisfação CSAT)", () => {
  it("renderiza pesquisa de satisfação e permite enviar avaliação com nota e comentário", async () => {
    const user = userEvent.setup();
    const onAvaliado = vi.fn();
    const api = mockBackend({
      "POST v1/atendimento/solicitacoes/sol-csat-1/comentarios": { status: 201, data: {} },
    });

    render(
      <AvaliacaoAtendimentoCard
        solicitacao={solicitacaoResolvida}
        solicitante={true}
        tecnico={false}
        onAvaliado={onAvaliado}
      />,
    );

    expect(screen.getByTestId("card-csat-avaliacao")).toBeInTheDocument();
    expect(screen.getByText("Pesquisa de Satisfação (CSAT)")).toBeInTheDocument();
    expect(screen.getByText(/Como você avalia a resolução deste chamado\?/i)).toBeInTheDocument();

    // Clica na 5ª estrela
    await user.click(screen.getByRole("button", { name: /5 estrelas: Excelente/i }));
    // Digita comentário
    await user.type(screen.getByLabelText("Comentário adicional (opcional)"), "Atendimento impecável e muito rápido!");

    // Envia avaliação
    await user.click(screen.getByRole("button", { name: /Enviar avaliação/i }));

    await waitFor(() => {
      expect(api.to("POST v1/atendimento/solicitacoes/sol-csat-1/comentarios")).toHaveLength(1);
    });

    const comentarioEnviado = api.to("POST v1/atendimento/solicitacoes/sol-csat-1/comentarios")[0]!.body as Record<string, unknown>;
    expect(comentarioEnviado.texto).toContain("[Pesquisa de Satisfação - CSAT]");
    expect(comentarioEnviado.texto).toContain("5/5 estrelas");
    expect(comentarioEnviado.texto).toContain("Atendimento impecável e muito rápido!");

    // Feedback exibido
    expect(await screen.findByText("Obrigado por avaliar o atendimento!")).toBeInTheDocument();
    expect(screen.getByText(/5\/5 — Excelente \/ Muito satisfeito/)).toBeInTheDocument();
  });

  it("não exibe para solicitação ainda em aberto", () => {
    const aberta: Solicitacao = { ...solicitacaoResolvida, estado: "em_atendimento" };
    render(
      <AvaliacaoAtendimentoCard
        solicitacao={aberta}
        solicitante={true}
        tecnico={false}
      />,
    );

    expect(screen.queryByTestId("card-csat-avaliacao")).not.toBeInTheDocument();
  });

  it("não exibe para técnico se o usuário ainda não avaliou", () => {
    render(
      <AvaliacaoAtendimentoCard
        solicitacao={solicitacaoResolvida}
        solicitante={false}
        tecnico={true}
      />,
    );

    expect(screen.queryByTestId("card-csat-avaliacao")).not.toBeInTheDocument();
  });
});
