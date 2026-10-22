import { Fragment, type ReactNode } from "react";
import { AlertCircle, AlertTriangle, Info, Lightbulb, ShieldAlert } from "lucide-react";

import { slugifyHeading } from "@/lib/blog/slugify";
import { safeResourceUrl } from "@/lib/security/safe-url";

/** Link permitido no Markdown: caminho interno ("/wiki/x", nunca
 * "//host") ou URL absoluta http/https. javascript:, data: etc. viram
 * texto. */
export function safeMarkdownHref(href: string): string | null {
  if (/^\/(?![\/\\])/.test(href)) return href;
  return safeResourceUrl(href, { allowHttp: true });
}

function inline(text: string, keyBase: string): ReactNode[] {
  const out: ReactNode[] = [];
  // Regex atualizada para incluir imagens ![alt](url) antes de links normais [label](url)
  const re = /(!\[[^\]]*\]\([^)\s]+\)|\*\*[^*]+\*\*|\*[^*]+\*|`[^`]+`|\[[^\]]+\]\([^)\s]+\))/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let i = 0;
  while ((m = re.exec(text))) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const tok = m[0];
    const key = `${keyBase}-${i++}`;

    if (tok.startsWith("![")) {
      const match = /^!\[([^\]]*)\]\(([^)\s]+)\)$/.exec(tok);
      if (match) {
        const alt = match[1] || "";
        const href = match[2] || "";
        const safe = safeMarkdownHref(href);
        if (safe) {
          out.push(
            <span key={key} className="my-4 block text-center">
              {/* Imagem de qualquer origem e tamanho escrita pelo autor: o next/image exigiria liberar domínios e medidas. */}
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={safe}
                alt={alt}
                loading="lazy"
                className="mx-auto max-h-[500px] rounded-xl border border-surface-border object-contain shadow-xs"
              />
              {alt && <span className="mt-1.5 block text-center text-xs text-muted">{alt}</span>}
            </span>,
          );
        } else {
          out.push(<Fragment key={key}>{alt ? `[Imagem: ${alt}]` : ""}</Fragment>);
        }
      }
    } else if (tok.startsWith("**")) {
      out.push(<strong key={key}>{tok.slice(2, -2)}</strong>);
    } else if (tok.startsWith("`")) {
      out.push(
        <code key={key} className="rounded bg-surface-hover px-1 py-0.5 font-mono text-[0.9em]">
          {tok.slice(1, -1)}
        </code>,
      );
    } else if (tok.startsWith("[")) {
      const [, label, href] = /^\[([^\]]+)\]\(([^)\s]+)\)$/.exec(tok) ?? [];
      const safe = href ? safeMarkdownHref(href) : null;
      out.push(
        safe ? (
          <a
            key={key}
            href={safe}
            className="text-primary underline underline-offset-2 transition-colors hover:text-accent"
            rel="noopener noreferrer"
            target={safe.startsWith("/") ? undefined : "_blank"}
          >
            {label}
          </a>
        ) : (
          <Fragment key={key}>{label}</Fragment>
        ),
      );
    } else {
      out.push(<em key={key}>{tok.slice(1, -1)}</em>);
    }
    last = m.index + tok.length;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

const ADMONITION_CONFIG: Record<
  string,
  { label: string; icon: typeof Info; borderCls: string; bgCls: string; textCls: string }
> = {
  NOTE: {
    label: "Nota",
    icon: Info,
    borderCls: "border-sky-500/40",
    bgCls: "bg-sky-500/5",
    textCls: "text-sky-600 dark:text-sky-400",
  },
  INFO: {
    label: "Informação",
    icon: Info,
    borderCls: "border-sky-500/40",
    bgCls: "bg-sky-500/5",
    textCls: "text-sky-600 dark:text-sky-400",
  },
  TIP: {
    label: "Dica",
    icon: Lightbulb,
    borderCls: "border-emerald-500/40",
    bgCls: "bg-emerald-500/5",
    textCls: "text-emerald-600 dark:text-emerald-400",
  },
  IMPORTANT: {
    label: "Importante",
    icon: AlertCircle,
    borderCls: "border-indigo-500/40",
    bgCls: "bg-indigo-500/5",
    textCls: "text-indigo-600 dark:text-indigo-400",
  },
  WARNING: {
    label: "Atenção",
    icon: AlertTriangle,
    borderCls: "border-amber-500/40",
    bgCls: "bg-amber-500/5",
    textCls: "text-amber-600 dark:text-amber-400",
  },
  CAUTION: {
    label: "Cuidado",
    icon: ShieldAlert,
    borderCls: "border-rose-500/40",
    bgCls: "bg-rose-500/5",
    textCls: "text-rose-600 dark:text-rose-400",
  },
};

