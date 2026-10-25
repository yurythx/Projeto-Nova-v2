import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, moduleStatus, renderApp } from "@/test/backend";

import { SegredoVinculadoCard } from "./SegredoVinculadoCard";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

describe("Senha de serviço no chamado", () => {
  it("vincula o id e só revela depois da senha de acesso", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes({}, [moduleStatus("cofre", { enabled: true, name: "Cofre" })]),
      "GET v1/cofre/segredos": { data: [{ id: "s1", nome: "Wifi", usuario: "rede" }] },
      "PUT v1/atendimento/solicitacoes/t1/segredo": { data: {} },
      "POST v1/cofre/segredos/s1/revelar": { data: { id: "s1", nome: "Wifi", usuario: "rede", senha: "chave-da-rede", notas: "" } },
    });
    renderApp(
      <SegredoVinculadoCard solicitacaoId="t1" segredoId="s1" tecnico finalizada={false} onMudou={() => {}} />,
    );

    expect(await screen.findByText("Wifi")).toBeInTheDocument();
    expect(screen.queryByDisplayValue("chave-da-rede")).not.toBeInTheDocument();

    await user.selectOptions(screen.getByLabelText("Segredo"), "s1");
    await user.click(screen.getByRole("button", { name: "Vincular segredo" }));
    await waitFor(() => expect(api.to("PUT v1/atendimento/solicitacoes/t1/segredo")).toHaveLength(1));
    expect(api.to("PUT v1/atendimento/solicitacoes/t1/segredo")[0]!.body).toEqual({ segredo_id: "s1" });

    await user.type(screen.getByLabelText("Sua senha de acesso"), "acesso");
    await user.click(screen.getByRole("button", { name: "Revelar senha do chamado" }));
    expect(await screen.findByDisplayValue("chave-da-rede")).toBeInTheDocument();
    expect(api.to("POST v1/cofre/segredos/s1/revelar")[0]!.body).toEqual({ senha_de_acesso: "acesso" });
    expect(screen.queryByDisplayValue("acesso")).not.toBeInTheDocument();
  });
});
