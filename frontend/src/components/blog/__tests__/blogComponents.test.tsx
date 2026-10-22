import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";

import { ReadingProgressBar } from "../ReadingProgressBar";
import { TableOfContents } from "../TableOfContents";
import { SocialShareBar } from "../SocialShareBar";
import { BlogCoverImage } from "../BlogCoverImage";
import { RelatedPosts } from "../RelatedPosts";
import { PostEditor } from "../PostEditor";
import { ToastProvider } from "@/components/notifications/ToastProvider";
import type { Post } from "@/lib/plataforma/types";

// Mock IntersectionObserver
class MockIntersectionObserver {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}
window.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver;

describe("ReadingProgressBar", () => {
  it("renderiza o elemento progressbar e atualiza conforme o scroll", () => {
    render(<ReadingProgressBar />);
    const bar = screen.getByRole("progressbar");
    expect(bar).toBeInTheDocument();
    expect(bar).toHaveAttribute("aria-valuenow", "0");

    // Simula scroll
    Object.defineProperty(document.documentElement, "scrollHeight", { value: 2000, configurable: true });
    Object.defineProperty(window, "innerHeight", { value: 1000, configurable: true });
    Object.defineProperty(window, "scrollY", { value: 500, configurable: true });

    fireEvent.scroll(window);
    expect(bar).toHaveAttribute("aria-valuenow", "50");
  });
});

describe("TableOfContents", () => {
  it("não renderiza nada se o artigo não tiver títulos H2 ou H3", () => {
    const { container } = render(<TableOfContents source="Apenas um parágrafo normal." />);
    expect(container.firstChild).toBeNull();
  });

  it("extrai títulos H2 e H3 do Markdown e permite navegação", () => {
    const md = `
# Título H1 Ignorado
## Seção Um
Texto da seção um.
### Subseção A
Texto da subseção A.
## Seção Dois
Texto da seção dois.
    `;
    render(<TableOfContents source={md} />);
    expect(screen.getByText("Neste Artigo")).toBeInTheDocument();
    expect(screen.getByText("Seção Um")).toBeInTheDocument();
    expect(screen.getByText("Subseção A")).toBeInTheDocument();
    expect(screen.getByText("Seção Dois")).toBeInTheDocument();

    const link = screen.getByText("Seção Um");
    const mockEl = document.createElement("div");
    mockEl.id = "secao-um";
    mockEl.scrollIntoView = vi.fn();
    document.body.appendChild(mockEl);

    fireEvent.click(link);
    expect(mockEl.scrollIntoView).toHaveBeenCalledWith({ behavior: "smooth" });
  });
});

describe("SocialShareBar", () => {
  it("renderiza os botões de compartilhamento", () => {
    render(<SocialShareBar title="Novidades Projeto-nova-v2" url="https://nova.local/blog/novidades" />);
    expect(screen.getByLabelText("Compartilhar no WhatsApp")).toHaveAttribute(
      "href",
      expect.stringContaining("api.whatsapp.com"),
    );
    expect(screen.getByLabelText("Compartilhar no LinkedIn")).toHaveAttribute(
      "href",
      expect.stringContaining("linkedin.com"),
    );
    expect(screen.getByLabelText("Compartilhar no X")).toHaveAttribute(
      "href",
      expect.stringContaining("twitter.com"),
    );
  });

  it("permite copiar o link para a área de transferência", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: { writeText },
    });

    render(<SocialShareBar title="Novidades Projeto-nova-v2" url="https://nova.local/blog/novidades" />);
    const copyBtn = screen.getByLabelText("Copiar link do artigo para a área de transferência");
    fireEvent.click(copyBtn);

    expect(writeText).toHaveBeenCalledWith("https://nova.local/blog/novidades");
    await waitFor(() => {
      expect(screen.getByText("Copiado!")).toBeInTheDocument();
    });
  });
});