export function Markdown({ source, className = "" }: { source: string; className?: string }) {
  const lines = source.replace(/\r\n/g, "\n").split("\n");
  const blocks: ReactNode[] = [];
  let i = 0;
  let k = 0;

  while (i < lines.length) {
    const line = lines[i] ?? "";

    if (line.startsWith("```")) {
      const code: string[] = [];
      i++;
      while (i < lines.length && !(lines[i] ?? "").startsWith("```")) {
        code.push(lines[i++] ?? "");
      }
      i++;
      blocks.push(
        <pre
          key={k++}
          className="max-w-full overflow-x-auto rounded-lg bg-surface-hover p-3.5 sm:p-4 font-mono text-xs sm:text-sm leading-relaxed"
        >
          <code>{code.join("\n")}</code>
        </pre>,
      );
      continue;
    }

    // Tabela Markdown GFM (| col1 | col2 | seguido de | --- | --- |)
    if (
      /^\s*\|(.+)\|\s*$/.test(line) &&
      i + 1 < lines.length &&
      /^\s*\|?\s*:?-+:?\s*(\|?\s*:?-+:?\s*)*\|?\s*$/.test(lines[i + 1] ?? "")
    ) {
      const parseCells = (rowText: string): string[] => {
        const trimmed = rowText.trim();
        const withoutEdges = trimmed.replace(/^\|/, "").replace(/\|$/, "");
        return withoutEdges.split("|").map((c) => c.trim());
      };

      const headerCells = parseCells(line);
      const sepCells = parseCells(lines[i + 1] ?? "");
      const alignments = sepCells.map((sep) => {
        const left = sep.startsWith(":");
        const right = sep.endsWith(":");
        if (left && right) return "text-center";
        if (right) return "text-right";
        return "text-left";
      });

      i += 2; // pula cabeçalho e separador
      const rows: string[][] = [];
      while (i < lines.length && /^\s*\|(.+)\|\s*$/.test(lines[i] ?? "")) {
        rows.push(parseCells(lines[i++] ?? ""));
      }

      blocks.push(
        <div
          key={k++}
          className="my-6 w-full max-w-full overflow-x-auto rounded-xl border border-surface-border bg-surface shadow-xs [touch-action:auto] [webkit-overflow-scrolling:touch]"
        >
          <table className="w-full min-w-[560px] divide-y divide-surface-border text-left text-sm">
            <thead className="bg-surface-2 text-xs font-bold uppercase tracking-wider text-muted">
              <tr>
                {headerCells.map((h, hIdx) => (
                  <th
                    key={hIdx}
                    className={`whitespace-nowrap px-4 py-3 font-semibold ${alignments[hIdx] ?? "text-left"}`}
                  >
                    {inline(h, `th${k}-${hIdx}`)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-border">
              {rows.map((row, rIdx) => (
                <tr
                  key={rIdx}
                  className="transition-colors hover:bg-surface-hover/50 even:bg-surface-1/40"
                >
                  {row.map((cell, cIdx) => (
                    <td
                      key={cIdx}
                      className={`px-4 py-3 text-sm text-foreground align-top ${alignments[cIdx] ?? "text-left"}`}
                    >
                      {inline(cell, `td${k}-${rIdx}-${cIdx}`)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>,
      );
      continue;
    }

    const h = /^(#{1,4})\s+(.*)$/.exec(line);
    if (h) {
      const level = h[1]!.length;
      const rawTitle = h[2]!;
      const headingId = slugifyHeading(rawTitle);
      const cls = ["", "text-2xl", "text-xl", "text-lg", "text-base"][level];
      const content = inline(rawTitle, `h${k}`);
      const Tag = `h${Math.min(level + 1, 5)}` as "h2" | "h3" | "h4" | "h5";

      blocks.push(
        <Tag
          key={k++}
          id={headingId}
          className={`group relative mt-8 font-semibold tracking-tight scroll-mt-24 ${cls}`}
        >
          {content}
          <a
            href={`#${headingId}`}
            aria-hidden="true"
            tabIndex={-1}
            className="ml-2 inline-block text-muted opacity-0 transition-opacity group-hover:opacity-100 hover:text-accent"
          >
            #
          </a>
        </Tag>,
      );
      i++;
      continue;
    }

    if (/^\s*([-*]|\d+\.)\s+/.test(line)) {
      const ordered = /^\s*\d+\./.test(line);
      const items: string[] = [];
      while (i < lines.length && /^\s*([-*]|\d+\.)\s+/.test(lines[i] ?? "")) {
        items.push((lines[i++] ?? "").replace(/^\s*([-*]|\d+\.)\s+/, ""));
      }
      const List = ordered ? "ol" : "ul";
      blocks.push(
        <List
          key={k++}
          className={`my-3 ml-6 flex flex-col gap-1.5 ${ordered ? "list-decimal" : "list-disc"}`}
        >
          {items.map((it, j) => (
            <li key={j} className="leading-relaxed">
              {inline(it, `l${k}-${j}`)}
            </li>
          ))}
        </List>,
      );
      continue;
    }

    if (line.startsWith(">")) {
      const quote: string[] = [];
      while (i < lines.length && (lines[i] ?? "").startsWith(">")) {
        quote.push((lines[i++] ?? "").replace(/^>\s?/, ""));
      }

      // Detecta Admonition [!NOTE], [!TIP], etc.
      const firstLine = quote[0] ?? "";
      const admonitionMatch = /^\[!([A-Z]+)\]\s*(.*)$/.exec(firstLine);

      if (admonitionMatch) {
        const type = admonitionMatch[1] ?? "";
        const cfg = ADMONITION_CONFIG[type] ?? {
          label: "Nota",
          icon: Info,
          borderCls: "border-sky-500/40",
          bgCls: "bg-sky-500/5",
          textCls: "text-sky-600 dark:text-sky-400",
        };
        const Icon = cfg.icon;
        const restOfFirst = admonitionMatch[2];
        const bodyLines = restOfFirst ? [restOfFirst, ...quote.slice(1)] : quote.slice(1);

        blocks.push(
          <aside
            key={k++}
            role="note"
            className={`my-4 flex flex-col gap-2 rounded-xl border p-4 ${cfg.borderCls} ${cfg.bgCls}`}
          >
            <div
              className={`flex items-center gap-2 text-xs font-bold uppercase tracking-wider ${cfg.textCls}`}
            >
              <Icon size={16} aria-hidden="true" />
              <span>{cfg.label}</span>
            </div>
            <div className="text-sm leading-relaxed text-foreground">
              {inline(bodyLines.join(" "), `adm${k}`)}
            </div>
          </aside>,
        );
      } else {
        blocks.push(
          <blockquote
            key={k++}
            className="my-4 border-l-4 border-primary/40 pl-4 italic text-muted"
          >
            {inline(quote.join(" "), `q${k}`)}
          </blockquote>,
        );
      }
      continue;
    }

    if (line.trim() === "") {
      i++;
      continue;
    }

    const para: string[] = [];
    while (
      i < lines.length &&
      (lines[i] ?? "").trim() !== "" &&
      !/^(#{1,4}\s|```|>|\s*([-*]|\d+\.)\s|\|)/.test(lines[i] ?? "")
    ) {
      para.push(lines[i++] ?? "");
    }
    blocks.push(
      <p key={k++} className="my-3 text-base leading-relaxed text-foreground">
        {inline(para.join(" "), `p${k}`)}
      </p>,
    );
  }

  return <div className={`text-foreground ${className}`}>{blocks}</div>;
}
