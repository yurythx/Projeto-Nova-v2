"use client";

import { useState, useEffect, type FormEvent } from "react";

import { PublicoAlvoPicker, TODOS } from "@/components/iam/PublicoAlvo";
import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { Markdown } from "@/components/plataforma/Markdown";
import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { TipTapEditor } from "@/components/blog/TipTapEditor";
import { apiClient } from "@/lib/api/client";
import type { Post, UploadTicket } from "@/lib/plataforma/types";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";

/** Editor de publicação (rascunho): corpo em Markdown com pré-visualização
 * segura e capa enviada direto ao MinIO por URL pré-assinada. */
export function PostEditor({ post, onSaved }: { post?: Post; onSaved: (p: Post) => void }) {
  const { run, pending } = useAction();
  const [body, setBody] = useState(post?.body ?? "");
  const [preview, setPreview] = useState(false);
  const [coverKey, setCoverKey] = useState(post?.cover_object_key ?? "");
  const [uploading, setUploading] = useState(false);
  const [publico, setPublico] = useState(post?.publico ?? TODOS);

  const storageKey = post ? `nova_blog_draft_${post.id}` : "nova_blog_draft_new";
  const [hasDraft, setHasDraft] = useState(() => {
    if (typeof window === "undefined") return false;
    try {
      const raw = localStorage.getItem(storageKey);
      if (raw) {
        const parsed = JSON.parse(raw);
        return Boolean(parsed.body && parsed.body !== (post?.body ?? ""));
      }
    } catch {
      return false;
    }
    return false;
  });
  const [lastSaved, setLastSaved] = useState<string | null>(null);

  useEffect(() => {
    if (!body || body === (post?.body ?? "")) return;
    const timer = setTimeout(() => {
      try {
        localStorage.setItem(storageKey, JSON.stringify({ body, savedAt: Date.now() }));
        const now = new Date();
        const timeStr = `${String(now.getHours()).padStart(2, "0")}:${String(now.getMinutes()).padStart(2, "0")}:${String(now.getSeconds()).padStart(2, "0")}`;
        setLastSaved(timeStr);
      } catch {
        // Ignora erros de cota do localStorage
      }
    }, 1000);
    return () => clearTimeout(timer);
  }, [body, storageKey, post?.body]);

  function restoreDraft() {
    try {
      const raw = localStorage.getItem(storageKey);
      if (raw) {
        const parsed = JSON.parse(raw);
        if (parsed.body) {
          setBody(parsed.body);
        }
      }
    } catch {
      // noop
    }
    setHasDraft(false);
  }

  function discardDraft() {
    try {
      localStorage.removeItem(storageKey);
    } catch {
      // noop
    }
    setHasDraft(false);
    setLastSaved(null);
  }

  async function uploadCover(file: File) {
    setUploading(true);
    await run(async () => {
      const { data: ticket } = await apiClient.post<UploadTicket>("v1/blog/uploads", { filename: file.name, content_type: mimeOf(file) });
      await putToTicket(ticket, file);
      setCoverKey(ticket.object_key);
    }, "Capa enviada");
    setUploading(false);
  }

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const payload = {
      title: String(fd.get("title") ?? "").trim(),
      slug: String(fd.get("slug") ?? "").trim(),
      summary: String(fd.get("summary") ?? "").trim(),
      kind: String(fd.get("kind") ?? "noticia"),
      pinned: fd.get("pinned") === "on",
      cover_object_key: coverKey,
      unidade_id: String(fd.get("unidade_id") ?? "") || null,
      publico,
      body,
    };
    const res = await run(
      () => (post ? apiClient.put<Post>(`v1/blog/posts/${post.id}`, payload) : apiClient.post<Post>("v1/blog/posts", payload)),
      "Publicação salva",
    );
    if (res) {
      try {
        localStorage.removeItem(storageKey);
      } catch {
        // noop
      }
      onSaved(res.data);
    }
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-3">
      {hasDraft && (
        <div className="flex items-center justify-between rounded-lg border border-primary/20 bg-primary/5 px-4 py-2 text-sm text-foreground">
          <span>Existe um rascunho anterior salvo localmente no seu navegador.</span>
          <div className="flex items-center gap-2">
            <Button type="button" size="sm" variant="secondary" onClick={restoreDraft}>
              Restaurar
            </Button>
            <Button type="button" size="sm" variant="ghost" onClick={discardDraft}>
              Descartar
            </Button>
          </div>
        </div>
      )}

      <Input id="post-title" name="title" label="Título *" required maxLength={200} defaultValue={post?.title} />
      <div className="grid gap-3 sm:grid-cols-3">
        <Input id="post-slug" name="slug" label="Slug" maxLength={100} defaultValue={post?.slug} placeholder="gerado do título" />
        <Select
          id="post-kind"
          name="kind"
          label="Tipo"
          defaultValue={post?.kind ?? "noticia"}
          options={[
            { value: "noticia", label: "Notícia" },
            { value: "comunicado", label: "Comunicado interno" },
          ]}
        />
        <label className="flex items-center gap-2 self-end pb-2 text-sm">
          <input type="checkbox" name="pinned" defaultChecked={post?.pinned} className="h-4 w-4 accent-primary" /> Fixar no topo
        </label>
      </div>
      <UnidadeDonaSelect id="post-unidade" permission="blog:manage" defaultValue={post?.unidade_id} />
      <PublicoAlvoPicker idPrefix="post-publico" value={publico} onChange={setPublico} />
      <Textarea id="post-summary" name="summary" label="Resumo" rows={2} maxLength={500} defaultValue={post?.summary} />
      <div className="flex flex-col gap-1">
        <label htmlFor="post-cover" className="text-sm font-medium">
          Imagem de capa
        </label>
        <input
          id="post-cover"
          type="file"
          accept="image/png,image/jpeg,image/webp"
          disabled={uploading}
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) void uploadCover(f);
          }}
          className="text-sm"
        />
        {coverKey && <span className="text-xs text-muted">Capa: {coverKey.split("/").pop()}</span>}
      </div>
      <div className="flex flex-col gap-1.5">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">Conteúdo (Markdown)</span>
          <Button type="button" variant="ghost" size="sm" onClick={() => setPreview((v) => !v)} aria-pressed={preview}>
            {preview ? "Editar" : "Pré-visualizar"}
          </Button>
        </div>
        {preview ? (
          <div className="min-h-48 rounded-lg border border-surface-border p-6 bg-surface">
            <Markdown source={body || "_(vazio)_"} />
          </div>
        ) : (
          <TipTapEditor value={body} onChange={setBody} />
        )}
      </div>
      <div className="flex items-center justify-end gap-3">
        {lastSaved && (
          <span className="text-xs text-muted">
            ✓ Rascunho salvo localmente às {lastSaved}
          </span>
        )}
        <Button type="submit" loading={pending || uploading}>
          Salvar
        </Button>
      </div>
    </form>
  );
}
