import "server-only";

import { PublicApiError, publicGet } from "@/lib/api/publicServer";
import type { Post } from "@/lib/plataforma/types";

export async function listPublicPosts(
  query = "",
  kind = "",
  page = 1,
  pageSize = 50,
): Promise<Post[]> {
  const params = new URLSearchParams();
  if (query) params.set("q", query);
  if (kind) params.set("kind", kind);
  params.set("page", String(page));
  params.set("page_size", String(pageSize));

  try {
    const res = await publicGet<{ items?: Post[] } | Post[]>(`blog/posts?${params.toString()}`, 0);
    if (Array.isArray(res.data)) return res.data;
    if (res.data && Array.isArray(res.data.items)) return res.data.items;
    return [];
  } catch (err) {
    if (err instanceof PublicApiError && err.status === 404) return [];
    return [];
  }
}

export async function getPublicPost(ref: string): Promise<Post | null> {
  try {
    const res = await publicGet<Post>(`blog/posts/${encodeURIComponent(ref)}`, 0);
    return res.data;
  } catch (err) {
    if (err instanceof PublicApiError && err.status === 404) return null;
    return null;
  }
}
