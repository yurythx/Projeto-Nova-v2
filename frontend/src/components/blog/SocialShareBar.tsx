"use client";

import { useState } from "react";
import { Check, Copy, MessageCircle, Share2 } from "lucide-react";

interface SocialShareBarProps {
  title: string;
  url: string;
}

export function SocialShareBar({ title, url }: SocialShareBarProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      setTimeout(() => setCopied(false), 2500);
    } catch {
      // Fallback silencioso
    }
  };

  const encodedUrl = encodeURIComponent(url);
  const encodedTitle = encodeURIComponent(title);

  const shareWhatsapp = `https://api.whatsapp.com/send?text=${encodedTitle}%20${encodedUrl}`;
  const shareLinkedin = `https://www.linkedin.com/sharing/share-offsite/?url=${encodedUrl}`;
  const shareTwitter = `https://twitter.com/intent/tweet?text=${encodedTitle}&url=${encodedUrl}`;

  return (
    <div
      role="region"
      aria-label="Compartilhar publicação"
      className="flex flex-wrap items-center gap-2"
    >
      <span className="text-xs font-semibold text-muted">Compartilhar:</span>

      {/* WhatsApp */}
      <a
        href={shareWhatsapp}
        target="_blank"
        rel="noopener noreferrer"
        aria-label="Compartilhar no WhatsApp"
        className="inline-flex items-center gap-1.5 rounded-lg border border-surface-border bg-surface px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-emerald-500/10 hover:text-emerald-600 dark:hover:text-emerald-400"
      >
        <MessageCircle size={14} aria-hidden="true" />
        <span>WhatsApp</span>
      </a>

      {/* LinkedIn */}
      <a
        href={shareLinkedin}
        target="_blank"
        rel="noopener noreferrer"
        aria-label="Compartilhar no LinkedIn"
        className="inline-flex items-center gap-1.5 rounded-lg border border-surface-border bg-surface px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-blue-500/10 hover:text-blue-600 dark:hover:text-blue-400"
      >
        <Share2 size={14} aria-hidden="true" />
        <span>LinkedIn</span>
      </a>

      {/* X / Twitter */}
      <a
        href={shareTwitter}
        target="_blank"
        rel="noopener noreferrer"
        aria-label="Compartilhar no X"
        className="inline-flex items-center gap-1.5 rounded-lg border border-surface-border bg-surface px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-surface-2"
      >
        <span className="text-xs font-bold leading-none">𝕏</span>
        <span>Post</span>
      </a>

      {/* Copiar Link */}
      <button
        type="button"
        onClick={handleCopy}
        aria-label="Copiar link do artigo para a área de transferência"
        className="inline-flex items-center gap-1.5 rounded-lg border border-surface-border bg-surface px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-surface-2"
      >
        {copied ? (
          <>
            <Check size={14} className="text-emerald-500" aria-hidden="true" />
            <span className="text-emerald-600 dark:text-emerald-400">Copiado!</span>
          </>
        ) : (
          <>
            <Copy size={14} aria-hidden="true" />
            <span>Copiar Link</span>
          </>
        )}
      </button>
    </div>
  );
}
