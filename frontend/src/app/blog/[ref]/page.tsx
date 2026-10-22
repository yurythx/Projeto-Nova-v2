import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft, Calendar, Clock, User } from "lucide-react";

import { BlogCoverImage } from "@/components/blog/BlogCoverImage";
import { ReadingProgressBar } from "@/components/blog/ReadingProgressBar";
import { RelatedPosts } from "@/components/blog/RelatedPosts";
import { SocialShareBar } from "@/components/blog/SocialShareBar";
import { TableOfContents } from "@/components/blog/TableOfContents";
import { PublicShell } from "@/components/layout/PublicShell";
import { Markdown } from "@/components/sismarq/Markdown";
import { getPublicPost, listPublicPosts } from "@/lib/blog/publicBlog";
import { APP_URL } from "@/lib/env";

type Props = { params: Promise<{ ref: string }> };

export const dynamic = "force-dynamic";

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const post = await getPublicPost((await params).ref);
  if (!post) return { title: "Artigo não encontrado" };
  const canonicalUrl = `${APP_URL}/blog/${post.slug}`;

  return {
    title: `${post.title} — Blog & Novidades`,
    description: post.summary,
    alternates: { canonical: canonicalUrl },
    openGraph: {
      title: post.title,
      description: post.summary,
      type: "article",
      url: canonicalUrl,
      images: post.cover_url ? [{ url: post.cover_url, alt: post.title }] : undefined,
    },
    twitter: {
      card: "summary_large_image",
      title: post.title,
      description: post.summary,
      images: post.cover_url ? [post.cover_url] : undefined,
    },
  };
}

export default async function PublicPostPage({ params }: Props) {
  const { ref } = await params;
  const [post, allPosts] = await Promise.all([
    getPublicPost(ref),
    listPublicPosts("", "", 1, 10).catch(() => []),
  ]);

  if (!post) notFound();

  const formatDate = (dateStr?: string | null) => {
    if (!dateStr) return "";
    try {
      return new Intl.DateTimeFormat("pt-BR", {
        day: "2-digit",
        month: "long",
        year: "numeric",
      }).format(new Date(dateStr));
    } catch {
      return dateStr;
    }
  };

  const words = (post.body || "").split(/\s+/).length;
  const readTime = `${Math.max(1, Math.ceil(words / 200))} min de leitura`;
  const postUrl = `${APP_URL}/blog/${post.slug}`;

  // Schema.org JSON-LD para SEO técnico (BlogPosting)
  const jsonLd = {
    "@context": "https://schema.org",
    "@type": "BlogPosting",
    headline: post.title,
    description: post.summary,
    image: post.cover_url ? [post.cover_url] : undefined,
    datePublished: post.published_at || post.created_at,
    dateModified: post.updated_at,
    author: {
      "@type": "Organization",
      name: post.author_name || "Projeto-nova-v2",
    },
    publisher: {
      "@type": "Organization",
      name: "Projeto-nova-v2",
      url: APP_URL,
    },
    mainEntityOfPage: {
      "@type": "WebPage",
      "@id": postUrl,
    },
  };

  return (
    <PublicShell>
      {/* Barra de Progresso de Leitura */}
      <ReadingProgressBar />

      {/* Dados Estruturados Schema.org */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />

      <article className="mx-auto flex w-full max-w-5xl flex-col gap-8 px-6 py-12">
        <Link
          href="/blog"
          className="inline-flex items-center gap-1.5 text-sm font-medium text-muted transition-colors hover:text-foreground"
        >
          <ArrowLeft size={15} aria-hidden="true" /> Voltar para Blog & Novidades
        </Link>

        {/* Cabeçalho do Artigo */}
        <header className="flex flex-col gap-6 border-b border-surface-border pb-8">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <div className="flex flex-wrap items-center gap-3 text-xs text-muted">
              <span className="inline-flex items-center gap-1 font-medium">
                <Calendar size={13} aria-hidden="true" />
                {formatDate(post.published_at || post.updated_at)}
              </span>
              <span>•</span>
              <span className="inline-flex items-center gap-1">
                <Clock size={13} aria-hidden="true" />
                {readTime}
              </span>
              {post.author_name && (
                <>
                  <span>•</span>
                  <span className="inline-flex items-center gap-1">
                    <User size={13} aria-hidden="true" />
                    {post.author_name}
                  </span>
                </>
              )}
            </div>

            {/* Compartilhamento Social */}
            <SocialShareBar title={post.title} url={postUrl} />
          </div>

          <h1 className="text-3xl font-extrabold tracking-tight text-foreground sm:text-4xl md:text-5xl leading-tight">
            {post.title}
          </h1>

          {post.summary && (
            <p className="text-lg leading-relaxed text-muted">
              {post.summary}
            </p>
          )}

          {/* Imagem de Capa do Artigo (MinIO ou Fallback Editorial) */}
          <div className="mt-2 overflow-hidden rounded-xl">
            <BlogCoverImage
              coverUrl={post.cover_url}
              title={post.title}
              slug={post.slug}
              aspect="article"
            />
          </div>
        </header>

        {/* Layout do Corpo com ToC Lateral */}
        <div className="grid gap-10 lg:grid-cols-[1fr_240px]">
          <div className="min-w-0">
            {/* Sumário Móvel Colapsável para Leitura em Celulares */}
            {post.body && (
              <div className="mb-6 lg:hidden">
                <TableOfContents source={post.body} mobile />
              </div>
            )}

            {post.body && (
              <div className="prose prose-neutral dark:prose-invert max-w-none text-base leading-relaxed">
                <Markdown source={post.body} />
              </div>
            )}

            {/* Para onde seguir */}
            <div className="mt-12 rounded-xl border border-accent/20 bg-accent/5 p-6 sm:p-8">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div className="flex flex-col gap-1">
                  <span className="text-xs font-bold uppercase tracking-wider text-accent">Projeto-nova-v2</span>
                  <h3 className="text-lg font-bold text-foreground">Precisa de um serviço?</h3>
                  <p className="text-sm text-muted">Veja a carta de serviços publicada no site.</p>
                </div>
                <div className="flex shrink-0 items-center gap-3 pt-2 sm:pt-0">
                  <Link
                    href="/servicos"
                    className="inline-flex items-center justify-center rounded-lg bg-accent px-4 py-2.5 text-sm font-semibold text-accent-foreground shadow-xs transition-colors hover:bg-accent/90"
                  >
                    Carta de serviços
                  </Link>
                </div>
              </div>
            </div>

            {/* Artigos Recomendados */}
            <div className="mt-12">
              <RelatedPosts currentSlug={post.slug} posts={allPosts} />
            </div>
          </div>

          {/* Sumário Flutuante Lateral */}
          <aside className="hidden lg:block">
            <TableOfContents source={post.body || ""} />
          </aside>
        </div>
      </article>
    </PublicShell>
  );
}
