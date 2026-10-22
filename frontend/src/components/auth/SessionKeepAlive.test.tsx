import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const update = vi.fn(async () => null);
const getSession = vi.fn(async (): Promise<{ expiresAt?: number } | null> => null);
let expiresAt: number | undefined;
vi.mock("next-auth/react", () => ({
  useSession: () => ({ data: expiresAt ? { expiresAt } : null, update }),
  getSession: () => getSession(),
}));
const redirectToLogin = vi.fn();
vi.mock("@/lib/auth/sessionExpired", () => ({ redirectToLogin: () => redirectToLogin() }));

import { SessionKeepAlive } from "./SessionKeepAlive";

const MIN = 60_000;

beforeEach(() => {
  vi.useFakeTimers();
  update.mockClear();
  getSession.mockClear();
  redirectToLogin.mockClear();
});
afterEach(() => vi.useRealTimers());

describe("SessionKeepAlive", () => {
  it("longe do vencimento não renova nem avisa", () => {
    expiresAt = Date.now() + 50 * MIN;
    render(<SessionKeepAlive />);
    fireEvent.keyDown(window, { key: "a" });
    expect(update).not.toHaveBeenCalled();
    expect(screen.queryByRole("alertdialog")).toBeNull();
  });

  it("perto do vencimento e com uso recente, renova a sessão", () => {
    expiresAt = Date.now() + 5 * MIN;
    render(<SessionKeepAlive />);
    expect(update).toHaveBeenCalledTimes(1);
    // Uso logo em seguida não dispara outra renovação (espera entre tentativas).
    fireEvent.pointerDown(window);
    expect(update).toHaveBeenCalledTimes(1);
  });

  it("inativo: avisa com contagem regressiva e 'Continuar conectado' renova", async () => {
    expiresAt = Date.now() + 25 * MIN;
    render(<SessionKeepAlive />);
    // 24 min sem uso: falta 1 min e a janela de renovação abriu com o
    // usuário já inativo havia mais de 10.
    await act(async () => {
      vi.advanceTimersByTime(24 * MIN);
    });
    expect(update).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toHaveTextContent(/vai expirar por inatividade/i);
    expect(screen.getByText(/^\d:\d\d$/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Continuar conectado" }));
    expect(update).toHaveBeenCalledTimes(1);
  });

  it("no vencimento confere a sessão e volta ao login", async () => {
    expiresAt = Date.now() + 25 * MIN;
    render(<SessionKeepAlive />);
    await act(async () => {
      vi.advanceTimersByTime(26 * MIN);
    });
    expect(getSession).toHaveBeenCalled();
    expect(redirectToLogin).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("alertdialog")).toBeNull();
  });

  it("não volta ao login se outra aba já renovou", async () => {
    expiresAt = Date.now() + 25 * MIN;
    getSession.mockResolvedValueOnce({ expiresAt: Date.now() + 60 * MIN });
    render(<SessionKeepAlive />);
    await act(async () => {
      vi.advanceTimersByTime(26 * MIN);
    });
    expect(getSession).toHaveBeenCalled();
    expect(redirectToLogin).not.toHaveBeenCalled();
  });

  it("sem sessão não faz nada", () => {
    expiresAt = undefined;
    const { container } = render(<SessionKeepAlive />);
    expect(container).toBeEmptyDOMElement();
  });
});