describe("BlogCoverImage", () => {
  it("renderiza a tag <img> quando coverUrl for fornecida", () => {
    render(
      <BlogCoverImage
        coverUrl="/static/capa.jpg"
        title="Artigo com Imagem"
        slug="artigo-com-imagem"
        aspect="card"
      />,
    );
    const img = screen.getByRole("img", { name: /Artigo com Imagem/ });
    expect(img).toHaveAttribute("src", "/static/capa.jpg");
  });

  it("renderiza o fallback com ícone estilizado quando não há imagem", () => {
    const { container } = render(
      <BlogCoverImage
        coverUrl={null}
        title="Artigo do Tailscale"
        slug="artigo-tailscale"
        aspect="hero"
      />,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
    expect(screen.getByText("Artigo do Tailscale")).toBeInTheDocument();
  });
});

describe("RelatedPosts", () => {
  const posts: Post[] = [
    {
      id: "1",
      slug: "post-1",
      title: "Post 1",
      summary: "Resumo 1",
      body: "Corpo 1",
      kind: "noticia",
      status: "published",
      pinned: false,
      published_at: "2026-10-01T10:00:00Z",
      created_at: "2026-10-01T10:00:00Z",
      updated_at: "2026-10-01T10:00:00Z",
    },
    {
      id: "2",
      slug: "post-2",
      title: "Post 2",
      summary: "Resumo 2",
      body: "Corpo 2",
      kind: "noticia",
      status: "published",
      pinned: false,
      published_at: "2026-10-02T10:00:00Z",
      created_at: "2026-10-02T10:00:00Z",
      updated_at: "2026-10-02T10:00:00Z",
    },
  ];

  it("retorna null se não houver outros posts", () => {
    const { container } = render(<RelatedPosts currentSlug="post-1" posts={[posts[0]!]} />);
    expect(container.firstChild).toBeNull();
  });

  it("filtra o post atual e exibe posts relacionados", () => {
    render(<RelatedPosts currentSlug="post-1" posts={posts} />);
    expect(screen.getByText("Leia Também")).toBeInTheDocument();
    expect(screen.queryByText("Post 1")).not.toBeInTheDocument();
    expect(screen.getAllByText("Post 2").length).toBeGreaterThan(0);
  });
});

describe("PostEditor Autosave & Draft Management", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("salva rascunho automaticamente no localStorage com debounce ao digitar", async () => {
    const onSaved = vi.fn();
    render(
      <ToastProvider>
        <PostEditor onSaved={onSaved} />
      </ToastProvider>,
    );

    // Clica para alternar para modo Markdown puro
    const mdToggle = screen.getByText("Ver Markdown");
    fireEvent.click(mdToggle);

    const textarea = screen.getByLabelText("Código Markdown puro");
    fireEvent.change(textarea, { target: { value: "Novo rascunho em markdown de teste." } });

    act(() => {
      vi.advanceTimersByTime(1100);
    });

    const stored = localStorage.getItem("nova_blog_draft_new");
    expect(stored).toBeTruthy();
    expect(JSON.parse(stored!).body).toContain("Novo rascunho em markdown de teste.");
  });

  it("detecta rascunho anterior e permite restaurar ou descartar", async () => {
    localStorage.setItem(
      "nova_blog_draft_new",
      JSON.stringify({ body: "Texto restaurado do rascunho.", savedAt: Date.now() }),
    );

    const onSaved = vi.fn();
    render(
      <ToastProvider>
        <PostEditor onSaved={onSaved} />
      </ToastProvider>,
    );

    expect(screen.getByText("Existe um rascunho anterior salvo localmente no seu navegador.")).toBeInTheDocument();

    const restoreBtn = screen.getByRole("button", { name: "Restaurar" });
    fireEvent.click(restoreBtn);

    // Clica em pré-visualizar para checar o corpo renderizado
    const previewBtn = screen.getByRole("button", { name: "Pré-visualizar" });
    fireEvent.click(previewBtn);

    expect(screen.getByText("Texto restaurado do rascunho.")).toBeInTheDocument();
    expect(screen.queryByText("Existe um rascunho anterior salvo localmente no seu navegador.")).not.toBeInTheDocument();
  });

  it("descarta o rascunho ao clicar em Descartar", async () => {
    localStorage.setItem(
      "nova_blog_draft_new",
      JSON.stringify({ body: "Texto a descartar.", savedAt: Date.now() }),
    );

    const onSaved = vi.fn();
    render(
      <ToastProvider>
        <PostEditor onSaved={onSaved} />
      </ToastProvider>,
    );

    const discardBtn = screen.getByRole("button", { name: "Descartar" });
    fireEvent.click(discardBtn);

    expect(localStorage.getItem("nova_blog_draft_new")).toBeNull();
    expect(screen.queryByText("Existe um rascunho anterior salvo localmente no seu navegador.")).not.toBeInTheDocument();
  });
});
