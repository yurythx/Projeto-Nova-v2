import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { ApontamentosCard } from "@/components/atendimento/ApontamentosCard";
import { identityRoutes, mockBackend, renderApp } from "@/test/backend";

describe("ApontamentosCard (Timesheet no Atendimento)", () => {
  it("renderiza lista de apontamentos e total de horas", async () => {
    mockBackend({
      ...identityRoutes({ permissions: ["atendimento:atender"] }),
      "GET v1/atendimento/solicitacoes/s1/apontamentos": {
        data: {
          total_horas: 3.5,
          apontamentos: [
            {
              id: "ap1",
              solicitacao_id: "s1",
              usuario_id: "u1",
              usuario_nome: "Carlos Técnico",
              data_apontamento: "2026-10-22",
              horas: 3.5,
              descricao: "Diagnóstico e reparo de cabeamento",
              created_at: "2026-10-22T10:00:00Z",
            },
          ],
        },
      },
    });

    renderApp(<ApontamentosCard solicitacaoId="s1" tecnico={true} finalizada={false} />);

    expect(await screen.findByText("3.5h total")).toBeInTheDocument();
    expect(screen.getByText("Carlos Técnico")).toBeInTheDocument();
    expect(screen.getByText("Diagnóstico e reparo de cabeamento")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Apontar horas" })).toBeInTheDocument();
  });

  it("permite ao técnico registrar novas horas via modal", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes({ permissions: ["atendimento:atender"] }),
      "GET v1/atendimento/solicitacoes/s1/apontamentos": {
        data: {
          total_horas: 0,
          apontamentos: [],
        },
      },
      "POST v1/atendimento/solicitacoes/s1/apontamentos": {
        data: {
          id: "ap2",
          solicitacao_id: "s1",
          usuario_id: "u1",
          usuario_nome: "Carlos Técnico",
          data_apontamento: "2026-10-22",
          horas: 2,
          descricao: "Configuração remota de perfil",
          created_at: "2026-10-22T11:00:00Z",
        },
      },
    });

    renderApp(<ApontamentosCard solicitacaoId="s1" tecnico={true} finalizada={false} />);

    expect(
      await screen.findByText("Nenhum apontamento de esforço registrado para esta solicitação."),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Apontar horas" }));
    expect(screen.getByText("Lançar Apontamento de Horas")).toBeInTheDocument();

    const horasInput = screen.getByLabelText(/Horas Trabalhadas/);
    await user.clear(horasInput);
    await user.type(horasInput, "2");

    const descInput = screen.getByLabelText("Descrição das Atividades");
    await user.type(descInput, "Configuração remota de perfil");

    await user.click(screen.getByRole("button", { name: "Salvar Horas" }));

    await waitFor(() =>
      expect(api.to("POST v1/atendimento/solicitacoes/s1/apontamentos")).toHaveLength(1),
    );
    expect(api.to("POST v1/atendimento/solicitacoes/s1/apontamentos")[0]!.body).toMatchObject({
      horas: 2,
      descricao: "Configuração remota de perfil",
    });
  });
});
