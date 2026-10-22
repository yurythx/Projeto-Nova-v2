import type { MetadataRoute } from "next";

import { listPublicPosts } from "@/lib/blog/publicBlog";
import { listPublishedServices } from "@/lib/catalog/catalog";
import { APP_URL } from "@/lib/env";
import { getFeatures } from "@/lib/features/getFeatures";

// Só páginas públicas. As seções de plug-ins entram apenas quando o
// módulo está ativo no Kernel (GET /system/public-modules) — um módulo
// desativado responde 404 e não deve ser anunciado a crawlers.
const MODULE_PAGES: { path: string; module: string }[] = [
  { path: "/blog", module: "blog" },
  { path: "/eventos", module: "calendar" },
  { path: "/servicos", module: "catalog" },
  { path: "/diretorio", module: "directory" },
  { path: "/contato", module: "contact" },
];

export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const { features } = await getFeatures();
  const now = new Date();

  const blogPostEntries: MetadataRoute.Sitemap = [];
  if (features.blog) {
    try {
      const posts = await listPublicPosts("", "", 1, 100);
      for (const p of posts) {
        blogPostEntries.push({
          url: `${APP_URL}/blog/${p.slug}`,
          lastModified: p.published_at ? new Date(p.published_at) : now,
          changeFrequency: "weekly",
          priority: 0.7,
        });
      }
    } catch {
      // Silencia erros de fetch em build/offline
    }
  }

  const catalogEntries: MetadataRoute.Sitemap = [];
  if (features.catalog) {
    try {
      const services = await listPublishedServices();
      for (const svc of services) {
        catalogEntries.push({
          url: `${APP_URL}/servicos/${svc.slug}`,
          lastModified: now,
          changeFrequency: "weekly",
          priority: 0.8,
        });
      }
    } catch {
      // Silencia erros em build/offline
    }
  }

  return [
    { url: APP_URL, lastModified: now, changeFrequency: "daily", priority: 1.0 },
    { url: `${APP_URL}/sobre`, lastModified: now, changeFrequency: "monthly", priority: 0.8 },
    { url: `${APP_URL}/faq`, lastModified: now, changeFrequency: "monthly", priority: 0.8 },
    ...MODULE_PAGES.filter((p) => features[p.module]).map((p) => ({
      url: `${APP_URL}${p.path}`,
      lastModified: now,
      changeFrequency: "weekly" as const,
      priority: 0.8,
    })),
    ...catalogEntries,
    ...blogPostEntries,
    { url: `${APP_URL}/transparencia`, lastModified: now, changeFrequency: "daily", priority: 0.6 },
    {
      url: `${APP_URL}/acessibilidade`,
      lastModified: now,
      changeFrequency: "yearly",
      priority: 0.5,
    },
    { url: `${APP_URL}/privacidade`, lastModified: now, changeFrequency: "yearly", priority: 0.5 },
  ];
}

