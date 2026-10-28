import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { identityRoutes, mockBackend, renderApp } from "@/test/backend";
import { resetNavigation } from "@/test/navigation";
import { formatarWhatsApp } from "@/lib/plataforma/canais";

vi.mock("next/navigation", async () => (await import("@/test/navigation")).navigationModule);

import { FormularioDeContato, PedidoDeContato } from "./ContatoParaAvisos";

beforeEach(() => resetNavigation({}, "/atendimento"));

describe("Contato para avisos", () => {
  it("na primeira visita pede o contato, com o e-mail do login sugerido", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/me/contato": {
        data: { email: "", whatsapp: "", respondido: false, email_do_sso: "ana@x.gov.br" },
      },
      "PUT v1/me/contato": {
        data: { email: "ana@x.gov.br", whatsapp: "+5566999991234", respondido: true },
      },
    });
    renderApp(<PedidoDeContato />);
    const dialogo = await screen.findByRole("dialog", { name: "Como quer receber os avisos?" });
    expect(within(dialogo).getByLabelText("E-mail")).toHaveValue("ana@x.gov.br");
    await user.type(within(dialogo).getByLabelText("WhatsApp"), "(66) 99999-1234");
    await user.click(within(dialogo).getByRole("button", { name: "Salvar contato" }));
    await waitFor(() => expect(api.to("PUT v1/me/contato")).toHaveLength(1));
    expect(api.to("PUT v1/me/contato")[0]!.body).toEqual({
      email: "ana@x.gov.br",
      whatsapp: "(66) 99999-1234",
    });
    await waitFor(() => expect(dialogo).not.toHaveAttribute("open"));
  });

  it("agora não grava a resposta vazia; fechar só some por agora", async () => {
    const user = userEvent.setup();
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/me/contato": { data: { email: "", whatsapp: "", respondido: false } },
      "PUT v1/me/contato": { data: { email: "", whatsapp: "", respondido: true } },
    });
    const um = renderApp(<PedidoDeContato />);
    const dialogo = await screen.findByRole("dialog");
    await user.click(within(dialogo).getByRole("button", { name: "Agora não" }));
    await waitFor(() => expect(api.to("PUT v1/me/contato")).toHaveLength(1));
    expect(api.to("PUT v1/me/contato")[0]!.body).toEqual({ email: "", whatsapp: "" });
    await waitFor(() => expect(dialogo).not.toHaveAttribute("open"));
    um.unmount();

    mockBackend({
      ...identityRoutes(),
      "GET v1/me/contato": { data: { email: "", whatsapp: "", respondido: false } },
    });
    renderApp(<PedidoDeContato />);
    const aberto = await screen.findByRole("dialog");
    fireEvent(aberto, new Event("cancel"));
    await waitFor(() => expect(aberto).not.toHaveAttribute("open"));
  });

  it("quem já respondeu não é perguntado; o perfil mostra o número formatado", async () => {
    const api = mockBackend({
      ...identityRoutes(),
      "GET v1/me/contato": { data: { email: "", whatsapp: "+5566999991234", respondido: true } },
    });
    renderApp(<PedidoDeContato />);
    await waitFor(() => expect(api.to("GET v1/me/contato").length).toBeGreaterThan(0));
    expect(screen.queryByRole("dialog")).toBeNull();

    renderApp(
      <FormularioDeContato
        contato={{ email: "", whatsapp: "+5566999991234", respondido: true }}
        onSalvo={() => {}}
      />,
    );
    expect(screen.getByLabelText("WhatsApp")).toHaveValue("+55 (66) 99999-1234");
    expect(screen.queryByRole("button", { name: "Agora não" })).not.toBeInTheDocument();
    expect(formatarWhatsApp("+351912345678")).toBe("+351912345678");
  });
});
