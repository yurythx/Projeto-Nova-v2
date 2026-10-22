import type { ReactNode } from "react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

vi.mock("server-only", () => ({}));
vi.mock("@/components/layout/PublicShell", () => ({
  PublicShell: ({ children }: { children: ReactNode }) => <main>{children}</main>,
}));

// Mock publicBlog e catalog
vi.mock("@/lib/blog/publicBlog", () => ({
  listPublicPosts: vi.fn(),
  getPublicPost: vi.fn(),
}));

vi.mock("@/lib/catalog/catalog", () => ({
  listPublishedServices: vi.fn().mockResolvedValue([]),
}));

// Mock next/navigation notFound
const notFoundMock = vi.fn();
vi.mock("next/navigation", () => ({
  notFound: () => {
    notFoundMock();
    throw new Error("NEXT_NOT_FOUND");
  },
}));

// Mock IntersectionObserver
class MockIntersectionObserver {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}
window.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver;

import { PublicBlogListClient } from "../PublicBlogListClient";
import BlogPublicPage from "../page";
import PublicPostPage, { generateMetadata } from "../[ref]/page";
import { GET as getRss } from "../rss.xml/route";
import type { Post } from "@/lib/sismarq/types";

describe("PublicBlogListClient", () => {
  const samplePosts: Post[] = [
    {
      id: "1",
      slug: "post-infraestrutura",
      title: "Tabela de Temporalidade publicada",
      summary: "Prazos de guarda das secretarias.",
      body: "## Tópico 1\nConteúdo sobre a TTDD e o DIORONDON-e.",
      kind: "noticia",
      status: "published",
      pinned: true,
      published_at: "2026-10-01T10:00:00Z",
      created_at: "2026-10-01T10:00:00Z",
      updated_at: "2026-10-01T10:00:00Z",
    },
    {
      id: "2",
      slug: "post-seguranca",
      title: "Rito de Eliminação de Documentos",
      summary: "Relação, ata, edital e termo.",
      body: "Conteúdo sobre a eliminação e a amostragem.",
      kind: "comunicado",
      status: "published",
      pinned: false,
      published_at: "2026-10-02T10:00:00Z",
      created_at: "2026-10-02T10:00:00Z",
      updated_at: "2026-10-02T10:00:00Z",
    },
  ];

  it("renderiza a lista de posts e destaca o post fixado (hero)", () => {
    render(<PublicBlogListClient initialPosts={samplePosts} />);

    expect(screen.getByText(/destaque editorial/i)).toBeInTheDocument();
    expect(screen.getAllByText("Tabela de Temporalidade publicada").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Rito de Eliminação de Documentos").length).toBeGreaterThan(0);
  });

  it("filtra posts por termo de busca no input", () => {
    render(<PublicBlogListClient initialPosts={samplePosts} />);

    const searchInput = screen.getByPlaceholderText(/buscar por assunto/i);
    fireEvent.change(searchInput, { target: { value: "amostragem" } });

    expect(screen.getAllByText("Rito de Eliminação de Documentos").length).toBeGreaterThan(0);
    expect(screen.queryByRole("heading", { name: "Tabela de Temporalidade publicada" })).not.toBeInTheDocument();

    // Limpa a busca
    const clearBtn = screen.getByLabelText("Limpar busca");
    fireEvent.click(clearBtn);

    expect(screen.getAllByText("Tabela de Temporalidade publicada").length).toBeGreaterThan(0);
  });

  it("filtra posts por tag de categoria", () => {
    render(<PublicBlogListClient initialPosts={samplePosts} />);

    // Clica na tag de Eliminação
    const segurancaTag = screen.getByRole("tab", { name: "Eliminação" });
    fireEvent.click(segurancaTag);

    expect(screen.getAllByText("Rito de Eliminação de Documentos").length).toBeGreaterThan(0);
    expect(screen.queryByRole("heading", { name: "Tabela de Temporalidade publicada" })).not.toBeInTheDocument();
  });

  it("exibe estado vazio quando nenhum post corresponde ao filtro", () => {
    render(<PublicBlogListClient initialPosts={samplePosts} />);

    const searchInput = screen.getByPlaceholderText(/buscar por assunto/i);
    fireEvent.change(searchInput, { target: { value: "palavra-inexistente-xyz" } });

    expect(screen.getByText(/nenhuma publicação encontrada/i)).toBeInTheDocument();

    const resetBtn = screen.getByRole("button", { name: "Limpar filtros" });
    fireEvent.click(resetBtn);

    expect(screen.getAllByText("Tabela de Temporalidade publicada").length).toBeGreaterThan(0);
  });
});

describe("Páginas Públicas do Blog (Server Components)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renderiza BlogPublicPage com os posts retornados", async () => {
    const { listPublicPosts } = await import("@/lib/blog/publicBlog");
    vi.mocked(listPublicPosts).mockResolvedValue([
      {
        id: "1",
        slug: "post-1",
        title: "Artigo Inicial de Testes",
        summary: "Resumo do artigo inicial",
        body: "Corpo",
        kind: "noticia",
        status: "published",
        pinned: false,
        published_at: "2026-10-01T10:00:00Z",
        created_at: "2026-10-01T10:00:00Z",
        updated_at: "2026-10-01T10:00:00Z",
      },
    ]);

    const page = await BlogPublicPage();
    render(page);

    expect(screen.getAllByText("Artigo Inicial de Testes").length).toBeGreaterThan(0);
    expect(screen.getByText("Resumo do artigo inicial")).toBeInTheDocument();
  });

  it("renderiza PublicPostPage quando o post existe", async () => {
    const { getPublicPost, listPublicPosts } = await import("@/lib/blog/publicBlog");
    const mockPost: Post = {
      id: "10",
      slug: "post-detalhado",
      title: "Artigo Detalhado da Plataforma",
      summary: "Resumo do artigo detalhado.",
      body: "## Introdução\nTexto de introdução do artigo.",
      kind: "noticia",
      status: "published",
      pinned: false,
      published_at: "2026-10-05T12:00:00Z",
      created_at: "2026-10-05T12:00:00Z",
      updated_at: "2026-10-05T12:00:00Z",
    };
    vi.mocked(getPublicPost).mockResolvedValue(mockPost);
    vi.mocked(listPublicPosts).mockResolvedValue([]);

    const page = await PublicPostPage({ params: Promise.resolve({ ref: "post-detalhado" }) });
    render(page);

    expect(screen.getByRole("heading", { name: "Artigo Detalhado da Plataforma", level: 1 })).toBeInTheDocument();
    expect(screen.getByText("Resumo do artigo detalhado.")).toBeInTheDocument();
  });

  it("chama notFound() em PublicPostPage quando o post não existe", async () => {
    const { getPublicPost } = await import("@/lib/blog/publicBlog");
    vi.mocked(getPublicPost).mockResolvedValue(null);

    await expect(
      PublicPostPage({ params: Promise.resolve({ ref: "post-inexistente" }) }),
    ).rejects.toThrow("NEXT_NOT_FOUND");
    expect(notFoundMock).toHaveBeenCalled();
  });

  it("gera metadados corretos com generateMetadata", async () => {
    const { getPublicPost } = await import("@/lib/blog/publicBlog");
    vi.mocked(getPublicPost).mockResolvedValue({
      id: "10",
      slug: "post-meta",
      title: "Artigo SEO",
      summary: "Descrição para Google",
      body: "Corpo",
      kind: "noticia",
      status: "published",
      pinned: false,
      cover_url: "https://minio/cover.png",
      published_at: "2026-10-05T12:00:00Z",
      created_at: "2026-10-05T12:00:00Z",
      updated_at: "2026-10-05T12:00:00Z",
    });

    const meta = await generateMetadata({ params: Promise.resolve({ ref: "post-meta" }) });
    expect(meta.title).toContain("Artigo SEO");
    expect(meta.description).toBe("Descrição para Google");

    // Quando não existe
    vi.mocked(getPublicPost).mockResolvedValue(null);
    const meta404 = await generateMetadata({ params: Promise.resolve({ ref: "post-404" }) });
    expect(meta404.title).toBe("Artigo não encontrado");
  });
});

describe("RSS 2.0 Feed Route (app/blog/rss.xml/route.ts)", () => {
  it("gera o feed XML RSS válido com os posts públicos", async () => {
    const { listPublicPosts } = await import("@/lib/blog/publicBlog");
    vi.mocked(listPublicPosts).mockResolvedValue([
      {
        id: "1",
        slug: "post-rss",
        title: "Artigo no Feed RSS",
        summary: "Resumo do post no feed.",
        body: "Corpo",
        kind: "noticia",
        status: "published",
        pinned: false,
        published_at: "2026-10-01T10:00:00Z",
        created_at: "2026-10-01T10:00:00Z",
        updated_at: "2026-10-01T10:00:00Z",
      },
    ]);

    const res = await getRss();
    expect(res.status).toBe(200);
    expect(res.headers.get("Content-Type")).toContain("application/xml");

    const xml = await res.text();
    expect(xml).toContain("<rss version=\"2.0\"");
    expect(xml).toContain("<title><![CDATA[Artigo no Feed RSS]]></title>");
    expect(xml).toContain("/blog/post-rss</link>");
  });
});
