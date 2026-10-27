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
  it("envia a nota para o atendimento e não guarda no navegador", async () => {
    const user = userEvent.setup();
    const onAvaliado = vi.fn();
    const api = mockBackend({
      "POST v1/atendimento/solicitacoes/sol-csat-1/avaliacao": { status: 201, data: { nota: 5 } },
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
    await user.click(screen.getByRole("button", { name: /5 estrelas: Excelente/i }));
    await user.type(
      screen.getByLabelText("Comentário adicional (opcional)"),
      "Atendimento impecável",
    );
    await user.click(screen.getByRole("button", { name: /Enviar avaliação/i }));

    await waitFor(() => {
      expect(api.to("POST v1/atendimento/solicitacoes/sol-csat-1/avaliacao")).toHaveLength(1);
    });
    const enviado = api.to("POST v1/atendimento/solicitacoes/sol-csat-1/avaliacao")[0]!
      .body as Record<string, unknown>;
    expect(enviado.nota).toBe(5);
    expect(enviado.facilidade).toBeNull();
    expect(enviado.comentario).toBe("Atendimento impecável");
    expect(onAvaliado).toHaveBeenCalled();
    expect(localStorage.length).toBe(0);
  });

  it("envia a facilidade quando a pessoa responde; clicar de novo desmarca", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      "POST v1/atendimento/solicitacoes/sol-csat-1/avaliacao": { status: 201, data: { nota: 4 } },
    });
    render(
      <AvaliacaoAtendimentoCard solicitacao={solicitacaoResolvida} solicitante tecnico={false} />,
    );
    await user.click(screen.getByRole("button", { name: "Facilidade 2: Difícil" }));
    await user.click(screen.getByRole("button", { name: "Facilidade 2: Difícil" }));
    expect(screen.queryByText("Difícil")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Facilidade 4: Fácil" }));
    expect(screen.getByText("Fácil")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /Enviar avaliação/i }));
    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/sol-csat-1/avaliacao")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/sol-csat-1/avaliacao")[0]!.body).toMatchObject({
      facilidade: 4,
    });
  });

  it("mostra a avaliação que veio do servidor", () => {
    render(
      <AvaliacaoAtendimentoCard
        solicitacao={solicitacaoResolvida}
        avaliacao={{
          id: "av-1",
          solicitacao_id: "sol-csat-1",
          autor_id: "user-1",
          autor_nome: "Maria Oliveira",
          nota: 5,
          facilidade: 5,
          comentario: "Atendimento impecável",
          created_at: "2026-10-21T16:00:00Z",
        }}
        solicitante={true}
        tecnico={false}
      />,
    );
    expect(screen.getByText("Obrigado por avaliar o atendimento!")).toBeInTheDocument();
    expect(screen.getByText(/5\/5 — Excelente \/ Muito satisfeito/)).toBeInTheDocument();
    expect(screen.getByText("Facilidade: 5/5 — Muito fácil")).toBeInTheDocument();
  });

  it("não exibe para solicitação ainda em aberto", () => {
    const aberta: Solicitacao = { ...solicitacaoResolvida, estado: "em_atendimento" };
    render(<AvaliacaoAtendimentoCard solicitacao={aberta} solicitante={true} tecnico={false} />);
    expect(screen.queryByTestId("card-csat-avaliacao")).not.toBeInTheDocument();
  });

  it("não exibe o formulário para o técnico enquanto não houver nota", () => {
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
