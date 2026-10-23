import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { AtivoVinculadoCard } from "@/components/atendimento/AtivoVinculadoCard";
import { identityRoutes, mockBackend, moduleStatus, renderApp } from "@/test/backend";

const ativoMock = {
  id: "at-1",
  nome: "Workstation Dell OptiPlex",
  tipo: "hardware" as const,
  estado: "em_uso" as const,
  identificador: "PAT-2024-8841",
  descricao: "Computador do setor de faturamento",
};

describe("AtivoVinculadoCard (CMDB no Atendimento)", () => {
  it("renderiza opções de ativos e permite associar equipamento ao chamado", async () => {
    const user = userEvent.setup();
    mockBackend({
      ...identityRoutes({ permissions: ["ativos:read", "atendimento:atender"] }, [
        moduleStatus("ativos"),
      ]),
      "GET v1/ativos": {
        data: [ativoMock],
      },
    });

    renderApp(<AtivoVinculadoCard solicitacaoId="sol-10" tecnico={true} finalizada={false} />);

    expect(
      await screen.findByText("Nenhum ativo ou equipamento vinculado diretamente a esta solicitação."),
    ).toBeInTheDocument();

    const select = screen.getByLabelText("Vincular equipamento ou sistema");
    await user.selectOptions(select, "at-1");

    await user.click(screen.getByRole("button", { name: "Associar ao chamado" }));

    expect(await screen.findByText("Workstation Dell OptiPlex")).toBeInTheDocument();
    expect(screen.getByText("[PAT-2024-8841]")).toBeInTheDocument();
    expect(screen.getByText("Equipamento")).toBeInTheDocument();
    expect(screen.getByText("Em uso")).toBeInTheDocument();
  });
});
