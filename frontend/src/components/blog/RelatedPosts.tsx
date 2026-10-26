import Link from "next/link";
import { ArrowRight, Calendar, Clock } from "lucide-react";

import { BlogCoverImage } from "@/components/blog/BlogCoverImage";
import type { Post } from "@/lib/plataforma/types";

interface RelatedPostsProps {
  currentSlug: string;
  posts: Post[];
}

export function RelatedPosts({ currentSlug, posts }: RelatedPostsProps) {
  // Filtra o post atual e pega até 3 posts relevantes
  const related = posts.filter((p) => p.slug !== currentSlug).slice(0, 3);

  if (related.length === 0) return null;

  const formatDate = (dateStr?: string | null) => {
    if (!dateStr) return "";
    try {
      return new Intl.DateTimeFormat("pt-BR", {
        day: "2-digit",
        month: "short",
        year: "numeric",
      }).format(new Date(dateStr));
    } catch {
      return dateStr;
    }
  };

  return (
    <section aria-labelledby="related-posts" className="flex flex-col gap-6 pt-6">
      <div className="flex items-center justify-between border-b border-surface-border pb-3">
        <h2 id="related-posts" className="text-xl font-bold tracking-tight text-foreground">
          Leia Também
        </h2>
        <Link
          href="/blog"
          className="inline-flex items-center gap-1 text-xs font-semibold text-accent transition-colors hover:text-accent/80"
        >
          <span>Ver todos os artigos</span>
          <ArrowRight size={14} aria-hidden="true" />
        </Link>
      </div>

      <div className="grid gap-6 sm:grid-cols-2 md:grid-cols-3">
        {related.map((post) => {
          const words = (post.summary || "").split(/\s+/).length * 10;
          const readTime = `${Math.max(2, Math.ceil(words / 200))} min de leitura`;

          return (
            <Link
              key={post.id}
              href={`/blog/${post.slug}`}
              className="group flex flex-col overflow-hidden rounded-xl border border-surface-border bg-surface p-4 shadow-xs transition-all duration-200 hover:border-accent/40 hover:shadow-md"
            >
              <div className="mb-3 overflow-hidden rounded-lg">
                <BlogCoverImage
                  coverUrl={post.cover_url}
                  title={post.title}
                  slug={post.slug}
                  aspect="card"
                />
              </div>

              <div className="flex items-center gap-2 text-[11px] text-muted">
                <span className="inline-flex items-center gap-1">
                  <Calendar size={12} aria-hidden="true" />
                  {formatDate(post.published_at || post.updated_at)}
                </span>
                <span>•</span>
                <span className="inline-flex items-center gap-1">
                  <Clock size={12} aria-hidden="true" />
                  {readTime}
                </span>
              </div>

              <h3 className="mt-2 line-clamp-2 text-sm font-bold text-foreground transition-colors group-hover:text-accent">
                {post.title}
              </h3>

              {post.summary && (
                <p className="mt-1.5 line-clamp-2 text-xs text-muted">{post.summary}</p>
              )}
            </Link>
          );
        })}
      </div>
    </section>
  );
}
