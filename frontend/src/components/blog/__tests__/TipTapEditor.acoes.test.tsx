import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const post = vi.fn();
const putToTicket = vi.fn();
vi.mock("@/lib/api/client", () => ({ apiClient: { post: (...a: unknown[]) => post(...a) } }));
vi.mock("@/lib/plataforma/upload", () => ({
  mimeOf: (f: File) => f.type,
  putToTicket: (...a: unknown[]) => putToTicket(...a),
}));

import { TipTapEditor } from "../TipTapEditor";

const imagem = () => new File(["x"], "foto.png", { type: "image/png" });

afterEach(() => {
  vi.restoreAllMocks();
  post.mockReset();
  putToTicket.mockReset();
});

describe("TipTapEditor — ações", () => {
  it("todos os botões de formatação e o histórico", async () => {
    const onChange = vi.fn();
    render(<TipTapEditor value="Parágrafo" onChange={onChange} />);
    const rotulos = [
      "Negrito",
      "Itálico",
      "Riscado",
      "Código em linha",
      "Lista com marcadores",
      "Lista numerada",
      "Citação",
      "Bloco de código",
      "Título H2",
      "Subtítulo H3",
    ];
    for (const r of rotulos) fireEvent.click(await screen.findByLabelText(r));
    await waitFor(() => expect(onChange).toHaveBeenCalled());
    fireEvent.click(screen.getByLabelText("Desfazer"));
    fireEvent.click(screen.getByLabelText("Refazer"));
  });

  it("link: cancelar, remover e definir", async () => {
    render(<TipTapEditor value="Texto" onChange={vi.fn()} />);
    const prompt = vi.spyOn(window, "prompt");
    const botao = await screen.findByLabelText("Inserir Link");
    prompt.mockReturnValueOnce(null);
    fireEvent.click(botao);
    prompt.mockReturnValueOnce("");
    fireEvent.click(botao);
    prompt.mockReturnValueOnce("https://rondonopolis.mt.gov.br");
    fireEvent.click(botao);
    expect(prompt).toHaveBeenCalledTimes(3);
    expect(prompt).toHaveBeenCalledWith("URL do link:", "https://");
  });

  it("imagem: sobe pelo ticket e insere a URL limpa; falha só registra", async () => {
    const { container } = render(<TipTapEditor value="Texto" onChange={vi.fn()} />);
    await screen.findByLabelText("Inserir imagem no corpo (MinIO)");
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    const clique = vi.spyOn(input, "click");
    fireEvent.click(screen.getByLabelText("Inserir imagem no corpo (MinIO)"));
    expect(clique).toHaveBeenCalled();

    post.mockResolvedValueOnce({
      data: { upload_url: "https://minio/blog/foto.png?X-Amz=1", method: "PUT", headers: {} },
    });
    putToTicket.mockResolvedValueOnce(undefined);
    fireEvent.change(input, { target: { files: [imagem()] } });
    await waitFor(() =>
      expect(container.querySelector('img[src="https://minio/blog/foto.png"]')).not.toBeNull(),
    );
    expect(post).toHaveBeenCalledWith("v1/blog/uploads", {
      filename: "foto.png",
      content_type: "image/png",
    });

    const erro = vi.spyOn(console, "error").mockImplementation(() => {});
    post.mockRejectedValueOnce(new Error("sem permissão"));
    fireEvent.change(input, { target: { files: [imagem()] } });
    await waitFor(() => expect(erro).toHaveBeenCalled());
    await waitFor(() =>
      expect(screen.getByLabelText("Inserir imagem no corpo (MinIO)")).not.toBeDisabled(),
    );

    post.mockResolvedValueOnce({ data: { upload_url: "", method: "PUT", headers: {} } });
    putToTicket.mockResolvedValueOnce(undefined);
    fireEvent.change(input, { target: { files: [imagem()] } });
    await waitFor(() => expect(post).toHaveBeenCalledTimes(3));
    fireEvent.change(input, { target: { files: [] } });
    expect(post).toHaveBeenCalledTimes(3);
  });

  it("acompanha o valor externo quando ele muda", async () => {
    const { rerender, container } = render(<TipTapEditor value="Antes" onChange={vi.fn()} />);
    await screen.findByLabelText("Negrito");
    rerender(<TipTapEditor value="Depois do rascunho" onChange={vi.fn()} />);
    await waitFor(() => expect(container).toHaveTextContent("Depois do rascunho"));
  });
});
