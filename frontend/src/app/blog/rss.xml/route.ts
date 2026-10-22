import { NextResponse } from "next/server";

import { listPublicPosts } from "@/lib/blog/publicBlog";
import { APP_URL } from "@/lib/env";

export const dynamic = "force-dynamic";

export async function GET() {
  const posts = await listPublicPosts("", "", 1, 100).catch(() => []);

  const itemsXml = posts
    .map((post) => {
      const pubDate = new Date(post.published_at || post.created_at).toUTCString();
      const link = `${APP_URL}/blog/${post.slug}`;
      const description = post.summary ? `<![CDATA[${post.summary}]]>` : "";

      return `
    <item>
      <title><![CDATA[${post.title}]]></title>
      <link>${link}</link>
      <guid isPermaLink="true">${link}</guid>
      <pubDate>${pubDate}</pubDate>
      <description>${description}</description>
    </item>`;
    })
    .join("\n");

  const rssFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
  <channel>
    <title>Blog &amp; Novidades — Projeto-nova-v2</title>
    <link>${APP_URL}/blog</link>
    <description>Notícias e comunicados do Projeto-nova-v2.</description>
    <language>pt-BR</language>
    <atom:link href="${APP_URL}/blog/rss.xml" rel="self" type="application/rss+xml"/>
${itemsXml}
  </channel>
</rss>`;

  return new NextResponse(rssFeed, {
    headers: {
      "Content-Type": "application/xml; charset=utf-8",
      "Cache-Control": "public, s-maxage=3600, stale-while-revalidate=86400",
    },
  });
}
