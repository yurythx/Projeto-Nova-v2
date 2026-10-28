import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { fail, identityRoutes, mockBackend, renderApp } from "@/test/backend";
import type { WhatsAppCanal } from "@/lib/plataforma/canais";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import { WhatsAppCard } from "./WhatsAppCard";

const gravado: WhatsAppCanal = {
  ativo: false,
  provedor: "meta",
  meta_numero_id: "1234567",
  tem_token: true,
  meta_modelo: "aviso_chamado",
  meta_idioma: "pt_BR",
  evolution_url: "",
  evolution_instancia: "",
  tem_chave: false,
  receber: false,
  tem_app_secret: false,
  updated_at: "2026-10-28T12:00:00Z",
  updated_by: "admin",
};

describe("WhatsApp dos avisos", () => {
  it("liga a API oficial mantendo o token e testa", async () => {
    const user = userEvent.setup();
    const onSalvo = vi.fn();
    let teste = 0;
    const api = mockBackend({
      ...identityRoutes({ permissions: ["canais:manage"] }),
      "PUT v1/canais/whatsapp": { data: { ...gravado, ativo: true } },
      "POST v1/canais/whatsapp/testar": () =>
        ++teste === 1
          ? { data: { ok: true } }
          : { data: { ok: false, erro: "modelo não aprovado" } },
    });
    renderApp(<WhatsAppCard whatsapp={gravado} onSalvo={onSalvo} />);

    const form = screen.getByRole("form", { name: "Provedor do WhatsApp" });
    expect(screen.getByText(/por admin/)).toBeInTheDocument();
    expect(within(form).getByLabelText("Token")).toHaveAttribute(
      "placeholder",
      "Gravado — deixe em branco para manter",
    );
    expect(screen.getByText(/três variáveis/)).toBeInTheDocument();
    await user.click(within(form).getByRole("switch", { name: "Enviar avisos pelo WhatsApp" }));
    await user.click(within(form).getByRole("button", { name: "Salvar" }));
    await waitFor(() => expect(api.to("PUT v1/canais/whatsapp")).toHaveLength(1));
    expect(api.to("PUT v1/canais/whatsapp")[0]!.body).toEqual({
      ativo: true,
      provedor: "meta",
      meta_numero_id: "1234567",
      meta_token: "",
      remover_token: false,
      meta_modelo: "aviso_chamado",
      meta_idioma: "pt_BR",
      evolution_url: "",
      evolution_instancia: "",
      evolution_chave: "",
      remover_chave: false,
      receber: false,
      meta_app_secret: "",
      novo_webhook: false,
    });
    await waitFor(() => expect(onSalvo).toHaveBeenCalled());

    const envio = screen.getByRole("button", { name: /Enviar teste/ });
    await user.type(screen.getByLabelText("Mandar o teste para"), "(66) 99999-1234");
    await user.click(envio);
    expect(await screen.findByText("Mensagem enviada. Confira o WhatsApp.")).toBeInTheDocument();
    expect(api.to("POST v1/canais/whatsapp/testar")[0]!.body).toEqual({ para: "(66) 99999-1234" });
    await user.click(envio);
    expect(await screen.findByText("O provedor recusou: modelo não aprovado")).toBeInTheDocument();
  });

  it("troca para a Evolution, com o aviso de risco, e apaga a credencial", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes({ permissions: ["canais:manage"] }),
      "PUT v1/canais/whatsapp": fail(422, "para ligar o WhatsApp, informe o endereço"),
    });
    renderApp(
      <WhatsAppCard
        whatsapp={{ ...gravado, ativo: true, tem_chave: true, evolution_url: "http://evo:8080" }}
        onSalvo={() => {}}
      />,
    );
    expect(screen.getByText("Ligado")).toBeInTheDocument();
    const form = screen.getByRole("form", { name: "Provedor do WhatsApp" });
    await user.selectOptions(within(form).getByLabelText("Provedor"), "evolution");
    expect(within(form).getByRole("note")).toHaveTextContent(/fere os termos da Meta/);
    await user.clear(within(form).getByLabelText("Instância"));
    await user.type(within(form).getByLabelText("Instância"), "central");
    await user.type(within(form).getByLabelText("Chave"), "nova-chave");
    await user.click(within(form).getByLabelText("Apagar chave gravado"));
    expect(within(form).getByLabelText("Chave")).toBeDisabled();
    await user.click(within(form).getByRole("button", { name: "Salvar" }));
    expect(
      await screen.findByText("para ligar o WhatsApp, informe o endereço"),
    ).toBeInTheDocument();
    expect(api.to("PUT v1/canais/whatsapp")[0]!.body).toMatchObject({
      provedor: "evolution",
      meta_numero_id: "1234567",
      meta_token: "",
      evolution_url: "http://evo:8080",
      evolution_instancia: "central",
      remover_chave: true,
    });

    await user.selectOptions(within(form).getByLabelText("Provedor"), "meta");
    await user.click(within(form).getByLabelText("Apagar token gravado"));
    expect(within(form).getByLabelText("Token")).toBeDisabled();
  });

  it("liga a entrada, mostra o endereço do webhook e pede outro", async () => {
    const user = userEvent.setup();
    const escrever = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: escrever },
      configurable: true,
    });
    const api = mockBackend({
      ...identityRoutes({ permissions: ["canais:manage"] }),
      "PUT v1/canais/whatsapp": { data: gravado },
    });
    const endereco = "https://api.nova.gov.br/api/v1/canais/whatsapp/webhook/abc123";
    const um = renderApp(
      <WhatsAppCard
        whatsapp={{
          ...gravado,
          ativo: true,
          receber: true,
          tem_app_secret: true,
          webhook_url: endereco,
        }}
        onSalvo={() => {}}
      />,
    );
    expect(screen.getByText(endereco)).toBeInTheDocument();
    expect(screen.getByLabelText(/App Secret/)).toHaveAttribute(
      "placeholder",
      "Gravado — deixe em branco para manter",
    );
    expect(screen.getByText(/URL de callback/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Copiar" }));
    expect(escrever).toHaveBeenCalledWith(endereco);
    expect(await screen.findByRole("button", { name: "Copiado" })).toBeInTheDocument();
    await user.type(screen.getByLabelText(/App Secret/), "novo-segredo");
    await user.click(screen.getByLabelText(/Gerar outro endereço/));
    await user.click(screen.getByRole("button", { name: "Salvar" }));
    await waitFor(() => expect(api.to("PUT v1/canais/whatsapp")).toHaveLength(1));
    expect(api.to("PUT v1/canais/whatsapp")[0]!.body).toMatchObject({
      receber: true,
      meta_app_secret: "novo-segredo",
      novo_webhook: true,
    });
    um.unmount();

    // Na Evolution, as instruções mudam; sem endereço, ele aparece depois
    // de salvar; a cópia que falha não quebra a tela.
    escrever.mockRejectedValue(new Error("negado"));
    const dois = renderApp(
      <WhatsAppCard
        whatsapp={{ ...gravado, provedor: "evolution", receber: true, webhook_url: endereco }}
        onSalvo={() => {}}
      />,
    );
    expect(screen.getByText(/MESSAGES_UPSERT/)).toBeInTheDocument();
    expect(screen.queryByLabelText(/App Secret/)).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Copiar" }));
    expect(screen.getByRole("button", { name: "Copiar" })).toBeInTheDocument();
    dois.unmount();

    renderApp(<WhatsAppCard whatsapp={gravado} onSalvo={() => {}} />);
    await user.click(screen.getByRole("switch", { name: "Receber respostas pelo WhatsApp" }));
    expect(
      screen.getByText("O endereço do webhook aparece aqui depois de salvar."),
    ).toBeInTheDocument();
  });
});
