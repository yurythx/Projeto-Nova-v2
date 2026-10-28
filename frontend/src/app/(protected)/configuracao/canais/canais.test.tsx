import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import CanaisPage from "./page";

const gravado = {
  ativo: false,
  host: "smtp.x.gov.br",
  porta: 587,
  usuario: "central",
  tem_senha: true,
  remetente: "Central <at@x.gov.br>",
  origem: "tela",
  updated_at: "2026-10-27T12:00:00Z",
  updated_by: "admin",
};

const rotas = (extra: Record<string, unknown> = {}) => ({
  ...identityRoutes({ permissions: ["canais:manage"] }),
  "GET v1/canais/email": { data: gravado },
  "PUT v1/canais/email": { data: { ...gravado, ativo: true } },
  ...extra,
});

beforeEach(() => resetNavigation({}, "/configuracao/canais"));

describe("Configurações — canais de aviso", () => {
  it("liga o e-mail mantendo a senha e testa o envio", async () => {
    const user = userEvent.setup();
    let teste = 0;
    const api = mockBackend(
      rotas({
        "POST v1/canais/email/testar": () =>
          ++teste === 1
            ? { data: { ok: true } }
            : { data: { ok: false, erro: "535 credenciais recusadas" } },
      }),
    );
    renderApp(<CanaisPage />);

    const form = await screen.findByRole("form", { name: "Servidor de e-mail" });
    expect(screen.getByText("Desligado")).toBeInTheDocument();
    expect(screen.getByText("Configurado aqui")).toBeInTheDocument();
    expect(screen.getByText(/por admin/)).toBeInTheDocument();
    expect(within(form).getByLabelText("Senha")).toHaveAttribute(
      "placeholder",
      "Gravada — deixe em branco para manter",
    );
    await user.click(within(form).getByRole("switch", { name: "Enviar avisos por e-mail" }));
    await user.click(within(form).getByRole("button", { name: "Salvar" }));
    await waitFor(() => expect(api.to("PUT v1/canais/email")).toHaveLength(1));
    expect(api.to("PUT v1/canais/email")[0]!.body).toEqual({
      ativo: true,
      host: "smtp.x.gov.br",
      porta: 587,
      usuario: "central",
      senha: "",
      remover_senha: false,
      remetente: "Central <at@x.gov.br>",
    });

    const envio = screen.getByRole("button", { name: /Enviar teste/ });
    expect(envio).toBeDisabled();
    await user.type(screen.getByLabelText("Mandar o teste para"), "eu@x.gov.br");
    await user.click(envio);
    expect(await screen.findByText(/Mensagem enviada/)).toBeInTheDocument();
    expect(api.to("POST v1/canais/email/testar")[0]!.body).toEqual({ para: "eu@x.gov.br" });
    await user.click(envio);
    expect(
      await screen.findByText("O servidor recusou: 535 credenciais recusadas"),
    ).toBeInTheDocument();
  });

  it("apaga a senha, mostra a recusa da validação e a origem do ambiente", async () => {
    const user = userEvent.setup();
    const api = mockBackend(
      rotas({
        "PUT v1/canais/email": fail(422, "para ligar o e-mail, informe o servidor e o remetente"),
      }),
    );
    const tela = renderApp(<CanaisPage />);
    const form = await screen.findByRole("form", { name: "Servidor de e-mail" });
    await user.click(within(form).getByLabelText("Apagar a senha gravada"));
    expect(within(form).getByLabelText("Senha")).toBeDisabled();
    await user.click(within(form).getByRole("button", { name: "Salvar" }));
    expect(
      await screen.findByText("para ligar o e-mail, informe o servidor e o remetente"),
    ).toBeInTheDocument();
    expect(api.to("PUT v1/canais/email")[0]!.body).toMatchObject({ remover_senha: true });
    tela.unmount();

    mockBackend(
      rotas({
        "GET v1/canais/email": {
          data: {
            ativo: true,
            host: "smtp.env",
            porta: 0,
            usuario: "",
            tem_senha: false,
            remetente: "",
            origem: "ambiente",
          },
        },
      }),
    );
    renderApp(<CanaisPage />);
    expect(await screen.findByText("Vindo das variáveis de ambiente")).toBeInTheDocument();
    expect(screen.getByText("Ligado")).toBeInTheDocument();
    expect(screen.getByLabelText(/Porta/)).toHaveValue(587);
    expect(screen.queryByLabelText("Apagar a senha gravada")).not.toBeInTheDocument();
  });

  it("falha ao ler e tenta de novo", async () => {
    const user = userEvent.setup();
    const api = mockBackend(rotas({ "GET v1/canais/email": fail(500, "banco fora") }));
    renderApp(<CanaisPage />);
    await user.click(await screen.findByRole("button", { name: "Tentar novamente" }));
    await waitFor(() => expect(api.to("GET v1/canais/email").length).toBeGreaterThan(1));
  });
});
