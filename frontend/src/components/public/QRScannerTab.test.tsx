import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { QRScannerTab } from "./QRScannerTab";

const UUID = "0A1B2C3D-0000-4000-8000-123456789ABC";

/** Image falsa: dispara onload (ou onerror) quando recebe o src. */
function imagem(ok: boolean) {
  vi.stubGlobal(
    "Image",
    class {
      onload: (() => void) | null = null;
      onerror: (() => void) | null = null;
      set src(_v: string) {
        setTimeout(() => (ok ? this.onload?.() : this.onerror?.()), 0);
      }
    },
  );
}

function camera(getUserMedia?: () => Promise<MediaStream>) {
  Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: getUserMedia ? { getUserMedia } : undefined });
}

const enviar = (container: HTMLElement, nome: string) =>
  fireEvent.change(container.querySelector('input[type="file"]')!, { target: { files: [new File(["x"], nome, { type: "image/png" })] } });

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  camera();
});

describe("QRScannerTab — foto", () => {
  it("lê o código pelo nome do arquivo", async () => {
    const onCode = vi.fn();
    const { container } = render(<QRScannerTab onCodeDetected={onCode} />);
    enviar(container, `qr-${UUID}.png`);
    await waitFor(() => expect(onCode).toHaveBeenCalledWith(UUID.toLowerCase()));
  });

  it("orienta quando a imagem não traz o código e avisa imagem inválida", async () => {
    const onCode = vi.fn();
    const { container } = render(<QRScannerTab onCodeDetected={onCode} />);
    imagem(true);
    enviar(container, "foto.png");
    expect(screen.getByText("Analisando imagem...")).toBeInTheDocument();
    expect(await screen.findByText(/Foto carregada/)).toBeInTheDocument();
    imagem(false);
    enviar(container, "foto.png");
    expect(await screen.findByText("Não foi possível processar o arquivo de imagem selecionado.")).toBeInTheDocument();
    fireEvent.change(container.querySelector('input[type="file"]')!, { target: { files: [] } });
    expect(onCode).not.toHaveBeenCalled();
  });
});

describe("QRScannerTab — câmera", () => {
  it("sem suporte à câmera", async () => {
    camera();
    render(<QRScannerTab onCodeDetected={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: /Usar Câmera/ }));
    expect(await screen.findByRole("alert")).toHaveTextContent("não possui suporte");
  });

  it("permissão negada", async () => {
    camera(() => Promise.reject(new Error("negada")));
    render(<QRScannerTab onCodeDetected={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: /Usar Câmera/ }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Permissão para câmera negada");
    expect(screen.getByRole("button", { name: "Ativar Câmera" })).toBeInTheDocument();
  });

  it("ativa, desativa e solta a câmera ao trocar de aba e ao sair", async () => {
    const stop = vi.fn();
    const stream = { getTracks: () => [{ stop }] } as unknown as MediaStream;
    camera(() => Promise.resolve(stream));
    vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);
    const { unmount } = render(<QRScannerTab onCodeDetected={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: /Usar Câmera/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Desativar Câmera" }));
    expect(stop).toHaveBeenCalledTimes(1);
    fireEvent.click(await screen.findByRole("button", { name: "Ativar Câmera" }));
    await screen.findByRole("button", { name: "Desativar Câmera" });
    fireEvent.click(screen.getByRole("button", { name: /Carregar Foto/ }));
    expect(stop).toHaveBeenCalledTimes(2);

    fireEvent.click(screen.getByRole("button", { name: /Usar Câmera/ }));
    await screen.findByRole("button", { name: "Desativar Câmera" });
    unmount();
    expect(stop).toHaveBeenCalledTimes(3);
  });
});
