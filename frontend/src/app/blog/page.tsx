import type { Metadata } from "next";

import { PublicShell } from "@/components/layout/PublicShell";
import { listPublicPosts } from "@/lib/blog/publicBlog";
import { APP_URL } from "@/lib/env";
import { PublicBlogListClient } from "./PublicBlogListClient";

export const dynamic = "force-dynamic";

const description =
  "Notícias e comunicados do Projeto Nova. O blog nasce desligado e só aparece quando a instalação o liga.";

export const metadata: Metadata = {
  title: "Blog & Novidades",
  description,
  alternates: { canonical: `${APP_URL}/blog` },
  openGraph: {
    title: "Blog & Novidades",
    description,
    type: "website",
    url: `${APP_URL}/blog`,
  },
};

export default async function BlogPublicPage() {
  const posts = await listPublicPosts("", "", 1, 100).catch(() => []);

  return (
    <PublicShell>
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-10 px-6 py-12">
        <div className="flex flex-col gap-3">
          <p className="dateline">Blog & Novidades</p>
          <h1 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Blog & Novidades
          </h1>
          <p className="max-w-3xl text-base text-muted">{description}</p>
        </div>

        <PublicBlogListClient initialPosts={posts} />
      </div>
    </PublicShell>
  );
}
