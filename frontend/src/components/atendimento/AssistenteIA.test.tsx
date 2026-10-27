import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AssistenteIA } from "@/components/atendimento/AssistenteIA";
import { taxaDeAceite } from "@/lib/atendimento/ia";
import { fail, identityRoutes, mockBackend, moduleStatus, renderApp } from "@/test/backend";

const base = identityRoutes({ permissions: ["atendimento:atender"] }, [
  moduleStatus("atendimento"),
]);

const classificacao = {
  id: "sug-1",
  solicitacao_id: "sol-1",
  tipo: "classificacao",
  conexao: "IA local",
  classificacao: { impacto: 3, urgencia: 2, motivo: "O setor inteiro está sem e-mail." },
  created_at: "2026-10-27T10:00:00Z",
};

const resumo = {
  id: "sug-2",
  solicitacao_id: "sol-1",
  tipo: "resumo",
  conexao: "IA local",
  resumo: "Sem e-mail desde cedo.\nFalta testar o certificado.",
  created_at: "2026-10-27T10:01:00Z",
};

describe("AssistenteIA (ADR 065)", () => {
  it("não aparece sem IA configurada", async () => {
    const backend = mockBackend({
      ...base,
      "GET v1/atendimento/ia": { data: { disponivel: false } },
    });
    renderApp(<AssistenteIA solicitacaoId="sol-1" editavel onAplicarClassificacao={vi.fn()} />);
    await waitFor(() => expect(backend.to("GET v1/atendimento/ia")).toHaveLength(1));
    expect(screen.queryByTestId("assistente-ia")).not.toBeInTheDocument();
  });

  it("aplica a classificação pela rota de sempre e só então registra o aceite", async () => {
    const user = userEvent.setup();
    const ordem: string[] = [];
    const aplicar = vi.fn(async () => {
      ordem.push("classificou");
      return true;
    });
    const backend = mockBackend({
      ...base,
      "GET v1/atendimento/ia": { data: { disponivel: true } },
      "POST v1/atendimento/solicitacoes/sol-1/ia/classificacao": {
        status: 201,
        data: classificacao,
      },
      "POST v1/atendimento/ia/sugestoes/sug-1/decisao": () => {
        ordem.push("decidiu");
        return { data: { ...classificacao, decisao: "aceita" } };
      },
    });
    renderApp(<AssistenteIA solicitacaoId="sol-1" editavel onAplicarClassificacao={aplicar} />);

    await user.click(await screen.findByRole("button", { name: "Sugerir classificação" }));
    expect(await screen.findByText("Impacto alto · Urgência média")).toBeInTheDocument();
    expect(screen.getByText("O setor inteiro está sem e-mail.")).toBeInTheDocument();
    expect(screen.getByText("Sugerido por: IA local")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Aplicar esta classificação" }));
    expect(await screen.findByText("Você aceitou esta sugestão.")).toBeInTheDocument();
    expect(aplicar).toHaveBeenCalledWith(3, 2);
    expect(ordem).toEqual(["classificou", "decidiu"]);
    expect(backend.to("POST v1/atendimento/ia/sugestoes/sug-1/decisao")[0]?.body).toEqual({
      aceita: true,
    });
  });

  it("não registra aceite quando a classificação falha", async () => {
    const user = userEvent.setup();
    const backend = mockBackend({
      ...base,
      "GET v1/atendimento/ia": { data: { disponivel: true } },
      "POST v1/atendimento/solicitacoes/sol-1/ia/classificacao": {
        status: 201,
        data: classificacao,
      },
    });
    renderApp(
      <AssistenteIA solicitacaoId="sol-1" editavel onAplicarClassificacao={async () => false} />,
    );
    await user.click(await screen.findByRole("button", { name: "Sugerir classificação" }));
    await user.click(await screen.findByRole("button", { name: "Aplicar esta classificação" }));
    expect(backend.to("POST v1/atendimento/ia/sugestoes/sug-1/decisao")).toHaveLength(0);
  });

  it("descarta a sugestão e avalia o resumo", async () => {
    const user = userEvent.setup();
    const backend = mockBackend({
      ...base,
      "GET v1/atendimento/ia": { data: { disponivel: true } },
      "POST v1/atendimento/solicitacoes/sol-1/ia/classificacao": {
        status: 201,
        data: classificacao,
      },
      "POST v1/atendimento/solicitacoes/sol-1/ia/resumo": { status: 201, data: resumo },
      "POST v1/atendimento/ia/sugestoes/sug-1/decisao": {
        data: { ...classificacao, decisao: "recusada" },
      },
      "POST v1/atendimento/ia/sugestoes/sug-2/decisao": { data: { ...resumo, decisao: "aceita" } },
    });
    renderApp(<AssistenteIA solicitacaoId="sol-1" editavel onAplicarClassificacao={vi.fn()} />);

    await user.click(await screen.findByRole("button", { name: "Sugerir classificação" }));
    await user.click(await screen.findByRole("button", { name: "Descartar" }));
    expect(await screen.findByText("Você descartou esta sugestão.")).toBeInTheDocument();
    expect(backend.to("POST v1/atendimento/ia/sugestoes/sug-1/decisao")[0]?.body).toEqual({
      aceita: false,
    });

    await user.click(screen.getByRole("button", { name: "Resumir o chamado" }));
    expect(await screen.findByText(/Falta testar o certificado/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Ajudou" }));
    expect(await screen.findByText("Obrigado: o resumo ajudou.")).toBeInTheDocument();
  });

  it("em chamado encerrado só oferece o resumo, e a falha da IA vira aviso", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...base,
      "GET v1/atendimento/ia": { data: { disponivel: true } },
      "POST v1/atendimento/solicitacoes/sol-1/ia/resumo": fail(
        503,
        "a IA não respondeu agora; tente de novo ou siga sem ela",
      ),
    });
    renderApp(
      <AssistenteIA solicitacaoId="sol-1" editavel={false} onAplicarClassificacao={vi.fn()} />,
    );
    await user.click(await screen.findByRole("button", { name: "Resumir o chamado" }));
    expect(screen.queryByRole("button", { name: "Sugerir classificação" })).not.toBeInTheDocument();
    expect(
      await screen.findByText("a IA não respondeu agora; tente de novo ou siga sem ela"),
    ).toBeInTheDocument();
  });
});

describe("taxaDeAceite", () => {
  it("fica vazia sem decisão e conta só as decididas", () => {
    expect(
      taxaDeAceite({ tipo: "resumo", total: 4, aceitas: 0, recusadas: 0, pendentes: 4 }),
    ).toBeNull();
    expect(
      taxaDeAceite({ tipo: "classificacao", total: 5, aceitas: 2, recusadas: 1, pendentes: 2 }),
    ).toBe(67);
  });
});
