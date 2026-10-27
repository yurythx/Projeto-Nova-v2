import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";

import IntegracoesPage from "./page";

const conexao = {
  id: "c1",
  tipo: "glpi",
  nome: "GLPI da prefeitura",
  modo: "conector",
  url: "",
  perfil: "",
  fuso: "America/Sao_Paulo",
  tem_credenciais: false,
  tem_token: true,
  extraido_em: "2026-10-27T15:00:00Z",
  created_at: "2026-10-27T12:00:00Z",
  created_by: "admin",
};

describe("Integrações", () => {
  it("lista as conexões e cria uma direta com o endereço da API", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/integracoes/conexoes": {
        data: [
          conexao,
          { ...conexao, id: "c2", nome: "Outra", modo: "direto", extraido_em: undefined },
        ],
      },
      "POST v1/integracoes/conexoes": { data: conexao },
    });
    renderApp(<IntegracoesPage />);

    expect(await screen.findByRole("link", { name: "GLPI da prefeitura" })).toHaveAttribute(
      "href",
      "/integracoes/c1",
    );
    expect(screen.getByText(/pelo conector · extraído em/)).toBeInTheDocument();
    expect(screen.getByText(/direto · ainda não extraído/)).toBeInTheDocument();

    expect(screen.queryByLabelText(/Endereço da API/)).not.toBeInTheDocument();
    await user.type(screen.getByLabelText("Nome *"), "GLPI novo");
    await user.selectOptions(screen.getByLabelText("Como ler a origem"), "direto");
    await user.type(screen.getByLabelText(/Endereço da API/), "https://glpi.test/apirest.php");
    await user.click(screen.getByRole("button", { name: "Criar conexão" }));

    expect(await screen.findByText("Conexão criada")).toBeInTheDocument();
    expect(api.to("POST v1/integracoes/conexoes")[0]?.body).toEqual({
      tipo: "glpi",
      nome: "GLPI novo",
      modo: "direto",
      url: "https://glpi.test/apirest.php",
      perfil: "",
    });
  });
});
