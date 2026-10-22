"use client";

import { useEffect, useRef, useState } from "react";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import Image from "@tiptap/extension-image";
import Link from "@tiptap/extension-link";
import Placeholder from "@tiptap/extension-placeholder";
import { Markdown } from "tiptap-markdown";
import {
  Bold,
  Code,
  FileCode,
  Heading2,
  Heading3,
  ImageIcon,
  Italic,
  Link as LinkIcon,
  List,
  ListOrdered,
  Quote,
  Strikethrough,
  Undo2,
  Redo2,
  CodeXml,
  Loader2,
} from "lucide-react";

import { apiClient } from "@/lib/api/client";
import type { UploadTicket } from "@/lib/plataforma/types";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";

interface TipTapEditorProps {
  value: string;
  onChange: (markdown: string) => void;
  placeholder?: string;
}

export function TipTapEditor({ value, onChange, placeholder = "Escreva seu artigo aqui..." }: TipTapEditorProps) {
  const [mode, setMode] = useState<"visual" | "markdown">("visual");
  const [uploading, setUploading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: {
          levels: [2, 3],
        },
        link: false,
      }),
      Image.configure({
        allowBase64: false,
        HTMLAttributes: {
          class: "rounded-xl border border-surface-border max-h-[500px] my-4 mx-auto object-contain",
        },
      }),
      Link.configure({
        openOnClick: false,
        HTMLAttributes: {
          class: "text-primary underline underline-offset-2 hover:text-accent",
        },
      }),
      Placeholder.configure({
        placeholder,
      }),
      Markdown.configure({
        html: false,
        transformCopiedText: true,
        transformPastedText: true,
      }),
    ],
    content: value,
    editorProps: {
      attributes: {
        id: "post-body",
        "aria-label": "Conteúdo em Markdown",
        class:
          "prose prose-neutral dark:prose-invert max-w-none focus:outline-hidden min-h-[360px] p-4 text-foreground text-base leading-relaxed",
      },
    },
    onUpdate: ({ editor }) => {
      // @ts-expect-error tiptap-markdown extension adds storage.markdown
      const md = editor.storage.markdown?.getMarkdown?.() ?? editor.getHTML();
      onChange(md);
    },
    immediatelyRender: false,
  });

  // Mantém o editor sincronizado se o valor inicial mudar externamente
  useEffect(() => {
    if (!editor) return;
    // @ts-expect-error tiptap-markdown extension adds storage.markdown
    const currentMd = editor.storage.markdown?.getMarkdown?.() ?? "";
    if (value !== currentMd && !editor.isFocused) {
      editor.commands.setContent(value);
    }
  }, [value, editor]);

  async function handleImageUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file || !editor) return;

    setUploading(true);
    try {
      const { data: ticket } = await apiClient.post<UploadTicket>("v1/blog/uploads", {
        filename: file.name,
        content_type: mimeOf(file),
      });

      await putToTicket(ticket, file);

      // A URL de visualização é a URL base da pré-assinada sem os parâmetros temporários de upload
      const cleanUrl = ticket.upload_url.split("?")[0] || "";
      if (cleanUrl) {
        editor.chain().focus().setImage({ src: cleanUrl, alt: file.name }).run();
      }
    } catch (err) {
      console.error("Falha ao subir imagem para o MinIO:", err);
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  function handleSetLink() {
    if (!editor) return;
    const prev = editor.getAttributes("link").href;
    const url = window.prompt("URL do link:", prev || "https://");
    if (url === null) return;
    if (url === "") {
      editor.chain().focus().unsetLink().run();
      return;
    }
    editor.chain().focus().setLink({ href: url }).run();
  }

  return (
    <div className="flex flex-col rounded-xl border border-surface-border bg-surface shadow-xs">
      {/* Barra de Ferramentas */}
      <div className="flex flex-wrap items-center justify-between gap-1 border-b border-surface-border bg-surface-1/70 p-2 backdrop-blur-xs">
        <div className="flex flex-wrap items-center gap-1">
          {mode === "visual" && editor && (
            <>
              {/* Títulos */}
              <button
                type="button"
                onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
                className={`rounded-md p-1.5 text-xs font-semibold transition-colors ${
                  editor.isActive("heading", { level: 2 })
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Título H2"
                aria-label="Título H2"
              >
                <Heading2 size={16} />
              </button>

              <button
                type="button"
                onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
                className={`rounded-md p-1.5 text-xs font-semibold transition-colors ${
                  editor.isActive("heading", { level: 3 })
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Subtítulo H3"
                aria-label="Subtítulo H3"
              >
                <Heading3 size={16} />
              </button>

              <span className="mx-1 h-4 w-px bg-surface-border" />

              {/* Formatação Básica */}
              <button
                type="button"
                onClick={() => editor.chain().focus().toggleBold().run()}
                className={`rounded-md p-1.5 transition-colors ${
                  editor.isActive("bold")
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Negrito (Ctrl+B)"
                aria-label="Negrito"
              >
                <Bold size={16} />
              </button>

              <button
                type="button"
                onClick={() => editor.chain().focus().toggleItalic().run()}
                className={`rounded-md p-1.5 transition-colors ${
                  editor.isActive("italic")
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Itálico (Ctrl+I)"
                aria-label="Itálico"
              >
                <Italic size={16} />
              </button>

              <button
                type="button"
                onClick={() => editor.chain().focus().toggleStrike().run()}
                className={`rounded-md p-1.5 transition-colors ${
                  editor.isActive("strike")
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Riscado"
                aria-label="Riscado"
              >
                <Strikethrough size={16} />
              </button>

              <button
                type="button"
                onClick={() => editor.chain().focus().toggleCode().run()}
                className={`rounded-md p-1.5 transition-colors ${
                  editor.isActive("code")
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Código em linha"
                aria-label="Código em linha"
              >
                <Code size={16} />
              </button>

              <span className="mx-1 h-4 w-px bg-surface-border" />

              {/* Listas e Citações */}
              <button
                type="button"
                onClick={() => editor.chain().focus().toggleBulletList().run()}
                className={`rounded-md p-1.5 transition-colors ${
                  editor.isActive("bulletList")
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Lista com marcadores"
                aria-label="Lista com marcadores"
              >
                <List size={16} />
              </button>

              <button
                type="button"
                onClick={() => editor.chain().focus().toggleOrderedList().run()}
                className={`rounded-md p-1.5 transition-colors ${
                  editor.isActive("orderedList")
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Lista numerada"
                aria-label="Lista numerada"
              >
                <ListOrdered size={16} />
              </button>

              <button
                type="button"
                onClick={() => editor.chain().focus().toggleBlockquote().run()}
                className={`rounded-md p-1.5 transition-colors ${
                  editor.isActive("blockquote")
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Citação"
                aria-label="Citação"
              >
                <Quote size={16} />
              </button>

              <button
                type="button"
                onClick={() => editor.chain().focus().toggleCodeBlock().run()}
                className={`rounded-md p-1.5 transition-colors ${
                  editor.isActive("codeBlock")
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Bloco de código"
                aria-label="Bloco de código"
              >
                <FileCode size={16} />
              </button>

              <span className="mx-1 h-4 w-px bg-surface-border" />

              {/* Links e Imagens */}
              <button
                type="button"
                onClick={handleSetLink}
                className={`rounded-md p-1.5 transition-colors ${
                  editor.isActive("link")
                    ? "bg-accent/15 text-accent"
                    : "text-muted hover:bg-surface-2 hover:text-foreground"
                }`}
                title="Inserir Link"
                aria-label="Inserir Link"
              >
                <LinkIcon size={16} />
              </button>

              {/* Upload de Imagem MinIO */}
              <input
                ref={fileInputRef}
                type="file"
                accept="image/png,image/jpeg,image/webp,image/gif"
                onChange={handleImageUpload}
                className="hidden"
                id="tiptap-image-upload"
              />
              <button
                type="button"
                disabled={uploading}
                onClick={() => fileInputRef.current?.click()}
                className="inline-flex items-center gap-1 rounded-md p-1.5 text-muted transition-colors hover:bg-surface-2 hover:text-foreground disabled:opacity-50"
                title="Inserir imagem no corpo (MinIO)"
                aria-label="Inserir imagem no corpo (MinIO)"
              >
                {uploading ? (
                  <Loader2 size={16} className="animate-spin text-accent" />
                ) : (
                  <ImageIcon size={16} />
                )}
              </button>

              <span className="mx-1 h-4 w-px bg-surface-border" />

              {/* Histórico */}
              <button
                type="button"
                onClick={() => editor.chain().focus().undo().run()}
                disabled={!editor.can().undo()}
                className="rounded-md p-1.5 text-muted transition-colors hover:bg-surface-2 hover:text-foreground disabled:opacity-40"
                title="Desfazer (Ctrl+Z)"
                aria-label="Desfazer"
              >
                <Undo2 size={16} />
              </button>

              <button
                type="button"
                onClick={() => editor.chain().focus().redo().run()}
                disabled={!editor.can().redo()}
                className="rounded-md p-1.5 text-muted transition-colors hover:bg-surface-2 hover:text-foreground disabled:opacity-40"
                title="Refazer (Ctrl+Y)"
                aria-label="Refazer"
              >
                <Redo2 size={16} />
              </button>
            </>
          )}
        </div>

        {/* Alternância Modo Visual / Código Markdown */}
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => setMode(mode === "visual" ? "markdown" : "visual")}
            className="inline-flex items-center gap-1.5 rounded-lg border border-surface-border bg-surface px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-surface-2"
          >
            <CodeXml size={14} aria-hidden="true" />
            <span>{mode === "visual" ? "Ver Markdown" : "Modo Visual"}</span>
          </button>
        </div>
      </div>

      {/* Área de Edição */}
      {mode === "visual" ? (
        <div className="relative">
          <EditorContent editor={editor} />
        </div>
      ) : (
        <textarea
          rows={16}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          aria-label="Código Markdown puro"
          className="w-full bg-surface-1 p-4 font-mono text-sm leading-relaxed text-foreground focus:outline-hidden"
          placeholder="# Digite em Markdown puro..."
        />
      )}
    </div>
  );
}
