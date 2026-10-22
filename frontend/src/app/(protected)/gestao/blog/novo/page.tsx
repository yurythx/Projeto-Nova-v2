"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";

import { PostEditor } from "@/components/blog/PostEditor";
import { PageHeader } from "@/components/sismarq/PageHeader";
import { Card } from "@/components/ui/Card";

export default function NovoPostPage() {
  const router = useRouter();

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-6">
      <Link
        href="/gestao/blog"
        className="inline-flex items-center gap-1.5 text-sm text-muted transition-colors hover:text-foreground"
      >
        <ArrowLeft size={16} aria-hidden="true" /> Voltar para o painel de publicações
      </Link>

      <PageHeader
        eyebrow="Comunicação & Redação"
        title="Nova publicação"
        description="Escreva uma nova notícia ou comunicado interno utilizando o editor rico TipTap, upload direto de imagens no MinIO e pré-visualização."
      />

      <Card className="p-6">
        <PostEditor onSaved={(p) => router.push(`/gestao/blog/${p.slug}`)} />
      </Card>
    </div>
  );
}
