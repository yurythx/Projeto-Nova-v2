"use client";

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";

import { PostEditor } from "@/components/blog/PostEditor";
import { DataState } from "@/components/sismarq/DataState";
import { PageHeader } from "@/components/sismarq/PageHeader";
import { Card } from "@/components/ui/Card";
import { useApiQuery } from "@/lib/api/swr";
import type { Post } from "@/lib/sismarq/types";

export default function EditarPostPage() {
  const { ref } = useParams<{ ref: string }>();
  const router = useRouter();
  const post = useApiQuery<Post>(`v1/blog/admin/posts/${encodeURIComponent(ref)}`);
  const p = post.data;

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
      <Link
        href={`/gestao/blog/${ref}`}
        className="inline-flex items-center gap-1.5 text-sm text-muted transition-colors hover:text-foreground"
      >
        <ArrowLeft size={16} aria-hidden="true" /> Voltar para os detalhes da publicação
      </Link>

      <PageHeader
        eyebrow="Edição de conteúdo"
        title={`Editar: ${p?.title || "Publicação"}`}
        description="Ajuste o texto, imagens, metadados ou público-alvo com o editor visual TipTap."
      />

      <DataState loading={post.isLoading} error={post.error} empty={!p}>
        {p && (
          <Card className="p-6">
            <PostEditor
              post={p}
              onSaved={(np) => {
                if (np.slug !== ref) router.replace(`/gestao/blog/${np.slug}`);
                else router.push(`/gestao/blog/${ref}`);
              }}
            />
          </Card>
        )}
      </DataState>
    </div>
  );
}
